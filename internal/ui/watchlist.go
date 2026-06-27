package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// watchlist is a persisted set of saved event IDs.
type watchlist struct {
	path string
	ids  map[string]bool
}

func loadWatchlist() *watchlist {
	w := &watchlist{ids: map[string]bool{}}
	dir, err := os.UserConfigDir()
	if err == nil {
		w.path = filepath.Join(dir, "polytui", "watchlist.json")
		if data, err := os.ReadFile(w.path); err == nil {
			var ids []string
			if json.Unmarshal(data, &ids) == nil {
				for _, id := range ids {
					w.ids[id] = true
				}
			}
		}
	}
	return w
}

func (w *watchlist) has(id string) bool { return w.ids[id] }

func (w *watchlist) toggle(id string) bool {
	if w.ids[id] {
		delete(w.ids, id)
	} else {
		w.ids[id] = true
	}
	w.save()
	return w.ids[id]
}

func (w *watchlist) list() []string {
	out := make([]string, 0, len(w.ids))
	for id := range w.ids {
		out = append(out, id)
	}
	return out
}

func (w *watchlist) save() {
	if w.path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(w.path), 0o755)
	data, err := json.Marshal(w.list())
	if err == nil {
		_ = os.WriteFile(w.path, data, 0o644)
	}
}

// sortOption describes an ordering for the events list.
type sortOption struct {
	label string
	order string // gamma `order` field
	asc   bool
}

var sortOptions = []sortOption{
	{"24h Volume", "volume24hr", false},
	{"Total Volume", "volume", false},
	{"Liquidity", "liquidity", false},
	{"Ending Soon", "endDate", true},
	{"Newest", "startDate", false},
	{"Competitive", "competitive", false},
}
