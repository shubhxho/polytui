# ◆ polytui

A fully-fledged **Polymarket terminal client** — browse live prediction
markets, inspect order books and price history, and track a watchlist, all
inside a smooth, animated TUI.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss) and
[Harmonica](https://github.com/charmbracelet/harmonica) (spring physics power
the animated probability bars).

Styled in the canonical Charm aesthetic — connected **tab bar**, pink/purple
accents, underlined section headers and a status bar — kept deliberately minimal.

## Features

- **Category tabs** — Trending, Politics, Sports, Crypto, World, Tech, Culture
  and your Watch list, navigated with `tab` / arrows.
- **Animated probability bars** — each market's odds spring smoothly to value
  using Harmonica spring physics, and animate up on first reveal.
- **Sorting** — by 24h volume, total volume, liquidity, ending-soon, newest or
  competitiveness.
- **Market detail view** — multi-outcome breakdown, an area **price-history
  chart** (6H / 1D / 1W / 1M / ALL ranges) and a live **order book** with depth
  bars and spread.
- **Search** — full-text search across active markets.
- **Watchlist** — star markets; persisted to disk (`$XDG_CONFIG_HOME/polytui`)
  and re-fetched by id on launch.
- **Auto-refresh** every 15s, plus mouse-wheel navigation.
- **Animated splash** with a minimal gradient reveal.

All data comes from Polymarket's public APIs — `gamma-api.polymarket.com`
(markets/events) and `clob.polymarket.com` (order book + price history). No
account or API key required; this is a read-only client.

## Run

```sh
go run .
# or build a binary
go build -o polytui . && ./polytui
```

## Keybindings

| Key | Action |
| --- | --- |
| `↑`/`k` `↓`/`j` | move selection |
| `tab` / `shift+tab` · `←` `→` | switch category tab |
| `g` / `G` | jump to top / bottom |
| `ctrl+u` / `ctrl+d` | page up / down |
| `enter` / `l` | open market detail |
| `esc` | back |
| `s` / `S` | cycle sort order |
| `/` | search markets |
| `w` | star / unstar (watch) |
| `t` / `T` | (detail) change chart range |
| `d` | (detail) expand description |
| `r` | refresh |
| `?` | help |
| `q` / `ctrl+c` | quit |

## Project layout

```
main.go                 program entry
internal/api/           Polymarket Gamma + CLOB client and types
internal/ui/            Bubble Tea model, views, components, animations
```

## Tests

```sh
go test ./...            # unit / render tests
NET=1 go test ./...      # also run live API integration tests
DUMP=1 go test ./internal/ui -run TestRender   # print rendered screens
```
