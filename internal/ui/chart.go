package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"shubhxho/polytui/internal/api"
)

// zoneChart is the bubblezone id marking the price-chart body for mouse hover.
const zoneChart = "chart"

// detailCache memoises the two most expensive detail-screen renders — the price
// chart and the order book — so the 60fps animation loop doesn't rebuild them
// every frame. Both bodies only change when their inputs do (a 15s refresh, a
// range switch, or a resize), so each frame is an O(1) signature compare and a
// cached-string return instead of an O(width·height) redraw.
//
// Held behind a pointer on the model so the value-receiver View can populate it.
type detailCache struct {
	ts        timeserieslinechart.Model // reused canvas; resized in place
	tsW, tsH  int                       // dims the canvas was built at
	chartSig  string
	chartBody string
	bookSig   string
	bookBody  string
	tabSig    string // tab bar is static unless activeTab/width change
	tabBody   string
}

func newDetailCache() *detailCache { return &detailCache{} }

// chart returns the rendered price chart, rebuilding only when the data, size,
// or trend direction changed since the last call.
func (c *detailCache) chart(pts []api.PricePoint, w, h int) string {
	if w < 8 {
		w = 8
	}
	if h < 3 {
		h = 3
	}
	sig := chartSig(pts, w, h)
	if sig == c.chartSig && c.chartBody != "" {
		return c.chartBody
	}
	c.chartSig = sig
	c.chartBody = c.buildChart(pts, w, h)
	return c.chartBody
}

// chartSig captures everything that affects the rendered chart cheaply: the
// span (first/last sample), the count, and the canvas size. The endpoints also
// determine the trend colour, so no separate term is needed for it.
func chartSig(pts []api.PricePoint, w, h int) string {
	if len(pts) == 0 {
		return "empty"
	}
	first, last := pts[0], pts[len(pts)-1]
	var b []byte
	b = strconv.AppendInt(b, int64(len(pts)), 10)
	b = append(b, '|')
	b = strconv.AppendInt(b, first.T, 10)
	b = append(b, ':')
	b = strconv.AppendFloat(b, first.P, 'f', 4, 64)
	b = append(b, '|')
	b = strconv.AppendInt(b, last.T, 10)
	b = append(b, ':')
	b = strconv.AppendFloat(b, last.P, 'f', 4, 64)
	b = append(b, '|')
	b = strconv.AppendInt(b, int64(w), 10)
	b = append(b, 'x')
	b = strconv.AppendInt(b, int64(h), 10)
	return string(b)
}

func (c *detailCache) buildChart(pts []api.PricePoint, w, h int) string {
	if len(pts) == 0 {
		return ""
	}
	lineCol := green
	if len(pts) > 1 && pts[len(pts)-1].P < pts[0].P {
		lineCol = pink
	}

	// Build the canvas once per size; reuse it across refreshes/range switches.
	if c.tsW != w || c.tsH != h {
		axisStyle := lipgloss.NewStyle().Foreground(faint)
		labelStyle := lipgloss.NewStyle().Foreground(faint)
		c.ts = timeserieslinechart.New(w, h,
			timeserieslinechart.WithXYSteps(3, 2),
			timeserieslinechart.WithAxesStyles(axisStyle, labelStyle),
			timeserieslinechart.WithLineStyle(runes.ThinLineStyle),
			timeserieslinechart.WithYLabelFormatter(func(_ int, v float64) string {
				return fmtPct(v)
			}),
		)
		c.tsW, c.tsH = w, h
	} else {
		c.ts.ClearAllData()
	}

	c.ts.SetStyle(lipgloss.NewStyle().Foreground(lineCol))
	// Span-aware X labels: clock for intraday, dates for longer ranges. This
	// replaces ntcharts' default formatter (which printed a stray "'70" year
	// line). Set outside the size guard so a range switch updates it.
	span := pts[len(pts)-1].T - pts[0].T
	c.ts.XLabelFormatter = func(_ int, v float64) string {
		t := time.Unix(int64(v), 0)
		if span < 36*3600 {
			return t.Format("15:04")
		}
		return t.Format("Jan 2")
	}
	mn, mx := pts[0].P, pts[0].P
	for _, p := range pts {
		if p.P < mn {
			mn = p.P
		}
		if p.P > mx {
			mx = p.P
		}
		c.ts.Push(timeserieslinechart.TimePoint{Time: time.Unix(p.T, 0), Value: p.P})
	}
	// Fit the viewport to the data so the line fills the panel instead of being
	// squashed against the default 0–100% range.
	pad := (mx - mn) * 0.08
	if pad < 0.005 {
		pad = 0.005
	}
	c.ts.SetViewYRange(mn-pad, mx+pad)
	c.ts.SetViewTimeRange(time.Unix(pts[0].T, 0), time.Unix(pts[len(pts)-1].T, 0))
	c.ts.DrawBraille()
	return c.ts.View()
}

