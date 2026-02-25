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

## Implemented Features

### Config System
- [x] TOML config parsing from `~/.config/tqbtui/config.toml`
- [x] Support multiple clients (qBittorrent & Transmission, unlimited instances)
- [x] Environment variable expansion in passwords
- [x] Theme loading (default dark theme with optional omarchy colors.toml)
- [x] Full test coverage

### Client Adapters
- [x] **qBittorrent** (HTTP API v2)
  - Login with credentials
  - List torrents with progress, speed, peers
  - Pause/resume individual or all
  - Add torrents (magnet links & .torrent files)
  - Delete torrents with optional file deletion
  - Status mapping (downloading, seeding, paused, error)

- [x] **Transmission** (JSON-RPC)
  - Session authentication with header-based session ID
  - List torrents with progress, speed, peers
  - Pause/resume individual or all
  - Add torrents (magnet links & .torrent files)
  - Delete torrents with optional file deletion
  - Basic auth support
  - Status mapping (downloading, seeding, paused)

### Application State
- [x] Multi-client switching with cycling
- [x] Torrent filtering (all, active, paused, completed)
- [x] Torrent sorting (name, progress, speed, seeders)
- [x] Multi-selection support (prepared for bulk operations)
- [x] Theme management
- [x] Hints bar toggle

### Testing
- [x] Unit tests for config loading
- [x] Unit tests for app state management
- [x] Filter/sort cycling tests
- [x] Client switching tests
- All tests passing

## Development Status

- [x] Project scaffold
- [x] Config system with TOML parsing
- [x] Client abstraction interface
- [x] qBittorrent adapter (full HTTP API implementation)
- [x] Transmission adapter (full JSON-RPC implementation)
- [x] Application state management
- [x] Unit & integration tests
- [ ] TUI implementation with bubbletea
- [ ] Keybinding system
- [ ] omarchy colors.toml integration

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
4. Display results

## Next Steps

1. Build TUI with bubbletea
2. Implement torrent list view with progress bars, speeds, seeds/leechs
3. Implement keybinding system (p/P for pause, r/R for resume, etc.)
4. Add hints bar and multi-select UI
5. Integrate with omarchy colors.toml for theming
