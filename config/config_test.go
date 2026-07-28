package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig tests loading a valid config file
func TestLoadConfig(t *testing.T) {
	// Create temporary config directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "torgo")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.toml")

	configContent := `
[ui]
default_client = "qbt-local"
show_hints = true
default_color_scheme = "default"

[[clients]]
type = "qbittorrent"
id = "qbt-local"
name = "Local qBittorrent"
host = "localhost"
port = 8080
username = "admin"
password = "admin"

[[clients]]
type = "transmission"
id = "trans-local"
name = "Local Transmission"
host = "localhost"
port = 6969
username = "transmission"
password = "transmission"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Mock getConfigDir to return our temp directory
	originalOverride := getConfigDirOverride
	getConfigDirOverride = func() (string, error) {
		return configDir, nil
	}
	defer func() {
		getConfigDirOverride = originalOverride
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify UI config
	if cfg.UI.DefaultClient != "qbt-local" {
		t.Errorf("Expected default_client 'qbt-local', got %s", cfg.UI.DefaultClient)
	}
	if !cfg.UI.ShowHints {
		t.Errorf("Expected show_hints true, got %v", cfg.UI.ShowHints)
	}
	if cfg.UI.DefaultColorScheme != "default" {
		t.Errorf("Expected default_color_scheme 'default', got %s", cfg.UI.DefaultColorScheme)
	}

	// Verify clients loaded
	if len(cfg.Clients) != 2 {
		t.Fatalf("Expected 2 clients, got %d", len(cfg.Clients))
	}

	// Verify qBittorrent client
	qbt := cfg.Clients[0]
	if qbt.Type != "qbittorrent" {
		t.Errorf("Expected type 'qbittorrent', got %s", qbt.Type)
	}
	if qbt.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", qbt.Host)
	}
	if qbt.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", qbt.Port)
	}

	// Verify Transmission client
	trans := cfg.Clients[1]
	if trans.Type != "transmission" {
		t.Errorf("Expected type 'transmission', got %s", trans.Type)
	}
	if trans.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", trans.Host)
	}
	if trans.Port != 6969 {
		t.Errorf("Expected port 6969, got %d", trans.Port)
	}

	// Verify theme scheme resolved
	if cfg.UI.DefaultColorScheme != "default" {
		t.Errorf("Expected default_color_scheme 'default', got %s", cfg.UI.DefaultColorScheme)
	}
}