// book returns the rendered order book, rebuilding only when the book contents
// or layout changed.
func (c *detailCache) book(b *api.OrderBook, w, rows int) string {
	sig := bookSig(b, w, rows)
	if sig == c.bookSig && c.bookBody != "" {
		return c.bookBody
	}
	c.bookSig = sig
	c.bookBody = orderBookView(b, w, rows)
	return c.bookBody
}

// bookSig folds the spread-nearest levels (the only ones rendered) plus layout
// into a key. A new book object with identical visible levels reuses the cache.
func bookSig(b *api.OrderBook, w, rows int) string {
	if b == nil {
		return "nil"
	}
	out := make([]byte, 0, 64)
	out = strconv.AppendInt(out, int64(w), 10)
	out = append(out, 'x')
	out = strconv.AppendInt(out, int64(rows), 10)
	appendLevels := func(levels []api.OrderLevel) {
		for _, l := range topLevels(levels, true, rows) {
			out = append(out, '|')
			out = append(out, l.Price...)
			out = append(out, ',')
			out = append(out, l.Size...)
		}
	}
	out = append(out, ';')
	appendLevels(b.Bids)
	out = append(out, ';')
	appendLevels(b.Asks)
	return string(out)
}

// tabs returns the rendered tab bar, rebuilding only when the active tab or
// width changes (it's otherwise static across the 60fps frame loop).
func (c *detailCache) tabs(m model, width int) string {
	sig := strconv.Itoa(m.activeTab) + "x" + strconv.Itoa(width)
	if sig == c.tabSig && c.tabBody != "" {
		return c.tabBody
	}
	c.tabSig = sig
	c.tabBody = m.renderTabs(width)
	return c.tabBody
}

// ---- chart hover crosshair ----------------------------------------------

// chartHover composites a vertical crosshair and a date/price tooltip onto the
// cached chart body, without touching the data cache. col is the chart-local
// column of the hovered sample; idx is its index into pts.
func chartHover(body string, w, col, idx int, pts []api.PricePoint) string {
	if body == "" || idx < 0 || idx >= len(pts) {
		return body
	}
	lines := strings.Split(body, "\n")
	bar := stylePink.Render("│")
	for i := range lines {
		lines[i] = spliceCell(lines[i], col, bar)
	}
	// Tooltip pill on the top row, e.g. "Mar 12 · 41%".
	p := pts[idx]
	tip := styleCrosshairTip.Render(time.Unix(p.T, 0).Format("Jan 2") + " · " + fmtPct(p.P))
	lw := lipgloss.Width(tip)
	tipCol := col + 1
	if tipCol+lw > w {
		tipCol = col - lw // not enough room on the right; flip to the left
	}
	if tipCol < 0 {
		tipCol = 0
	}
	if len(lines) > 0 {
		lines[0] = spliceCell(lines[0], tipCol, tip)
	}
	return strings.Join(lines, "\n")
}

// spliceCell overwrites the cells of line starting at display column col with s,
// preserving the surrounding ANSI styling. Short lines are space-padded.
func spliceCell(line string, col int, s string) string {
	sw := lipgloss.Width(s)
	left := ansi.Truncate(line, col, "")
	if pad := col - ansi.StringWidth(left); pad > 0 {
		left += strings.Repeat(" ", pad)
	}
	right := ansi.TruncateLeft(line, col+sw, "")
	return left + s + right
}
