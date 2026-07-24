package tui

import (
	"strings"
	"tqbtui/client"
)

// normalizeForSearch replaces common torrent title separators with spaces
func normalizeForSearch(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// SearchFilter filters torrents by name using case-insensitive substring matching
type SearchFilter struct {
	query   string
	results []client.Torrent
}

// NewSearchFilter creates a new search filter
func NewSearchFilter() *SearchFilter {
	return &SearchFilter{
		query:   "",
		results: []client.Torrent{},
	}
}

// SetQuery updates the search query and filters torrents
func (s *SearchFilter) SetQuery(query string, torrents []client.Torrent) {
	s.query = strings.TrimSpace(query)
	s.results = []client.Torrent{}

	if s.query == "" {
		// No query, return all torrents
		s.results = torrents
		return
	}

	// Filter torrents by case-insensitive substring matching
	// with separator normalization (dots, dashes, underscores)
	lowerQuery := strings.ToLower(normalizeForSearch(s.query))
	for _, t := range torrents {
		if strings.Contains(strings.ToLower(normalizeForSearch(t.Name)), lowerQuery) {
			s.results = append(s.results, t)
		}
	}
}

// GetQuery returns the current search query
func (s *SearchFilter) GetQuery() string {
	return s.query
}

// GetResults returns the filtered torrents
func (s *SearchFilter) GetResults() []client.Torrent {
	return s.results
}

// Clear clears the search query and results
func (s *SearchFilter) Clear() {
	s.query = ""
	s.results = []client.Torrent{}
}

// IsActive returns true if there's an active search query
func (s *SearchFilter) IsActive() bool {
	return s.query != ""
}

// GetMatchCount returns the number of matching torrents
func (s *SearchFilter) GetMatchCount() int {
	return len(s.results)
}
