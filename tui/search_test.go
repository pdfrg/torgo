package tui

import (
	"testing"
	"tqbtui/client"
)

func TestNormalizeForSearch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"already fine", "already fine"},
		{"21.jump.street", "21 jump street"},
		{"21-jump-street", "21 jump street"},
		{"21_jump_street", "21 jump street"},
		{"21.jump_street-2024", "21 jump street 2024"},
		{"no change", "no change"},
	}
	for _, tt := range tests {
		got := normalizeForSearch(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeForSearch(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSearchFilter(t *testing.T) {
	// Create test torrents
	torrents := []client.Torrent{
		{ID: "1", Name: "The Linux Kernel"},
		{ID: "2", Name: "Ubuntu 22.04 LTS"},
		{ID: "3", Name: "Fedora 37"},
		{ID: "4", Name: "Debian 12"},
		{ID: "5", Name: "Linux Mint"},
		{ID: "6", Name: "21.jump.street.2024"},
		{ID: "7", Name: "the-walking-dead-s01"},
		{ID: "8", Name: "some_movie_2024"},
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
			expectedCount: 8,
			expectedIDs:   []string{"1", "2", "3", "4", "5", "6", "7", "8"},
		},
		{
			name:          "whitespace query returns all",
			query:         "   ",
			expectedCount: 8,
			expectedIDs:   []string{"1", "2", "3", "4", "5", "6", "7", "8"},
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
		{
			name:          "space query matches dot-separated name",
			query:         "21 jump",
			expectedCount: 1,
			expectedIDs:   []string{"6"},
		},
		{
			name:          "dot query matches dot-separated name",
			query:         "21.jump",
			expectedCount: 1,
			expectedIDs:   []string{"6"},
		},
		{
			name:          "space query matches dash-separated name",
			query:         "walking dead",
			expectedCount: 1,
			expectedIDs:   []string{"7"},
		},
		{
			name:          "dash query matches dash-separated name",
			query:         "walking-dead",
			expectedCount: 1,
			expectedIDs:   []string{"7"},
		},
		{
			name:          "space query matches underscore-separated name",
			query:         "some movie",
			expectedCount: 1,
			expectedIDs:   []string{"8"},
		},
		{
			name:          "query separator different from name separator",
			query:         "21-jump-street",
			expectedCount: 1,
			expectedIDs:   []string{"6"},
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
