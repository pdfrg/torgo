package tui

import (
	"fmt"
	"strings"
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
		return s.styles.StatusBar.Render(fmt.Sprintf("%-"+fmt.Sprintf("%d", width)+"s", "No clients configured"))
	}

	// Connection status
	connStatus := "⚠ Offline"
	if current.Adapter.IsConnected() {
		connStatus = "✓ Online"
	}

	// Client info
	clientInfo := fmt.Sprintf("[%s] %s (%s)",
		current.ID, current.Name, current.Type)

	// Torrent count
	filtered := appState.FilteredTorrents()
	torrentInfo := fmt.Sprintf("%d/%d torrents",
		len(filtered), len(appState.Torrents))

	// Build status line
	left := fmt.Sprintf("%s  %s", connStatus, clientInfo)
	right := fmt.Sprintf("%s  Filter: %s  Sort: %s",
		torrentInfo, appState.Filter, appState.SortBy)

	// Calculate padding to reach exact width
	contentWidth := len(left) + len(right)
	padding := width - contentWidth
	if padding < 1 {
		padding = 1
	}

	status := fmt.Sprintf("%s%s%s",
		left, strings.Repeat(" ", padding), right)

	// Ensure exact width
	runes := []rune(status)
	if len(runes) > width {
		status = string(runes[:width])
	} else if len(runes) < width {
		status = status + strings.Repeat(" ", width-len(runes))
	}

	return s.styles.StatusBar.Render(status)
}
