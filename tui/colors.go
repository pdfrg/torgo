package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/teacat/noire"
)

// Theme defines a color palette for the multi-line view
type Theme struct {
	// Status gradient colors: [status] = (color1, color2)
	StatusGradients map[string][2]color.Color
	// Text colors
	TextNormal color.Color
	TextMuted  color.Color
	TextError  color.Color
	// Background
	BgNormal   color.Color
	BgSelected color.Color
	// Progress bar empty color
	ProgressBarEmptyColor color.Color
	
	// Omarchy semantic colors
	AccentColor     color.Color  // For titles, keys, highlights
	CursorColor     color.Color  // For cursor indicator
	ForegroundColor color.Color  // For variable text
	
	// Bar styling (derived)
	StatusBarBg    color.Color
	StatusBarFg    color.Color
	StatusBarAccent color.Color
	
	// Hex string versions for convenient access (used by search bar, etc)
	AccentColorHex string  // Hex version of AccentColor
	BgNormalHex    string  // Hex version of BgNormal
}

// DefaultTheme returns the default light-on-dark theme
func DefaultTheme() Theme {
	return Theme{
		StatusGradients: map[string][2]color.Color{
			"downloading": {
				lipgloss.Color("#0087FF"), // Blue
				lipgloss.Color("#00D26A"), // Green
			},
			"seeding": {
				lipgloss.Color("#00D26A"), // Green
				lipgloss.Color("#00D9FF"), // Cyan
			},
			"paused": {
				lipgloss.Color("#FFD700"), // Yellow
				lipgloss.Color("#FFA500"), // Orange
			},
			"completed": {
				lipgloss.Color("#9D00FF"), // Purple
				lipgloss.Color("#FF00FF"), // Magenta
			},
			"error": {
				lipgloss.Color("#FFA500"), // Orange
				lipgloss.Color("#FF3333"), // Red
			},
			"queueing": {
				lipgloss.Color("#FF00FF"), // Magenta
				lipgloss.Color("#9D00FF"), // Purple
			},
			// Fallback for unknown statuses
			"unknown": {
				lipgloss.Color("#606060"), // Dark gray
				lipgloss.Color("#808080"), // Light gray
			},
		},
		TextNormal:            lipgloss.Color("#FFFFFF"), // White
		TextMuted:             lipgloss.Color("#888888"), // Gray
		TextError:             lipgloss.Color("#FF3333"), // Red
		BgNormal:              lipgloss.Color(""),        // Transparent (terminal default)
		BgSelected:            lipgloss.Color("#1E1E1E"), // Slight highlight
		ProgressBarEmptyColor: lipgloss.Color("#333333"), // Dark gray for empty progress bar
		
		// Omarchy semantic colors (fallback values)
		AccentColor:     lipgloss.Color("51"),   // Bright cyan
		CursorColor:     lipgloss.Color("214"),  // Orange
		ForegroundColor: lipgloss.Color("252"),  // Light gray
		
		// Bar styling
		StatusBarBg:    lipgloss.Color("237"),   // Dark gray (darkened from 252)
		StatusBarFg:    lipgloss.Color("252"),   // Light gray text
		StatusBarAccent: lipgloss.Color("51"),   // Bright cyan for keys
		
		// Hex versions (for built-in theme, use defaults)
		AccentColorHex: "#00d7ff",  // Bright cyan
		BgNormalHex:    "#000000",  // Black (terminal default)
	}
}

// DarkTheme is an alias for DefaultTheme
func DarkTheme() Theme {
	return DefaultTheme()
}

