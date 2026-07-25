package state

import (
	"context"
	"fmt"
	"sort"
	"tqbtui/client"
	"tqbtui/config"
)

// ClientInstance wraps a client adapter with metadata
type ClientInstance struct {
	ID      string
	Name    string
	Type    string
	Adapter client.ClientAdapter
}

// AppState manages the overall application state
type AppState struct {
	Clients           []ClientInstance
	CurrentClientIdx  int
	Torrents          []client.Torrent
	Selected          map[string]bool // Torrent ID -> selected
	Filter            FilterType
	SortBy            SortType
	SortAscending     bool
	ShowHints         bool
	ErrorMsg          string
	InputMode         string // "", "add", "search", "command"
	Config            *config.Config
	SpeedLimitEnabled bool     // Cache of speed limit status
	SpeedLimitDownKBs int      // Cache of down speed limit
	SpeedLimitUpKBs   int      // Cache of up speed limit
	Categories        []string // Cache of category/label names for current client
}

// FilterType represents torrent filtering options
type FilterType string

const (
	FilterAll         FilterType = "all"
	FilterDownloading FilterType = "downloading"
	FilterSeeding     FilterType = "seeding"
	FilterCompleted   FilterType = "completed"
	FilterPaused      FilterType = "paused"
	FilterStalled     FilterType = "stalled"
	FilterError       FilterType = "error"
	FilterQueued      FilterType = "queued"
)

// SortType represents torrent sorting options
type SortType string

const (
	SortByName     SortType = "name"
	SortByProgress SortType = "progress"
	SortBySpeed    SortType = "speed"
	SortBySeeds    SortType = "seeds"
	SortByRatio    SortType = "ratio"
	SortBySize     SortType = "size"
	SortByAge      SortType = "age"
	SortByLeech    SortType = "leechers"
	SortByETA      SortType = "eta"
)

// NewAppState initializes the app state from config
func NewAppState(cfg *config.Config) (*AppState, error) {
	as := &AppState{
		Config:           cfg,
		CurrentClientIdx: 0,
		Selected:         make(map[string]bool),
		Filter:           FilterAll,
		SortBy:           SortByName,
		SortAscending:    true,
		ShowHints:        cfg.UI.ShowHints,
		Torrents:         []client.Torrent{},
		Clients:          []ClientInstance{},
	}

	// Initialize clients from config
	for _, clientCfg := range cfg.Clients {
		var adapter client.ClientAdapter

		switch clientCfg.Type {
		case "qbittorrent":
			raw := client.NewQBittorrentAdapter(
				clientCfg.Host,
				clientCfg.Port,
				clientCfg.Username,
				clientCfg.Password,
			)
			adapter = client.NewResilientAdapter(raw, client.DefaultResilienceConfig())
		case "transmission":
			raw := client.NewTransmissionAdapter(
				clientCfg.Host,
				clientCfg.Port,
				clientCfg.Username,
				clientCfg.Password,
			)
			adapter = client.NewResilientAdapter(raw, client.DefaultResilienceConfig())
		default:
			return nil, fmt.Errorf("unknown client type: %s", clientCfg.Type)
		}

		as.Clients = append(as.Clients, ClientInstance{
			ID:      clientCfg.ID,
			Name:    clientCfg.Name,
			Type:    clientCfg.Type,
			Adapter: adapter,
		})
	}

	if len(as.Clients) == 0 {
		return nil, fmt.Errorf("no clients configured")
	}

	// Select default client by ID if configured
	if cfg.UI.DefaultClient != "" {
		for i, c := range as.Clients {
			if c.ID == cfg.UI.DefaultClient {
				as.CurrentClientIdx = i
				break
			}
		}
	}

	return as, nil
}

// CurrentClient returns the currently selected client
func (as *AppState) CurrentClient() *ClientInstance {
	if as.CurrentClientIdx >= 0 && as.CurrentClientIdx < len(as.Clients) {
		return &as.Clients[as.CurrentClientIdx]
	}
	return nil
}

// SwitchClient moves to the next client
func (as *AppState) SwitchClient() {
	as.CurrentClientIdx = (as.CurrentClientIdx + 1) % len(as.Clients)
	as.Selected = make(map[string]bool) // Clear selection
}

// RefreshTorrents fetches torrents from the current client
func (as *AppState) RefreshTorrents(ctx context.Context) error {
	current := as.CurrentClient()
	if current == nil {
		return fmt.Errorf("no current client")
	}

	// Ensure connected
	if !current.Adapter.IsConnected() {
		if err := current.Adapter.Connect(ctx); err != nil {
			as.ErrorMsg = fmt.Sprintf("Failed to connect: %v", err)
			return err
		}
	}

	torrents, err := current.Adapter.ListTorrents(ctx)
	if err != nil {
		as.ErrorMsg = fmt.Sprintf("Failed to list torrents: %v", err)
		return err
	}

	as.Torrents = torrents
	as.ErrorMsg = ""
	return nil
}

// FilteredTorrents returns torrents matching current filter and sort order
func (as *AppState) FilteredTorrents() []client.Torrent {
	filtered := []client.Torrent{}
	for _, t := range as.Torrents {
		if as.matchesFilter(t) {
			filtered = append(filtered, t)
		}
	}

	// Apply sorting
	as.sortTorrents(filtered)

	return filtered
}

