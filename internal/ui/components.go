package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
	"shubhxho/polytui/internal/api"
)

// ---- Spring-animated bar -------------------------------------------------

// springBar is a probability bar whose fill smoothly animates toward a target
// using harmonica spring physics.
type springBar struct {
	spring harmonica.Spring
	pos    float64 // current animated value 0..1
	vel    float64
	target float64 // desired value 0..1
}

func newSpringBar() springBar {
	return springBar{
		// ~60fps, lively frequency, slightly underdamped for a subtle settle.
		spring: harmonica.NewSpring(harmonica.FPS(animFPS), 7.0, 0.78),
	}
}

func (b *springBar) setTarget(v float64) { b.target = clamp01(v) }

// snap jumps instantly to v (used on first load to avoid a fill-up sweep
// when you'd rather show the value immediately).
func (b *springBar) snap(v float64) {
	b.target = clamp01(v)
	b.pos = b.target
	b.vel = 0
}

// update advances the physics one frame; returns true while still moving.
func (b *springBar) update() bool {
	b.pos, b.vel = b.spring.Update(b.pos, b.vel, b.target)
	moving := math.Abs(b.pos-b.target) > 0.0005 || math.Abs(b.vel) > 0.0005
	if !moving {
		b.pos = b.target
		b.vel = 0
	}
	return moving
}

// render draws the bar at the given width using the animated position, while
// the numeric label reflects the true target value.
func (b springBar) render(width int, label string, col lipgloss.Color) string {
	if width < 4 {
		width = 4
	}
	barW := width - len(label) - 1
	if barW < 3 {
		barW = 3
	}
	filledF := b.pos * float64(barW)
	full := int(filledF)
	frac := filledF - float64(full)
	var sb strings.Builder
	fillStyle := lipgloss.NewStyle().Foreground(col)
	emptyStyle := lipgloss.NewStyle().Foreground(faint)
	for i := 0; i < barW; i++ {
		switch {
		case i < full:
			sb.WriteString(fillStyle.Render("█"))
		case i == full && frac >= 0.15:
			sb.WriteString(fillStyle.Render(partialBlock(frac)))
		default:
			sb.WriteString(emptyStyle.Render("░"))
		}
	}
	lab := lipgloss.NewStyle().Foreground(col).Bold(true).Render(label)
	return sb.String() + " " + lab
}

// renderPlain draws just the animated bar (no label) at the given width.
func (b springBar) renderPlain(width int, col lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	filledF := b.pos * float64(width)
	full := int(filledF)
	frac := filledF - float64(full)
	var sb strings.Builder
	fillStyle := lipgloss.NewStyle().Foreground(col)
	emptyStyle := lipgloss.NewStyle().Foreground(faint)
	for i := 0; i < width; i++ {
		switch {
		case i < full:
			sb.WriteString(fillStyle.Render("█"))
		case i == full && frac >= 0.15:
			sb.WriteString(fillStyle.Render(partialBlock(frac)))
		default:
			sb.WriteString(emptyStyle.Render("░"))
		}
	}
	return sb.String()
}

var partialBlocks = []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

