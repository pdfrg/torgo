package tui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

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
	BgNormal color.Color
	// Progress bar empty color
	ProgressBarEmptyColor color.Color

	// Omarchy semantic colors
	AccentColor     color.Color // For titles, keys, highlights
	CursorColor     color.Color // For cursor indicator
	ForegroundColor color.Color // For variable text

	// Bar styling (derived)
	StatusBarBg     color.Color
	StatusBarFg     color.Color
	StatusBarAccent color.Color

	// Hex string versions for convenient access (used by search bar, etc)
	AccentColorHex string // Hex version of AccentColor
	BgNormalHex    string // Hex version of BgNormal

	// Detail view colors
	DetailTabActiveBorder   color.Color // Active tab border color (bright, stands out)
	DetailTabInactiveBorder color.Color // Inactive tab border color (muted)
	DetailLabelColor        color.Color // Field labels and section headers
	DetailCursorColor       color.Color // Tree/list cursor indicator color

	// Oneline view solid status colors (not gradients)
	StatusOnlineColors    map[string]color.Color // Solid colors for oneline view by status
	StatusOnlineColorsHex map[string]string      // Hex versions for luminance check

	// SolidProgressBars renders multiline progress bars as a single palette
	// color instead of a blended gradient. Required for themes whose colors
	// are ANSI slots: blending converts slots to fixed RGB and would bypass
	// the terminal palette. Only TerminalTheme sets this.
	SolidProgressBars bool
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
		BgNormal:              lipgloss.Color("#1A1A1A"), // Very dark gray
		ProgressBarEmptyColor: lipgloss.Color("#333333"), // Dark gray for empty progress bar

		// Semantic colors for dark theme
		AccentColor:     lipgloss.Color("#00FF00"), // Terminal green
		CursorColor:     lipgloss.Color("214"),     // Orange
		ForegroundColor: lipgloss.Color("#00d7d7"), // Cyan

		// Bar styling
		StatusBarBg:     lipgloss.Color("237"),     // Dark gray (darkened from 252)
		StatusBarFg:     lipgloss.Color("252"),     // Light gray text
		StatusBarAccent: lipgloss.Color("#00FF00"), // Terminal green for keys

		// Hex versions
		AccentColorHex: "#00ff00", // Terminal green
		BgNormalHex:    "#1A1A1A", // Very dark gray

		// Detail view colors for dark theme
		DetailTabActiveBorder:   lipgloss.Color("#00FF00"), // Terminal green - stands out
		DetailTabInactiveBorder: lipgloss.Color("242"),     // Medium gray - muted
		DetailLabelColor:        lipgloss.Color("#00FF00"), // Terminal green - same as accent
		DetailCursorColor:       lipgloss.Color("214"),     // Orange - matches cursor

		// Oneline view solid status colors (using gradient endpoints per revised planning)
		StatusOnlineColors: map[string]color.Color{
			"downloading": lipgloss.Color("#00D26A"), // Bright green (gradient[1])
			"seeding":     lipgloss.Color("#00D9FF"), // Bright cyan (gradient[1])
			"paused":      lipgloss.Color("#FFD700"), // Yellow (gradient[0] - color3)
			"completed":   lipgloss.Color("#9D00FF"), // Purple (gradient[0] - distinct from seeding)
			"error":       lipgloss.Color("#FF3333"), // Red (gradient[0] - color1)
			"queueing":    lipgloss.Color("#FF00FF"), // Magenta (gradient[0] - color5)
			"stalled":     lipgloss.Color("#FFA500"), // Orange (gradient[0] - color6)
			"unknown":     lipgloss.Color("#606060"), // Dark gray
		},
		StatusOnlineColorsHex: map[string]string{
			"downloading": "#00D26A",
			"seeding":     "#00D9FF",
			"paused":      "#FFD700",
			"completed":   "#9D00FF",
			"error":       "#FF3333",
			"queueing":    "#FF00FF",
			"stalled":     "#FFA500",
			"unknown":     "#606060",
		},
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
				lipgloss.Color("#009944"), // Dark green
				lipgloss.Color("#0099BB"), // Dark cyan
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
		BgNormal:              lipgloss.Color("#C0C0C0"), // Medium light gray
		ProgressBarEmptyColor: lipgloss.Color("#AAAAAA"), // Subtle darker than background

		// Semantic colors for light theme (no omarchy)
		AccentColor:     lipgloss.Color("33"),      // Dark cyan
		CursorColor:     lipgloss.Color("#00AA00"), // Bright green
		ForegroundColor: lipgloss.Color("#880088"), // Dark magenta

		// Bar styling for light theme
		StatusBarBg:     lipgloss.Color("#A0A0A0"), // Darker gray than background
		StatusBarFg:     lipgloss.Color("16"),      // Black text
		StatusBarAccent: lipgloss.Color("33"),      // Dark cyan for keys

		// Hex versions
		AccentColorHex: "#0099aa", // Dark cyan
		BgNormalHex:    "#C0C0C0", // Medium light gray

		// Detail view colors for light theme
		DetailTabActiveBorder:   lipgloss.Color("33"),      // Dark cyan - same as accent
		DetailTabInactiveBorder: lipgloss.Color("245"),     // Light gray - muted
		DetailLabelColor:        lipgloss.Color("33"),      // Dark cyan - same as accent
		DetailCursorColor:       lipgloss.Color("#00AA00"), // Bright green - matches cursor

		// Oneline view solid status colors for light theme
		StatusOnlineColors: map[string]color.Color{
			"downloading": lipgloss.Color("#0055CC"), // Dark blue (gradient[1])
			"seeding":     lipgloss.Color("#0099AA"), // Dark cyan (gradient[1])
			"paused":      lipgloss.Color("#CC8800"), // Dark yellow (gradient[0])
			"completed":   lipgloss.Color("#BB00BB"), // Dark magenta (gradient[1])
			"error":       lipgloss.Color("#CC0000"), // Dark red (gradient[0])
			"queueing":    lipgloss.Color("#BB00BB"), // Dark magenta (gradient[0])
			"stalled":     lipgloss.Color("#CC6600"), // Dark orange (gradient[0])
			"unknown":     lipgloss.Color("#999999"), // Medium gray
		},
		StatusOnlineColorsHex: map[string]string{
			"downloading": "#0055CC",
			"seeding":     "#0099AA",
			"paused":      "#CC8800",
			"completed":   "#BB00BB",
			"error":       "#CC0000",
			"queueing":    "#BB00BB",
			"stalled":     "#CC6600",
			"unknown":     "#999999",
		},
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
		BgNormal:              lipgloss.Color("#000000"), // Pure black
		ProgressBarEmptyColor: lipgloss.Color("#333333"), // Dark gray for empty progress bar

		// Semantic colors for high contrast theme
		AccentColor:     lipgloss.Color("226"),     // Bright yellow
		CursorColor:     lipgloss.Color("51"),      // Bright cyan
		ForegroundColor: lipgloss.Color("#00FF00"), // Bright green

		// Bar styling for high contrast theme
		StatusBarBg:     lipgloss.Color("0"),   // Black
		StatusBarFg:     lipgloss.Color("255"), // White text
		StatusBarAccent: lipgloss.Color("226"), // Bright yellow for keys

		// Hex versions
		AccentColorHex: "#ffff00", // Bright yellow
		BgNormalHex:    "#000000", // Pure black

		// Detail view colors for high contrast theme
		DetailTabActiveBorder:   lipgloss.Color("226"), // Bright yellow - stands out
		DetailTabInactiveBorder: lipgloss.Color("8"),   // Bright black/gray - muted but visible
		DetailLabelColor:        lipgloss.Color("226"), // Bright yellow - same as accent
		DetailCursorColor:       lipgloss.Color("51"),  // Bright cyan - matches cursor

		// Oneline view solid status colors for high contrast theme
		StatusOnlineColors: map[string]color.Color{
			"downloading": lipgloss.Color("#00FF00"), // Bright green (gradient[1])
			"seeding":     lipgloss.Color("#00FFFF"), // Bright cyan (gradient[1])
			"paused":      lipgloss.Color("#FFFF00"), // Bright yellow (gradient[0])
			"completed":   lipgloss.Color("#FF00FF"), // Bright magenta (gradient[1])
			"error":       lipgloss.Color("#FF0000"), // Bright red (gradient[0])
			"queueing":    lipgloss.Color("#FF00FF"), // Bright magenta (gradient[0])
			"stalled":     lipgloss.Color("#FF8800"), // Bright orange (gradient[0])
			"unknown":     lipgloss.Color("#CCCCCC"), // Light gray
		},
		StatusOnlineColorsHex: map[string]string{
			"downloading": "#00FF00",
			"seeding":     "#00FFFF",
			"paused":      "#FFFF00",
			"completed":   "#FF00FF",
			"error":       "#FF0000",
			"queueing":    "#FF00FF",
			"stalled":     "#FF8800",
			"unknown":     "#CCCCCC",
		},
	}
}

