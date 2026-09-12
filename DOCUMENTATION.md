# torgo Documentation

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [CLI Flags](#cli-flags)
- [Configuration](#configuration)
- [Theme System](#theme-system)
- [Client Adapters](#client-adapters)
- [UI Views](#ui-views)
- [Detail View](#detail-view)
- [Torrent Management](#torrent-management)
- [Multi-Client Support](#multi-client-support)
- [Filtering & Sorting](#filtering--sorting)
- [Search](#search)
- [Category / Label Icons](#category--label-icons)
- [Auto-Refresh](#auto-refresh)
- [Keybindings](#keybindings)
- [Build & Development](#build--development)

---

## Overview

torgo is a terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (v2) that lets you manage **qBittorrent** and **Transmission** torrent clients from a single terminal session. You can connect to unlimited instances of either type, switch between them with one key, and perform all common operations without leaving your terminal.

Architecture:

```
main.go → config.LoadConfig() → state.NewAppState() → tui.NewApp() → tea.NewProgram(App)
                                ├─ qBittorrentAdapter (+ ResilientAdapter wrapper)
                                ├─ TransmissionAdapter (+ ResilientAdapter wrapper)
                                └─ AppState (filter, sort, multi-select, speed limit, categories)
```

---

## Installation

### From source

```bash
git clone https://github.com/pdfrg/torgo
cd torgo
make build
```

### Via `go install`

```bash
go install github.com/pdfrg/torgo@latest
```

### Version embedding

The binary version is injected via ldflags. The `Makefile` handles this automatically:

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.Version=$(VERSION)

make build                 # version from git describe, e.g. "v0.1.0" or "dev"
make build VERSION=1.0.0   # explicit override
```

### Multi-platform builds

```bash
make build-multiplatform
# Produces dist/torgo-linux-amd64 and dist/torgo-linux-arm64
```

Other `make` targets:
- `make install` — `go install` with version
- `make test` — `go test ./...`
- `make clean` — remove binary and dist/

---

## CLI Flags

```
Usage: torgo [flags]

Flags:
  -h, --help            Show this help text and exit
  -v, --version         Print version and exit
  -c, --config <path>   Path to config file (default: ~/.config/torgo/config.toml)
  -i, --client <idx>    Client index to start with (0-based)
```

---

## Configuration

Config file: `~/.config/torgo/config.toml`

### `[ui]` section

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_client` | string | first client | Client ID to select on startup |
| `show_hints` | bool | `true` | Show keybinding hints bar at bottom |
| `default_color_scheme` | string | `"dark"` | Theme name (see [Theme System](#theme-system)) |

If `default_color_scheme` is not set, auto-detection picks: `custom` > `omarchy` > `dark`.

### `[[clients]]` blocks

| Key | Required | Description |
|-----|----------|-------------|
| `type` | yes | `"qbittorrent"` or `"transmission"` |
| `id` | yes | Unique identifier (shown in status bar) |
| `name` | yes | Display name |
| `host` | yes | Hostname or IP |
| `port` | yes | API port |
| `username` | no | Auth username (empty = anonymous) |
| `password` | no | Auth password (supports `$ENV_VAR` expansion) |

**Example** (two qBittorrent + one Transmission):

```toml
[ui]
default_client = "qbt-local"
show_hints = true
default_color_scheme = "dark"

[[clients]]
type = "qbittorrent"
id = "qbt-local"
name = "Local qBittorrent"
host = "localhost"
port = 8080
username = "admin"
password = "$QBT_PASS"

[[clients]]
type = "transmission"
id = "trans-local"
name = "Local Transmission"
host = "localhost"
port = 9091
username = "transmission"
password = "transmission"
```

Passwords are expanded via `os.ExpandEnv()` at startup, so `$QBT_PASS`, `${QBT_PASS}`, or `$VAR` all work.

---

## Theme System

### Built-in themes

| Name | Abbreviation | Description |
|------|-------------|-------------|
| `dark` | dark | Light text on dark background (default) |
| `light` | lite | Dark text on light background |
| `highcontrast` | HC | Bright, high contrast for accessibility |
| `terminal` | terminal | Inherits your terminal emulator's palette (ANSI 0–15 + default fg/bg). Ideal over SSH: the local terminal's theme shows through, no theme files needed on the remote host |

All built-in themes use consistent [semantic color roles](#semantic-color-roles).

### External themes

| Name | Abbreviation | Source file |
|------|-------------|-------------|
| `omarchy` | omarchy | Active Omarchy theme's `colors.toml` — resolved from `$XDG_STATE_HOME/omarchy/current/theme/` (usually `~/.local/state/omarchy/current/theme/`), falling back to legacy `~/.config/omarchy/current/theme/` (Omarchy v3 layout) |
| `custom` | custom | `~/.config/torgo/colors.toml` |

Omarchy and custom themes are auto-discovered on startup. The hints bar shows the current theme abbreviation after `t:theme (...)`.

### Theme auto-detection

If `default_color_scheme` is empty:
1. If `~/.config/torgo/colors.toml` exists → `custom`
2. Else if an Omarchy `colors.toml` exists (`$XDG_STATE_HOME/omarchy/current/theme/`, usually `~/.local/state/omarchy/current/theme/`, or legacy `~/.config/omarchy/current/theme/`) → `omarchy`
3. Else → `dark`

### Theme cycling

Press **`t`** to cycle through all discovered themes (built-in + omarchy + custom). The order is `dark → light → highcontrast → terminal → omarchy → custom → dark...` (`terminal` is always available; `omarchy`/`custom` appear only when their files exist).

### Hot-reload

For `omarchy` and `custom` themes, the file's `ModTime` is checked on every refresh tick (every 2 seconds). If the file changed, the theme is re-read from disk and applied immediately.

### Color transformations (omarchy/custom)

External themes use the `teacat/noire` library for:
- **Lighten/darken** — status bar background and empty progress bar areas
- **Desaturate** — gradient endpoints for distinct status colors
- **Auto-adjusted background** — lighten dark backgrounds, darken light backgrounds

### Status gradients

Progress bar gradients are not decorative — they are the fastest way to read torrent
status at a glance. Each gradient follows one of two visual directions:

- **Filling up** (muted → bright): the torrent is actively progressing. The bar
  desaturates to muted tones and builds intensity toward full saturation, visually
  suggesting momentum.
- **Fading out** (bright → background): the torrent is not actively moving. The bar
  dissolves into the terminal background, visually de-emphasizing it.

| Status | Gradient (left → right) | Design intent |
|--------|------------------------|---------------|
| downloading | desaturated color4 → color4 | "Filling up" — building intensity from muted to pure, suggesting active progress |
| seeding | desaturated color4 → color2 | "Complete, still active" — starts with a hint of the download color, resolves to a distinct post-download state |
| completed | desaturated color2 → color2 | "Ready to remove" — built from the same endpoint as seeding but lowered saturation, visually separate |
| paused | color3 → background | "On hold" — fades into the background, de-emphasized |
| error | color1 → background | "Broken / stuck" — urgency (red/orange) dissolves into nothing |
| queueing | color5 → background | "Waiting its turn" — fades out, not progressing |
| stalled | color6 → background | "Hung" — no activity, washed out |

The same visual language applies to built-in themes:

| Status | Gradient (left → right) | What it tells you |
|--------|------------------------|-------------------|
| downloading | Blue → Green | Cold to warm — the fill is *happening* |
| seeding | Green → Cyan | Resolved to cool tones — steady state |
| completed | Purple → Magenta | Distinct from seeding — celebratory coloration |
| paused | Yellow → Orange | Cautionary warm — fading out |
| error | Orange → Red | Escalating urgency — dissolving |
| queueing | Magenta → Purple | Waiting — fading out |
| stalled | Dark gray → Light gray | Washed out — stuck |

This means you can tell a torrent's status from the progress bar appearance alone,
without reading the status label. A bar that fades into the background is not
moving; one that builds intensity is actively transferring data.

### Semantic color roles

| Role | Used for |
|------|----------|
| `AccentColor` | Titles, separator lines, selection indicators, dialog borders, help keys |
| `CursorColor` | Cursor indicator, sort/filter header, list header |
| `ForegroundColor` | Torrent names, metric values (variable data) |
| `TextNormal` | Labels, progress percentage, row numbers (non-cursor) |
| `TextMuted` | Hints, secondary text, inactive elements |
| `TextError` | Error messages, validation errors |
| `BgNormal` | Main background |
| `DetailTabActiveBorder` | Active tab border |
| `DetailTabInactiveBorder` | Inactive tab border |
| `DetailLabelColor` | Field labels in detail view |
| `DetailCursorColor` | Tree/list cursor in detail view |

### Single-line status colors

The single-line view uses solid colors (instead of gradients) with smart contrast text. The text color auto-selects light or dark based on background luminance.

---

## Client Adapters

### qBittorrent (HTTP API v2)

- Login with session cookies
- List torrents (progress, speeds, peers, seeds, ETA, etc.)
- Pause/Resume individual or all
- Add torrents: magnet links, HTTP(S) URLs, `.torrent` file paths
- Delete torrents with optional file deletion
- Force recheck and reannounce
- Copy magnet URI to clipboard
- Queue priority management (up/down/top/bottom)
- Status mapping (downloading, uploading, paused, stalled, error, queued, etc.)
- Speed limit (get/set/toggle)
- Categories (get/set)
- Tags (get/set, comma-separated in edit tab)
- Rename torrent
- Change save path
- Set file priorities (individual file selection)
- Get file list

### Transmission (JSON-RPC)

- Session authentication with `X-Transmission-Session-Id` header
- All operations listed above (except rename — qBittorrent only)
- Labels (used instead of tags)
- Basic auth support
- Labels auto-derived from save path directory name on location change

### Resilience layer

Both adapters are wrapped in a `ResilientAdapter` that provides automatic reconnection on transient failures. Configuration:

- `MaxRetries`: 3
- `InitialBackoff`: 500ms
- `BackoffFactor`: 2.0

---

## UI Views

### Single-line view

Compact layout with columns:

```
 > 1  Torrent Name.............   4.2G   65%  1.2MB  340KB   12    5   d/l
   2  Another Torrent..........   1.1G  100%     -      -     -    -   done
```

Columns: `#` (cursor + optional selection dot), **Name** (with progress fill as background, 🔒 suffix for private trackers), **Size**, **Prog** (%), **↓Down**, **↑Up**, **Seed**, **Leech**, **Status**.

**Seed** shows connected seeds with the tracker's total in parentheses when they differ (e.g. `12(45)`); same for **Leech**. `-`/`N/A` when unknown.

The name field acts as a progress bar: filled portion uses the status color with smart-contrast text, unfilled portion uses normal foreground color.

### Multi-line view

Three lines per torrent with a blank line separator:

```
 > 1  Torrent Name (truncated to fit)
      [████████░░░░░░░░░░░░░░░░░░░] (3.2 GB / 4.2 GB)
      🎬 Downloading  ↓ 1.2 MB/s  ↑ 340 KB/s  Ratio: 0.85  Seeds: 12  Peers: 5  ETA: 2h 15m
```

- **Line 1**: Row number + cursor indicator + selection dot + name (🔒 suffix for private trackers)
- **Line 2**: Gradient progress bar (status-colored) + downloaded/total sizes
- **Line 3**: Category icon + status label + speeds + ratio + seeds + peers + ETA (seed/peer counts use the same `connected(total)` format as single-line view)

### View cycling

Press **`v`** to toggle between single-line and multi-line view. Cursor position and selection state are preserved across view switches.

---

## Detail View

Press **`enter`** on a torrent to open the detail view — a tabbed interface for inspecting and editing torrent properties. Press **`enter`** again (with changes) to save, or **`esc`** to close.

### Info tab (read-only)

Displays current torrent properties:
- Name (shows pending edits)
- Category (shows pending edits)
- Tags (comma-separated, shows pending edits)
- Comments
- Tracker URL (🔒 suffix for private trackers)

### Edit tab

Editable fields:
- **Name** — rename the torrent
- **Tags** — comma-separated tag list
- **Location/Save Path** — change download directory. For local hosts (localhost, 127.0.0.1, or hostname resolving to loopback), a subdirectory listing dialog appears: `/` to navigate into folders, `Esc` to go up.
  - On Transmission, changing location also auto-derives the category from the directory name.

### Category tab

Browse available categories from the client. Arrow keys to navigate, Enter to select. Shows "None" as the first option.

### Files tab

Browse all files in the torrent. Navigate with arrow keys, **space** to toggle file selection (wanted/unwanted). Files set to unwanted will not be downloaded.

### Tab navigation

| Key | Action |
|-----|--------|
| Tab | Next tab |
| Shift+Tab | Previous tab |
| e | Jump from Info to Edit tab |
| Enter | Save changes and close |
| Esc | Back to Info / discard changes / close |

---

## Torrent Management

### Pause / Resume

- **`p`** — pause selected torrent(s)
- **`P`** — pause all torrents
- **`r`** — resume selected torrent(s)
- **`R`** — resume all torrents

### Delete

- **`x`** — delete selected torrent(s) (keep files). Shows confirmation dialog with torrent name(s).
- **`X`** — delete selected torrent(s) **with data**. Confirmation dialog shows "Delete with data (files will be removed)".

In the confirmation dialog: **`y`** or **Enter** to confirm, **`Esc`** to cancel.

### Add torrents

Press **`a`** to open the add dialog. Supported input types:
- **Magnet links**: `magnet:?xt=urn:btih:...` (validated for `xt` parameter)
- **HTTP(S) URLs**: `https://example.com/file.torrent`
- **Local file paths**: `/path/to/file.torrent` (supports `~` expansion, must end in `.torrent`)

The add dialog includes:
- Real-time input validation with error display
- `Ctrl+P` to paste from clipboard
- Category selector (arrow keys to choose from available categories, or "None")

### Recheck / Reannounce

- **`!`** — force recheck (hash check) selected torrent(s)
- **`n`** — reannounce to trackers

### Queue priority

- **`=`** — move selected up in queue
- **`-`** — move selected down in queue
- **`+`** — move selected to top of queue
- **`_`** — move selected to bottom of queue

### Speed limit

- **`l`** — toggle speed limit on/off
- Status bar shows `🐢▼limit ▲limit` when speed limit is enabled
- Speed limit status is polled every 12 seconds to catch external changes (scheduler, web UI, etc.)

### Copy magnet link

**`y`** — copies the current torrent's magnet URI to the system clipboard. Shows "Magnet link copied ✓" on success.

---

## Multi-Client Support

- Define unlimited `[[clients]]` blocks in config
- Press **`c`** to cycle through clients
- Status bar shows: `✓ Online  Client Name host:port  🐢▼limit ▲limit    filtered/total torrents`
- Connection is tested on client switch
- Filter, sort, and search are per-client (reset on switch)

---

## Filtering & Sorting

### Filter popup

Press **`f`** to open the filter popup. Options:

| Filter | Shows |
|--------|-------|
| all | All torrents |
| downloading | Currently downloading |
| seeding | Actively seeding |
| completed | 100% complete |
| paused | Paused/stopped |
| stalled | Stalled (no peers) |
| error | Error state |
| queued | Queued for download |

Navigation: `↑`/`↓` or `k`/`j`, **Enter** to select, **`Esc`** to cancel.

### Sort popup

Press **`s`** to open the sort popup. Options (each available ascending ↑ and descending ↓):

| Sort | Field |
|------|-------|
| name | Name (alphabetical) |
| progress | Completion % |
| speed | Total speed (download + upload) |
| seeds | Seed count |
| ratio | Upload/download ratio |
| size | Total size |
| leechers | Leecher count |
| eta | ETA (sentinel/unknown always last) |

Navigation: `↑`/`↓` or `k`/`j`, **Enter** to select, **`Esc`** to cancel.

The status bar shows current filter and sort: `filter: downloading  sort: name ↑`.

---

## Search

Press **`/`** to enter search mode.

- **Real-time filtering**: results update as you type
- **Case-insensitive**: "ubuntu" matches "Ubuntu"
- **Separator normalization**: dots, dashes, and underscores are treated as spaces
- **Tracker filter**: tokens of the form `tr:<text>` or `tracker:<text>` narrow results to torrents whose tracker hostname contains `<text>` (case-insensitive, separators ignored — `tr:torrent-leech`, `tr:torrent leech`, and `tr:torrentleech` all match `tracker.torrentleech.org`). Short substrings work too, so `tr:tl` is a fast shortcut. Torrents with no tracker URL are excluded whenever a tracker term is present.
- **Combined search**: tracker tokens and title text compose — `tr:torrentleech ubuntu` means "on that tracker AND title contains ubuntu". Comma-separated values match OR across trackers: `tr:tl,mao arch`. Bare text with no `tr:` token is title-only, exactly as before.
- **Visible filter**: confirmed tracker filters are shown explicitly in the results bar, e.g. `🔍 ubuntu [tr:torrentleech] (3 matches)` — or `🔍 * [tr:...]` when there is no title text — so it's always visible that a tracker filter is active
- **Match count**: shows `(N matches)` in the search bar
- **Visual modes**:
  - *Editing* (bright accent background): shows current input with match count
  - *Active* (subtle background): shows query and match count after confirmation

Workflow:
1. Press `/` → search input opens in editing mode
2. Type query → results filter in real-time
3. Press **Enter** → lock filter, exit editing (results remain filtered)
4. Press `/` again → re-enter editing mode to modify query
5. Press **Esc** → clear search, show all torrents

Search applies on top of the current filter (e.g., filter=downloading + search="ubuntu" shows downloading torrents with "ubuntu" in name).

Search survives background refreshes (every 2s) and sort/filter popup confirmations — the query is re-applied on top of the refreshed list, so results don't silently reset while you work.

---

## Category / Label Icons

The app maps torrent categories/labels to emoji icons shown in multi-line view (status line). Mapping is case-insensitive with substring matching, so `"TV-english"` matches `"tv"` → 📺.

### Full mapping

| Category keywords | Icon |
|-------------------|------|
| movie, movies, film, cinema, radarr, feature | 🎬 |
| tv, tvshow, series, episode, sonarr, show | 📺 |
| music, audio, song, album, lidarr, podcast | 🎵 |
| audiobook, audiobooks | 📖 |
| readarr, book, books, ebook, document | 📚 |
| software, app, application, program, utility | 💻/🔧 |
| game, games, gaming, playstation, xbox, nintendo | 👾 |
| whisparr, adult, xxx | 🔞 |
| mylar, comic, comics | 💥 |
| photo, photos, image, pictures, wallpaper | 📷 |
| video, videos, tutorial, stream | 🎥 |
| linux, ubuntu, debian, fedora, arch, distro | 🐧 |
| windows | 🪟 |
| macos, osx | 🍎 |
| iso | 💿 |
| news, documentary | 📰 |
| sports, football, soccer, basketball, etc. | ⚽ |
| anime | 🌸 |
| manga, manhua | 📖 |
| development, code, source code, git, repository | 💾 |
| archive, compressed, tar, zip, rar | 📦 |
| education, course, training, school, university | 🎓 |
| art, design, graphic, 3d | 🎨 |
| fitness, workout, yoga | 💪 |
| health, medical | ⚕️ |
| nature, wildlife, animals, pet | 🌿/🦁/🐾 |
| travel, tourism, destination, guide | ✈️/🗺️ |
| podcast | 📻 |
| default, misc, other, various | 📁 |

### qBittorrent vs Transmission

- **qBittorrent**: uses categories directly (single category per torrent)
- **Transmission**: uses labels (multiple labels per torrent, first label used as category)

---

## Auto-Refresh

| Cycle | Interval | What refreshes |
|-------|----------|---------------|
| Torrent list | Every 2 seconds | Torrent data, filter/sort, view render |
| Speed limit status | Every 12 seconds | Speed limit state (catches external changes like scheduler) |
| Theme file check | Every 2 seconds | ModTime check for omarchy/custom themes |

Errors are displayed at the bottom and auto-clear after 3 seconds.

---

## Keybindings

### List view

| Key | Action |
|-----|--------|
| ↑/k | Move cursor up |
| ↓/j | Move cursor down |
| PgUp/^U | Page up |
| PgDn/^D | Page down |
| space | Toggle selection |
| A | Select all / deselect all |
| p | Pause selected |
| P | Pause all |
| r | Resume selected |
| R | Resume all |
| x | Delete (keep files) |
| X | Delete with data |
| enter | View details |
| a | Add torrent |
| y | Copy magnet link |
| ! | Force recheck |
| n | Reannounce to trackers |
| = | Queue priority up |
| - | Queue priority down |
| + | Queue to top |
| _ | Queue to bottom |
| c | Cycle clients |
| v | Toggle view (single-line ↔ multi-line) |
| t | Cycle themes |
| s | Sort popup |
| f | Filter popup |
| / | Search |
| h | Toggle hints bar |
| l | Toggle speed limit |
| ? | Help dialog |
| q | Quit |

### Detail view

| Key | Action |
|-----|--------|
| Tab | Next tab |
| Shift+Tab | Previous tab |
| e | Jump to Edit tab |
| enter | Save changes and close |
| esc | Back to info / discard / close |

### Search mode

| Key | Action |
|-----|--------|
| / | Enter / edit search |
| Enter | Confirm search filter |
| Esc | Clear search |
| (type) | Real-time filtering |

### Input modes (add torrent / delete confirm)

| Key | Action |
|-----|--------|
| Ctrl+P | Paste from clipboard (add dialog) |
| y / Enter | Confirm delete |
| Esc | Cancel |

---

## Build & Development

### Prerequisites

- Go 1.24+

### Commands

```bash
make build              # Build binary (torgo)
make build-release      # Release build
make build-multiplatform # Cross-compile for linux/amd64 + linux/arm64
make install            # go install
make test               # Run all tests
make clean              # Remove build artifacts

# Manual checks
go fmt ./...
golangci-lint run ./...
go vet ./...
go build ./...
go test ./...
```

### Project layout

```
torgo/
├── main.go                   # Entry point
├── config/
│   ├── config.go             # TOML parsing, theme loading
│   └── config_test.go
├── client/
│   ├── interface.go          # ClientAdapter interface + Torrent/TorrentDetail/TorrentFile types
│   ├── qbittorrent.go        # qBittorrent HTTP API v2
│   ├── transmission.go       # Transmission JSON-RPC
│   ├── resilience.go         # Retry/backoff wrapper
│   └── resilience_test.go
├── state/
│   ├── app_state.go          # AppState: filter, sort, client switching, speed limit
│   └── app_state_test.go
├── tui/
│   ├── app.go                # Main Bubble Tea model
│   ├── colors.go             # Theme definitions (dark, light, highcontrast, omarchy)
│   ├── styles.go             # Styles struct and SyncFromTheme
│   ├── keybindings.go        # KeyMap definition
│   ├── status_bar.go         # Status bar renderer
│   ├── hints_bar.go          # Hints bar renderer
│   ├── torrent_list.go       # Single-line compact view
│   ├── torrent_list_multiline.go # Multi-line detailed view
│   ├── progressbars.go       # Gradient progress bar builder
│   ├── detail_view.go        # Tabbed detail view (info, edit, category, files)
│   ├── categories.go         # Category/label emoji mapping
│   ├── search.go             # Search filter engine
│   ├── validation.go         # Input validation (magnet, URL, file path)
│   ├── formatting.go         # Formatting helpers (bytes, speed, duration, etc.)
│   ├── subdirectory_helper.go # Subdirectory listing for local hosts
│   ├── subdirectory_helper_test.go
│   ├── search_test.go
│   └── validation_test.go
├── config.example.toml       # Example configuration
├── Makefile
├── go.mod
└── go.sum
```
