package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	zone "github.com/lrstanley/bubblezone"
	"shubhxho/polytui/internal/api"
)

type screen int

const (
	screenSplash screen = iota
	screenBrowse
	screenDetail
)

// model is the root Bubble Tea model.
type model struct {
	client *api.Client
	width  int
	height int

	screen screen

	// ---- splash ----
	splashSpring harmonica.Spring
	splashPos    float64
	splashVel    float64

	// ---- browse ----
	events      []api.Event
	cursor      int
	scroll      int
	loading     bool
	loadErr     error
	sortIdx     int
	offset      int
	hasMore     bool
	loadingMore bool

	activeTab int

	searchMode  bool
	searchInput textinput.Model
	searchTerm  string

	bars      map[string]*springBar
	knownBars map[string]bool

	watch       *watchlist
	watchEvents map[string]api.Event

	// ---- detail ----
	detail       *api.Event
	detailCursor int
	detailBars   []springBar
	book         *api.OrderBook
	bookToken    string
	history      []api.PricePoint
	histToken    string
	histIdx      int // index into historyIntervals
	descExpanded bool

	// renderCache memoises the chart + order book + tab bar so the 60fps frame
	// loop doesn't redraw them every tick (held behind a pointer so View can fill it).
	renderCache *detailCache

	// zone hit-tests on-screen regions (the price chart) for mouse hover.
	zone *zone.Manager

	// chart hover crosshair state, set on mouse motion over the price chart.
	hoverChart bool // pointer currently over the chart
	hoverIdx   int  // nearest history index under the pointer
	hoverCol   int  // chart-local column to draw the crosshair at

	// animRunning is true while a 60fps animTick chain is live. The loop idles
	// itself when every spring has settled and re-arms on the next interaction
	// or data message, so a still screen costs 0fps instead of 60.
	animRunning bool

	// ---- kitty images ----
	imgOK      bool                   // terminal speaks the Kitty graphics protocol
	imgCache   map[string]*kittyImage // built thumbnails, keyed by source URL
	imgLoading map[string]bool        // in-flight image fetches, keyed by URL
	imgSeq     uint32                 // monotonic Kitty image-id counter

	// ---- overlays ----
	showHelp bool

	spinFrame int
	frameNo   int
}

type histInterval struct {
	label    string
	interval string
	fidelity int
}

var historyIntervals = []histInterval{
	{"6H", "6h", 5},
	{"1D", "1d", 15},
	{"1W", "1w", 60},
	{"1M", "1m", 180},
	{"ALL", "max", 720},
}

// New constructs the root model.
func New() model {
	ti := textinput.New()
	ti.Placeholder = "search markets…"
	ti.CharLimit = 80
	ti.Prompt = "  "

	return model{
		client:       api.New(),
		screen:       screenSplash,
		splashSpring: harmonica.NewSpring(harmonica.FPS(animFPS), 5.0, 0.6),
		searchInput:  ti,
		bars:         map[string]*springBar{},
		knownBars:    map[string]bool{},
		watch:        loadWatchlist(),
		watchEvents:  map[string]api.Event{},
		renderCache:  newDetailCache(),
		zone:         zone.New(),
		histIdx:      2, // default 1W
		loading:      true,
		hasMore:      true,
		animRunning:  true, // Init() starts the animTick chain
		imgOK:        kittyEnabled(),
		imgCache:     map[string]*kittyImage{},
		imgLoading:   map[string]bool{},
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		animTick(),
		refreshTick(),
		loadEvents(m.client, m.query(0), false),
	}
	if ids := m.watch.list(); len(ids) > 0 {
		cmds = append(cmds, loadWatchEvents(m.client, ids))
	}
	return tea.Batch(cmds...)
}

// query builds the current events query from the active tab + sort + search.
func (m model) query(offset int) api.EventQuery {
	so := sortOptions[m.sortIdx]
	q := api.EventQuery{
		Limit:  60,
		Offset: offset,
		Order:  so.order,
		Asc:    so.asc,
		Search: m.searchTerm,
	}
	if t := m.currentTab(); !t.watch && t.tagID != "" {
		q.TagID = t.tagID
	}
	return q
}

func (m *model) currentSort() sortOption { return sortOptions[m.sortIdx] }