func partialBlock(frac float64) string {
	idx := int(frac * float64(len(partialBlocks)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(partialBlocks) {
		idx = len(partialBlocks) - 1
	}
	return partialBlocks[idx]
}

// ---- Sparkline price chart ----------------------------------------------

var brailleRamp = []rune("⣀⣄⣤⣦⣶⣷⣿")
var blockRamp = []rune(" ▁▂▃▄▅▆▇█")

// sparkline renders a compact single-line chart of the series.
func sparkline(points []float64, width int) string {
	if len(points) == 0 || width <= 0 {
		return strings.Repeat(" ", max(width, 0))
	}
	sampled := resample(points, width)
	mn, mx := minMax(sampled)
	rng := mx - mn
	var sb strings.Builder
	for _, v := range sampled {
		var idx int
		if rng == 0 {
			idx = len(blockRamp) / 2
		} else {
			idx = int(((v - mn) / rng) * float64(len(blockRamp)-1))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blockRamp) {
			idx = len(blockRamp) - 1
		}
		sb.WriteRune(blockRamp[idx])
	}
	return sb.String()
}

// chartBlock renders a multi-row line chart (height rows tall) with a Y axis.
func chartBlock(points []float64, width, height int) string {
	if height < 2 {
		height = 2
	}
	if width < 4 {
		width = 4
	}
	axisW := 5
	plotW := width - axisW
	if plotW < 2 {
		plotW = 2
	}
	if len(points) == 0 {
		empty := lipgloss.NewStyle().Foreground(faint)
		rows := make([]string, height)
		for i := range rows {
			rows[i] = empty.Render(strings.Repeat(" ", axisW) + strings.Repeat("·", plotW))
		}
		return strings.Join(rows, "\n")
	}
	sampled := resample(points, plotW)
	mn, mx := minMax(sampled)
	if mn == mx {
		mn -= 0.01
		mx += 0.01
	}
	rng := mx - mn

	// Map each column to a fractional row position.
	grid := make([][]rune, height)
	for r := range grid {
		grid[r] = []rune(strings.Repeat(" ", plotW))
	}
	colorGrid := make([][]bool, height)
	for r := range colorGrid {
		colorGrid[r] = make([]bool, plotW)
	}
	rising := sampled[len(sampled)-1] >= sampled[0]
	for x, v := range sampled {
		frac := (v - mn) / rng
		// row 0 is top (max), height-1 bottom (min)
		level := frac * float64((height-1)*len(blockRamp[1:]))
		row := height - 1 - int(level)/(len(blockRamp)-1)
		sub := int(level) % (len(blockRamp) - 1)
		if row < 0 {
			row = 0
		}
		if row >= height {
			row = height - 1
		}
		ch := blockRamp[sub+1]
		grid[row][x] = ch
		colorGrid[row][x] = true
		// fill below with solid blocks for an area-chart feel
		for rr := row + 1; rr < height; rr++ {
			grid[rr][x] = '█'
			colorGrid[rr][x] = true
		}
	}

	lineCol := green
	if !rising {
		lineCol = pink
	}
	fillStyle := lipgloss.NewStyle().Foreground(lineCol)
	axisStyle := lipgloss.NewStyle().Foreground(faint)

	var sb strings.Builder
	for r := 0; r < height; r++ {
		// Y axis label at top, mid, bottom
		var label string
		switch r {
		case 0:
			label = fmtCents(mx)
		case height - 1:
			label = fmtCents(mn)
		case height / 2:
			label = fmtCents((mn + mx) / 2)
		default:
			label = ""
		}
		sb.WriteString(axisStyle.Render(padLeft(label, axisW-1) + "│"))
		// row content
		var rowStr strings.Builder
		for x := 0; x < plotW; x++ {
			if colorGrid[r][x] {
				rowStr.WriteString(fillStyle.Render(string(grid[r][x])))
			} else {
				rowStr.WriteString(" ")
			}
		}
		sb.WriteString(rowStr.String())
		if r < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// ---- Order book depth ----------------------------------------------------

// orderBookView renders bids (left/green) and asks (right/red) with depth bars.
func orderBookView(book *api.OrderBook, width, rows int) string {
	if book == nil {
		return styleMuted.Render("loading order book…")
	}
	// Bids come ascending by price; take the highest. Asks ascending; take lowest.
	bids := topLevels(book.Bids, true, rows)
	asks := topLevels(book.Asks, false, rows)

	maxSize := 0.0
	for _, l := range append(append([]api.OrderLevel{}, bids...), asks...) {
		if l.SizeF() > maxSize {
			maxSize = l.SizeF()
		}
	}
	if maxSize == 0 {
		maxSize = 1
	}

	colW := (width - 3) / 2
	if colW < 10 {
		colW = 10
	}
	barW := colW - 12
	if barW < 3 {
		barW = 3
	}

	greenBar := lipgloss.NewStyle().Foreground(green)
	redBar := lipgloss.NewStyle().Foreground(pink)
	gap := "   "

	var sb strings.Builder
	header := joinH(
		padRight(styleMuted.Render("  bid size  price"), colW),
		gap,
		styleMuted.Render("price  ask size"),
	)
	sb.WriteString(header + "\n")

	n := rows
	for i := 0; i < n; i++ {
		var left, right string
		if i < len(bids) {
			b := bids[i]
			w := int(b.SizeF() / maxSize * float64(barW))
			bar := greenBar.Render(strings.Repeat("▰", w) + strings.Repeat(" ", barW-w))
			left = padLeft(fmtNum(b.SizeF()), 8) + " " + styleGreen.Render(fmtCents(b.PriceF())) + " " + bar
		} else {
			left = strings.Repeat(" ", colW)
		}
		if i < len(asks) {
			a := asks[i]
			w := int(a.SizeF() / maxSize * float64(barW))
			bar := redBar.Render(strings.Repeat(" ", barW-w) + strings.Repeat("▰", w))
			right = bar + " " + stylePink.Render(fmtCents(a.PriceF())) + " " + padRight(fmtNum(a.SizeF()), 8)
		} else {
			right = ""
		}
		sb.WriteString(joinH(padRight(left, colW), gap, right))
		if i < n-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// topLevels returns the n order levels nearest the spread, best price first.
// Polymarket's CLOB orders both sides with the best price at the tail (bids
// ascending, asks descending), so the levels closest to the mid are the last
// entries; we take those and reverse to surface the best price first. The bid
// flag is accepted for call-site clarity but the operation is identical.
func topLevels(levels []api.OrderLevel, bid bool, n int) []api.OrderLevel {
	_ = bid
	if len(levels) == 0 {
		return nil
	}
	start := len(levels) - n
	if start < 0 {
		start = 0
	}
	tail := levels[start:]
	out := make([]api.OrderLevel, len(tail))
	for i, l := range tail {
		out[len(tail)-1-i] = l // reverse: best (spread-nearest) first
	}
	return out
}

// ---- helpers -------------------------------------------------------------

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func resample(points []float64, width int) []float64 {
	if len(points) <= width {
		// stretch by nearest-neighbour so short series fill the width
		out := make([]float64, width)
		for i := range out {
			src := int(float64(i) / float64(width) * float64(len(points)))
			if src >= len(points) {
				src = len(points) - 1
			}
			out[i] = points[src]
		}
		return out
	}
	// downsample by bucket averaging
	out := make([]float64, width)
	bucket := float64(len(points)) / float64(width)
	for i := 0; i < width; i++ {
		start := int(float64(i) * bucket)
		end := int(float64(i+1) * bucket)
		if end > len(points) {
			end = len(points)
		}
		if end <= start {
			end = start + 1
		}
		sum := 0.0
		for j := start; j < end && j < len(points); j++ {
			sum += points[j]
		}
		out[i] = sum / float64(end-start)
	}
	return out
}

func minMax(s []float64) (float64, float64) {
	if len(s) == 0 {
		return 0, 0
	}
	mn, mx := s[0], s[0]
	for _, v := range s {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}
