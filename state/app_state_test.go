package state

import (
	"testing"
	"tqbtui/client"
	"tqbtui/config"
)

// TestNewAppState tests creating a new app state from config
func TestNewAppState(t *testing.T) {
	cfg := &config.Config{
		UI: config.UIConfig{
			DefaultClient:      "qbt-local",
			ShowHints:          true,
			DefaultColorScheme: "default",
		},
		Clients: []config.ClientConfig{
			{
				Type:     "qbittorrent",
				ID:       "qbt-local",
				Name:     "Local qBittorrent",
				Host:     "localhost",
				Port:     8080,
				Username: "admin",
				Password: "admin",
			},
			{
				Type:     "transmission",
				ID:       "trans-local",
				Name:     "Local Transmission",
				Host:     "localhost",
				Port:     6969,
				Username: "transmission",
				Password: "transmission",
			},
		},
	}

	as, err := NewAppState(cfg)
	if err != nil {
		t.Fatalf("NewAppState failed: %v", err)
	}

	// Verify clients were initialized
	if len(as.Clients) != 2 {
		t.Fatalf("Expected 2 clients, got %d", len(as.Clients))
	}

	// Verify current client
	current := as.CurrentClient()
	if current == nil {
		t.Fatalf("Expected current client, got nil")
	}
	if current.Type != "qbittorrent" {
		t.Errorf("Expected current client type 'qbittorrent', got %s", current.Type)
	}

	// Verify defaults
	if as.Filter != FilterAll {
		t.Errorf("Expected filter FilterAll, got %v", as.Filter)
	}
	if as.SortBy != SortByName {
		t.Errorf("Expected sort SortByName, got %v", as.SortBy)
	}
	if !as.ShowHints {
		t.Errorf("Expected ShowHints true, got %v", as.ShowHints)
	}
}

// TestSwitchClient tests switching between clients
func TestSwitchClient(t *testing.T) {
	cfg := &config.Config{
		UI: config.UIConfig{
			DefaultClient:      "qbt-local",
			ShowHints:          true,
			DefaultColorScheme: "default",
		},
		Clients: []config.ClientConfig{
			{
				Type: "qbittorrent",
				ID:   "qbt-local",
				Name: "qBittorrent",
				Host: "localhost",
				Port: 8080,
			},
			{
				Type: "transmission",
				ID:   "trans-local",
				Name: "Transmission",
				Host: "localhost",
				Port: 6969,
			},
		},
	}

	as, _ := NewAppState(cfg)

	// Initially first client
	if as.CurrentClientIdx != 0 {
		t.Errorf("Expected CurrentClientIdx 0, got %d", as.CurrentClientIdx)
	}

	// Switch to next
	as.SwitchClient()
	if as.CurrentClientIdx != 1 {
		t.Errorf("Expected CurrentClientIdx 1, got %d", as.CurrentClientIdx)
	}

	// Switch again (wraps around)
	as.SwitchClient()
	if as.CurrentClientIdx != 0 {
		t.Errorf("Expected CurrentClientIdx 0 (wrapped), got %d", as.CurrentClientIdx)
	}
}

// TestFilterOptions verifies FilterOptions returns the expected list
func TestFilterOptions(t *testing.T) {
	opts := FilterOptions()
	expected := []FilterType{
		FilterAll, FilterDownloading, FilterSeeding, FilterCompleted,
		FilterPaused, FilterStalled, FilterError, FilterQueued,
	}
	if len(opts) != len(expected) {
		t.Fatalf("Expected %d filter options, got %d", len(expected), len(opts))
	}
	for i, opt := range opts {
		if opt.Filter != expected[i] {
			t.Errorf("FilterOptions[%d]: expected %v, got %v", i, expected[i], opt.Filter)
		}
	}
}

// TestSetFilter verifies SetFilter
func TestSetFilter(t *testing.T) {
	as := &AppState{Filter: FilterAll}
	as.SetFilter(FilterPaused)
	if as.Filter != FilterPaused {
		t.Errorf("Expected FilterPaused, got %v", as.Filter)
	}
}

// TestSortOptions verifies SortOptions returns the expected list
func TestSortOptions(t *testing.T) {
	opts := SortOptions()
	expected := []SortType{
		SortByName, SortByName,
		SortByProgress, SortByProgress,
		SortBySpeed, SortBySpeed,
		SortBySeeds, SortBySeeds,
		SortByRatio, SortByRatio,
		SortBySize, SortBySize,
		SortByLeech, SortByLeech,
		SortByETA, SortByETA,
	}
	if len(opts) != len(expected) {
		t.Fatalf("Expected %d sort options, got %d", len(expected), len(opts))
	}
	for i, opt := range opts {
		if opt.SortBy != expected[i] {
			t.Errorf("SortOptions[%d]: expected %v, got %v", i, expected[i], opt.SortBy)
		}
	}
}

// TestSetSort verifies SetSort
func TestSetSort(t *testing.T) {
	as := &AppState{SortBy: SortByName, SortAscending: true}
	as.SetSort(SortBySpeed, false)
	if as.SortBy != SortBySpeed {
		t.Errorf("Expected SortBySpeed, got %v", as.SortBy)
	}
	if as.SortAscending {
		t.Errorf("Expected SortAscending false, got true")
	}
}

