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
	Clients              []ClientInstance
	CurrentClientIdx     int
	Torrents             []client.Torrent
	Selected             map[string]bool     // Torrent ID -> selected
	Filter               FilterType
	SortBy               SortType
	ShowHints            bool
	ErrorMsg             string
	InputMode            string // "", "add", "search", "command"
	Config               *config.Config
}

// FilterType represents torrent filtering options
type FilterType string

const (
	FilterAll         FilterType = "all"
	FilterActive      FilterType = "active"
	FilterPaused      FilterType = "paused"
	FilterCompleted   FilterType = "completed"
)

// SortType represents torrent sorting options
type SortType string

const (
	SortByName     SortType = "name"
	SortByProgress SortType = "progress"
	SortBySpeed    SortType = "speed"
	SortBySeeds    SortType = "seeds"
)

// NewAppState initializes the app state from config
func NewAppState(cfg *config.Config) (*AppState, error) {
	as := &AppState{
		Config:        cfg,
		CurrentClientIdx: 0,
		Selected:      make(map[string]bool),
		Filter:        FilterAll,
		SortBy:        SortByName,
		ShowHints:     cfg.UI.ShowHints,
		Torrents:      []client.Torrent{},
		Clients:       []ClientInstance{},
	}

	// Initialize clients from config
	for _, clientCfg := range cfg.Clients {
		var adapter client.ClientAdapter

		switch clientCfg.Type {
		case "qbittorrent":
			adapter = client.NewQBittorrentAdapter(
				clientCfg.Host,
				clientCfg.Port,
				clientCfg.Username,
				clientCfg.Password,
			)
		case "transmission":
			adapter = client.NewTransmissionAdapter(
				clientCfg.Host,
				clientCfg.Port,
				clientCfg.Username,
				clientCfg.Password,
			)
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

// sortTorrents sorts torrents in place based on current SortBy
func (as *AppState) sortTorrents(torrents []client.Torrent) {
	switch as.SortBy {
	case SortByName:
		sort.Slice(torrents, func(i, j int) bool {
			return torrents[i].Name < torrents[j].Name
		})
	case SortByProgress:
		sort.Slice(torrents, func(i, j int) bool {
			return torrents[i].Progress > torrents[j].Progress
		})
	case SortBySpeed:
		sort.Slice(torrents, func(i, j int) bool {
			return (torrents[i].SpeedDown + torrents[i].SpeedUp) > 
				   (torrents[j].SpeedDown + torrents[j].SpeedUp)
		})
	case SortBySeeds:
		sort.Slice(torrents, func(i, j int) bool {
			return torrents[i].Seeds > torrents[j].Seeds
		})
	}
}

// matchesFilter checks if a torrent matches the current filter
func (as *AppState) matchesFilter(t client.Torrent) bool {
	switch as.Filter {
	case FilterActive:
		return t.Status == client.StatusDownloading
	case FilterPaused:
		return t.Status == client.StatusPaused
	case FilterCompleted:
		return t.Status == client.StatusSeeding
	default: // FilterAll
		return true
	}
}

// CycleFilter moves to the next filter
func (as *AppState) CycleFilter() {
	switch as.Filter {
	case FilterAll:
		as.Filter = FilterActive
	case FilterActive:
		as.Filter = FilterPaused
	case FilterPaused:
		as.Filter = FilterCompleted
	case FilterCompleted:
		as.Filter = FilterAll
	}
}

// CycleSort moves to the next sort option
func (as *AppState) CycleSort() {
	switch as.SortBy {
	case SortByName:
		as.SortBy = SortByProgress
	case SortByProgress:
		as.SortBy = SortBySpeed
	case SortBySpeed:
		as.SortBy = SortBySeeds
	case SortBySeeds:
		as.SortBy = SortByName
	}
}
