package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// detailWithHistory returns a settled detail-screen model showing the
// multi-outcome event with price history loaded.
func detailWithHistory(t *testing.T) model {
	t.Helper()
	m := New()
	m = drive(m,
		tea.WindowSizeMsg{Width: 110, Height: 38},
		eventsMsg{events: sampleEvents()},
	)
	m.cursor = 1
	nm, _ := m.handleBrowseKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(model)
	m = drive(m, historyMsg{tokenID: m.histToken, points: makeHistory(60)})
	return m
}

// TestChartHoverOverlay verifies the mouse-hover crosshair + tooltip are drawn
// onto the chart and that hovering changes the rendered output. makeHistory
// spans under 36h, so the X-axis uses clock labels (no "Jan"); the tooltip
// always formats as "Jan 2", giving a clean discriminator.
func TestChartHoverOverlay(t *testing.T) {
	m := detailWithHistory(t)

	m.hoverChart = false
	base := m.viewDetail()
	if strings.Contains(base, "Jan") {
		t.Fatal("precondition failed: base chart should not contain a 'Jan' label")
	}

	m.hoverChart = true
	m.hoverIdx = 10
	m.hoverCol = 20
	hov := m.viewDetail()

	if hov == base {
		t.Fatal("hover did not change the rendered detail view")
	}
	if !strings.Contains(hov, "Jan") {
		t.Fatal("hover tooltip (date) not rendered")
	}
}

// TestSpliceCell checks the ANSI-aware cell splice used by the crosshair.
func TestSpliceCell(t *testing.T) {
	got := spliceCell("abcdef", 2, "X")
	if got != "abXdef" {
		t.Fatalf("splice into middle: got %q want %q", got, "abXdef")
	}
	// Splicing past the end pads with spaces.
	got = spliceCell("ab", 4, "X")
	if got != "ab  X" {
		t.Fatalf("splice past end: got %q want %q", got, "ab  X")
	}
}

// TestAnimIdlesWhenSettled is the core performance property: once every spring
// has settled the 60fps tick stops (returns a nil cmd) instead of running forever.
func TestAnimIdlesWhenSettled(t *testing.T) {
	m := New()
	m = drive(m,
		tea.WindowSizeMsg{Width: 110, Height: 38},
		eventsMsg{events: sampleEvents()},
	)
	m.screen = screenBrowse

	nm, cmd := m.Update(frameMsg{})
	m = nm.(model)
	if cmd != nil {
		t.Fatal("expected the animation tick to idle (nil cmd) after springs settled")
	}
	if m.animRunning {
		t.Fatal("animRunning should be false while idle")
	}
}

// TestAnimRearmsOnInteraction verifies the idle tick is restarted when an
// interaction starts new motion (a sort reload that animates fresh bars).
func TestAnimRearmsOnInteraction(t *testing.T) {
	m := New()
	m = drive(m,
		tea.WindowSizeMsg{Width: 110, Height: 38},
		eventsMsg{events: sampleEvents()},
	)
	m.screen = screenBrowse
	// Settle into the idle state.
	nm, _ := m.Update(frameMsg{})
	m = nm.(model)
	if m.animRunning {
		t.Fatal("precondition: expected idle before interaction")
	}

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = nm.(model)
	if !m.animRunning {
		t.Fatal("expected the animation loop to re-arm after a sort keypress")
	}
	if cmd == nil {
		t.Fatal("expected a batched command (reload + animTick)")
	}
}

// TestHoverClearedOnExit ensures leaving the detail screen clears hover state.
func TestHoverClearedOnExit(t *testing.T) {
	m := detailWithHistory(t)
	m.hoverChart = true
	nm, _ := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(model)
	if m.hoverChart {
		t.Fatal("hover should be cleared when leaving the detail screen")
	}
}
