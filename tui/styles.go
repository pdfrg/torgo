package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all UI styling
type Styles struct {
	// Base colors
	FgColor    lipgloss.Color
	BgColor    lipgloss.Color
	SelectColor lipgloss.Color
	HintColor  lipgloss.Color
	ErrorColor lipgloss.Color

	// Status colors
	DownloadColor     lipgloss.Color
	QueuedDLColor     lipgloss.Color
	StalledDLColor    lipgloss.Color
	PauseColor        lipgloss.Color
	SeedColor         lipgloss.Color
	CompletedColor    lipgloss.Color
	ErrorStatusColor  lipgloss.Color

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
		FgColor:     lipgloss.Color("252"),  // Light gray
		BgColor:     lipgloss.Color("235"),  // Dark gray
		SelectColor: lipgloss.Color("39"),   // Cyan
		HintColor:   lipgloss.Color("242"),  // Medium gray
		ErrorColor:  lipgloss.Color("196"),  // Red
		DownloadColor:    lipgloss.Color("26"),  // Dark blue
		QueuedDLColor:    lipgloss.Color("130"), // Dark orange
		StalledDLColor:   lipgloss.Color("130"), // Dark orange (same as queuedDL)
		PauseColor:       lipgloss.Color("240"), // Subdued gray
		SeedColor:        lipgloss.Color("22"),  // Dark green
		CompletedColor:   lipgloss.Color("178"), // Gold
		ErrorStatusColor: lipgloss.Color("124"), // Dark red
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
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.SelectColor).
		Padding(1)

	return s
}

// StatusColor returns the color for a torrent status
func (s *Styles) StatusColor(status string) lipgloss.Color {
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
func (s *Styles) ProgressBarColors(status string, progress uint8) (filledColor, unfilledColor lipgloss.Color) {
	// unfilledColor is transparent (empty string means no background, allowing terminal bg to show)
	unfilledColor = lipgloss.Color("") // No background = transparent
	
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
