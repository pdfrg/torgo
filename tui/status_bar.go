package tui

import (
	"fmt"
	"tqbtui/state"
)

// StatusBar displays connection and application status
type StatusBar struct {
	styles *Styles
}

// NewStatusBar creates a new status bar
func NewStatusBar(styles *Styles) *StatusBar {
	return &StatusBar{
		styles: styles,
	}
}

// Render returns the rendered status bar
func (s *StatusBar) Render(appState *state.AppState, width int) string {
	current := appState.CurrentClient()
	if current == nil {
		return s.styles.StatusBar.Render("No clients configured")
	}

	// Connection status
	connStatus := "⚠ Offline"
	if current.Adapter.IsConnected() {
		connStatus = "✓ Online"
	}

	// Client info
	clientInfo := fmt.Sprintf("[%s] %s (%s:%d)",
		current.ID, current.Name, current.Type)

	// Torrent count
	filtered := appState.FilteredTorrents()
	torrentInfo := fmt.Sprintf("%d/%d torrents",
		len(filtered), len(appState.Torrents))

	// Build status line
	left := fmt.Sprintf("%s  %s", connStatus, clientInfo)
	right := fmt.Sprintf("%s  Filter: %s  Sort: %s",
		torrentInfo, appState.Filter, appState.SortBy)

	// Pad to width
	padding := width - len(left) - len(right) - 4
	if padding < 1 {
		padding = 1
	}

	status := fmt.Sprintf("%s%s%s",
		left, fmt.Sprintf("%*s", padding, ""), right)

	if len(status) > width {
		status = status[:width]
	}

	return s.styles.StatusBar.Render(status)
}
