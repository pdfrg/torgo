package tui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SubdirectoryCache manages subdirectory enumeration with debouncing
type SubdirectoryCache struct {
	BasePath         string
	Subdirectories   []string
	LastFetchTime    time.Time
	LastQueryPath    string
	DebounceTimer    *time.Timer
	DebounceCallback func([]string)
}

// SubdirectoryHelper provides utilities for subdirectory detection and navigation
type SubdirectoryHelper struct {
	cache *SubdirectoryCache
}

// NewSubdirectoryHelper creates a new helper
func NewSubdirectoryHelper() *SubdirectoryHelper {
	return &SubdirectoryHelper{
		cache: &SubdirectoryCache{
			Subdirectories: []string{},
		},
	}
}

// ListSubdirectories lists immediate subdirectories of basePath
// Returns only directories that exist and are writeable
func ListSubdirectories(basePath string) ([]string, error) {
	// Ensure path exists
	info, err := os.Stat(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Path doesn't exist, return empty list
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	// Ensure it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", basePath)
	}

	// Check if writeable by trying to open
	if err := canWrite(basePath); err != nil {
		return []string{}, nil // Not writeable, return empty list
	}

	// List entries
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if subdir is writeable
			fullPath := filepath.Join(basePath, entry.Name())
			if err := canWrite(fullPath); err == nil {
				subdirs = append(subdirs, entry.Name())
			}
		}
	}

	// Sort alphabetically
	sort.Strings(subdirs)
	return subdirs, nil
}

// GetSubdirectoryFromPath extracts the final directory name from a path
// e.g., "/path/to/movies" returns "movies"
func GetSubdirectoryFromPath(path string) string {
	if path == "" {
		return ""
	}

	// Remove trailing slash if present
	path = strings.TrimSuffix(path, "/")

	// Get base name
	return filepath.Base(path)
}

// GetParentPath returns the parent directory of the given path
// e.g., "/path/to/dl-dir/movies" returns "/path/to/dl-dir"
func GetParentPath(path string) string {
	if path == "" {
		return ""
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Get parent
	return filepath.Dir(path)
}

// IsPathWithTrailingSlash checks if path ends with separator
func IsPathWithTrailingSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

// GetBasePathForListing determines which path to list subdirectories from
// If path ends with /, list that path's contents
// If path doesn't end with /, list the parent of that path
func GetBasePathForListing(path string) string {
	if path == "" {
		return ""
	}

	if IsPathWithTrailingSlash(path) {
		// Path ends with slash: list this path's subdirectories
		return strings.TrimSuffix(path, "/")
	}

	// Path doesn't end with slash: list parent's subdirectories
	return GetParentPath(path)
}

// canWrite checks if a path is writeable
func canWrite(path string) error {
	// Try to write a test file
	testFile := filepath.Join(path, ".tqbtui_write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return err
	}
	// Clean up
	_ = os.Remove(testFile)
	return nil
}

// FetchSubdirectories fetches subdirectories from a path
// Returns empty list if path doesn't exist or isn't writeable
func (h *SubdirectoryHelper) FetchSubdirectories(path string) ([]string, error) {
	if path == "" {
		return []string{}, nil
	}

	basePath := GetBasePathForListing(path)
	if basePath == "" {
		return []string{}, nil
	}

	// Check if we've already fetched this
	if h.cache.LastQueryPath == basePath && time.Since(h.cache.LastFetchTime) < 1*time.Second {
		return h.cache.Subdirectories, nil
	}

	subdirs, err := ListSubdirectories(basePath)
	if err != nil {
		// Log error but return empty list rather than failing
		return []string{}, nil
	}

	// Cache result
	h.cache.BasePath = basePath
	h.cache.Subdirectories = subdirs
	h.cache.LastFetchTime = time.Now()
	h.cache.LastQueryPath = basePath

	return subdirs, nil
}

// InvalidateCache clears the cache
func (h *SubdirectoryHelper) InvalidateCache() {
	h.cache.Subdirectories = []string{}
	h.cache.LastQueryPath = ""
}

// IsLocalhost checks if a hostname refers to the local machine
// Returns true for: localhost, 127.0.0.1, ::1, or the local hostname
func IsLocalhost(hostname string) bool {
	if hostname == "" {
		return false
	}

	// Direct string matches
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return true
	}

	// Check if hostname resolves to localhost
	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		if addr.IsLoopback() {
			return true
		}
	}

	// Check against local hostname
	localHostname, err := os.Hostname()
	if err == nil && hostname == localHostname {
		return true
	}

	return false
}
