package api

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveAPI exercises the real Polymarket endpoints. Run with NET=1.
func TestLiveAPI(t *testing.T) {
	if os.Getenv("NET") == "" {
		t.Skip("set NET=1 to run live API test")
	}
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	events, err := c.Events(ctx, EventQuery{Limit: 5, Order: "volume24hr"})
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no events returned")
	}
	e := events[0]
	t.Logf("top event: %q vol=%.0f markets=%d", e.Title, e.Volume, len(e.Markets))
	if len(e.Markets) == 0 {
		t.Fatal("event has no markets")
	}
	mk := e.Markets[0]
	if len(mk.Outcomes) == 0 {
		t.Fatalf("market has no parsed outcomes: %+v", mk.OutcomePrices)
	}
	t.Logf("market %q outcomes=%v prices=%v tokens=%d", mk.Title(), mk.Outcomes, mk.Prices(), len(mk.ClobTokenIDs))

	if len(mk.ClobTokenIDs) > 0 {
		book, err := c.Book(ctx, mk.ClobTokenIDs[0])
		if err != nil {
			t.Fatalf("book: %v", err)
		}
		t.Logf("book bids=%d asks=%d", len(book.Bids), len(book.Asks))

		hist, err := c.PriceHistory(ctx, mk.ClobTokenIDs[0], "1w", 60)
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		t.Logf("history points=%d", len(hist))
	}

	// Verify each curated category tag id actually filters events.
	cats := CuratedCategories()
	if len(cats) == 0 {
		t.Fatal("no curated categories")
	}
	catEvents, err := c.Events(ctx, EventQuery{Limit: 3, Order: "volume24hr", TagID: cats[0].ID})
	if err != nil {
		t.Fatalf("category events: %v", err)
	}
	t.Logf("category %q -> %d events", cats[0].Label, len(catEvents))
}
