package tui

import (
	"testing"
	"torgo/client"
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

func TestTrackerHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://tracker.torrentleech.org:443/announce", "tracker.torrentleech.org"},
		{"udp://tracker.opentrackr.org:1337/announce", "tracker.opentrackr.org"},
		{"http://www.example.com/announce", "example.com"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		if got := trackerHost(tt.input); got != tt.expected {
			t.Errorf("trackerHost(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseSearchQuery(t *testing.T) {
	title, terms := parseSearchQuery("tr:torrentleech arch linux")
	if title != "arch linux" {
		t.Errorf("expected title %q, got %q", "arch linux", title)
	}
	if len(terms) != 1 || terms[0] != "torrentleech" {
		t.Errorf("expected terms [torrentleech], got %v", terms)
	}

	title, terms = parseSearchQuery("tracker:tl,mao arch")
	if title != "arch" {
		t.Errorf("expected title %q, got %q", "arch", title)
	}
	if len(terms) != 2 || terms[0] != "tl" || terms[1] != "mao" {
		t.Errorf("expected terms [tl mao], got %v", terms)
	}

	title, terms = parseSearchQuery("TR:TorrentLeech arch")
	if title != "arch" || len(terms) != 1 || terms[0] != "TorrentLeech" {
		t.Errorf("case-insensitive prefix failed: title=%q terms=%v", title, terms)
	}

	title, terms = parseSearchQuery("plain title only")
	if title != "plain title only" || len(terms) != 0 {
		t.Errorf("plain query failed: title=%q terms=%v", title, terms)
	}
}

func TestSearchFilterTracker(t *testing.T) {
	torrents := []client.Torrent{
		{ID: "1", Name: "Ubuntu 22.04", TrackerURL: "https://tracker.torrentleech.org:443/announce"},
		{ID: "2", Name: "Ubuntu 20.04", TrackerURL: "udp://tracker.opentrackr.org:1337/announce"},
		{ID: "3", Name: "Fedora 37", TrackerURL: "https://tracker.torrentleech.org:443/announce"},
		{ID: "4", Name: "Debian 12", TrackerURL: ""},
	}

	tests := []struct {
		name        string
		query       string
		expectedIDs []string
	}{
		{"tracker only short", "tr:torrentleech", []string{"1", "3"}},
		{"tracker long prefix", "tracker:torrentleech", []string{"1", "3"}},
		{"tracker abbreviation substring", "tr:tl", []string{"1", "3"}},
		{"tracker combined with title", "tr:torrentleech ubuntu", []string{"1"}},
		{"tracker combined no title match", "tr:opentrackr ubuntu", []string{"2"}},
		{"tracker combined excludes", "tr:torrentleech fedora", []string{"3"}},
		{"tracker OR list", "tr:torrentleech,opentrackr", []string{"1", "2", "3"}},
		{"tracker no match", "tr:nonexistent", []string{}},
		{"title only ignores tracker", "ubuntu", []string{"1", "2"}},
		{"empty tracker value is title-only", "tr: ubuntu", []string{"1", "2"}},
		{"torrent without tracker excluded", "tr:torrentleech debian", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSearchFilter()
			filter.SetQuery(tt.query, torrents)
			results := filter.GetResults()
			if len(results) != len(tt.expectedIDs) {
				t.Fatalf("query %q: expected %d results %v, got %d (%v)",
					tt.query, len(tt.expectedIDs), tt.expectedIDs, len(results), results)
			}
			for i, id := range tt.expectedIDs {
				if results[i].ID != id {
					t.Errorf("query %q: expected ID %s at %d, got %s", tt.query, id, i, results[i].ID)
				}
			}
		})
	}
}
