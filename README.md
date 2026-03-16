# tqbtui – Torrent Client TUI

A terminal user interface for managing torrents across multiple qBittorrent and Transmission instances.

## Project Structure

```
tqbtui/
├── main.go                 # Entry point (config loading, demo)
├── config/                 # Configuration loading & theming
│   ├── config.go          # Config parsing (TOML)
│   ├── config_test.go     # Config tests
│   ├── theme.go           # Color themes
├── client/                 # Torrent client abstractions
│   ├── interface.go       # ClientAdapter interface & Torrent model
│   ├── qbittorrent.go     # qBittorrent HTTP API implementation (~230 lines)
│   └── transmission.go     # Transmission JSON-RPC implementation (~340 lines)
├── state/                  # Application state
│   ├── app_state.go       # AppState manager
│   └── app_state_test.go  # AppState tests
├── tui/                    # Terminal UI (TBD)
│   ├── app.go             # Main TUI app (bubbletea)
│   ├── components/        # UI components
│   └── keybindings.go     # Keybinding definitions
├── config.example.toml     # Example configuration (4 client instances)
├── go.mod                  # Go module definition
└── go.sum                  # Dependency versions
```

## Setup

1. Copy `config.example.toml` to `~/.config/tqbtui/config.toml`
2. Edit with your torrent client credentials
3. Support environment variables in passwords (e.g., `$QBT_PASS`)

### Theme Configuration

**Built-in themes:** Set `default_color_scheme` in config to one of:
- `dark` (default) - light text on dark background
- `light` - dark text on light background
- `highcontrast` - bright colors for accessibility

**Omarchy theme:** If you use omarchy, place its colors.toml at:
- `~/.config/omarchy/current/theme/colors.toml` (automatic discovery)
- Or copy to `~/.config/tqbtui/colors.toml` (as "custom")

Then set `default_color_scheme = "omarchy"` or `default_color_scheme = "custom"`.

At runtime, press `t` to cycle through all available themes (built-in + any discovered omarchy/custom themes).

## Implemented Features

### Config System
- [x] TOML config parsing from `~/.config/tqbtui/config.toml`
- [x] Support multiple clients (qBittorrent & Transmission, unlimited instances)
- [x] Environment variable expansion in passwords
- [x] Full test coverage

### Theme System
- [x] Built-in themes: dark, light, high-contrast with consistent semantic color roles
  - Semantic roles: AccentColor, CursorColor, ForegroundColor, TextNormal, TextMuted
  - All UI elements use the same role across all themes for consistency
  - `syncComponentStyles()` ensures cached components update on theme switch
- [x] **Omarchy theme support** with smart color transformations
  - Auto-discover omarchy theme at `~/.config/omarchy/current/theme/colors.toml`
  - Custom theme support at `~/.config/tqbtui/colors.toml`
  - Color manipulation (lighten, darken, desaturate) via `teacat/noire` library
  - 7 distinct progress bar gradients for different torrent statuses
  - Auto-adjusted background contrast (lighten dark backgrounds, darken light backgrounds)
- [x] Theme cycling with `t` key (hot-swap any available theme)
- [x] Hints bar displays current theme name (abbreviations: dark, lite, HC, omarchy, custom)

### Client Adapters
- [x] **qBittorrent** (HTTP API v2)
  - Login with credentials
  - List torrents with progress, speed, peers
  - Pause/resume individual or all
  - Add torrents (magnet links & .torrent files)
  - Delete torrents with optional file deletion
  - Force recheck and reannounce
  - Copy magnet link to clipboard
  - Queue priority management (up/down/top/bottom)
  - Status mapping (downloading, seeding, paused, error)

- [x] **Transmission** (JSON-RPC)
  - Session authentication with header-based session ID
  - List torrents with progress, speed, peers
  - Pause/resume individual or all
  - Add torrents (magnet links & .torrent files)
  - Delete torrents with optional file deletion
  - Force recheck and reannounce
  - Copy magnet link to clipboard
  - Queue priority management (up/down/top/bottom)
  - Basic auth support
  - Status mapping (downloading, seeding, paused)

