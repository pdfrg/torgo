package state

import (
	"testing"
	"tqbtui/config"
)

// TestNewAppState tests creating a new app state from config
func TestNewAppState(t *testing.T) {
	cfg := &config.Config{
		UI: config.UIConfig{
			DefaultClient: "qbt-local",
			ShowHints:     true,
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
		Theme: config.DefaultTheme(),
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
		t.Errorf("Expected current client, got nil")
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
			DefaultClient: "qbt-local",
			ShowHints:     true,
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
		Theme: config.DefaultTheme(),
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

// TestCycleFilter tests filter cycling
func TestCycleFilter(t *testing.T) {
	as := &AppState{Filter: FilterAll}

	tests := []struct {
		start    FilterType
		expected FilterType
	}{
		{FilterAll, FilterActive},
		{FilterActive, FilterPaused},
		{FilterPaused, FilterCompleted},
		{FilterCompleted, FilterAll},
	}

	for _, tt := range tests {
		as.Filter = tt.start
		as.CycleFilter()
		if as.Filter != tt.expected {
			t.Errorf("CycleFilter from %v: expected %v, got %v", tt.start, tt.expected, as.Filter)
		}
	}
}

// TestCycleSort tests sort cycling
func TestCycleSort(t *testing.T) {
	as := &AppState{SortBy: SortByName}

	tests := []struct {
		start    SortType
		expected SortType
	}{
		{SortByName, SortByProgress},
		{SortByProgress, SortBySpeed},
		{SortBySpeed, SortBySeeds},
		{SortBySeeds, SortByName},
	}

	for _, tt := range tests {
		as.SortBy = tt.start
		as.CycleSort()
		if as.SortBy != tt.expected {
			t.Errorf("CycleSort from %v: expected %v, got %v", tt.start, tt.expected, as.SortBy)
		}
	}
}

// TestSpeedLimitInitialized tests that speed limit is initialized as false
func TestSpeedLimitInitialized(t *testing.T) {
	cfg := &config.Config{
		UI: config.UIConfig{
			DefaultClient: "qbt-local",
			ShowHints:     true,
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
		Theme: config.DefaultTheme(),
	}

	as, _ := NewAppState(cfg)

	if as.SpeedLimitEnabled {
		t.Errorf("Expected SpeedLimitEnabled to be false on init, got %v", as.SpeedLimitEnabled)
	}
}
