package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"shubhxho/polytui/internal/api"
)

// ---- Messages ------------------------------------------------------------

type eventsMsg struct {
	events []api.Event
	append bool
	watch  bool
	err    error
}

type bookMsg struct {
	tokenID string
	book    *api.OrderBook
	err     error
}

type historyMsg struct {
	tokenID  string
	interval string
	points   []api.PricePoint
	err      error
}

type imageMsg struct {
	url string
	png []byte
	err error
}

// frameMsg drives the animation loop.
type frameMsg time.Time

// refreshMsg triggers a periodic data refresh of the current view.
type refreshMsg time.Time

const (
	animFPS       = 60
	refreshPeriod = 15 * time.Second
)

func animTick() tea.Cmd {
	return tea.Tick(time.Second/animFPS, func(t time.Time) tea.Msg {
		return frameMsg(t)
	})
}

func refreshTick() tea.Cmd {
	return tea.Tick(refreshPeriod, func(t time.Time) tea.Msg {
		return refreshMsg(t)
	})
}

// ---- Commands ------------------------------------------------------------

func loadEvents(c *api.Client, q api.EventQuery, appendMode bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		events, err := c.Events(ctx, q)
		return eventsMsg{events: events, append: appendMode, err: err}
	}
}

func loadWatchEvents(c *api.Client, ids []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		events, err := c.EventsByIDs(ctx, ids)
		return eventsMsg{events: events, watch: true, err: err}
	}
}

// loadImage fetches and prepares an event thumbnail for the Kitty renderer.
func loadImage(url string) tea.Cmd {
	return func() tea.Msg {
		imageSem <- struct{}{} // bound concurrent fetch+decode jobs
		defer func() { <-imageSem }()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		png, err := fetchAndPreparePNG(ctx, url, imgMaxPixels)
		return imageMsg{url: url, png: png, err: err}
	}
}

func loadBook(c *api.Client, tokenID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		book, err := c.Book(ctx, tokenID)
		return bookMsg{tokenID: tokenID, book: book, err: err}
	}
}

func loadHistory(c *api.Client, tokenID, interval string, fidelity int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		pts, err := c.PriceHistory(ctx, tokenID, interval, fidelity)
		return historyMsg{tokenID: tokenID, interval: interval, points: pts, err: err}
	}
}
