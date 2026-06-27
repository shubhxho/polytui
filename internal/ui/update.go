package ui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"shubhxho/polytui/internal/api"
)

func (m model) onEvents(msg eventsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.loading = false
		m.loadingMore = false
		m.loadErr = msg.err
		return m, nil
	}
	if msg.watch {
		// Watchlist refresh: keep full event data for starred ids.
		for _, e := range msg.events {
			m.watchEvents[e.ID] = e
			m.barFor(e.ID, e.TopPrice())
		}
		return m, nil
	}
	m.loading = false
	m.loadingMore = false
	m.loadErr = nil
	if msg.append {
		m.events = append(m.events, msg.events...)
	} else {
		m.events = msg.events
		if m.cursor >= len(m.events) {
			m.cursor = 0
		}
		m.scroll = 0
	}
	m.hasMore = len(msg.events) >= 60
	// Register animated bars; new events animate up from zero.
	for _, e := range m.events {
		m.barFor(e.ID, e.TopPrice())
		m.knownBars[e.ID] = true
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search input mode swallows most keys.
	if m.searchMode {
		return m.handleSearchKey(msg)
	}
	// Help overlay: any key closes.
	if m.showHelp {
		switch msg.String() {
		case "?", "esc", "q", "enter", " ":
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "Q":
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	}

	if m.screen == screenDetail {
		return m.handleDetailKey(msg)
	}
	return m.handleBrowseKey(msg)
}

func (m model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.filteredEvents()
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(visible)-1 {
			m.cursor++
		}
		return m.maybeLoadMore(visible)
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "g", "home":
		m.cursor = 0
		m.scroll = 0
		return m, nil
	case "G", "end":
		m.cursor = len(visible) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil
	case "ctrl+d", "pgdown":
		m.cursor += 8
		if m.cursor > len(visible)-1 {
			m.cursor = len(visible) - 1
		}
		return m.maybeLoadMore(visible)
	case "ctrl+u", "pgup":
		m.cursor -= 8
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil
	case "enter", "l":
		if m.cursor >= 0 && m.cursor < len(visible) {
			return m.openDetail(visible[m.cursor])
		}
		return m, nil
	case "tab", "right", "L":
		return m.switchTab(1)
	case "shift+tab", "left", "H":
		return m.switchTab(-1)
	case "s":
		m.sortIdx = (m.sortIdx + 1) % len(sortOptions)
		m.loading = true
		m.cursor = 0
		return m, loadEvents(m.client, m.query(0), false)
	case "S":
		m.sortIdx = (m.sortIdx - 1 + len(sortOptions)) % len(sortOptions)
		m.loading = true
		m.cursor = 0
		return m, loadEvents(m.client, m.query(0), false)
	case "/":
		m.searchMode = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, nil
	case "w":
		if m.cursor >= 0 && m.cursor < len(visible) {
			m.toggleWatch(visible[m.cursor])
		}
		return m, nil
	case "r":
		m.loading = true
		return m, loadEvents(m.client, m.query(0), false)
	}
	return m, nil
}

// switchTab moves the active tab by delta and reloads its events.
func (m model) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.activeTab = (m.activeTab + delta + len(browseTabs)) % len(browseTabs)
	m.cursor = 0
	m.scroll = 0
	m.searchTerm = ""
	if m.currentTab().watch {
		if ids := m.watch.list(); len(ids) > 0 {
			return m, loadWatchEvents(m.client, ids)
		}
		return m, nil
	}
	m.loading = true
	return m, loadEvents(m.client, m.query(0), false)
}

// toggleWatch stars/unstars an event, keeping its full data for the Watch tab.
func (m *model) toggleWatch(e api.Event) {
	if m.watch.toggle(e.ID) {
		m.watchEvents[e.ID] = e
	} else {
		delete(m.watchEvents, e.ID)
	}
}

