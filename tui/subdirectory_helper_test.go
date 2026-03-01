package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSubdirectoryFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/path/to/movies",
			expected: "movies",
		},
		{
			name:     "path with trailing slash",
			path:     "/path/to/movies/",
			expected: "movies",
		},
		{
			name:     "single directory",
			path:     "movies",
			expected: "movies",
		},
		{
			name:     "relative path",
			path:     "relative/to/tv",
			expected: "tv",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "root",
			path:     "/",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSubdirectoryFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("GetSubdirectoryFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "nested path",
			path:     "/path/to/movies",
			expected: "/path/to",
		},
		{
			name:     "path with trailing slash",
			path:     "/path/to/movies/",
			expected: "/path/to",
		},
		{
			name:     "single level",
			path:     "movies",
			expected: ".",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetParentPath(tt.path)
			if result != tt.expected {
				t.Errorf("GetParentPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsPathWithTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "with trailing slash",
			path:     "/path/to/movies/",
			expected: true,
		},
		{
			name:     "without trailing slash",
			path:     "/path/to/movies",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "just slash",
			path:     "/",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithTrailingSlash(tt.path)
			if result != tt.expected {
				t.Errorf("IsPathWithTrailingSlash(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetBasePathForListing(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "path with trailing slash",
			path:     "/path/to/dl-dir/",
			expected: "/path/to/dl-dir",
		},
		{
			name:     "path without trailing slash",
			path:     "/path/to/dl-dir",
			expected: "/path/to",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBasePathForListing(tt.path)
			if result != tt.expected {
				t.Errorf("GetBasePathForListing(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestListSubdirectories(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	
	// Create some subdirectories
	subDirs := []string{"movies", "tv", "music", "other"}
	for _, dir := range subDirs {
		err := os.Mkdir(filepath.Join(tmpDir, dir), 0755)
		if err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
	}

	// Create a file (should be ignored)
	file := filepath.Join(tmpDir, "somefile.txt")
	err := os.WriteFile(file, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Test listing
	result, err := ListSubdirectories(tmpDir)
	if err != nil {
		t.Fatalf("ListSubdirectories failed: %v", err)
	}

	// Should return 4 directories in alphabetical order
	expected := []string{"movies", "music", "other", "tv"}
	if len(result) != len(expected) {
		t.Errorf("expected %d subdirectories, got %d", len(expected), len(result))
	}

	for i, dir := range expected {
		if i >= len(result) || result[i] != dir {
			t.Errorf("subdirectory %d: expected %q, got %q", i, dir, result[i])
		}
	}
}

func TestListSubdirectoriesNonExistent(t *testing.T) {
	result, err := ListSubdirectories("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("ListSubdirectories should not error for missing path: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("expected empty list for nonexistent path, got %v", result)
	}
}

func TestSubdirectoryHelper(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create subdirectories
	os.Mkdir(filepath.Join(tmpDir, "movies"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "tv"), 0755)

	helper := NewSubdirectoryHelper()
	
	// Test fetching
	result, err := helper.FetchSubdirectories(tmpDir + "/")
	if err != nil {
		t.Fatalf("FetchSubdirectories failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 subdirectories, got %d", len(result))
	}

	// Test cache
	result2, err := helper.FetchSubdirectories(tmpDir + "/")
	if err != nil {
		t.Fatalf("FetchSubdirectories (cached) failed: %v", err)
	}

	if len(result2) != len(result) {
		t.Errorf("cached result differs: expected %d, got %d", len(result), len(result2))
	}

	// Test cache invalidation
	helper.InvalidateCache()
	if len(helper.cache.Subdirectories) != 0 {
		t.Errorf("cache not invalidated properly")
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{
			name:     "localhost string",
			hostname: "localhost",
			expected: true,
		},
		{
			name:     "loopback IPv4",
			hostname: "127.0.0.1",
			expected: true,
		},
		{
			name:     "loopback IPv6",
			hostname: "::1",
			expected: true,
		},
		{
			name:     "empty string",
			hostname: "",
			expected: false,
		},
		{
			name:     "remote hostname",
			hostname: "remote.example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalhost(tt.hostname)
			if result != tt.expected {
				t.Errorf("IsLocalhost(%q) = %v, want %v", tt.hostname, result, tt.expected)
			}
		})
	}
}
