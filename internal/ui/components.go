package ui

import (
	"fmt"
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

// Hoisted bar styles — reused every frame instead of allocating a fresh
// lipgloss.NewStyle() per call. .Foreground(col) returns a cheap value copy.
var (
	barFillBase = lipgloss.NewStyle()
	// barTrack is the unfilled rail: the *same* █ glyph as the fill but in a
	// dim cool grey, so filled and empty share one seamless bar and only the
	// colour changes (no ░ texture clash).
	barTrack     = lipgloss.Color("#2C2C38")
	barTrackBase = lipgloss.NewStyle().Foreground(barTrack)
)

// renderBarRun draws a width-cell bar as a single continuous rail: a dim track
// behind, a vivid fill in front, joined by one sub-cell glyph at the crest so
// the spring animation reads smoothly. The fill keeps its accent hue all the way
// and only lifts toward white at the leading edge — a glossy highlight rather
// than a wash. The gradient is quantised into at most gradientSegs colour bands,
// so the whole bar is a bounded handful of lipgloss.Render calls regardless of
// width — cheap at 60fps across many bars.
func renderBarRun(width int, pos float64, col lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	filledF := pos * float64(width)
	full := int(filledF)
	if full > width {
		full = width
	}
	frac := filledF - float64(full)
	rest := width - full

	var sb strings.Builder
	if full > 0 {
		sb.WriteString(gradientFill(full, col))
	}
	// The crest cell is painted over the track background so the partial glyph's
	// empty side keeps the rail colour instead of punching a hole to the ground.
	if full < width && frac >= 0.12 {
		sb.WriteString(barFillBase.Foreground(barBright(col)).Background(barTrack).Render(partialBlock(frac)))
		rest--
	}
	if rest > 0 {
		sb.WriteString(barTrackBase.Render(strings.Repeat("█", rest)))
	}
	return sb.String()
}

// gradientSegs caps how many colour bands the fill is split into; each band is
// one Render call, so the whole gradient is at most this many regardless of bar
// width.
const gradientSegs = 8

// gradientFill renders n solid cells shaded from the full accent at the tail to
// a white-lifted crest at the leading edge — a subtle gloss, never grey.
func gradientFill(n int, col lipgloss.Color) string {
	if n <= 0 {
		return ""
	}
	base := rgbOf(col)
	crest := lerpRGB(base, rgbWhite, 0.30)
	segs := gradientSegs
	if segs > n {
		segs = n
	}
	var sb strings.Builder
	for s := 0; s < segs; s++ {
		cnt := n*(s+1)/segs - n*s/segs
		if cnt == 0 {
			continue
		}
		t := 0.0
		if segs > 1 {
			t = float64(s) / float64(segs-1)
		}
		sb.WriteString(barFillBase.Foreground(lerpHex(base, crest, t)).Render(strings.Repeat("█", cnt)))
	}
	return sb.String()
}

// barBright is the crest colour — the accent lifted toward white for the gloss
// highlight at the leading edge.
func barBright(col lipgloss.Color) lipgloss.Color {
	return packHex(lerpRGB(rgbOf(col), rgbWhite, 0.30))
}

var rgbWhite = [3]float64{0xff, 0xff, 0xff}

// ---- small RGB lerp helpers (hex #RRGGBB only) ---------------------------

func rgbOf(c lipgloss.Color) [3]float64 {
	s := string(c)
	if len(s) != 7 || s[0] != '#' {
		return [3]float64{}
	}
	return [3]float64{float64(hexByte(s[1], s[2])), float64(hexByte(s[3], s[4])), float64(hexByte(s[5], s[6]))}
}

func hexByte(hi, lo byte) int { return hexNib(hi)<<4 | hexNib(lo) }

func hexNib(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func lerpRGB(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

func lerpHex(a, b [3]float64, t float64) lipgloss.Color { return packHex(lerpRGB(a, b, t)) }

func packHex(c [3]float64) lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", clampByte(c[0]), clampByte(c[1]), clampByte(c[2])))
}

func clampByte(v float64) int {
	i := int(v + 0.5)
	if i < 0 {
		return 0
	}
	if i > 255 {
		return 255
	}
	return i
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
	lab := barFillBase.Foreground(col).Bold(true).Render(label)
	return renderBarRun(barW, b.pos, col) + " " + lab
}

// renderPlain draws just the animated bar (no label) at the given width.
func (b springBar) renderPlain(width int, col lipgloss.Color) string {
	return renderBarRun(width, b.pos, col)
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

// ---- Order book depth ----------------------------------------------------

// orderBookView renders bids (left/green) and asks (right/pink) as a depth
// ladder mirrored around a faint center divider — bars grow toward the spread,
// so the touch sits in the middle and sizes read outward.
func orderBookView(book *api.OrderBook, width, rows int) string {
	if book == nil {
		return styleSubtle.Render("  loading order book…")
	}
	// Bids come ascending by price; take the highest. Asks ascending; take lowest.
	bids := topLevels(book.Bids, true, rows)
	asks := topLevels(book.Asks, false, rows)
	if len(bids) == 0 && len(asks) == 0 {
		return styleSubtle.Render("  no resting orders")
	}

	maxSize := 0.0
	for _, l := range append(append([]api.OrderLevel{}, bids...), asks...) {
		if l.SizeF() > maxSize {
			maxSize = l.SizeF()
		}
	}
	if maxSize == 0 {
		maxSize = 1
	}

	side := (width - 1) / 2 // columns each side of the divider
	if side < 12 {
		side = 12
	}
	barW := side - 6 - 1 - 3 - 1 // size(6) sp price(3) sp bar
	if barW < 3 {
		barW = 3
	}

	greenBar := lipgloss.NewStyle().Foreground(green)
	redBar := lipgloss.NewStyle().Foreground(pink)
	div := styleFaint.Render("│")

	// Only draw rows that carry a level — avoids a divider dangling into empty space.
	n := len(bids)
	if len(asks) > n {
		n = len(asks)
	}
	if n > rows {
		n = rows
	}

	var sb strings.Builder
	sb.WriteString(joinH(
		padRight(styleSubtle.Render("  bids"), side),
		" ",
		styleSubtle.Render("asks"),
	) + "\n")

	for i := 0; i < n; i++ {
		var left, right string
		if i < len(bids) {
			b := bids[i]
			n := int(b.SizeF() / maxSize * float64(barW))
			bar := greenBar.Render(strings.Repeat(" ", barW-n) + strings.Repeat("▰", n))
			left = padLeft(fmtNum(b.SizeF()), 6) + " " +
				styleGreen.Render(padLeft(fmtCents(b.PriceF()), 3)) + " " + bar
		} else {
			left = strings.Repeat(" ", side)
		}
		if i < len(asks) {
			a := asks[i]
			n := int(a.SizeF() / maxSize * float64(barW))
			bar := redBar.Render(strings.Repeat("▰", n) + strings.Repeat(" ", barW-n))
			right = bar + " " +
				stylePink.Render(padRight(fmtCents(a.PriceF()), 3)) + " " + fmtNum(a.SizeF())
		}
		sb.WriteString(padRight(left, side) + div + right)
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