// Update is the central dispatcher. frameMsg drives the animation loop; every
// other message is routed and then allowed to re-arm that loop, so motion that
// began while the loop was idle (new data, a keypress) gets animated.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(frameMsg); ok {
		return m.tickAnim()
	}
	nm, cmd := m.route(msg)
	if mm, ok := nm.(model); ok {
		if arm := mm.maybeArm(); arm != nil {
			cmd = tea.Batch(cmd, arm)
		}
		if pf := mm.imagePrefetch(); pf != nil {
			cmd = tea.Batch(cmd, pf)
		}
		return mm, cmd
	}
	return nm, cmd
}

// imagePrefetch warms thumbnails ahead of the user: the focused browse event and
// the next couple, so opening the detail screen finds the image already cached.
// imageCmd dedupes against in-flight/cached urls, so this is cheap to call on
// every interaction; the work is bounded by imageSem.
func (m *model) imagePrefetch() tea.Cmd {
	if !m.imgOK {
		return nil
	}
	var cmds []tea.Cmd
	switch m.screen {
	case screenBrowse:
		ev := m.filteredEvents()
		for i := m.cursor; i < m.cursor+3 && i < len(ev); i++ {
			if c := m.imageCmd(&ev[i]); c != nil {
				cmds = append(cmds, c)
			}
		}
	case screenDetail:
		if c := m.imageCmd(m.detail); c != nil {
			cmds = append(cmds, c)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// maybeArm restarts the 60fps frame loop if it isn't already running. Bubble
// Tea serializes Update on one goroutine, so animRunning has no race and only
// tickAnim ever clears it — there can never be two concurrent tick chains.
func (m *model) maybeArm() tea.Cmd {
	if m.animRunning {
		return nil
	}
	m.animRunning = true
	return animTick()
}

// route dispatches a non-frame message to the right handler.
func (m model) route(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, refreshTick())
		if m.screen == screenBrowse && !m.searchMode {
			cmds = append(cmds, loadEvents(m.client, m.query(0), false))
		}
		if m.screen == screenDetail && m.bookToken != "" {
			cmds = append(cmds, loadBook(m.client, m.bookToken))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case eventsMsg:
		return m.onEvents(msg)

	case bookMsg:
		if msg.tokenID == m.bookToken && msg.err == nil {
			m.book = msg.book
		}
		return m, nil

	case historyMsg:
		if msg.tokenID == m.histToken && msg.err == nil {
			m.history = msg.points
		}
		return m, nil

	case imageMsg:
		return m.onImage(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "initializing…"
	}
	switch m.screen {
	case screenSplash:
		return m.viewSplash()
	case screenDetail:
		s := m.withOverlays(m.viewDetail())
		if m.zone != nil {
			s = m.zone.Scan(s) // records the chart zone bounds + strips the markers
		}
		return s
	default:
		return m.withOverlays(m.viewBrowse())
	}
}

func (m model) withOverlays(base string) string {
	if m.showHelp {
		return m.viewHelp()
	}
	return base
}

// tickAnim advances every active spring/animation one frame, then stops the
// 60fps chain once nothing is moving (re-armed by maybeArm on the next message).
func (m model) tickAnim() (tea.Model, tea.Cmd) {
	m.frameNo++
	if m.frameNo%6 == 0 {
		m.spinFrame++
	}

	// moving stays true while anything needs further frames: a spring in
	// flight, the splash reveal, or a spinner with a request still in flight.
	moving := m.loading || m.loadingMore

	if m.screen == screenSplash {
		m.splashPos, m.splashVel = m.splashSpring.Update(m.splashPos, m.splashVel, 1.0)
		if m.splashPos > 0.985 && !m.loading {
			m.screen = screenBrowse
		} else {
			moving = true
		}
	}
	// Keep the detail "loading history/book…" spinner alive during slow loads.
	if m.screen == screenDetail {
		if (m.histToken != "" && len(m.history) == 0) || (m.bookToken != "" && m.book == nil) {
			moving = true
		}
	}

	// Browse bars.
	for _, b := range m.bars {
		if b.update() {
			moving = true
		}
	}
	// Detail bars.
	for i := range m.detailBars {
		if m.detailBars[i].update() {
			moving = true
		}
	}

	if !moving {
		m.animRunning = false
		return m, nil // idle: stop ticking until the next interaction/data
	}
	return m, animTick()
}

// barFor returns (creating if needed) the spring bar for an event id. New bars
// start at zero so they animate up on first appearance.
func (m *model) barFor(id string, target float64) *springBar {
	b, ok := m.bars[id]
	if !ok {
		nb := newSpringBar()
		b = &nb
		m.bars[id] = b
	}
	b.setTarget(target)
	return b
}
