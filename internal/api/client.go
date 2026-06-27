package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	gammaBase = "https://gamma-api.polymarket.com"
	clobBase  = "https://clob.polymarket.com"
)

// Client talks to Polymarket's public Gamma and CLOB APIs.
type Client struct {
	http *http.Client
}

// New returns a ready-to-use API client.
func New() *Client {
	return &Client{http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "polytui/1.0")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d from %s", resp.StatusCode, u)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// EventQuery describes how to fetch the events list.
type EventQuery struct {
	Limit  int
	Offset int
	Order  string // e.g. "volume24hr", "volume", "liquidity", "startDate"
	Asc    bool
	TagID  string // filter by category tag id
	Search string // text search
}

// Events fetches active, open events with the given query.
func (c *Client) Events(ctx context.Context, q EventQuery) ([]Event, error) {
	v := url.Values{}
	limit := q.Limit
	if limit <= 0 {
		limit = 60
	}
	v.Set("limit", strconv.Itoa(limit))
	v.Set("offset", strconv.Itoa(q.Offset))
	v.Set("active", "true")
	v.Set("closed", "false")
	v.Set("archived", "false")
	if q.Order != "" {
		v.Set("order", q.Order)
		if q.Asc {
			v.Set("ascending", "true")
		} else {
			v.Set("ascending", "false")
		}
	}
	if q.TagID != "" {
		v.Set("tag_id", q.TagID)
	}
	endpoint := gammaBase + "/events?" + v.Encode()
	if q.Search != "" {
		// Gamma exposes search via a dedicated public endpoint.
		return c.searchEvents(ctx, q.Search, limit)
	}
	var events []Event
	if err := c.getJSON(ctx, endpoint, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// EventsByIDs fetches specific events by their ids (used for the watchlist).
func (c *Client) EventsByIDs(ctx context.Context, ids []string) ([]Event, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	v := url.Values{}
	for _, id := range ids {
		v.Add("id", id)
	}
	v.Set("limit", strconv.Itoa(len(ids)))
	endpoint := gammaBase + "/events?" + v.Encode()
	var events []Event
	if err := c.getJSON(ctx, endpoint, &events); err != nil {
		return nil, err
	}
	return events, nil
}

type searchResp struct {
	Events []Event `json:"events"`
}

func (c *Client) searchEvents(ctx context.Context, term string, limit int) ([]Event, error) {
	v := url.Values{}
	v.Set("q", term)
	v.Set("limit_per_type", strconv.Itoa(limit))
	v.Set("events_status", "active")
	endpoint := gammaBase + "/public-search?" + v.Encode()
	var sr searchResp
	if err := c.getJSON(ctx, endpoint, &sr); err != nil {
		return nil, err
	}
	// Keep only tradable, open events.
	out := sr.Events[:0]
	for _, e := range sr.Events {
		if !e.Closed && len(e.Markets) > 0 {
			out = append(out, e)
		}
	}
	return out, nil
}

// CuratedCategories returns the curated top-level Polymarket categories with
// their stable Gamma tag IDs.
func CuratedCategories() []Tag {
	return []Tag{
		{ID: "2", Label: "Politics", Slug: "politics"},
		{ID: "144", Label: "Elections", Slug: "elections"},
		{ID: "126", Label: "Trump", Slug: "trump"},
		{ID: "100265", Label: "Geopolitics", Slug: "geopolitics"},
		{ID: "101970", Label: "World", Slug: "world"},
		{ID: "1", Label: "Sports", Slug: "sports"},
		{ID: "100350", Label: "Soccer", Slug: "soccer"},
		{ID: "28", Label: "Basketball", Slug: "basketball"},
		{ID: "450", Label: "NFL", Slug: "nfl"},
		{ID: "21", Label: "Crypto", Slug: "crypto"},
		{ID: "100328", Label: "Economy", Slug: "economy"},
		{ID: "107", Label: "Business", Slug: "business"},
		{ID: "1401", Label: "Tech", Slug: "tech"},
		{ID: "439", Label: "AI", Slug: "ai"},
		{ID: "74", Label: "Science", Slug: "science"},
		{ID: "596", Label: "Culture", Slug: "pop-culture"},
		{ID: "53", Label: "Movies", Slug: "movies"},
		{ID: "100", Label: "Music", Slug: "music"},
	}
}

// Book fetches the CLOB order book for a single outcome token.
func (c *Client) Book(ctx context.Context, tokenID string) (*OrderBook, error) {
	endpoint := clobBase + "/book?token_id=" + url.QueryEscape(tokenID)
	var b OrderBook
	if err := c.getJSON(ctx, endpoint, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// PriceHistory fetches historical mid prices for a token.
// interval is one of: 1h, 6h, 1d, 1w, 1m, max. fidelity is in minutes.
func (c *Client) PriceHistory(ctx context.Context, tokenID, interval string, fidelity int) ([]PricePoint, error) {
	v := url.Values{}
	v.Set("market", tokenID)
	v.Set("interval", interval)
	if fidelity > 0 {
		v.Set("fidelity", strconv.Itoa(fidelity))
	}
	endpoint := clobBase + "/prices-history?" + v.Encode()
	var pr priceHistoryResp
	if err := c.getJSON(ctx, endpoint, &pr); err != nil {
		return nil, err
	}
	return pr.History, nil
}
