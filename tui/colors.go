package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
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
		TextNormal:   lipgloss.Color("#FFFFFF"), // White
		TextMuted:    lipgloss.Color("#888888"), // Gray
		TextError:    lipgloss.Color("#FF3333"), // Red
		BgNormal:     lipgloss.Color(""),        // Transparent (terminal default)
		BgSelected:   lipgloss.Color("#1E1E1E"), // Slight highlight
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
		TextNormal:   lipgloss.Color("#000000"), // Black
		TextMuted:    lipgloss.Color("#666666"), // Gray
		TextError:    lipgloss.Color("#CC0000"), // Red
		BgNormal:     lipgloss.Color(""),        // Transparent
		BgSelected:   lipgloss.Color("#E8E8E8"), // Light highlight
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
		TextNormal:   lipgloss.Color("#FFFFFF"), // White
		TextMuted:    lipgloss.Color("#AAAAAA"), // Gray
		TextError:    lipgloss.Color("#FF0000"), // Bright red
		BgNormal:     lipgloss.Color(""),        // Transparent
		BgSelected:   lipgloss.Color("#333333"), // Dark highlight
	}
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
