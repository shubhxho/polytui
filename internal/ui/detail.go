package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewDetail() string {
	w := m.width - 4
	if w < 24 {
		w = 24
	}
	innerH := m.height - 2
	if innerH < 10 {
		innerH = 10
	}
	e := m.detail
	if e == nil {
		return docStyle.Render("no market")
	}

	header := m.detailHeader(w)
	status := m.detailStatus(w)
	desc := ""
	if m.descExpanded && e.Description != "" {
		desc = m.descBlock(w)
	}

	bodyH := innerH - lipgloss.Height(header) - lipgloss.Height(status) - 1
	if desc != "" {
		bodyH -= lipgloss.Height(desc) + 1
	}
	if bodyH < 6 {
		bodyH = 6
	}

	wide := w >= 96
	var body string
	if wide {
		leftW := w * 9 / 20
		rightW := w - leftW - 3
		left := m.outcomesPanel(leftW, bodyH)
		right := m.chartAndBook(rightW, bodyH)
		body = joinH(left, "   ", right)
	} else {
		half := bodyH / 2
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.outcomesPanel(w, half), "",
			m.chartAndBook(w, bodyH-half-1))
	}

	parts := []string{header, "", body}
	if desc != "" {
		parts = append(parts, "", desc)
	}
	parts = append(parts, "", status)
	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m model) detailHeader(w int) string {
	e := m.detail
	star := ""
	if m.watch.has(e.ID) {
		star = stylePink.Render(" ★")
	}
	title := styleTitle.Render(truncate(e.Title, w-2)) + star
	meta := metaLine(w,
		"vol "+fmtUSD(e.Volume),
		"24h "+fmtUSD(e.Volume24hr),
		"liq "+fmtUSD(e.Liquidity),
		"OI "+fmtUSD(e.OpenInterest),
		"ends "+humanizeUntil(e.EndsAt()),
		plural(len(e.Markets), "outcome"),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, meta)
}

// outcomesPanel lists the markets within the event with animated bars.
func (m model) outcomesPanel(w, h int) string {
	e := m.detail
	rows := []string{sectionTitle("Outcomes", w), ""}

	avail := h - 2
	maxItems := avail / 2
	if maxItems < 1 {
		maxItems = 1
	}
	start := 0
	if m.detailCursor >= maxItems {
		start = m.detailCursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(e.Markets) {
		end = len(e.Markets)
	}

	for i := start; i < end; i++ {
		mk := e.Markets[i]
		selected := i == m.detailCursor
		p := mk.YesPrice()
		var bar springBar
		if i < len(m.detailBars) {
			bar = m.detailBars[i]
		} else {
			bar = newSpringBar()
			bar.snap(p)
		}
		marker := "  "
		nameStyle := styleMuted
		if selected {
			marker = stylePink.Render("▌ ")
			nameStyle = styleTitle
		}
		pct := lipgloss.NewStyle().Foreground(probColor(p)).Bold(true).Render(fmtPct(p))
		line1 := marker + hbar(w-2, nameStyle.Render(truncate(mk.Title(), w-10)), pct)
		line2 := "  " + bar.renderPlain(w-4, probColor(p))
		rows = append(rows, line1, line2)
	}
	if end < len(e.Markets) {
		rows = append(rows, styleFaint.Render(fmt.Sprintf("  … %d more", len(e.Markets)-end)))
	}
	return strings.Join(rows, "\n")
}

// chartAndBook renders the price chart and order book for the selected market.
func (m model) chartAndBook(w, h int) string {
	e := m.detail
	if m.detailCursor >= len(e.Markets) {
		return ""
	}
	mk := e.Markets[m.detailCursor]

	pts := make([]float64, len(m.history))
	for i, p := range m.history {
		pts[i] = p.P
	}
	cur := mk.YesPrice()
	change := ""
	if len(pts) > 1 {
		delta := pts[len(pts)-1] - pts[0]
		arrow, cstyle := "▲", styleGreen
		if delta < 0 {
			arrow, cstyle = "▼", stylePink
		}
		change = cstyle.Render(fmt.Sprintf("%s %.1f pts", arrow, delta*100))
	}

	priceLbl := "Chance  " +
		lipgloss.NewStyle().Foreground(probColor(cur)).Bold(true).Render(fmtPct(cur))
	if change != "" {
		priceLbl += "  " + change
	}
	chartHead := sectionHeader(styleTitle.Render(priceLbl), m.rangeTabs(), w)

	chartH := h/2 - 2
	if chartH < 4 {
		chartH = 4
	}
	var chartBody string
	if len(pts) == 0 {
		spin := m.spinner()
		if m.histToken == "" {
			spin = "—"
		}
		chartBody = lipgloss.Place(w, chartH, lipgloss.Center, lipgloss.Center,
			styleSubtle.Render(spin+" loading history…"))
	} else {
		chartBody = chartBlock(pts, w, chartH)
	}

	spread := ""
	if m.book != nil {
		bb := topLevels(m.book.Bids, true, 1)
		ba := topLevels(m.book.Asks, false, 1)
		if len(bb) > 0 && len(ba) > 0 {
			spread = styleSubtle.Render(fmt.Sprintf("spread %.0f¢", (ba[0].PriceF()-bb[0].PriceF())*100))
		}
	}
	bookRows := h - chartH - 6
	if bookRows < 2 {
		bookRows = 2
	}
	if bookRows > 8 {
		bookRows = 8
	}
	bookHead := sectionHeader(styleTitle.Render("Order book"), spread, w)
	bookBody := orderBookView(m.book, w, bookRows)

	return lipgloss.JoinVertical(lipgloss.Left,
		chartHead, chartBody, "",
		bookHead, bookBody,
	)
}

func (m model) rangeTabs() string {
	activePill := lipgloss.NewStyle().Foreground(fg).Background(purple).Padding(0, 1)
	var parts []string
	for i, hi := range historyIntervals {
		if i == m.histIdx {
			parts = append(parts, activePill.Render(hi.label))
		} else {
			parts = append(parts, styleSubtle.Render(" "+hi.label+" "))
		}
	}
	return strings.Join(parts, "")
}

func (m model) descBlock(w int) string {
	desc := strings.Join(reflow(m.detail.Description, w), "\n")
	return lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle("Details", w),
		styleMuted.Render(desc),
	)
}

func (m model) detailStatus(w int) string {
	mid := "↑↓ outcome  ·  t/T range  ·  d details  ·  w star  ·  r refresh  ·  esc back"
	return statusBar(w, "MARKET", mid, " polytui ")
}
