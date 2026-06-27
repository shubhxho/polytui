package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"shubhxho/polytui/internal/api"
)

var docStyle = lipgloss.NewStyle().Padding(1, 2)

// hbar builds a full-width line with left content and right-aligned content.
func hbar(width int, left, right string) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// statusBar renders the bottom bar: a pink key badge, a gray fill with context,
// and purple nuggets on the right.
func statusBar(width int, key, mid string, nuggets ...string) string {
	left := styleStatusKey.Render(key)
	right := ""
	for _, n := range nuggets {
		right += styleStatusPurp.Render(n)
	}
	midW := width - lipgloss.Width(left) - lipgloss.Width(right)
	if midW < 0 {
		midW = 0
	}
	midStr := styleStatusBar.Width(midW).Render(" " + truncate(mid, midW-1))
	return left + midStr + right
}

// ---- splash --------------------------------------------------------------

func (m model) viewSplash() string {
	reveal := clamp01(m.splashPos)
	mark := wordmark()
	barMax := 24
	bw := int(reveal * float64(barMax))
	rule := stylePink.Render(strings.Repeat("─", bw)) + styleFaint.Render(strings.Repeat("─", barMax-bw))

	var status string
	switch {
	case m.loadErr != nil:
		status = stylePink.Render("✗ " + truncate(m.loadErr.Error(), 40))
	case m.loading:
		status = styleMuted.Render(m.spinner() + " connecting…")
	default:
		status = styleGreen.Render("✓ ready")
	}

	block := lipgloss.JoinVertical(lipgloss.Center,
		mark,
		styleSubtle.Render("the polymarket terminal"),
		"",
		rule,
		"",
		status,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
}

// ---- browse --------------------------------------------------------------

func (m model) viewBrowse() string {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	innerH := m.height - 2
	if innerH < 8 {
		innerH = 8
	}

	header := m.browseTop(w)
	tabs := m.renderTabs(w)
	status := m.browseStatus(w)

	used := lipgloss.Height(header) + lipgloss.Height(tabs) + lipgloss.Height(status) + 1 // +1 spacer
	listH := innerH - used
	if listH < 3 {
		listH = 3
	}
	list := m.browseList(w, listH)

	content := lipgloss.JoinVertical(lipgloss.Left, header, tabs, list, "", status)
	return docStyle.Render(content)
}

func (m model) browseTop(w int) string {
	left := wordmark()
	right := styleSubtle.Render("sort ") + stylePurp.Render(m.currentSort().label)
	if m.loading {
		right += "  " + styleMuted.Render(m.spinner())
	}
	return hbar(w, left, right)
}

func (m model) browseList(w, h int) string {
	if m.searchMode {
		return m.searchOverlayBody(w, h)
	}
	events := m.filteredEvents()
	if len(events) == 0 {
		var msg string
		switch {
		case m.loading:
			msg = styleMuted.Render(m.spinner() + " loading markets…")
		case m.loadErr != nil:
			msg = stylePink.Render("✗ " + m.loadErr.Error())
		case m.currentTab().watch:
			msg = styleSubtle.Render("nothing here yet — press w on a market to star it")
		default:
			msg = styleSubtle.Render("no markets found")
		}
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
	}

	const rowH = 3
	perPage := h / rowH
	if perPage < 1 {
		perPage = 1
	}
	scroll := m.scroll
	if m.cursor < scroll {
		scroll = m.cursor
	}
	if m.cursor >= scroll+perPage {
		scroll = m.cursor - perPage + 1
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + perPage
	if end > len(events) {
		end = len(events)
	}

	var rows []string
	for i := scroll; i < end; i++ {
		rows = append(rows, m.eventRow(events[i], i == m.cursor, w))
	}
	body := strings.Join(rows, "\n")
	if bh := lipgloss.Height(body); bh < h {
		body += strings.Repeat("\n", h-bh)
	}
	return body
}

func (m model) eventRow(e api.Event, selected bool, w int) string {
	marker := "   "
	titleStyle := styleMuted
	if selected {
		marker = stylePink.Render(" ▌ ")
		titleStyle = styleTitle
	}
	inner := w - 3
	if inner < 16 {
		inner = 16
	}

	p := e.TopPrice()
	bar := m.bars[e.ID]
	if bar == nil {
		nb := newSpringBar()
		nb.snap(p)
		bar = &nb
	}

	star := " "
	if m.watch.has(e.ID) {
		star = stylePink.Render("★")
	}

	// Right cluster: animated bar + percentage + star.
	barW := 14
	pct := lipgloss.NewStyle().Foreground(probColor(p)).Bold(true).Render(padLeft(fmtPct(p), 4))
	rightCluster := bar.renderPlain(barW, probColor(p)) + " " + pct + " " + star
	titleW := inner - lipgloss.Width(rightCluster) - 1
	if titleW < 8 {
		titleW = 8
	}
	line1 := hbar(inner, titleStyle.Render(truncate(e.Title, titleW)), rightCluster)

	// Meta line.
	meta := []string{"vol " + fmtUSD(e.Volume)}
	if !e.Binary() {
		if ld, _ := e.Leader(); ld != "" {
			meta = append(meta, truncate(ld, 22))
		}
	}
	meta = append(meta, "ends "+humanizeUntil(e.EndsAt()), plural(len(e.Markets), "outcome"))
	line2 := metaLine(inner, meta...)

	content := lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	return joinH(marker, content)
}

func (m model) browseStatus(w int) string {
	key := strings.ToUpper(strings.TrimPrefix(m.currentTab().label, "★ "))
	mid := plural(len(m.filteredEvents()), "market")
	if m.searchTerm != "" {
		mid += " · search: " + m.searchTerm
	}
	mid += "   ·   ⇄ tab  ↑↓ nav  ⏎ open  / search  w star  s sort  ? help"
	return statusBar(w, key, mid, " polytui ")
}

// ---- search --------------------------------------------------------------

func (m model) searchOverlayBody(w, h int) string {
	box := stylePanel.BorderForeground(purple).Width(maxi(w/2, 30)).Render(
		stylePink.Render("Search markets") + "\n\n" +
			m.searchInput.View() + "\n\n" +
			styleSubtle.Render("enter to search · esc to cancel"),
	)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

// ---- help ----------------------------------------------------------------

func (m model) viewHelp() string {
	rows := [][2]string{
		{"↑/k  ↓/j", "move selection"},
		{"tab / shift+tab", "switch category tab"},
		{"←  →", "switch category tab"},
		{"g / G", "jump to top / bottom"},
		{"ctrl+u / ctrl+d", "page up / down"},
		{"enter / l", "open market detail"},
		{"esc", "back"},
		{"s / S", "cycle sort order"},
		{"/", "search markets"},
		{"w", "star / unstar (watch)"},
		{"t / T", "detail: chart range"},
		{"d", "detail: description"},
		{"r", "refresh"},
		{"?", "toggle help"},
		{"q / ctrl+c", "quit"},
	}
	var b strings.Builder
	b.WriteString(wordmark() + styleSubtle.Render("  keybindings") + "\n\n")
	for _, r := range rows {
		b.WriteString("  " + styleHelpKey.Render(padRight(r[0], 18)) + styleHelpDesc.Render(r[1]) + "\n")
	}
	b.WriteString("\n" + styleSubtle.Render("  gamma-api.polymarket.com · clob.polymarket.com") + "\n")
	b.WriteString(styleSubtle.Render("  press any key to close"))

	box := stylePanel.BorderForeground(purple).Padding(1, 3).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
