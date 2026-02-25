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
	DownloadColor lipgloss.Color
	SeedColor     lipgloss.Color
	PauseColor    lipgloss.Color
	ErrorStatusColor lipgloss.Color

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
		DownloadColor: lipgloss.Color("46"), // Green
		SeedColor:     lipgloss.Color("82"), // Light green
		PauseColor:    lipgloss.Color("220"), // Yellow
		ErrorStatusColor: lipgloss.Color("160"), // Dark red
	}

	// Title
	s.Title = lipgloss.NewStyle().
		Foreground(s.SelectColor).
		Bold(true)

	// Status bar at bottom
	s.StatusBar = lipgloss.NewStyle().
		Foreground(s.FgColor).
		Background(lipgloss.Color("237")).
		Padding(0, 1)

	// Hints bar at bottom
	s.HintsBar = lipgloss.NewStyle().
		Foreground(s.HintColor).
		Background(lipgloss.Color("237")).
		Padding(0, 1)

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

	// Progress bar
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
	case "seeding":
		return s.SeedColor
	case "paused":
		return s.PauseColor
	case "error":
		return s.ErrorStatusColor
	default:
		return s.FgColor
	}
}
