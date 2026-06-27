# polytui — agent notes

Polymarket TUI client. Bubble Tea + Lip Gloss + Harmonica. Read-only against
Polymarket public APIs.

## Architecture

- `internal/api` — HTTP client + types.
  - Gamma (`gamma-api.polymarket.com`): `Events`, `searchEvents` (`/public-search`),
    curated category tag ids (`CuratedCategories`).
  - CLOB (`clob.polymarket.com`): `Book`, `PriceHistory`.
  - Polymarket quirk: `outcomes`, `outcomePrices`, `clobTokenIds` arrive as
    JSON arrays encoded *inside* a JSON string — handled by `jsonStringArray`.
  - Order book quirk: both sides put the **best price at the tail** (bids
    ascending, asks descending). `topLevels` takes the tail and reverses.
- `internal/ui` — Bubble Tea root `model` with three screens
  (`screenSplash`, `screenBrowse`, `screenDetail`) plus help / search overlays.
  Update routing lives in `update.go`; views in `views.go` + `detail.go`.
  - Aesthetic: canonical Lip Gloss look — `tabs.go` renders the connected tab
    bar (Trending / categories / Watch); `theme.go` holds the small pink/purple
    palette, `statusBar`, and `sectionTitle` (underlined headers). Keep it
    minimal — one accent gradient (the `wordmark`), no heavy boxes.
  - Watch tab: starred event ids persist (`watchlist`); full event data is held
    in `model.watchEvents`, populated on star and re-fetched by id on launch via
    `EventsByIDs`.
  - Animation: a continuous 60fps `frameMsg` tick (`animTick`) drives Harmonica
    `springBar`s (probability bars) and the splash reveal. `refreshMsg` (15s)
    re-fetches the current view.
  - Components in `components.go`: `springBar` (Harmonica), `sparkline` /
    `chartBlock` (area chart), `orderBookView` (depth bars). Effects/gradients
    in `effects.go`.

## Conventions

- Money via `fmtUSD`, probabilities via `fmtPct` / `fmtCents`.
- New animated bars start at 0 and animate up; refreshes update the target only.

## Testing

- `go test ./...` — render + unit tests (no network).
- `NET=1 go test ./...` — live API integration.
- `DUMP=1 go test ./internal/ui -run TestRender` — dump rendered screens to stdout.
- TUI needs a TTY; for non-interactive verification drive the model with
  messages (see `render_test.go`) rather than launching the program.