### Application State
- [x] Multi-client switching with cycling
- [x] Torrent filtering (all, active, paused, completed)
- [x] Torrent sorting (name, progress, speed, seeders)
- [x] Multi-selection support (prepared for bulk operations)
- [x] Theme management (cycle themes with `t`)
- [x] Hints bar toggle
- [x] Category/Label icons (mapped from categories/labels with emoji)

### UI Views
- [x] Single-line view (compact list display)
   - Solid status colors with smart contrast text (light/dark based on background luminance)
   - Theme-aware table header, separator, row numbers, and field styling
   - Selection indicators and cursor highlighting
- [x] Multi-line view (detailed display with 3 lines per torrent)
   - Line 1: Torrent number, name
   - Line 2: Progress bar with file sizes
   - Line 3: Status, speeds, ratio, seeds, peers, ETA with category icon
   - Gradient progress bars for visual distinction of status
- [x] View cycling with `v`
- [x] Progress bars with solid colors in single-line view, gradients in multi-line view
- [x] Textured empty progress bar areas
- [x] 7 distinct status colors (downloading, seeding, completed, paused, error, queueing, stalled)

### Testing
- [x] Unit tests for config loading
- [x] Unit tests for app state management
- [x] Filter/sort cycling tests
- [x] Client switching tests
- All tests passing

## Category/Label Icon Mappings

The application includes built-in category-to-emoji mappings for common torrent categories:

### Supported Applications
- **qBittorrent** uses categories, mapped directly
- **Transmission** uses labels (first label is used as category)
- **Radarr** - `radarr` → 🎬
- **Sonarr** - `sonarr` → 📺
- **Lidarr** - `lidarr` → 🎵
- **Readarr** - `readarr` → 📚
- **Whisparr** - `whisparr` → 🔞
- **Mylar3** - `mylar` → 💥

### Smart Matching
Categories/labels are matched via substring matching, so:
- "TV-english" matches "tv" → 📺
- "movie-drama" matches "movie" → 🎬
- Custom names like "Radarr" or "SONARR" also work

The complete mapping includes 50+ category keywords for movies, TV, music, software, games, books, images, OSes, development, and more.

## Development Status

- [x] Project scaffold
- [x] Config system with TOML parsing
- [x] Client abstraction interface
- [x] qBittorrent adapter (full HTTP API implementation)
- [x] Transmission adapter (full JSON-RPC implementation)
- [x] Application state management
- [x] Unit & integration tests
- [x] TUI implementation with bubbletea (BubbleTea v2)
- [x] Keybinding system with hints bar
- [x] Help dialog with 2-column keybinding layout
- [x] Multi-view support (single-line and multi-line views)
- [x] Progress bars with gradient coloring and texture
- [x] Category/Label icon mapping system

## Usage

```bash
go run main.go
# or
go build && ./tqbtui
```

The program will:
1. Load configuration from `~/.config/tqbtui/config.toml`
2. Initialize all configured clients
3. Attempt to connect to each client
4. Display torrent list

## Keybindings

| Key | Action |
|-----|--------|
| ↑/k, ↓/j | Move cursor |
| PgUp/^U, PgDn/^D | Page navigation |
| space | Toggle select |
| A | Select all |
| p/P | Pause (individual/all) |
| r/R | Resume (individual/all) |
| x/X | Delete (without/with data) |
| enter | View details |
| a | Add torrent |
| y | Copy magnet link to clipboard |
| ! | Force recheck |
| n | Reannounce to trackers |
| =/- | Queue priority up/down |
| +/_ | Queue to top/bottom |
| c | Cycle clients |
| v | Cycle views |
| t | Cycle themes |
| s | Sort |
| f | Filter |
| / | Search |
| h | Toggle hints bar |
| l | Toggle speed limit |
| ? | Help |
| q | Quit |
