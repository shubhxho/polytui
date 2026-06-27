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
  - Animation: a 60fps `frameMsg` tick (`animTick`) drives Harmonica
    `springBar`s (probability bars) and the splash reveal. The loop is
    **self-idling**: `tickAnim` stops scheduling frames once every spring has
    settled (and no spinner/load is pending), dropping a still screen to 0fps.
    `Update` routes non-frame messages through `route()` then calls `maybeArm`,
    which re-arms the tick on the next interaction/data message (`animRunning`
    guards against double chains; safe because Bubble Tea serializes `Update`).
    `refreshMsg` (15s) re-fetches the current view.
  - Components in `components.go`: `springBar` (Harmonica), `orderBookView`
    (mirrored bid/ask depth ladder around a center divider). Effects in
    `effects.go` (`wordmark` is rendered once via `sync.Once`). `springBar`
    renders a continuous rail (`renderBarRun`): a dim `█` track behind a vivid
    `█` fill that carries a glossy accent→white gradient, quantised into
    `gradientSegs` colour bands so each bar is a bounded handful of
    `lipgloss.Render` calls, not one per cell.
  - Price chart: `chart.go` renders an `ntcharts` braille
    `timeserieslinechart` (NimbleMarkets/ntcharts, the v0.x line — it tracks
    Charm v1; the `/v2` module needs Charm v2 and is incompatible). The view
    fits the data's min/max so the line fills the panel; green rising / pink
    falling; span-aware X labels (clock vs date) via `XLabelFormatter`.
  - Mouse hover: the chart body is wrapped with a `bubblezone` mark
    (`zoneChart`); `model.zone` (a `zone.Manager`) records its bounds when
    `View` calls `zone.Scan` (detail screen only). Mouse-motion events hit
    `updateChartHover`, which maps the column (via `ts.Origin().X` /
    `GraphWidth()`) to the nearest `history` sample; `chartHover` composites a
    pink crosshair + date/price tooltip onto the *cached* chart string
    (`spliceCell`, ANSI-aware) without busting the data cache.
  - Perf: the detail screen renders every frame, so the chart, order book, and
    tab bar are memoised in `detailCache` (`model.renderCache`, a pointer so the
    value-receiver `View` can fill it). Each frame is an O(1) signature compare
    (`chartSig`/`bookSig`/tab sig) returning a cached string; they rebuild only
    when the data, size, or range actually change.

## Conventions

- Money via `fmtUSD`, probabilities via `fmtPct` / `fmtCents`.
- New animated bars start at 0 and animate up; refreshes update the target only.

## Testing

- `go test ./...` — render + unit tests (no network).
- `NET=1 go test ./...` — live API integration.
- `DUMP=1 go test ./internal/ui -run TestRender` — dump rendered screens to stdout.
- TUI needs a TTY; for non-interactive verification drive the model with
  messages (see `render_test.go`) rather than launching the program.