func (m model) maybeLoadMore(visible []api.Event) (tea.Model, tea.Cmd) {
	// Prefetch next page when nearing the end (not on watch/search views).
	if m.currentTab().watch || m.searchTerm != "" {
		return m, nil
	}
	if m.hasMore && !m.loadingMore && m.cursor >= len(m.events)-6 {
		m.loadingMore = true
		m.offset = len(m.events)
		return m, loadEvents(m.client, m.query(m.offset), true)
	}
	return m, nil
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchInput.Blur()
		return m, nil
	case "enter":
		m.searchMode = false
		m.searchInput.Blur()
		m.searchTerm = m.searchInput.Value()
		m.loading = true
		m.cursor = 0
		m.scroll = 0
		return m, loadEvents(m.client, m.query(0), false)
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h", "left", "backspace":
		m.screen = screenBrowse
		m.detail = nil
		m.book = nil
		m.history = nil
		return m, nil
	case "j", "down":
		if m.detail != nil && m.detailCursor < len(m.detail.Markets)-1 {
			m.detailCursor++
			return m, m.loadDetailData()
		}
		return m, nil
	case "k", "up":
		if m.detailCursor > 0 {
			m.detailCursor--
			return m, m.loadDetailData()
		}
		return m, nil
	case "t":
		m.histIdx = (m.histIdx + 1) % len(historyIntervals)
		return m, m.loadDetailData()
	case "T":
		m.histIdx = (m.histIdx - 1 + len(historyIntervals)) % len(historyIntervals)
		return m, m.loadDetailData()
	case "d":
		m.descExpanded = !m.descExpanded
		return m, nil
	case "w":
		if m.detail != nil {
			m.toggleWatch(*m.detail)
		}
		return m, nil
	case "r":
		return m, m.loadDetailData()
	}
	return m, nil
}

// handleMouse maps wheel events to navigation in the active screen.
func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showHelp || m.searchMode || m.screen == screenSplash {
		return m, nil
	}
	var up bool
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		up = true
	case tea.MouseButtonWheelDown:
		up = false
	default:
		return m, nil
	}
	key := tea.KeyMsg{Type: tea.KeyDown}
	if up {
		key = tea.KeyMsg{Type: tea.KeyUp}
	}
	if m.screen == screenDetail {
		return m.handleDetailKey(key)
	}
	return m.handleBrowseKey(key)
}

// openDetail switches to the detail screen for an event.
func (m model) openDetail(e api.Event) (tea.Model, tea.Cmd) {
	ev := e
	m.detail = &ev
	m.screen = screenDetail
	m.detailCursor = 0
	m.descExpanded = false
	m.book = nil
	m.history = nil
	// fresh animated bars per market, revealing from zero
	m.detailBars = make([]springBar, len(ev.Markets))
	for i, mk := range ev.Markets {
		b := newSpringBar()
		b.setTarget(mk.YesPrice())
		m.detailBars[i] = b
	}
	return m, m.loadDetailData()
}

// loadDetailData fetches the order book + price history for the selected market.
func (m *model) loadDetailData() tea.Cmd {
	if m.detail == nil || m.detailCursor >= len(m.detail.Markets) {
		return nil
	}
	mk := m.detail.Markets[m.detailCursor]
	if len(mk.ClobTokenIDs) == 0 {
		m.bookToken = ""
		m.histToken = ""
		m.book = nil
		m.history = nil
		return nil
	}
	token := mk.ClobTokenIDs[0]
	m.bookToken = token
	m.histToken = token
	m.book = nil
	m.history = nil
	hi := historyIntervals[m.histIdx]
	return tea.Batch(
		loadBook(m.client, token),
		loadHistory(m.client, token, hi.interval, hi.fidelity),
	)
}

// filteredEvents returns the events for the active tab. The Watch tab draws
// from the in-memory starred set; all others use the loaded list.
func (m model) filteredEvents() []api.Event {
	if !m.currentTab().watch {
		return m.events
	}
	out := make([]api.Event, 0, len(m.watchEvents))
	for _, id := range m.watch.list() {
		if e, ok := m.watchEvents[id]; ok {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Volume > out[j].Volume })
	return out
}
