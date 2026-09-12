package tui

import (
	"github.com/pdfrg/torgo/client"
	"net/url"
	"strings"
)

// normalizeForSearch replaces common torrent title separators with spaces
func normalizeForSearch(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// trackerHost extracts the lowercase hostname from a tracker URL for
// matching and display. It strips any port and a leading "www." prefix.
// Returns "" when the URL is empty or unparseable.
// Examples:
//
//	"https://tracker.torrentleech.org:443/announce" -> "tracker.torrentleech.org"
//	"udp://tracker.opentrackr.org:1337/announce"      -> "tracker.opentrackr.org"
//	""                                                -> ""
func trackerHost(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	// udp:// URLs parse fine with net/url; if there is no scheme at all,
	// prepend "//" so url.Parse treats it as a host.
	toParse := rawURL
	if !strings.Contains(toParse, "://") {
		toParse = "//" + toParse
	}
	u, err := url.Parse(toParse)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		// Fallback: bare "host/path" without scheme handling above
		host = strings.ToLower(u.Path)
		if i := strings.IndexAny(host, "/:?#"); i >= 0 {
			host = host[:i]
		}
	}
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "www.")
	return host
}

// trackerNormalize strips separators and lowercases so that
// "torrent leech", "torrent-leech" and "torrentleech" all match each other.
func trackerNormalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// parseSearchQuery splits a raw search query into a title substring and
// tracker terms. Tokens of the form "tracker:<v>" or "tr:<v>" (case-insensitive
// prefix) are tracker filters; everything else is title text. Comma-separated
// values inside one token mean OR, e.g. "tr:tl,mao arch" -> tracker terms
// ["tl" "mao"], title "arch".
func parseSearchQuery(query string) (titleQuery string, trackerTerms []string) {
	var titleParts []string
	for _, tok := range strings.Fields(query) {
		lower := strings.ToLower(tok)
		var val string
		switch {
		case strings.HasPrefix(lower, "tracker:"):
			val = tok[len("tracker:"):]
		case strings.HasPrefix(lower, "tr:"):
			val = tok[len("tr:"):]
		default:
			titleParts = append(titleParts, tok)
			continue
		}
		for _, part := range strings.Split(val, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				trackerTerms = append(trackerTerms, part)
			}
		}
	}
	return strings.Join(titleParts, " "), trackerTerms
}

// SearchFilter filters torrents by name using case-insensitive substring matching,
// optionally narrowed to specific trackers via "tracker:<v>" / "tr:<v>" tokens.
type SearchFilter struct {
	query        string
	titleQuery   string
	trackerTerms []string
	results      []client.Torrent
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
	s.titleQuery, s.trackerTerms = parseSearchQuery(s.query)
	s.results = []client.Torrent{}

	titleQuery := strings.TrimSpace(s.titleQuery)
	if titleQuery == "" && len(s.trackerTerms) == 0 {
		// No constraints, return all torrents
		s.results = torrents
		return
	}

	// Filter torrents by case-insensitive substring matching
	// with separator normalization (dots, dashes, underscores)
	lowerTitle := strings.ToLower(normalizeForSearch(titleQuery))
	trackerNorms := make([]string, 0, len(s.trackerTerms))
	for _, term := range s.trackerTerms {
		if n := trackerNormalize(term); n != "" {
			trackerNorms = append(trackerNorms, n)
		}
	}

	for _, t := range torrents {
		if titleQuery != "" {
			if !strings.Contains(strings.ToLower(normalizeForSearch(t.Name)), lowerTitle) {
				continue
			}
		}
		if len(trackerNorms) > 0 {
			hostNorm := trackerNormalize(trackerHost(t.TrackerURL))
			if hostNorm == "" {
				continue
			}
			matched := false
			for _, term := range trackerNorms {
				if strings.Contains(hostNorm, term) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		s.results = append(s.results, t)
	}
}

// GetQuery returns the current raw search query
func (s *SearchFilter) GetQuery() string {
	return s.query
}

// GetTitleQuery returns the title portion of the query (tracker tokens removed)
func (s *SearchFilter) GetTitleQuery() string {
	return s.titleQuery
}

// GetTrackerTerms returns the parsed tracker filter terms
func (s *SearchFilter) GetTrackerTerms() []string {
	return s.trackerTerms
}

// GetResults returns the filtered torrents
func (s *SearchFilter) GetResults() []client.Torrent {
	return s.results
}

// Clear clears the search query and results
func (s *SearchFilter) Clear() {
	s.query = ""
	s.titleQuery = ""
	s.trackerTerms = nil
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
