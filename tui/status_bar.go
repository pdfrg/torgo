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

	// Torrent count and speeds
	filtered := appState.FilteredTorrents()
	totalDownSpeed := 0.0
	totalUpSpeed := 0.0
	for _, t := range appState.Torrents {
		totalDownSpeed += t.SpeedDown
		totalUpSpeed += t.SpeedUp
	}
	
	downSpeedStr := formatSpeedForBar(totalDownSpeed)
	upSpeedStr := formatSpeedForBar(totalUpSpeed)
	torrentInfo := fmt.Sprintf("%d/%d torrents  ↓%s ↑%s",
		len(filtered), len(appState.Torrents), downSpeedStr, upSpeedStr)

	// Speed limit status
	speedLimitStatus := ""
	if appState.SpeedLimitEnabled {
		speedLimitStatus = "  Limit: ON"
	}

	// Build status line
	left := fmt.Sprintf("%s  %s%s", connStatus, clientInfo, speedLimitStatus)
	right := fmt.Sprintf("%s  filter: %s  sort: %s",
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

// formatSpeedForBar formats speed for the status bar (more compact than torrent list)
func formatSpeedForBar(speed float64) string {
	if speed == 0 {
		return "0"
	}
	if speed < 1024 {
		return fmt.Sprintf("%.0fB", speed)
	}
	if speed < 1024*1024 {
		return fmt.Sprintf("%.1fKB", speed/1024)
	}
	return fmt.Sprintf("%.1fMB", speed/(1024*1024))
}
