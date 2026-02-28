package tui

import (
	"testing"
	"tqbtui/client"
)

func TestSearchFilter(t *testing.T) {
	// Create test torrents
	torrents := []client.Torrent{
		{ID: "1", Name: "The Linux Kernel"},
		{ID: "2", Name: "Ubuntu 22.04 LTS"},
		{ID: "3", Name: "Fedora 37"},
		{ID: "4", Name: "Debian 12"},
		{ID: "5", Name: "Linux Mint"},
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "empty query returns all",
			query:         "",
			expectedCount: 5,
			expectedIDs:   []string{"1", "2", "3", "4", "5"},
		},
		{
			name:          "whitespace query returns all",
			query:         "   ",
			expectedCount: 5,
			expectedIDs:   []string{"1", "2", "3", "4", "5"},
		},
		{
			name:          "search for Linux",
			query:         "Linux",
			expectedCount: 2,
			expectedIDs:   []string{"1", "5"},
		},
		{
			name:          "case-insensitive search",
			query:         "linux",
			expectedCount: 2,
			expectedIDs:   []string{"1", "5"},
		},
		{
			name:          "search for Ubuntu",
			query:         "Ubuntu",
			expectedCount: 1,
			expectedIDs:   []string{"2"},
		},
		{
			name:          "search no matches",
			query:         "Windows",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name:          "partial match",
			query:         "Deb",
			expectedCount: 1,
			expectedIDs:   []string{"4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSearchFilter()
			filter.SetQuery(tt.query, torrents)

			if filter.GetMatchCount() != tt.expectedCount {
				t.Errorf("expected %d matches, got %d", tt.expectedCount, filter.GetMatchCount())
			}

			results := filter.GetResults()
			if len(results) != len(tt.expectedIDs) {
				t.Errorf("expected %d results, got %d", len(tt.expectedIDs), len(results))
			}

			for i, expectedID := range tt.expectedIDs {
				if i < len(results) && results[i].ID != expectedID {
					t.Errorf("expected ID %s at position %d, got %s", expectedID, i, results[i].ID)
				}
			}
		})
	}
}

func TestSearchFilterClear(t *testing.T) {
	torrents := []client.Torrent{
		{ID: "1", Name: "Test 1"},
		{ID: "2", Name: "Test 2"},
	}

	filter := NewSearchFilter()
	filter.SetQuery("Test", torrents)

	if !filter.IsActive() {
		t.Error("expected filter to be active after SetQuery")
	}

	filter.Clear()

	if filter.IsActive() {
		t.Error("expected filter to be inactive after Clear")
	}

	if filter.GetQuery() != "" {
		t.Errorf("expected empty query, got %q", filter.GetQuery())
	}

	if filter.GetMatchCount() != 0 {
		t.Errorf("expected 0 matches after Clear, got %d", filter.GetMatchCount())
	}
}

func TestSearchFilterIsActive(t *testing.T) {
	filter := NewSearchFilter()

	if filter.IsActive() {
		t.Error("expected inactive filter initially")
	}

	torrents := []client.Torrent{{ID: "1", Name: "Test"}}
	filter.SetQuery("Test", torrents)

	if !filter.IsActive() {
		t.Error("expected active filter after SetQuery with non-empty query")
	}

	filter.SetQuery("", torrents)

	if filter.IsActive() {
		t.Error("expected inactive filter after setting empty query")
	}
}