// LightTheme returns a light-background theme
func LightTheme() Theme {
	return Theme{
		StatusGradients: map[string][2]color.Color{
			"downloading": {
				lipgloss.Color("#0055CC"), // Dark blue
				lipgloss.Color("#00A850"), // Dark green
			},
			"seeding": {
				lipgloss.Color("#00A850"), // Dark green
				lipgloss.Color("#0099AA"), // Dark cyan
			},
			"paused": {
				lipgloss.Color("#CC8800"), // Dark yellow
				lipgloss.Color("#CC6600"), // Dark orange
			},
			"completed": {
				lipgloss.Color("#7700BB"), // Dark purple
				lipgloss.Color("#BB00BB"), // Dark magenta
			},
			"error": {
				lipgloss.Color("#CC6600"), // Dark orange
				lipgloss.Color("#CC0000"), // Dark red
			},
			"queueing": {
				lipgloss.Color("#BB00BB"), // Dark magenta
				lipgloss.Color("#7700BB"), // Dark purple
			},
			"unknown": {
				lipgloss.Color("#999999"), // Medium gray
				lipgloss.Color("#CCCCCC"), // Light gray
			},
		},
		TextNormal:            lipgloss.Color("#000000"), // Black
		TextMuted:             lipgloss.Color("#666666"), // Gray
		TextError:             lipgloss.Color("#CC0000"), // Red
		BgNormal:              lipgloss.Color(""),        // Transparent
		BgSelected:            lipgloss.Color("#E8E8E8"), // Light highlight
		ProgressBarEmptyColor: lipgloss.Color("#333333"), // Dark gray for empty progress bar
		
		// Semantic colors for light theme (no omarchy)
		AccentColor:     lipgloss.Color("33"),   // Dark cyan
		CursorColor:     lipgloss.Color("130"),  // Dark orange
		ForegroundColor: lipgloss.Color("243"),  // Dark gray
		
		// Bar styling for light theme
		StatusBarBg:    lipgloss.Color("252"),   // Light gray
		StatusBarFg:    lipgloss.Color("16"),    // Black text
		StatusBarAccent: lipgloss.Color("33"),   // Dark cyan for keys
		
		// Hex versions
		AccentColorHex: "#0099aa",  // Dark cyan
		BgNormalHex:    "#ffffff",  // White (light background)
	}
}

// HighContrastTheme returns a high-contrast theme for accessibility
func HighContrastTheme() Theme {
	return Theme{
		StatusGradients: map[string][2]color.Color{
			"downloading": {
				lipgloss.Color("#0055FF"), // Bright blue
				lipgloss.Color("#00FF00"), // Bright green
			},
			"seeding": {
				lipgloss.Color("#00FF00"), // Bright green
				lipgloss.Color("#00FFFF"), // Bright cyan
			},
			"paused": {
				lipgloss.Color("#FFFF00"), // Bright yellow
				lipgloss.Color("#FF8800"), // Bright orange
			},
			"completed": {
				lipgloss.Color("#FF00FF"), // Bright magenta
				lipgloss.Color("#AA00FF"), // Bright purple
			},
			"error": {
				lipgloss.Color("#FF8800"), // Bright orange
				lipgloss.Color("#FF0000"), // Bright red
			},
			"queueing": {
				lipgloss.Color("#FF00FF"), // Bright magenta
				lipgloss.Color("#AA00FF"), // Bright purple
			},
			"unknown": {
				lipgloss.Color("#CCCCCC"), // Light gray
				lipgloss.Color("#FFFFFF"), // White
			},
		},
		TextNormal:            lipgloss.Color("#FFFFFF"), // White
		TextMuted:             lipgloss.Color("#AAAAAA"), // Gray
		TextError:             lipgloss.Color("#FF0000"), // Bright red
		BgNormal:              lipgloss.Color(""),        // Transparent
		BgSelected:            lipgloss.Color("#333333"), // Dark highlight
		ProgressBarEmptyColor: lipgloss.Color("#333333"), // Dark gray for empty progress bar
		
		// Semantic colors for high contrast theme
		AccentColor:     lipgloss.Color("226"),  // Bright yellow
		CursorColor:     lipgloss.Color("226"),  // Bright yellow
		ForegroundColor: lipgloss.Color("255"),  // White
		
		// Bar styling for high contrast theme
		StatusBarBg:    lipgloss.Color("0"),     // Black
		StatusBarFg:    lipgloss.Color("255"),   // White text
		StatusBarAccent: lipgloss.Color("226"),  // Bright yellow for keys
		
		// Hex versions
		AccentColorHex: "#ffff00",  // Bright yellow
		BgNormalHex:    "#000000",  // Black
	}
}

