package ui

import "github.com/charmbracelet/lipgloss"

// Palette — the Lip Gloss signature: pink + purple accents on a dark ground,
// with a green "special" for the up/positive side. Deliberately small.
var (
	pink   = lipgloss.Color("#FF5F87")
	pinkHi = lipgloss.Color("#F25D94")
	purple = lipgloss.Color("#7D56F4")
	mauve  = lipgloss.Color("#AD58F6")
	green  = lipgloss.Color("#73F59F")

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
		return lipgloss.Color("#D7D7D7")
	}
}

// sectionTitle renders a bold heading followed by a thin rule across width.
func sectionTitle(label string, width int) string {
	t := styleSectionTitle.Render(label)
	tw := lipgloss.Width(t)
	rule := width - tw - 1
	if rule < 0 {
		rule = 0
	}
	line := t + " " + styleFaint.Render(repeat("─", rule))
	return line
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
