package ui

import "github.com/charmbracelet/lipgloss"

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m model) spinner() string {
	return spinnerFrames[m.spinFrame%len(spinnerFrames)]
}

// wordmark is the compact one-color logo used in headers and overlays.
func wordmark() string {
	return lipgloss.NewStyle().Foreground(purple).Bold(true).Render("◆ polytui")
}

// wordmarkLarge is the framed logo shown on the splash — letter-spaced name in
// a single accent color inside a rounded frame.
func wordmarkLarge() string {
	name := lipgloss.NewStyle().Foreground(purple).Bold(true).Render("◆  p o l y t u i")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 4).
		Render(name)
}