// BuildThemeFromConfigColors creates a tui.Theme from config theme colors
// If the config theme has Colors set, builds an OmarchyTheme
// Otherwise returns a standard theme
func BuildThemeFromConfigColors(cfgTheme *map[string]string) Theme {
	if cfgTheme == nil || len(*cfgTheme) == 0 {
		return DefaultTheme()
	}
	return OmarchyTheme(*cfgTheme)
}

// OmarchyTheme builds a theme from omarchy colors.toml format
// colors: map with keys like "color1", "color2", "background", "accent", "cursor", "foreground"
func OmarchyTheme(colors map[string]string) Theme {
	theme := Theme{
		StatusGradients: make(map[string][2]color.Color),
		TextNormal:      lipgloss.Color("#FFFFFF"),
		TextMuted:       lipgloss.Color("#888888"),
		TextError:       lipgloss.Color("#FF3333"),
		BgNormal:        lipgloss.Color(""),
		BgSelected:      lipgloss.Color("#1E1E1E"),
	}

	bg := colors["background"]
	if bg == "" {
		bg = "#000000"
	}
	theme.BgNormalHex = bg

	// SEMANTIC COLORS FROM OMARCHY
	accentColor := colors["accent"]
	if accentColor == "" {
		accentColor = "#00d7ff"  // Fallback cyan
	}
	theme.AccentColor = lipgloss.Color(accentColor)
	theme.AccentColorHex = accentColor

	cursorColor := colors["cursor"]
	if cursorColor == "" {
		cursorColor = "#888888"  // Fallback gray
	}
	theme.CursorColor = lipgloss.Color(cursorColor)

	fgColor := colors["foreground"]
	if fgColor == "" {
		fgColor = "#888888"  // Fallback gray
	}
	theme.ForegroundColor = lipgloss.Color(fgColor)

	// TEXT COLORS: Smart calculation (soft grays, not extreme)
	if isColorDark(bg) {
		theme.TextNormal = lipgloss.Color("#d0d0d0")  // Light gray (252-like)
		theme.TextMuted = lipgloss.Color("#888888")   // Medium gray
	} else {
		theme.TextNormal = lipgloss.Color("#333333")  // Dark gray (not pure black)
		theme.TextMuted = lipgloss.Color("#666666")   // Medium gray
	}
	theme.TextError = lipgloss.Color("#ff3333")

	// STATUS BAR BACKGROUND: Darken foreground moderately
	statusBarBg := darkenColorHex(fgColor, 0.25)
	// Ensure it's dark enough to read light text
	if !isColorDark(statusBarBg) {
		statusBarBg = darkenColorHex(statusBarBg, 0.15)
	}
	theme.StatusBarBg = lipgloss.Color(statusBarBg)

	// STATUS BAR TEXT: Smart calc from bar background
	if isColorDark(statusBarBg) {
		theme.StatusBarFg = lipgloss.Color("#d0d0d0")  // Light gray
	} else {
		theme.StatusBarFg = lipgloss.Color("#333333")  // Dark gray
	}

	// STATUS BAR ACCENT: Use accent color
	theme.StatusBarAccent = theme.AccentColor

	// Get background adjustment for empty progress bar areas
	bgAdjusted := getAutoAdjustedBackground(bg)

	// Helper to get color or use fallback
	getColor := func(key string) string {
		if val, ok := colors[key]; ok && val != "" {
			return val
		}
		return "#888888" // Gray fallback
	}

	// Status gradients based on omarchy scheme
	color5 := getColor("color5")
	theme.StatusGradients["queueing"] = [2]color.Color{
		lipgloss.Color(color5),
		lipgloss.Color(bg),
	}

	color6 := getColor("color6")
	theme.StatusGradients["stalled"] = [2]color.Color{
		lipgloss.Color(color6),
		lipgloss.Color(bg),
	}
	theme.StatusGradients["stalledDL"] = theme.StatusGradients["stalled"]
	theme.StatusGradients["stalledUP"] = theme.StatusGradients["stalled"]

	color3 := getColor("color3")
	theme.StatusGradients["paused"] = [2]color.Color{
		lipgloss.Color(color3),
		lipgloss.Color(bg),
	}

	color1 := getColor("color1")
	theme.StatusGradients["error"] = [2]color.Color{
		lipgloss.Color(color1),
		lipgloss.Color(bg),
	}

	color4 := getColor("color4")
	color4Desat := desaturateColorHex(color4, 0.6)
	theme.StatusGradients["downloading"] = [2]color.Color{
		lipgloss.Color(color4Desat),
		lipgloss.Color(color4),
	}

	color2 := getColor("color2")
	color4Desat04 := desaturateColorHex(color4, 0.4)
	theme.StatusGradients["seeding"] = [2]color.Color{
		lipgloss.Color(color4Desat04),
		lipgloss.Color(color2),
	}

	color2Desat := desaturateColorHex(color2, 0.6)
	theme.StatusGradients["completed"] = [2]color.Color{
		lipgloss.Color(color2Desat),
		lipgloss.Color(color2),
	}

	theme.StatusGradients["unknown"] = [2]color.Color{
		lipgloss.Color("#606060"),
		lipgloss.Color("#808080"),
	}

	theme.ProgressBarEmptyColor = lipgloss.Color(bgAdjusted)

	return theme
}

