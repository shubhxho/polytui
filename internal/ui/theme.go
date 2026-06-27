package ui

import "github.com/charmbracelet/lipgloss"

// Palette — the Lip Gloss signature: pink + purple accents on a dark ground,
// with a green "special" for the up/positive side. Deliberately small.
var (
	pink    = lipgloss.Color("#FF5F87")
	pinkHi  = lipgloss.Color("#F25D94")
	purple  = lipgloss.Color("#7D56F4")
	mauve   = lipgloss.Color("#AD58F6")
	green   = lipgloss.Color("#73F59F")
	neutral = lipgloss.Color("#C0BFD6") // ~even odds, soft cool grey

	fg     = lipgloss.Color("#FAFAFA")
	muted  = lipgloss.Color("#9A9A9A")
	subtle = lipgloss.Color("#6C6C6C")
	faint  = lipgloss.Color("#3A3A3A")

	statusBg = lipgloss.Color("#303030")
)

var (
	styleTitle  = lipgloss.NewStyle().Foreground(fg).Bold(true)
	styleMuted  = lipgloss.NewStyle().Foreground(muted)
	styleSubtle = lipgloss.NewStyle().Foreground(subtle)
	styleFaint  = lipgloss.NewStyle().Foreground(faint)

	stylePink  = lipgloss.NewStyle().Foreground(pink)
	stylePurp  = lipgloss.NewStyle().Foreground(purple)
	styleGreen = lipgloss.NewStyle().Foreground(green)

	// Section header: bold label over a subtle rule (the lipgloss list look).
	styleSectionTitle = lipgloss.NewStyle().Foreground(fg).Bold(true)

	// Soft panel, used sparingly — thin purple border, no fill.
	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(faint).
			Padding(0, 1)

	// Status bar pieces.
	styleStatusBar  = lipgloss.NewStyle().Foreground(lipgloss.Color("#C8C8C8")).Background(statusBg)
	styleStatusKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(pink).Bold(true).Padding(0, 1)
	styleStatusPurp = lipgloss.NewStyle().Foreground(fg).Background(purple).Padding(0, 1)

	// Help key/desc.
	styleHelpKey  = lipgloss.NewStyle().Foreground(fg)
	styleHelpDesc = lipgloss.NewStyle().Foreground(subtle)
)

// probColor: green for the favored side, pink for the underdog, muted near 50.
func probColor(p float64) lipgloss.Color {
	switch {
	case p >= 0.55:
		return green
	case p <= 0.45:
		return pink
	default:
		return neutral
	}
}

// sectionHeader renders a (pre-styled) left label, a thin faint rule filling
// the middle, and optional right-aligned content — the one heading style used
// everywhere so sections read consistently.
func sectionHeader(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if right == "" {
		ruleW := width - lw - 1
		if ruleW < 0 {
			ruleW = 0
		}
		return left + " " + styleFaint.Render(repeat("─", ruleW))
	}
	ruleW := width - lw - rw - 2
	if ruleW < 1 {
		return hbar(width, left, right)
	}
	return left + " " + styleFaint.Render(repeat("─", ruleW)) + " " + right
}

// sectionTitle is the simple label-only section heading.
func sectionTitle(label string, width int) string {
	return sectionHeader(styleSectionTitle.Render(label), "", width)
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