// TerminalTheme returns a theme that inherits the terminal emulator's palette.
// Every color is either an ANSI 0-15 palette slot or the terminal default
// (empty string), so whatever the terminal is themed with — e.g. an Omarchy
// theme applied locally — shows through, even over SSH where no theme files
// exist on the remote host. Status indicators are solid palette colors;
// gradients collapse to a single slot (no truecolor blending).
func TerminalTheme() Theme {
	solid := func(slot string) [2]color.Color {
		c := lipgloss.Color(slot)
		return [2]color.Color{c, c}
	}
	flat := func(slot string) color.Color { return lipgloss.Color(slot) }
	return Theme{
		StatusGradients: map[string][2]color.Color{
			"downloading": solid("4"), // blue
			"seeding":     solid("2"), // green
			"paused":      solid("3"), // yellow
			"completed":   solid("5"), // magenta
			"error":       solid("1"), // red
			"queueing":    solid("6"), // cyan
			"stalled":     solid("3"), // yellow
			"stalledDL":   solid("3"),
			"stalledUP":   solid("3"),
			"unknown":     solid("8"), // bright black
		},
		TextNormal:            lipgloss.Color(""),  // terminal default fg (adapts to light/dark)
		TextMuted:             lipgloss.Color("8"), // bright black
		TextError:             lipgloss.Color("1"), // red
		BgNormal:              lipgloss.Color(""),  // terminal default bg (transparent)
		ProgressBarEmptyColor: lipgloss.Color("8"), // bright black

		AccentColor:     lipgloss.Color("4"), // blue
		CursorColor:     lipgloss.Color("6"), // cyan
		ForegroundColor: lipgloss.Color(""),  // terminal default fg

		StatusBarBg:     lipgloss.Color("8"), // bright black
		StatusBarFg:     lipgloss.Color(""),  // terminal default fg
		StatusBarAccent: lipgloss.Color("4"), // blue

		// Hex versions: ANSI slot strings pass through to lipgloss as-is;
		// empty means "use terminal default".
		AccentColorHex: "4",
		BgNormalHex:    "",

		DetailTabActiveBorder:   lipgloss.Color("4"), // blue
		DetailTabInactiveBorder: lipgloss.Color("8"), // bright black
		DetailLabelColor:        lipgloss.Color("4"), // blue
		DetailCursorColor:       lipgloss.Color("6"), // cyan

		StatusOnlineColors: map[string]color.Color{
			"downloading": flat("4"),
			"seeding":     flat("2"),
			"paused":      flat("3"),
			"completed":   flat("5"),
			"error":       flat("1"),
			"queueing":    flat("6"),
			"stalled":     flat("3"),
			"unknown":     flat("8"),
		},
		// Empty hex = "terminal palette fill": GetContrastTextColorForBg
		// answers dark overlay text (see below) instead of guessing luminance.
		SolidProgressBars: true,
		StatusOnlineColorsHex: map[string]string{
			"downloading": "",
			"seeding":     "",
			"paused":      "",
			"completed":   "",
			"error":       "",
			"queueing":    "",
			"stalled":     "",
			"unknown":     "",
		},
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
	}

	bg := colors["background"]
	if bg == "" {
		bg = "#000000"
	}
	theme.BgNormal = lipgloss.Color(bg)
	theme.BgNormalHex = bg

	// SEMANTIC COLORS FROM OMARCHY
	accentColor := colors["accent"]
	if accentColor == "" {
		accentColor = "#00d7ff" // Fallback cyan
	}
	theme.AccentColor = lipgloss.Color(accentColor)
	theme.AccentColorHex = accentColor

	cursorColor := colors["cursor"]
	if cursorColor == "" {
		cursorColor = "#888888" // Fallback gray
	}
	theme.CursorColor = lipgloss.Color(cursorColor)

	fgColor := colors["foreground"]
	if fgColor == "" {
		fgColor = "#888888" // Fallback gray
	}
	theme.ForegroundColor = lipgloss.Color(fgColor)

	// TEXT COLORS: Smart calculation (soft grays, not extreme)
	if isColorDark(bg) {
		theme.TextNormal = lipgloss.Color("#d0d0d0") // Light gray (252-like)
		theme.TextMuted = lipgloss.Color("#888888")  // Medium gray
	} else {
		theme.TextNormal = lipgloss.Color("#333333") // Dark gray (not pure black)
		theme.TextMuted = lipgloss.Color("#666666")  // Medium gray
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
		theme.StatusBarFg = lipgloss.Color("#d0d0d0") // Light gray
	} else {
		theme.StatusBarFg = lipgloss.Color("#333333") // Dark gray
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

	// DETAIL VIEW COLORS
	// Use accent color for active tabs (bright, stands out like search bar edit mode)
	theme.DetailTabActiveBorder = theme.AccentColor

	// Use muted foreground for inactive tabs
	theme.DetailTabInactiveBorder = theme.TextMuted

	// Use accent color for labels (matches search bar, draws attention)
	theme.DetailLabelColor = theme.AccentColor

	// Use cursor color for tree/list cursor indicator
	theme.DetailCursorColor = theme.CursorColor

	// ONELINE VIEW COLORS: Use gradient endpoints per revised planning
	// Active states (downloading, seeding) use gradient[1] (bright endpoint)
	// Completed uses gradient[0] for distinction from seeding
	// Passive states (paused, error, queueing, stalled) use gradient[0] (semantic color)
	theme.StatusOnlineColors = map[string]color.Color{
		"downloading": theme.StatusGradients["downloading"][1], // Bright color4
		"seeding":     theme.StatusGradients["seeding"][1],     // Bright color2
		"paused":      theme.StatusGradients["paused"][0],      // color3
		"completed":   theme.StatusGradients["completed"][0],   // Desaturated color2 (distinct from seeding)
		"error":       theme.StatusGradients["error"][0],       // color1
		"queueing":    theme.StatusGradients["queueing"][0],    // color5
		"stalled":     theme.StatusGradients["stalled"][0],     // color6
		"unknown":     theme.StatusGradients["unknown"][0],     // Dark gray
	}

	// Store hex versions for luminance calculation
	// Map gradient endpoints back to original color strings
	theme.StatusOnlineColorsHex = map[string]string{
		"downloading": color4,      // Bright color4
		"seeding":     color2,      // Bright color2
		"paused":      color3,      // color3
		"completed":   color2Desat, // Desaturated color2 (distinct from seeding)
		"error":       color1,      // color1
		"queueing":    color5,      // color5
		"stalled":     color6,      // color6
		"unknown":     "#606060",   // Fallback gray
	}

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

// currentTheme holds the active theme. It is unexported so every read goes
// through GetCurrentTheme(), keeping reads synchronized with SetTheme().
// Themes are treated as immutable once set (SetTheme replaces the whole
// value; no code mutates the maps in place), so returning the struct by value
// is safe as long as that invariant holds.
var currentTheme = DefaultTheme()
var currentThemeMu sync.RWMutex

// SetTheme changes the active theme
func SetTheme(t Theme) {
	currentThemeMu.Lock()
	currentTheme = t
	currentThemeMu.Unlock()
}

// GetCurrentTheme returns a snapshot of the active theme. Reads are protected
// by currentThemeMu so they can safely race against SetTheme(), which may be
// reached from the Bubble Tea main goroutine while a tea.Cmd goroutine renders
// theme-dependent components.
func GetCurrentTheme() Theme {
	currentThemeMu.RLock()
	defer currentThemeMu.RUnlock()
	return currentTheme
}

// GetCurrentStatusGradient returns the gradient colors for a status using the current theme
func GetCurrentStatusGradient(status string) [2]color.Color {
	return GetCurrentTheme().GetStatusGradient(status)
}

// GetStatusColorForOneline returns the solid color for oneline view for a given status
func (t Theme) GetStatusColorForOneline(status string) color.Color {
	if col, ok := t.StatusOnlineColors[status]; ok {
		return col
	}
	// Fallback to text normal if status not found
	return t.TextNormal
}

// GetStatusColorHexForOneline returns the hex string for oneline color for luminance calculation
func (t Theme) GetStatusColorHexForOneline(status string) string {
	if hex, ok := t.StatusOnlineColorsHex[status]; ok {
		return hex
	}
	// Fallback white
	return "#FFFFFF"
}

// GetContrastTextColorForBg returns light or dark text color based on background luminance.
// An empty bgHex means "terminal palette fill of unknown luminance" (used by
// TerminalTheme, whose StatusOnlineColorsHex entries are empty). Luminance
// can't be computed, so use palette black: correct on pastel-on-dark palettes
// (Omarchy style) and light terminals, mirroring the dark-on-light-fill look
// of the omarchy theme. May be low-contrast on stock-dark palettes.
func (t Theme) GetContrastTextColorForBg(bgHex string) color.Color {
	if bgHex == "" {
		return lipgloss.Color("0")
	}
	if isColorDark(bgHex) {
		return lipgloss.Color("#d0d0d0") // Light gray on dark background
	}
	return lipgloss.Color("#333333") // Dark gray on light background
}

// colorToHex converts a color.Color to a hex string like "#RRGGBB"
func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// Color transformation helper functions for omarchy themes

// isValidHex reports whether s is a parseable #rrggbb or #rgb hex color.
// The noire parser panics on malformed input, so every entry point that may
// receive ANSI slots ("4") or empty terminal-theme values must check first.
func isValidHex(s string) bool {
	cleanHex := strings.TrimPrefix(s, "#")
	if len(cleanHex) != 3 && len(cleanHex) != 6 {
		return false
	}
	for _, r := range cleanHex {
		isDigit := r >= '0' && r <= '9'
		isHexLetter := r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
		if !isDigit && !isHexLetter {
			return false
		}
	}
	return true
}

// lightenColorHex lightens a hex color by the given amount (0.0-1.0)
func lightenColorHex(hexColor string, amount float64) string {
	if !isValidHex(hexColor) {
		return hexColor
	}
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	lightened := c.Lighten(amount)
	return "#" + lightened.Hex()
}

// darkenColorHex darkens a hex color by the given amount (0.0-1.0)
func darkenColorHex(hexColor string, amount float64) string {
	if !isValidHex(hexColor) {
		return hexColor
	}
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	darkened := c.Darken(amount)
	return "#" + darkened.Hex()
}

// desaturateColorHex desaturates a hex color by the given amount (0.0-1.0)
func desaturateColorHex(hexColor string, amount float64) string {
	if !isValidHex(hexColor) {
		return hexColor
	}
	cleanHex := strings.TrimPrefix(hexColor, "#")
	c := noire.NewHex(cleanHex)
	desaturated := c.Desaturate(amount)
	return "#" + desaturated.Hex()
}

// isColorDark checks if a hex color is dark.
// Invalid (non-hex) input is treated as dark, matching dark terminal defaults.
func isColorDark(hexColor string) bool {
	if !isValidHex(hexColor) {
		return true
	}
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
