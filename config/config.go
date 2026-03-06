package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	UI           UIConfig       `toml:"ui"`
	Clients      []ClientConfig `toml:"clients"`
	Theme        *Theme         `toml:"-"` // Loaded separately
	LoadedColors map[string]map[string]string `toml:"-"` // Pre-loaded colors for omarchy/custom themes
}

type UIConfig struct {
	DefaultClient      string `toml:"default_client"`
	ShowHints          bool   `toml:"show_hints"`
	DefaultColorScheme string `toml:"default_color_scheme"` // "default", "dark", "light", "highcontrast", "omarchy", or "custom"
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

// GetConfigDirForThemeMonitoring is an exported version of getConfigDir for theme file monitoring
func GetConfigDirForThemeMonitoring() (string, error) {
	return getConfigDir()
}

// getConfigDirOverride can be set by tests
var getConfigDirOverride func() (string, error)

// AvailableThemes holds the list of theme names that can be cycled through
var AvailableThemes []string

// DiscoverThemes finds all available themes at startup
// Returns slice of theme names in load order
func DiscoverThemes(configDir string) []string {
	themes := []string{"dark", "light", "highcontrast"}

	// Check for omarchy theme
	home, err := os.UserHomeDir()
	if err == nil {
		omarchyPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
		if _, err := os.Stat(omarchyPath); err == nil {
			themes = append(themes, "omarchy")
		}
	}

	// Check for custom theme in tqbtui config dir
	customPath := filepath.Join(configDir, "colors.toml")
	if _, err := os.Stat(customPath); err == nil {
		themes = append(themes, "custom")
	}

	return themes
}

// loadTheme loads theme from default_color_scheme setting
// If default_color_scheme is not set, auto-prioritize: custom > omarchy > dark
func loadTheme(cfg *Config, configDir string) (*Theme, error) {
	scheme := cfg.UI.DefaultColorScheme

	// Discover available themes
	AvailableThemes = DiscoverThemes(configDir)
	
	// Pre-load colors for omarchy and custom themes
	cfg.LoadedColors = make(map[string]map[string]string)
	
	// Load omarchy colors if it exists
	home, err := os.UserHomeDir()
	if err == nil {
		omarchyPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
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
			home, err := os.UserHomeDir()
			if err == nil {
				omarchyPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
				if _, err := os.Stat(omarchyPath); err == nil {
					scheme = "omarchy"
				} else {
					// Fall back to dark theme
					scheme = "dark"
				}
			} else {
				// Can't access home dir, fall back to dark
				scheme = "dark"
			}
		}
	}

	switch {
	case scheme == "dark":
		return DefaultTheme(), nil
	case scheme == "light":
		return DefaultTheme(), nil // Will need to get this from tui package
	case scheme == "highcontrast":
		return DefaultTheme(), nil // Will need to get this from tui package
	case scheme == "omarchy":
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		omarchyPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
		return loadColorsToml(omarchyPath)
	case scheme == "custom":
		customPath := filepath.Join(configDir, "colors.toml")
		return loadColorsToml(customPath)
	default:
		// Try to load as direct path
		if _, err := os.Stat(scheme); err == nil {
			return loadColorsToml(scheme)
		}
		// Try relative to config dir
		relPath := filepath.Join(configDir, scheme)
		if _, err := os.Stat(relPath); err == nil {
			return loadColorsToml(relPath)
		}
		// Fall back to default with error logged
		fmt.Printf("Warning: color scheme '%s' not found, using default\n", scheme)
		return DefaultTheme(), nil
	}
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

// loadColorsToml loads omarchy colors.toml format
func loadColorsToml(path string) (*Theme, error) {
	colors, err := loadColorsFile(path)
	if err != nil {
		return nil, err
	}

	// Return the colors map wrapped in a Theme
	// The actual theme building happens in tui package
	return buildOmarchyTheme(colors), nil
}

// buildOmarchyTheme creates a theme with the colors stored for later processing
func buildOmarchyTheme(colors map[string]string) *Theme {
	t := DefaultTheme()
	// Store the raw colors map for tui package to process
	t.Colors = colors
	return t
}

// ReloadThemeColors refreshes the colors for a theme from disk
// Used when user switches to omarchy or custom theme to ensure fresh data
func (c *Config) ReloadThemeColors(themeName string) error {
	if themeName == "omarchy" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		omarchyPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
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
