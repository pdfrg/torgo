package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		// Magnet links
		{
			name:    "valid magnet link",
			input:   "magnet:?xt=urn:btih:abc123",
			wantErr: false,
		},
		{
			name:    "valid magnet with multiple params",
			input:   "magnet:?xt=urn:btih:abc123&dn=test&tr=http://tracker.example.com",
			wantErr: false,
		},
		{
			name:    "invalid magnet - missing xt",
			input:   "magnet:?dn=test",
			wantErr: true,
			errMsg:  "Magnet link missing xt parameter",
		},
		{
			name:    "invalid magnet - malformed",
			input:   "magnet:noparams",
			wantErr: true,
			errMsg:  "Invalid magnet link format",
		},

		// URLs
		{
			name:    "valid http URL",
			input:   "http://example.com/torrent.torrent",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			input:   "https://example.com/torrent.torrent",
			wantErr: false,
		},
		{
			name:    "invalid URL - bad format",
			input:   "http://",
			wantErr: true,
			errMsg:  "Invalid URL: missing host",
		},
		// Note: ftp:// is treated as file path, not URL
		// so it will fail with "File not found", which is correct behavior

		// File paths
		{
			name:    "empty string",
			input:   "",
			wantErr: false, // Empty is valid at validation stage
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(tt.input)
			if tt.wantErr && err == "" {
				t.Errorf("expected error, got none")
			}
			if !tt.wantErr && err != "" {
				t.Errorf("expected no error, got: %s", err)
			}
			if tt.wantErr && tt.errMsg != "" && err != tt.errMsg {
				t.Errorf("expected error %q, got %q", tt.errMsg, err)
			}
		})
	}
}

func TestValidateInputFilePath(t *testing.T) {
	// Create a temporary .torrent file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.torrent")
	if err := os.WriteFile(tmpFile, []byte("fake torrent"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid file path",
			input:   tmpFile,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			input:   filepath.Join(tmpDir, "nonexistent.torrent"),
			wantErr: true,
			errMsg:  "File not found",
		},
		{
			name:    "wrong extension",
			input:   filepath.Join(tmpDir, "test.txt"),
			wantErr: true,
			errMsg:  "File must be a .torrent file",
		},
	}

	// Create a non-torrent file
	nonTorrentFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(nonTorrentFile, []byte("not torrent"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(tt.input)
			if tt.wantErr && err == "" {
				t.Errorf("expected error, got none")
			}
			if !tt.wantErr && err != "" {
				t.Errorf("expected no error, got: %s", err)
			}
			if tt.wantErr && tt.errMsg != "" && err != tt.errMsg {
				t.Errorf("expected error %q, got %q", tt.errMsg, err)
			}
		})
	}
}

func TestIsValidForSubmit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid magnet",
			input:   "magnet:?xt=urn:btih:abc123",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "Please enter a magnet link, URL, or .torrent file path",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
			errMsg:  "Please enter a magnet link, URL, or .torrent file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidForSubmit(tt.input)
			if tt.wantErr && err == "" {
				t.Errorf("expected error, got none")
			}
			if !tt.wantErr && err != "" {
				t.Errorf("expected no error, got: %s", err)
			}
			if tt.wantErr && tt.errMsg != "" && err != tt.errMsg {
				t.Errorf("expected error %q, got %q", tt.errMsg, err)
			}
		})
	}
}
