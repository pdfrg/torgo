package tui

import (
	"charm.land/lipgloss/v2"
)

// Styles holds all UI styling
type Styles struct {
	// Base colors (using string representation for v2)
	FgColor    string
	BgColor    string
	SelectColor string
	HintColor  string
	ErrorColor string

	// Status colors
	DownloadColor     string
	QueuedDLColor     string
	StalledDLColor    string
	PauseColor        string
	SeedColor         string
	CompletedColor    string
	ErrorStatusColor  string

	// Styled components
	Title        lipgloss.Style
	StatusBar    lipgloss.Style
	HintsBar     lipgloss.Style
	ListHeader   lipgloss.Style
	ListItem     lipgloss.Style
	ListItemSelected lipgloss.Style
	ProgressBar  lipgloss.Style
	Dialog       lipgloss.Style
}

// DefaultStyles returns the default dark theme
func DefaultStyles() *Styles {
	s := &Styles{
		FgColor:     "252",  // Light gray
		BgColor:     "235",  // Dark gray
		SelectColor: "39",   // Cyan
		HintColor:   "242",  // Medium gray
		ErrorColor:  "196",  // Red
		DownloadColor:    "26",  // Dark blue
		QueuedDLColor:    "130", // Dark orange
		StalledDLColor:   "130", // Dark orange (same as queuedDL)
		PauseColor:       "240", // Subdued gray
		SeedColor:        "22",  // Dark green
		CompletedColor:   "178", // Gold
		ErrorStatusColor: "124", // Dark red
	}

	// Title
	s.Title = lipgloss.NewStyle().
		Foreground(s.SelectColor).
		Bold(true)

	// Status bar at bottom (styled in Render method with Width, Foreground, and Background)
	s.StatusBar = lipgloss.NewStyle()

	// Hints bar at bottom (styled in Render method with Width, Foreground, and Background)
	s.HintsBar = lipgloss.NewStyle()

	// List header row
	s.ListHeader = lipgloss.NewStyle().
		Foreground(s.SelectColor).
		Bold(true).
		Padding(0, 1)

	// Normal list item
	s.ListItem = lipgloss.NewStyle().
		Foreground(s.FgColor).
		Padding(0, 1)

	// Selected list item
	s.ListItemSelected = lipgloss.NewStyle().
		Foreground(s.BgColor).
		Background(s.SelectColor).
		Padding(0, 1)

	// Progress bar (for reference; actual implementation uses status-based colors)
	s.ProgressBar = lipgloss.NewStyle().
		Foreground(s.DownloadColor)

	// Dialog
	s.Dialog = lipgloss.NewStyle().
		Foreground(s.FgColor).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.SelectColor).
		Padding(1)

	return s
}

// StatusColor returns the color for a torrent status
func (s *Styles) StatusColor(status string) string {
	switch status {
	case "downloading":
		return s.DownloadColor
	case "queuedDL":
		return s.QueuedDLColor
	case "stalledDL":
		return s.StalledDLColor
	case "paused":
		return s.PauseColor
	case "seeding":
		return s.SeedColor
	case "completed":
		return s.CompletedColor
	case "error":
		return s.ErrorStatusColor
	default:
		return s.FgColor
	}
}

// ProgressBarColors returns background colors for progress bar based on status and progress
// filledColor is for the filled portion, unfilledColor for the empty portion
func (s *Styles) ProgressBarColors(status string, progress uint8) (filledColor, unfilledColor string) {
	// unfilledColor is transparent (empty string means no background, allowing terminal bg to show)
	unfilledColor = "" // No background = transparent
	
	// Use the status directly - we now have proper status mapping from qBittorrent API
	// including StatusCompleted, so we don't need to override based on progress
	switch status {
	case "downloading":
		filledColor = s.DownloadColor // Dark blue
	case "queuedDL":
		filledColor = s.QueuedDLColor // Dark orange
	case "stalledDL":
		filledColor = s.StalledDLColor // Dark orange
	case "paused":
		filledColor = s.PauseColor // Subdued gray
	case "seeding":
		filledColor = s.SeedColor // Dark green
	case "completed":
		filledColor = s.CompletedColor // Dark yellow
	case "error":
		filledColor = s.ErrorStatusColor // Dark red
	default:
		filledColor = s.FgColor // Light gray
	}
	
	return filledColor, unfilledColor
}
