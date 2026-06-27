package ui

import (
	"testing"

	"shubhxho/polytui/internal/api"
)

func makeHistory(n int) []api.PricePoint {
	pts := make([]api.PricePoint, n)
	v := 0.30
	for i := range pts {
		v += (float64((i*7)%11) - 5) / 200
		pts[i] = api.PricePoint{T: int64(i * 60), P: v}
	}
	return pts
}

// TestChartCacheReuse is the core performance guarantee: identical inputs must
// hit the cache (no rebuild), and any change to data or size must rebuild. This
// is what keeps the 60fps frame loop from redrawing the chart every tick.
func TestChartCacheReuse(t *testing.T) {
	c := newDetailCache()
	pts := makeHistory(50)

	first := c.chart(pts, 60, 12)
	if first == "" {
		t.Fatal("empty chart body")
	}
	sig := c.chartSig

	// Same inputs → same signature, byte-identical body, no rebuild.
	for i := 0; i < 5; i++ {
		if got := c.chart(pts, 60, 12); got != first {
			t.Fatal("cached chart body changed under identical inputs")
		}
		if c.chartSig != sig {
			t.Fatal("signature changed under identical inputs")
		}
	}

	// Changed data → rebuild.
	if c.chart(makeHistory(60), 60, 12); c.chartSig == sig {
		t.Fatal("signature did not change when data changed")
	}
	// Changed size → rebuild.
	prev := c.chartSig
	if c.chart(pts, 80, 12); c.chartSig == prev {
		t.Fatal("signature did not change when width changed")
	}
}

func TestBookCacheReuse(t *testing.T) {
	c := newDetailCache()
	book := &api.OrderBook{
		Bids: []api.OrderLevel{{Price: "0.39", Size: "1200"}, {Price: "0.40", Size: "800"}},
		Asks: []api.OrderLevel{{Price: "0.43", Size: "600"}, {Price: "0.42", Size: "1500"}},
	}
	first := c.book(book, 60, 4)
	sig := c.bookSig

	// A new book object with identical visible levels must reuse the cache.
	same := &api.OrderBook{
		Bids: []api.OrderLevel{{Price: "0.39", Size: "1200"}, {Price: "0.40", Size: "800"}},
		Asks: []api.OrderLevel{{Price: "0.43", Size: "600"}, {Price: "0.42", Size: "1500"}},
	}
	if got := c.book(same, 60, 4); got != first || c.bookSig != sig {
		t.Fatal("identical book contents did not hit cache")
	}

	// Changed size → rebuild.
	if c.book(book, 80, 4); c.bookSig == sig {
		t.Fatal("signature did not change when width changed")
	}
}
