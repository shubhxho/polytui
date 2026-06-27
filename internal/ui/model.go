package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
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
		histIdx:      2, // default 1W
		loading:      true,
		hasMore:      true,
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

// Update is the central dispatcher.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case frameMsg:
		return m.tickAnim()

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
		base := m.viewDetail()
		return m.withOverlays(base)
	default:
		base := m.viewBrowse()
		return m.withOverlays(base)
	}
}

func (m model) withOverlays(base string) string {
	if m.showHelp {
		return m.viewHelp()
	}
	return base
}

// tickAnim advances every active spring/animation one frame.
func (m model) tickAnim() (tea.Model, tea.Cmd) {
	m.frameNo++
	if m.frameNo%6 == 0 {
		m.spinFrame++
	}

	if m.screen == screenSplash {
		m.splashPos, m.splashVel = m.splashSpring.Update(m.splashPos, m.splashVel, 1.0)
		if m.splashPos > 0.985 && !m.loading {
			m.screen = screenBrowse
		}
	}

	// Browse bars.
	for _, b := range m.bars {
		b.update()
	}
	// Detail bars.
	for i := range m.detailBars {
		m.detailBars[i].update()
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