// TestMatchesFilter covers all filter types against appropriate torrent statuses
func TestMatchesFilter(t *testing.T) {
	as := &AppState{}

	tests := []struct {
		filter FilterType
		status client.TorrentStatus
		match  bool
	}{
		{FilterAll, client.StatusDownloading, true},
		{FilterAll, client.StatusPaused, true},
		{FilterDownloading, client.StatusDownloading, true},
		{FilterDownloading, client.StatusPaused, false},
		{FilterSeeding, client.StatusSeeding, true},
		{FilterSeeding, client.StatusDownloading, false},
		{FilterCompleted, client.StatusCompleted, true},
		{FilterCompleted, client.StatusDownloading, false},
		{FilterPaused, client.StatusPaused, true},
		{FilterPaused, client.StatusDownloading, false},
		{FilterStalled, client.StatusStalledDL, true},
		{FilterStalled, client.StatusDownloading, false},
		{FilterError, client.StatusError, true},
		{FilterError, client.StatusDownloading, false},
		{FilterQueued, client.StatusQueuedDL, true},
		{FilterQueued, client.StatusDownloading, false},
	}

	for _, tt := range tests {
		as.Filter = tt.filter
		result := as.matchesFilter(client.Torrent{Status: tt.status})
		if result != tt.match {
			t.Errorf("matchesFilter(%s, %s): expected %v, got %v", tt.filter, tt.status, tt.match, result)
		}
	}
}

// TestSortTorrents verifies sorting behavior with direction
func TestSortTorrents(t *testing.T) {
	torrents := []client.Torrent{
		{Name: "zeta", Progress: 50, SpeedDown: 100, SpeedUp: 10, Seeds: 5, Size: 300, Leechs: 2, Uploaded: 200, Downloaded: 100, ETA: 100},
		{Name: "alpha", Progress: 90, SpeedDown: 200, SpeedUp: 20, Seeds: 10, Size: 100, Leechs: 5, Uploaded: 500, Downloaded: 500, ETA: 50},
		{Name: "beta", Progress: 10, SpeedDown: 50, SpeedUp: 5, Seeds: 1, Size: 200, Leechs: 1, Uploaded: 0, Downloaded: 0, ETA: 8640000},
	}

	as := &AppState{}

	t.Run("ByNameAscending", func(t *testing.T) {
		as.SetSort(SortByName, true)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].Name != "alpha" || cp[2].Name != "zeta" {
			t.Errorf("ByName asc: expected alpha, beta, zeta; got %s, %s, %s", cp[0].Name, cp[1].Name, cp[2].Name)
		}
	})

	t.Run("ByNameDescending", func(t *testing.T) {
		as.SetSort(SortByName, false)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].Name != "zeta" || cp[2].Name != "alpha" {
			t.Errorf("ByName desc: expected zeta, beta, alpha; got %s, %s, %s", cp[0].Name, cp[1].Name, cp[2].Name)
		}
	})

	t.Run("ByProgressAscending", func(t *testing.T) {
		as.SetSort(SortByProgress, true)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].Progress != 10 || cp[2].Progress != 90 {
			t.Errorf("ByProgress asc: expected 10, 50, 90; got %d, %d, %d", cp[0].Progress, cp[1].Progress, cp[2].Progress)
		}
	})

	t.Run("ByProgressDescending", func(t *testing.T) {
		as.SetSort(SortByProgress, false)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].Progress != 90 || cp[2].Progress != 10 {
			t.Errorf("ByProgress desc: expected 90, 50, 10; got %d, %d, %d", cp[0].Progress, cp[1].Progress, cp[2].Progress)
		}
	})

	t.Run("ByETAAscending", func(t *testing.T) {
		as.SetSort(SortByETA, true)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].ETA != 50 || cp[1].ETA != 100 || cp[2].ETA != 8640000 {
			t.Errorf("ByETA asc: expected 50, 100, ∞")
		}
	})

	t.Run("ByETADescending", func(t *testing.T) {
		as.SetSort(SortByETA, false)
		cp := copyTorrents(torrents)
		as.sortTorrents(cp)
		if cp[0].ETA != 100 || cp[1].ETA != 50 || cp[2].ETA != 8640000 {
			t.Errorf("ByETA desc: expected 100, 50, ∞")
		}
	})
}

func copyTorrents(src []client.Torrent) []client.Torrent {
	dst := make([]client.Torrent, len(src))
	copy(dst, src)
	return dst
}

// TestSpeedLimitInitialized tests that speed limit is initialized as false
func TestSpeedLimitInitialized(t *testing.T) {
	cfg := &config.Config{
		UI: config.UIConfig{
			DefaultClient:      "qbt-local",
			ShowHints:          true,
			DefaultColorScheme: "default",
		},
		Clients: []config.ClientConfig{
			{
				Type:     "qbittorrent",
				ID:       "qbt-local",
				Name:     "Local qBittorrent",
				Host:     "localhost",
				Port:     8080,
				Username: "admin",
				Password: "admin",
			},
		},
	}

	as, _ := NewAppState(cfg)

	if as.SpeedLimitEnabled {
		t.Errorf("Expected SpeedLimitEnabled to be false on init, got %v", as.SpeedLimitEnabled)
	}
}
