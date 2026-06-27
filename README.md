# ◆ polytui

Polymarket, but it lives in your terminal.

Browse live prediction markets, watch the odds spring around in animated
probability bars, pull up a price chart and the live order book, and keep a
watchlist of the markets you care about — all without ever leaving your shell.

It's built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss) and
[Harmonica](https://github.com/charmbracelet/harmonica) (the spring physics
behind the bars), and styled in that clean Charm look — a connected tab bar,
pink/purple accents, glossy gradient bars, braille charts. No clutter.

One thing up front: **it's read-only.** It pulls straight from Polymarket's
public APIs, so there's no login, no API key, and nothing to configure. You also
can't place trades from it — it's for watching, not betting.

## Getting started

You'll need **Go 1.21 or newer** and a terminal that does 256/truecolor (most
modern ones do). A wider window is nicer — at ~96 columns the detail view opens
up into two side-by-side panes.

Grab it and run:

```sh
git clone https://github.com/shubhxho/polytui.git
cd polytui
go run .
```

Prefer a real binary you can drop on your `$PATH`? Build one:

```sh
go build -o polytui .
./polytui
```

That's the whole setup. No flags, no config files, no env vars to fiddle with —
just launch it. Hit `q` (or `ctrl+c`) whenever you want out.

## Getting around

There are three screens, and you move between them the obvious way: `enter` (or
`l`) goes deeper, `esc` backs out.

**The splash** is just a quick animated logo on launch — mash any key to skip it.

**Browse** is home base. Up top is a row of category tabs:

> `Trending · Politics · Sports · Crypto · World · Tech · Culture · ★ Watch`

Flip between them with `tab` / `shift+tab` (or `←` / `→`). Below sits the list of
markets for that tab, and it loads more as you scroll toward the bottom. Move
around with the arrow keys or `j`/`k`, jump to the ends with `g`/`G`, and the
mouse wheel works too.

A few things you'll reach for a lot here:

- `s` / `S` cycles how the list is sorted — **24h Volume → Total Volume →
  Liquidity → Ending Soon → Newest → Competitive**.
- `/` searches every active market. Type, `enter` to run, `esc` to back out.
- `w` stars whatever's selected and drops it in your Watch tab.
- `enter` / `l` opens a market up.

**Detail** is where a single market gets the full treatment:

- Every **outcome** gets its own probability bar that springs to value (and
  eases over when things move). Green means it's the favourite, pink the
  underdog. Use `j`/`k` to pick an outcome — the chart and order book follow
  along.
- The **price chart** is a braille line of that outcome's odds over time, green
  when it's climbing and pink when it's falling. Swap the time range with `t` /
  `T` (**6H → 1D → 1W → 1M → ALL**), and if you've got a mouse, hover the chart
  for a crosshair showing the exact date and price.
- The **order book** shows live bids and asks as a depth ladder mirrored around
  the spread — best prices in the middle, sizes fanning outward.
- `d` expands the market's full description, `w` stars it, `esc` takes you back.

Everything refreshes on its own every 15 seconds, and `r` forces it any time.

## The keys, all in one place

If you'd rather just have the cheat sheet (`?` brings this up in-app too):

**Anywhere**

| Key | Does |
| --- | --- |
| `?` | toggle the help overlay |
| `r` | refresh what you're looking at |
| `q` / `ctrl+c` | quit |

**Browse**

| Key | Does |
| --- | --- |
| `↑`/`k` · `↓`/`j` | move the selection |
| `tab` / `shift+tab` · `←` / `→` | switch category tab |
| `g` / `G` (`home` / `end`) | jump to top / bottom |
| `ctrl+u` / `ctrl+d` (`pgup` / `pgdn`) | page up / down |
| mouse wheel | scroll the list |
| `s` / `S` | cycle sort order |
| `/` | search markets |
| `w` | star / unstar |
| `enter` / `l` | open a market |

**Detail**

| Key | Does |
| --- | --- |
| `↑`/`k` · `↓`/`j` | pick an outcome |
| `t` / `T` | change the chart range |
| `d` | show / hide the description |
| `w` | star / unstar |
| hover the chart | crosshair with date + price |
| `esc` (`h` / `←` / `backspace`) | back to the list |

## Your watchlist sticks around

Star a market with `w` and it shows up under **★ Watch**. That list is saved to
disk, so it's still there next time you open the app — and each starred market
gets re-fetched fresh on launch. It lives at:

```
# macOS:  ~/Library/Application Support/polytui/watchlist.json
# Linux:  ~/.config/polytui/watchlist.json   (or $XDG_CONFIG_HOME/polytui/)
```

Want a market gone? Just press `w` on it again to toggle the star off — works
from the list or the detail screen.

## Under the hood

```
main.go                 program entry
internal/api/           the Polymarket Gamma + CLOB client and types
internal/ui/            the Bubble Tea model, views, components, animations
```

If you want to poke at it:

```sh
go test ./...            # unit + render tests, no network needed
NET=1 go test ./...      # also hits the live API
DUMP=1 go test ./internal/ui -run TestRender   # dump rendered screens to stdout
```

The TUI needs a real terminal, so for non-interactive checks it's easiest to
drive the model with messages (see `internal/ui/render_test.go`) instead of
launching the whole thing.
