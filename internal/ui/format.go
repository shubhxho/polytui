package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// fmtUSD formats a dollar amount compactly: $1.2M, $34.5K, $812.
func fmtUSD(v float64) string {
	neg := ""
	if v < 0 {
		neg = "-"
		v = -v
	}
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%s$%.1fB", neg, v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%s$%.1fM", neg, v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%s$%.1fK", neg, v/1e3)
	default:
		return fmt.Sprintf("%s$%.0f", neg, v)
	}
}

// fmtNum formats a plain compact number (no currency).
func fmtNum(v float64) string {
	switch {
	case v >= 1e6:
		return fmt.Sprintf("%.1fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.1fK", v/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// plural returns "n word" / "n words" with naive pluralization.
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// fmtPct renders a 0..1 probability as a percentage string.
func fmtPct(p float64) string {
	return fmt.Sprintf("%.0f%%", p*100)
}

// fmtCents renders a 0..1 price as cents, e.g. 52¢.
func fmtCents(p float64) string {
	return fmt.Sprintf("%.0f¢", p*100)
}

// humanizeUntil returns a short "ends in" string.
func humanizeUntil(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Until(t)
	if d <= 0 {
		return "ended"
	}
	days := int(d.Hours()) / 24
	switch {
	case days >= 365:
		return fmt.Sprintf("%dy", days/365)
	case days >= 1:
		return fmt.Sprintf("%dd", days)
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}

// truncate trims a string to width display cells, adding an ellipsis.
func truncate(s string, width int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if runewidth.StringWidth(s) <= width {
		return s
	}
	if width <= 1 {
		return runewidth.Truncate(s, width, "")
	}
	return runewidth.Truncate(s, width, "…")
}

// padRight pads s with spaces to exactly width display cells.
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// padLeft right-aligns s to width display cells.
func padLeft(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// reflow wraps plain text to width, returning lines.
func reflow(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		cur := ""
		for _, w := range words {
			if cur == "" {
				cur = w
			} else if runewidth.StringWidth(cur)+1+runewidth.StringWidth(w) <= width {
				cur += " " + w
			} else {
				lines = append(lines, cur)
				cur = w
			}
		}
		if cur != "" {
			lines = append(lines, cur)
		}
	}
	return lines
}

// metaLine joins metadata items with a faint "·" separator, dropping trailing
// items that would overflow width. Items are measured as plain text and styled
// afterwards, so truncation never splits an ANSI escape.
func metaLine(width int, parts ...string) string {
	const sep = " · "
	sepW := runewidth.StringWidth(sep)
	sepStyled := styleFaint.Render(sep)
	var kept []string
	used := 0
	for i, p := range parts {
		add := runewidth.StringWidth(p)
		if i > 0 {
			add += sepW
		}
		if width > 0 && used+add > width {
			break
		}
		used += add
		kept = append(kept, styleSubtle.Render(p))
	}
	return strings.Join(kept, sepStyled)
}

// joinH joins blocks horizontally with top alignment.
func joinH(blocks ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}
