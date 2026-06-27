package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"shubhxho/polytui/internal/api"
)

func sampleEvents() []api.Event {
	mk := func(title string, yes float64) api.Market {
		return api.Market{
			ID:            "m" + title,
			Question:      title + "?",
			GroupTitle:    title,
			Outcomes:      []string{"Yes", "No"},
			OutcomePrices: []string{fmt.Sprintf("%.2f", yes), fmt.Sprintf("%.2f", 1-yes)},
			ClobTokenIDs:  []string{"123", "456"},
			VolumeNum:     12345,
		}
	}
	return []api.Event{
		{
			ID: "1", Title: "Will Bitcoin hit $200k before 2027?",
			Volume: 4_200_000, Volume24hr: 152_000, Liquidity: 88_000,
			EndDate: "2027-01-01T00:00:00Z",
			Markets: []api.Market{mk("Yes", 0.62)},
		},
		{
			ID: "2", Title: "Who will win the 2028 election?",
			Volume: 18_900_000, Volume24hr: 980_000, Liquidity: 410_000,
			OpenInterest: 320_000, EndDate: "2028-11-07T00:00:00Z",
			Description: "This market resolves to the winner of the 2028 US presidential election as called by major outlets.",
			Markets: []api.Market{
				mk("Candidate A", 0.41), mk("Candidate B", 0.33),
				mk("Candidate C", 0.18), mk("Candidate D", 0.08),
			},
		},
		{
			ID: "3", Title: "New Rihanna Album before GTA VI?",
			Volume: 842_000, Volume24hr: 1_188, Liquidity: 19_300,
			EndDate: "2026-07-31T12:00:00Z",
			Markets: []api.Market{mk("Yes", 0.52)},
		},
	}
}

func drive(m model, msgs ...tea.Msg) model {
	for _, msg := range msgs {
		nm, _ := m.Update(msg)
		m = nm.(model)
	}
	// settle springs
	for i := 0; i < 200; i++ {
		nm, _ := m.Update(frameMsg{})
		m = nm.(model)
	}
	return m
}

func TestRenderBrowse(t *testing.T) {
	m := New()
	m = drive(m,
		tea.WindowSizeMsg{Width: 110, Height: 38},
		eventsMsg{events: sampleEvents()},
	)
	m.screen = screenBrowse
	out := m.View()
	if !strings.Contains(out, "Trending") {
		t.Fatal("missing tab bar")
	}
	if !strings.Contains(out, "Bitcoin") {
		t.Fatal("missing event title")
	}
	if os.Getenv("DUMP") != "" {
		fmt.Println("\n===== BROWSE =====")
		fmt.Println(out)
	}
}

func TestRenderDetail(t *testing.T) {
	m := New()
	m = drive(m,
		tea.WindowSizeMsg{Width: 110, Height: 38},
		eventsMsg{events: sampleEvents()},
	)
	// open the multi-outcome event
	m.cursor = 1
	nm, _ := m.handleBrowseKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(model)
	// feed history + book
	hist := make([]api.PricePoint, 0, 50)
	v := 0.30
	for i := 0; i < 50; i++ {
		v += (float64((i*7)%11) - 5) / 200
		hist = append(hist, api.PricePoint{T: int64(i), P: v})
	}
	// Polymarket orders both sides with the best price at the tail (bids
	// ascending, asks descending).
	book := &api.OrderBook{
		Bids: []api.OrderLevel{{Price: "0.39", Size: "1200"}, {Price: "0.40", Size: "800"}, {Price: "0.41", Size: "2500"}},
		Asks: []api.OrderLevel{{Price: "0.44", Size: "3300"}, {Price: "0.43", Size: "600"}, {Price: "0.42", Size: "1500"}},
	}
	m = drive(m,
		historyMsg{tokenID: m.histToken, points: hist},
		bookMsg{tokenID: m.bookToken, book: book},
	)
	m.descExpanded = true
	out := m.View()
	if !strings.Contains(out, "Order book") {
		t.Fatal("missing order book")
	}
	if os.Getenv("DUMP") != "" {
		fmt.Println("\n===== DETAIL =====")
		fmt.Println(out)
	}
}

func TestRenderLive(t *testing.T) {
	if os.Getenv("NET") == "" {
		t.Skip("set NET=1 for live render")
	}
	m := New()
	cmd := loadEvents(m.client, m.query(0), false)
	msg := cmd()
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, msg)
	m.screen = screenBrowse
	if os.Getenv("DUMP") != "" {
		fmt.Println("\n===== LIVE BROWSE =====")
		fmt.Println(m.View())
	}
	// open first event and fetch its real book + history
	nm, _ := m.handleBrowseKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(model)
	if m.bookToken != "" {
		m = drive(m, loadBook(m.client, m.bookToken)(), loadHistory(m.client, m.histToken, "1w", 60)())
	}
	if os.Getenv("DUMP") != "" {
		fmt.Println("\n===== LIVE DETAIL =====")
		fmt.Println(m.View())
		fmt.Println("\n===== WATCH TAB =====")
		m.screen = screenBrowse
		m.activeTab = len(browseTabs) - 1
		fmt.Println(m.View())
	}
}

func TestRenderSplashAndHelp(t *testing.T) {
	m := New()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 38})
	m = nm.(model)
	for i := 0; i < 20; i++ {
		x, _ := m.Update(frameMsg{})
		m = x.(model)
	}
	if os.Getenv("DUMP") != "" {
		fmt.Println("\n===== SPLASH =====")
		fmt.Println(m.viewSplash())
		fmt.Println("\n===== HELP =====")
		fmt.Println(m.viewHelp())
	}
}
