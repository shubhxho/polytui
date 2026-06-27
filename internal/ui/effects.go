package ui

import (
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m model) spinner() string {
	return spinnerFrames[m.spinFrame%len(spinnerFrames)]
}

// wordmark is the compact one-color logo used in headers and overlays. It's
// static, so it's rendered once (lazily, after lipgloss has detected the
// terminal's color profile) and reused — it appears on every browse frame.
var (
	wordmarkOnce sync.Once
	wordmarkStr  string
)

func wordmark() string {
	wordmarkOnce.Do(func() {
		wordmarkStr = lipgloss.NewStyle().Foreground(purple).Bold(true).Render("◆ polytui")
	})
	return wordmarkStr
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
