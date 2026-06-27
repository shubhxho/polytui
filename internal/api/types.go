package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// jsonStringArray decodes Polymarket's quirky fields that are JSON-encoded
// arrays stored *inside* a JSON string, e.g. "[\"Yes\", \"No\"]".
type jsonStringArray []string

func (a *jsonStringArray) UnmarshalJSON(data []byte) error {
	// First try a plain array.
	var direct []string
	if err := json.Unmarshal(data, &direct); err == nil {
		*a = direct
		return nil
	}
	// Otherwise it's a string containing a JSON array.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		*a = nil
		return nil
	}
	var inner []string
	if err := json.Unmarshal([]byte(s), &inner); err != nil {
		return err
	}
	*a = inner
	return nil
}

// Market is a single tradable outcome pair (Yes/No) on Polymarket.
type Market struct {
	ID            string          `json:"id"`
	Question      string          `json:"question"`
	Slug          string          `json:"slug"`
	Description   string          `json:"description"`
	Image         string          `json:"image"`
	Icon          string          `json:"icon"`
	ConditionID   string          `json:"conditionId"`
	GroupTitle    string          `json:"groupItemTitle"`
	Outcomes      jsonStringArray `json:"outcomes"`
	OutcomePrices jsonStringArray `json:"outcomePrices"`
	ClobTokenIDs  jsonStringArray `json:"clobTokenIds"`
	Volume        string          `json:"volume"`
	VolumeNum     float64         `json:"volumeNum"`
	Volume24hr    float64         `json:"volume24hr"`
	Liquidity     string          `json:"liquidity"`
	LiquidityNum  float64         `json:"liquidityNum"`
	EndDate       string          `json:"endDate"`
	Active        bool            `json:"active"`
	Closed        bool            `json:"closed"`
	EnableOrderBk bool            `json:"enableOrderBook"`
	BestBid       float64         `json:"bestBid"`
	BestAsk       float64         `json:"bestAsk"`
	TickSize      float64         `json:"orderPriceMinTickSize"`
}

// Prices returns the parsed outcome prices aligned with Outcomes.
func (m Market) Prices() []float64 {
	out := make([]float64, len(m.OutcomePrices))
	for i, p := range m.OutcomePrices {
		f, _ := strconv.ParseFloat(p, 64)
		out[i] = f
	}
	return out
}

// YesPrice returns the probability of the "Yes"/first outcome (0..1).
func (m Market) YesPrice() float64 {
	p := m.Prices()
	if len(p) == 0 {
		return 0
	}
	return p[0]
}

// Title prefers the grouped item title (used inside multi-outcome events).
func (m Market) Title() string {
	if m.GroupTitle != "" {
		return m.GroupTitle
	}
	return m.Question
}

// Event groups one or more related markets.
type Event struct {
	ID           string   `json:"id"`
	Ticker       string   `json:"ticker"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`
	Icon         string   `json:"icon"`
	Volume       float64  `json:"volume"`
	Volume24hr   float64  `json:"volume24hr"`
	Liquidity    float64  `json:"liquidity"`
	OpenInterest float64  `json:"openInterest"`
	CommentCount int      `json:"commentCount"`
	EndDate      string   `json:"endDate"`
	Active       bool     `json:"active"`
	Closed       bool     `json:"closed"`
	Featured     bool     `json:"featured"`
	New          bool     `json:"new"`
	Markets      []Market `json:"markets"`
	Tags         []Tag    `json:"tags"`
}

// Binary reports whether this event is a simple single Yes/No market.
func (e Event) Binary() bool {
	return len(e.Markets) == 1
}

// TopPrice returns the headline probability to show in a list row.
func (e Event) TopPrice() float64 {
	if len(e.Markets) == 0 {
		return 0
	}
	if e.Binary() {
		return e.Markets[0].YesPrice()
	}
	// For multi-outcome events surface the current front-runner.
	best := 0.0
	for _, m := range e.Markets {
		if p := m.YesPrice(); p > best {
			best = p
		}
	}
	return best
}

// Leader returns the leading market title and price for multi-outcome events.
func (e Event) Leader() (string, float64) {
	best := 0.0
	title := ""
	for _, m := range e.Markets {
		if p := m.YesPrice(); p > best {
			best = p
			title = m.Title()
		}
	}
	return title, best
}

// EndsAt parses the end date, returning zero time on failure.
func (e Event) EndsAt() time.Time {
	t, err := time.Parse(time.RFC3339, e.EndDate)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Tag is a Polymarket category label.
type Tag struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

// OrderLevel is one price level in the order book.
type OrderLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

func (l OrderLevel) PriceF() float64 { f, _ := strconv.ParseFloat(l.Price, 64); return f }
func (l OrderLevel) SizeF() float64  { f, _ := strconv.ParseFloat(l.Size, 64); return f }

// OrderBook is the CLOB order book for a single token.
type OrderBook struct {
	Market    string       `json:"market"`
	AssetID   string       `json:"asset_id"`
	Timestamp string       `json:"timestamp"`
	Bids      []OrderLevel `json:"bids"`
	Asks      []OrderLevel `json:"asks"`
}

// PricePoint is a single historical price sample.
type PricePoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

type priceHistoryResp struct {
	History []PricePoint `json:"history"`
}