// GetStatusGradient returns the gradient colors for a given status
func (t Theme) GetStatusGradient(status string) [2]color.Color {
	if colors, ok := t.StatusGradients[status]; ok {
		return colors
	}
	// Return unknown/fallback gradient
	if fallback, ok := t.StatusGradients["unknown"]; ok {
		return fallback
	}
	// Last resort fallback (should never happen)
	return [2]color.Color{lipgloss.Color("#606060"), lipgloss.Color("#808080")}
}

// UpdateStatusColor changes a status gradient color in the theme
func (t *Theme) UpdateStatusColor(status string, color1, color2 color.Color) {
	if t.StatusGradients == nil {
		t.StatusGradients = make(map[string][2]color.Color)
	}
	t.StatusGradients[status] = [2]color.Color{color1, color2}
}

// CurrentTheme holds the active theme
var CurrentTheme = DefaultTheme()

// SetTheme changes the active theme
func SetTheme(t Theme) {
	CurrentTheme = t
}

// GetCurrentStatusGradient returns the gradient colors for a status using the current theme
func GetCurrentStatusGradient(status string) [2]color.Color {
	return CurrentTheme.GetStatusGradient(status)
}

// Color transformation helper functions for omarchy themes

// lightenColorHex lightens a hex color by the given amount (0.0-1.0)
func lightenColorHex(hexColor string, amount float64) string {
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	lightened := c.Lighten(amount)
	return "#" + lightened.Hex()
}

// darkenColorHex darkens a hex color by the given amount (0.0-1.0)
func darkenColorHex(hexColor string, amount float64) string {
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	darkened := c.Darken(amount)
	return "#" + darkened.Hex()
}

// desaturateColorHex desaturates a hex color by the given amount (0.0-1.0)
func desaturateColorHex(hexColor string, amount float64) string {
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	desaturated := c.Desaturate(amount)
	return "#" + desaturated.Hex()
}

// isColorDark checks if a hex color is dark
func isColorDark(hexColor string) bool {
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	return c.IsDark()
}

// getAutoAdjustedBackground returns a background color adjusted for contrast
// If the input is dark, lighten by 15%; if light, darken by 15%
func getAutoAdjustedBackground(hexColor string) string {
	if isColorDark(hexColor) {
		return lightenColorHex(hexColor, 0.15)
	}
	return darkenColorHex(hexColor, 0.15)
}