// sortTorrents sorts torrents in place based on current SortBy and SortAscending
func (as *AppState) sortTorrents(torrents []client.Torrent) {
	asc := as.SortAscending

	// Handle ETA separately because sentinel values need special treatment
	// in both ascending and descending modes
	if as.SortBy == SortByETA {
		sort.Slice(torrents, func(i, j int) bool {
			return etaCompare(torrents[i].ETA, torrents[j].ETA, asc)
		})
		return
	}

	sort.Slice(torrents, func(i, j int) bool {
		var less bool

		switch as.SortBy {
		case SortByName:
			less = torrents[i].Name < torrents[j].Name
		case SortByProgress:
			less = torrents[i].Progress < torrents[j].Progress
		case SortBySpeed:
			less = (torrents[i].SpeedDown + torrents[i].SpeedUp) <
				(torrents[j].SpeedDown + torrents[j].SpeedUp)
		case SortBySeeds:
			less = torrents[i].Seeds < torrents[j].Seeds
		case SortByRatio:
			ri := ratio(torrents[i])
			rj := ratio(torrents[j])
			less = ri < rj
		case SortBySize:
			less = torrents[i].Size < torrents[j].Size
		case SortByLeech:
			less = torrents[i].Leechs < torrents[j].Leechs
		default:
			less = torrents[i].Name < torrents[j].Name
		}

		if asc {
			return less
		}
		return !less
	})
}

// ratio computes the share ratio, handling zero-download to avoid division by zero
func ratio(t client.Torrent) float64 {
	if t.Downloaded == 0 {
		return 0
	}
	return float64(t.Uploaded) / float64(t.Downloaded)
}

// etaCompare compares two ETA values, treating sentinel (>= 8640000) as always last
// regardless of sort direction. asc=true → shortest ETA first; asc=false → longest first.
func etaCompare(a, b int64, asc bool) bool {
	const sentinel = 8640000
	aInf := a >= sentinel
	bInf := b >= sentinel

	if aInf && bInf {
		return false
	}
	if aInf {
		return false // sentinel always last
	}
	if bInf {
		return true // non-sentinel always before sentinel
	}
	if asc {
		return a < b
	}
	return a > b
}

// matchesFilter checks if a torrent matches the current filter
func (as *AppState) matchesFilter(t client.Torrent) bool {
	switch as.Filter {
	case FilterDownloading:
		return t.Status == client.StatusDownloading
	case FilterSeeding:
		return t.Status == client.StatusSeeding
	case FilterCompleted:
		return t.Status == client.StatusCompleted
	case FilterPaused:
		return t.Status == client.StatusPaused
	case FilterStalled:
		return t.Status == client.StatusStalledDL
	case FilterError:
		return t.Status == client.StatusError
	case FilterQueued:
		return t.Status == client.StatusQueuedDL
	default: // FilterAll
		return true
	}
}

// SetFilter sets the current filter directly
func (as *AppState) SetFilter(f FilterType) {
	as.Filter = f
}

// SetSort sets the current sort field and direction directly
func (as *AppState) SetSort(sortBy SortType, ascending bool) {
	as.SortBy = sortBy
	as.SortAscending = ascending
}

// FilterOption represents a selectable filter option for the popup
type FilterOption struct {
	Filter FilterType
	Label  string
}

// SortOption represents a selectable sort option for the popup
type SortOption struct {
	SortBy    SortType
	Ascending bool
	Label     string
}

// FilterOptions returns all available filter options
func FilterOptions() []FilterOption {
	return []FilterOption{
		{FilterAll, "all"},
		{FilterDownloading, "downloading"},
		{FilterSeeding, "seeding"},
		{FilterCompleted, "completed"},
		{FilterPaused, "paused"},
		{FilterStalled, "stalled"},
		{FilterError, "error"},
		{FilterQueued, "queued"},
	}
}

// SortOptions returns all available sort options (field × direction pairs)
func SortOptions() []SortOption {
	return []SortOption{
		{SortByName, true, "name ↑"},
		{SortByName, false, "name ↓"},
		{SortByProgress, true, "progress ↑"},
		{SortByProgress, false, "progress ↓"},
		{SortBySpeed, true, "speed ↑"},
		{SortBySpeed, false, "speed ↓"},
		{SortBySeeds, true, "seeds ↑"},
		{SortBySeeds, false, "seeds ↓"},
		{SortByRatio, true, "ratio ↑"},
		{SortByRatio, false, "ratio ↓"},
		{SortBySize, true, "size ↑"},
		{SortBySize, false, "size ↓"},
		{SortByLeech, true, "leechers ↑"},
		{SortByLeech, false, "leechers ↓"},
		{SortByETA, true, "ETA ↑"},
		{SortByETA, false, "ETA ↓"},
	}
}

// RefreshSpeedLimitStatus updates the cached speed limit status and values from the current client
func (as *AppState) RefreshSpeedLimitStatus(ctx context.Context) error {
	current := as.CurrentClient()
	if current == nil {
		return fmt.Errorf("no current client")
	}

	enabled, err := current.Adapter.GetSpeedLimitEnabled(ctx)
	if err != nil {
		return err
	}

	as.SpeedLimitEnabled = enabled

	// Also fetch the actual speed limit values
	downKBs, upKBs, err := current.Adapter.GetSpeedLimits(ctx)
	if err == nil {
		as.SpeedLimitDownKBs = downKBs
		as.SpeedLimitUpKBs = upKBs
	}
	return nil
}

// ToggleSpeedLimit toggles the speed limit on the current client
func (as *AppState) ToggleSpeedLimit(ctx context.Context) error {
	current := as.CurrentClient()
	if current == nil {
		return fmt.Errorf("no current client")
	}

	newState := !as.SpeedLimitEnabled
	if err := current.Adapter.SetSpeedLimitEnabled(ctx, newState); err != nil {
		return err
	}

	as.SpeedLimitEnabled = newState

	// Also fetch the speed limit values after toggling
	downKBs, upKBs, err := current.Adapter.GetSpeedLimits(ctx)
	if err == nil {
		as.SpeedLimitDownKBs = downKBs
		as.SpeedLimitUpKBs = upKBs
	}
	return nil
}
