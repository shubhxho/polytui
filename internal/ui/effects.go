package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	colorful "github.com/lucasb-eyer/go-colorful"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m model) spinner() string {
	return spinnerFrames[m.spinFrame%len(spinnerFrames)]
}

// gradientText colors each rune of s along a smooth gradient from→to. Used
// once, for the wordmark — the single accent gradient in the whole UI.
func gradientText(s string, from, to lipgloss.Color) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	c1, _ := colorful.Hex(string(from))
	c2, _ := colorful.Hex(string(to))
	var sb strings.Builder
	n := len(runes)
	for i, r := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		c := c1.BlendLab(c2, t).Clamped()
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex())).Bold(true).Render(string(r)))
	}
	return sb.String()
}

// wordmark renders the app name in the pink→purple gradient.
func wordmark() string {
	return gradientText("polytui", pink, purple)
}
