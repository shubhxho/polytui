package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// browseTab is a top-level navigation tab. A tab is either Trending (no tag),
// a category (tagID), or the Watch list.
type browseTab struct {
	label string
	tagID string
	watch bool
}

var browseTabs = []browseTab{
	{label: "Trending"},
	{label: "Politics", tagID: "2"},
	{label: "Sports", tagID: "1"},
	{label: "Crypto", tagID: "21"},
	{label: "World", tagID: "101970"},
	{label: "Tech", tagID: "1401"},
	{label: "Culture", tagID: "596"},
	{label: "★ Watch", watch: true},
}

// Connected tab borders, à la the lipgloss tabs example: the active tab's
// bottom opens into the content area below it.
var (
	activeTabBorder = lipgloss.Border{
		Top: "─", Bottom: " ", Left: "│", Right: "│",
		TopLeft: "╭", TopRight: "╮", BottomLeft: "┘", BottomRight: "└",
	}
	inactiveTabBorder = lipgloss.Border{
		Top: "─", Bottom: "─", Left: "│", Right: "│",
		TopLeft: "╭", TopRight: "╮", BottomLeft: "┴", BottomRight: "┴",
	}
	tabStyle = lipgloss.NewStyle().
			Border(inactiveTabBorder, true).
			BorderForeground(faint).
			Foreground(muted).
			Padding(0, 1)
	activeTabStyle = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(purple).
			Foreground(fg).
			Bold(true).
			Padding(0, 1)
	tabGapStyle = lipgloss.NewStyle().
			Border(inactiveTabBorder, false, false, true, false).
			BorderForeground(faint)
)

// renderTabs draws the tab bar, windowing around the active tab if it would
// overflow the available width.
func (m model) renderTabs(width int) string {
	// Pre-render every tab and record its display width.
	rendered := make([]string, len(browseTabs))
	widths := make([]int, len(browseTabs))
	for i, t := range browseTabs {
		st := tabStyle
		if i == m.activeTab {
			st = activeTabStyle
		}
		rendered[i] = st.Render(t.label)
		widths[i] = lipgloss.Width(rendered[i])
	}

	// Choose a contiguous window [start,end) containing activeTab that fits.
	start, end := windowTabs(widths, m.activeTab, width-2)

	var parts []string
	if start > 0 {
		parts = append(parts, tabStyle.Render("‹"))
	}
	for i := start; i < end; i++ {
		parts = append(parts, rendered[i])
	}
	if end < len(browseTabs) {
		parts = append(parts, tabStyle.Render("›"))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
	used := lipgloss.Width(row)
	if gap := width - used; gap > 0 {
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, tabGapStyle.Render(strings.Repeat(" ", gap)))
	}
	return row
}

func windowTabs(widths []int, active, budget int) (int, int) {
	// Grow outward from the active tab while it fits.
	start, end := active, active+1
	total := widths[active]
	for {
		grew := false
		if end < len(widths) && total+widths[end] <= budget {
			total += widths[end]
			end++
			grew = true
		}
		if start > 0 && total+widths[start-1] <= budget {
			total += widths[start-1]
			start--
			grew = true
		}
		if !grew {
			break
		}
	}
	return start, end
}

func (m model) currentTab() browseTab { return browseTabs[m.activeTab] }
