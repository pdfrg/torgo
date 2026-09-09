package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	UI           UIConfig                     `toml:"ui"`
	Clients      []ClientConfig               `toml:"clients"`
	LoadedColors map[string]map[string]string `toml:"-"` // Pre-loaded colors for omarchy/custom themes
}

type UIConfig struct {
	DefaultClient      string `toml:"default_client"`
	ShowHints          bool   `toml:"show_hints"`
	DefaultColorScheme string `toml:"default_color_scheme"` // "default", "dark", "light", "highcontrast", "omarchy", "custom", or "terminal"
}

type ClientConfig struct {
	Type     string `toml:"type"` // "qbittorrent" or "transmission"
	ID       string `toml:"id"`   // Unique identifier
	Name     string `toml:"name"` // Display name
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// LoadConfig loads config from the given path.
// If cfgPath is empty, defaults to ~/.config/torgo/config.toml.
func LoadConfig(cfgPath string) (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	if cfgPath == "" {
		cfgPath = filepath.Join(configDir, "config.toml")
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Load theme
	if err := loadTheme(&cfg, configDir); err != nil {
		return nil, fmt.Errorf("failed to load theme: %w", err)
	}

	// Expand env vars in passwords
	for i := range cfg.Clients {
		cfg.Clients[i].Password = os.ExpandEnv(cfg.Clients[i].Password)
	}

	return &cfg, nil
}

// getConfigDirInternal returns ~/.config/torgo, creating it if needed
func getConfigDirInternal() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "torgo")
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

// GetConfigDirForThemeMonitoring is an exported version of getConfigDir for theme file monitoring
func GetConfigDirForThemeMonitoring() (string, error) {
	return getConfigDir()
}

// getConfigDirOverride can be set by tests
var getConfigDirOverride func() (string, error)

// AvailableThemes holds the list of theme names that can be cycled through
var AvailableThemes []string

// omarchyColorsCandidates returns paths to check for the active Omarchy theme,
// newest location first. Omarchy 4 (quattro) renders the active theme under
// $XDG_STATE_HOME/omarchy/current/theme/ (default ~/.local/state/...); older
// releases used ~/.config/omarchy/current/theme/.
func omarchyColorsCandidates() []string {
	var dirs []string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "omarchy", "current", "theme", "colors.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "state", "omarchy", "current", "theme", "colors.toml"))
		dirs = append(dirs, filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml"))
	}
	return dirs
}

// OmarchyColorsPath returns the path of the active Omarchy colors.toml,
// or "" if no Omarchy theme is present.
func OmarchyColorsPath() string {
	for _, p := range omarchyColorsCandidates() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// DiscoverThemes finds all available themes at startup
// Returns slice of theme names in load order
func DiscoverThemes(configDir string) []string {
	themes := []string{"dark", "light", "highcontrast", "terminal"}

	// Check for omarchy theme
	if OmarchyColorsPath() != "" {
		themes = append(themes, "omarchy")
	}

	// Check for custom theme in torgo config dir
	customPath := filepath.Join(configDir, "colors.toml")
	if _, err := os.Stat(customPath); err == nil {
		themes = append(themes, "custom")
	}

	return themes
}

// loadTheme loads theme from default_color_scheme setting
// If default_color_scheme is not set, auto-prioritize: custom > omarchy > dark
func loadTheme(cfg *Config, configDir string) error {
	scheme := cfg.UI.DefaultColorScheme

	// Discover available themes
	AvailableThemes = DiscoverThemes(configDir)

	// Pre-load colors for omarchy and custom themes
	cfg.LoadedColors = make(map[string]map[string]string)

	// Load omarchy colors if it exists
	if omarchyPath := OmarchyColorsPath(); omarchyPath != "" {
		if colors, err := loadColorsFile(omarchyPath); err == nil {
			cfg.LoadedColors["omarchy"] = colors
		}
	}

	// Load custom colors if it exists
	customPath := filepath.Join(configDir, "colors.toml")
	if colors, err := loadColorsFile(customPath); err == nil {
		cfg.LoadedColors["custom"] = colors
	}

	// If default_color_scheme not explicitly set, auto-detect based on file presence
	if scheme == "" {
		// Check for custom colors.toml first (highest priority)
		if _, err := os.Stat(customPath); err == nil {
			scheme = "custom"
		} else {
			// Check for omarchy colors second
			if OmarchyColorsPath() != "" {
				scheme = "omarchy"
			} else {
				// Fall back to dark theme
				scheme = "dark"
			}
		}
	}

	// Save the resolved scheme for use by the tui package at startup
	cfg.UI.DefaultColorScheme = scheme

	return nil
}

// loadColorsFile reads and parses a colors.toml file, returning raw colors map
func loadColorsFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read colors.toml: %w", err)
	}

	var colors map[string]string
	if err := toml.Unmarshal(data, &colors); err != nil {
		return nil, fmt.Errorf("failed to parse colors.toml: %w", err)
	}

	return colors, nil
}

// ReloadThemeColors refreshes the colors for a theme from disk
// Used when user switches to omarchy or custom theme to ensure fresh data
func (c *Config) ReloadThemeColors(themeName string) error {
	if themeName == "omarchy" {
		omarchyPath := OmarchyColorsPath()
		if omarchyPath == "" {
			return fmt.Errorf("no omarchy colors.toml found")
		}
		colors, err := loadColorsFile(omarchyPath)
		if err != nil {
			return err
		}
		c.LoadedColors["omarchy"] = colors
	} else if themeName == "custom" {
		configDir, err := getConfigDir()
		if err != nil {
			return err
		}
		customPath := filepath.Join(configDir, "colors.toml")
		colors, err := loadColorsFile(customPath)
		if err != nil {
			return err
		}
		c.LoadedColors["custom"] = colors
	}
	return nil
}
