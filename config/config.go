package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	UI      UIConfig       `toml:"ui"`
	Clients []ClientConfig `toml:"clients"`
	Theme   *Theme         `toml:"-"` // Loaded separately
}

type UIConfig struct {
	DefaultClient string `toml:"default_client"`
	ShowHints     bool   `toml:"show_hints"`
	ColorScheme   string `toml:"color_scheme"` // "default", path to colors.toml, or "custom"
}

type ClientConfig struct {
	Type     string `toml:"type"`     // "qbittorrent" or "transmission"
	ID       string `toml:"id"`       // Unique identifier
	Name     string `toml:"name"`     // Display name
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// LoadConfig loads config from ~/.config/tqbtui/config.toml
func LoadConfig() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Load theme
	theme, err := loadTheme(&cfg, configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load theme: %w", err)
	}
	cfg.Theme = theme

	// Expand env vars in passwords
	for i := range cfg.Clients {
		cfg.Clients[i].Password = os.ExpandEnv(cfg.Clients[i].Password)
	}

	return &cfg, nil
}

// getConfigDirInternal returns ~/.config/tqbtui, creating it if needed
func getConfigDirInternal() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "tqbtui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}

	return configDir, nil
}

// getConfigDir is the testable wrapper
func getConfigDir() (string, error) {
	if getConfigDirOverride != nil {
		return getConfigDirOverride()
	}
	return getConfigDirInternal()
}

// getConfigDirOverride can be set by tests
var getConfigDirOverride func() (string, error)

// loadTheme loads theme from color_scheme setting
func loadTheme(cfg *Config, configDir string) (*Theme, error) {
	scheme := cfg.UI.ColorScheme
	if scheme == "" {
		scheme = "default"
	}

	switch {
	case scheme == "default":
		return DefaultTheme(), nil
	case strings.HasSuffix(scheme, ".toml"):
		// Treat as path to colors.toml
		return loadColorsToml(scheme)
	default:
		// Try as path relative to config dir
		return loadColorsToml(filepath.Join(configDir, scheme))
	}
}

// loadColorsToml loads omarchy colors.toml format
func loadColorsToml(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read colors.toml: %w", err)
	}

	var colors map[string]interface{}
	if err := toml.Unmarshal(data, &colors); err != nil {
		return nil, fmt.Errorf("failed to parse colors.toml: %w", err)
	}

	// Parse omarchy format and build Theme
	// TODO: Implement full omarchy colors.toml parsing
	return DefaultTheme(), nil
}
