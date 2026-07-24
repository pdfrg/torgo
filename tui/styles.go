package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds all UI styling
type Styles struct {
	// Base color strings (for later use with lipgloss.Color() wrapper)
	FgColorStr     string
	BgColorStr     string
	SelectColorStr string
	HintColorStr   string
	ErrorColorStr  string

	// Status color strings
	DownloadColorStr    string
	QueuedDLColorStr    string
	StalledDLColorStr   string
	PauseColorStr       string
	SeedColorStr        string
	CompletedColorStr   string
	ErrorStatusColorStr string

	// Styled components
	Title            lipgloss.Style
	StatusBar        lipgloss.Style
	HintsBar         lipgloss.Style
	ListHeader       lipgloss.Style
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ProgressBar      lipgloss.Style
	Dialog           lipgloss.Style
}

// Convenience accessors that wrap color strings with lipgloss.Color()
func (s *Styles) FgColor() color.Color          { return lipgloss.Color(s.FgColorStr) }
func (s *Styles) BgColor() color.Color          { return lipgloss.Color(s.BgColorStr) }
func (s *Styles) SelectColor() color.Color      { return lipgloss.Color(s.SelectColorStr) }
func (s *Styles) HintColor() color.Color        { return lipgloss.Color(s.HintColorStr) }
func (s *Styles) ErrorColor() color.Color       { return lipgloss.Color(s.ErrorColorStr) }
func (s *Styles) DownloadColor() color.Color    { return lipgloss.Color(s.DownloadColorStr) }
func (s *Styles) QueuedDLColor() color.Color    { return lipgloss.Color(s.QueuedDLColorStr) }
func (s *Styles) StalledDLColor() color.Color   { return lipgloss.Color(s.StalledDLColorStr) }
func (s *Styles) PauseColor() color.Color       { return lipgloss.Color(s.PauseColorStr) }
func (s *Styles) SeedColor() color.Color        { return lipgloss.Color(s.SeedColorStr) }
func (s *Styles) CompletedColor() color.Color   { return lipgloss.Color(s.CompletedColorStr) }
func (s *Styles) ErrorStatusColor() color.Color { return lipgloss.Color(s.ErrorStatusColorStr) }

// SyncFromTheme updates style colors to match the current theme
func (s *Styles) SyncFromTheme(t Theme) {
	if t.TextNormal != nil {
		s.FgColorStr = colorToHex(t.TextNormal)
	}
	if t.BgNormal != nil {
		s.BgColorStr = colorToHex(t.BgNormal)
	}
	if t.AccentColor != nil {
		s.SelectColorStr = colorToHex(t.AccentColor)
	}
	if t.TextMuted != nil {
		s.HintColorStr = colorToHex(t.TextMuted)
	}
	if t.TextError != nil {
		s.ErrorColorStr = colorToHex(t.TextError)
	}

	// Rebuild pre-built lipgloss style objects from synced strings
	s.Title = lipgloss.NewStyle().
		Foreground(s.SelectColor()).
		Bold(true)
	s.ListHeader = lipgloss.NewStyle().
		Foreground(s.SelectColor()).
		Bold(true).
		Padding(0, 1)
	s.ListItem = lipgloss.NewStyle().
		Foreground(s.FgColor()).
		Padding(0, 1)
	s.ListItemSelected = lipgloss.NewStyle().
		Foreground(s.BgColor()).
		Background(s.SelectColor()).
		Padding(0, 1)
	s.Dialog = lipgloss.NewStyle().
		Foreground(s.FgColor()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.SelectColor()).
		Padding(1)
}

// DefaultStyles returns the default dark theme
func DefaultStyles() *Styles {
	s := &Styles{
		FgColorStr:          "252", // Light gray
		BgColorStr:          "235", // Dark gray
		SelectColorStr:      "39",  // Cyan
		HintColorStr:        "242", // Medium gray
		ErrorColorStr:       "196", // Red
		DownloadColorStr:    "26",  // Dark blue
		QueuedDLColorStr:    "130", // Dark orange
		StalledDLColorStr:   "130", // Dark orange (same as queuedDL)
		PauseColorStr:       "240", // Subdued gray
		SeedColorStr:        "22",  // Dark green
		CompletedColorStr:   "178", // Gold
		ErrorStatusColorStr: "124", // Dark red
	}

	// Title
	s.Title = lipgloss.NewStyle().
		Foreground(s.SelectColor()).
		Bold(true)

	// Status bar at bottom (styled in Render method with Width, Foreground, and Background)
	s.StatusBar = lipgloss.NewStyle()

	// Hints bar at bottom (styled in Render method with Width, Foreground, and Background)
	s.HintsBar = lipgloss.NewStyle()

	// List header row
	s.ListHeader = lipgloss.NewStyle().
		Foreground(s.SelectColor()).
		Bold(true).
		Padding(0, 1)

	// Normal list item
	s.ListItem = lipgloss.NewStyle().
		Foreground(s.FgColor()).
		Padding(0, 1)

	// Selected list item
	s.ListItemSelected = lipgloss.NewStyle().
		Foreground(s.BgColor()).
		Background(s.SelectColor()).
		Padding(0, 1)

	// Progress bar (for reference; actual implementation uses status-based colors)
	s.ProgressBar = lipgloss.NewStyle().
		Foreground(s.DownloadColor())

	// Dialog
	s.Dialog = lipgloss.NewStyle().
		Foreground(s.FgColor()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.SelectColor()).
		Padding(1)

	return s
}

// StatusColor returns the color for a torrent status
func (s *Styles) StatusColor(status string) color.Color {
	switch status {
	case "downloading":
		return s.DownloadColor()
	case "queuedDL":
		return s.QueuedDLColor()
	case "stalledDL":
		return s.StalledDLColor()
	case "paused":
		return s.PauseColor()
	case "seeding":
		return s.SeedColor()
	case "completed":
		return s.CompletedColor()
	case "error":
		return s.ErrorStatusColor()
	default:
		return s.FgColor()
	}
}

// ProgressBarColors returns background colors for progress bar based on status and progress
// filledColor is for the filled portion, unfilledColor for the empty portion
func (s *Styles) ProgressBarColors(status string, progress uint8) (filledColor, unfilledColor color.Color) {
	// unfilledColor is transparent (empty string means no background, allowing terminal bg to show)
	unfilledColor = lipgloss.Color("") // No background = transparent

	// Use the status directly - we now have proper status mapping from qBittorrent API
	// including StatusCompleted, so we don't need to override based on progress
	switch status {
	case "downloading":
		filledColor = s.DownloadColor() // Dark blue
	case "queuedDL":
		filledColor = s.QueuedDLColor() // Dark orange
	case "stalledDL":
		filledColor = s.StalledDLColor() // Dark orange
	case "paused":
		filledColor = s.PauseColor() // Subdued gray
	case "seeding":
		filledColor = s.SeedColor() // Dark green
	case "completed":
		filledColor = s.CompletedColor() // Dark yellow
	case "error":
		filledColor = s.ErrorStatusColor() // Dark red
	default:
		filledColor = s.FgColor() // Light gray
	}

	return filledColor, unfilledColor
}
