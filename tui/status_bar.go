package tui

import (
	"fmt"
	"strings"
	"tqbtui/state"

	"github.com/charmbracelet/lipgloss"
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
	// Make speeds fixed width (9 chars each) to prevent text from jumping
	downSpeedStr = fmt.Sprintf("%9s", downSpeedStr)
	upSpeedStr = fmt.Sprintf("%9s", upSpeedStr)
	torrentInfo := fmt.Sprintf("%d/%d torrents  ↓%s ↑%s",
		len(filtered), len(appState.Torrents), downSpeedStr, upSpeedStr)

	// Speed limit status
	speedLimitStatus := ""
	if appState.SpeedLimitEnabled {
		speedLimitStatus = "  Limit: ON"
	}

	// Format filter and sort with colored first letters
	// Note: Build plain strings first, then apply color to avoid ANSI codes breaking width calculations
	filterPlain := "filter: " + string(appState.Filter)
	sortPlain := "sort: " + string(appState.SortBy)

	// Build status line (plain strings for width calculation)
	left := fmt.Sprintf("%s  %s%s", connStatus, clientInfo, speedLimitStatus)
	right := fmt.Sprintf("%s  %s  %s",
		torrentInfo, filterPlain, sortPlain)

	// Calculate padding to reach exact width (using plain text length)
	contentWidth := len(left) + len(right)
	padding := width - contentWidth
	if padding < 1 {
		padding = 1
	}

	// Build final status with padding
	status := fmt.Sprintf("%s%s%s",
		left, strings.Repeat(" ", padding), right)

	// Ensure exact width
	runes := []rune(status)
	if len(runes) > width {
		status = string(runes[:width])
	} else if len(runes) < width {
		status = status + strings.Repeat(" ", width-len(runes))
	}

	// Now apply colors to the final string (after width is correct)
	status = applyStatusBarColors(status, appState)

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

// applyStatusBarColors applies cyan color to 'f' in 'filter:' and 's' in 'sort:'
func applyStatusBarColors(status string, appState *state.AppState) string {
	keyColor := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	
	// Replace 'filter:' with colored 'f' + 'ilter:'
	status = strings.Replace(status, "filter:", keyColor.Render("f")+"ilter:", 1)
	
	// Replace 'sort:' with colored 's' + 'ort:'
	status = strings.Replace(status, "sort:", keyColor.Render("s")+"ort:", 1)
	
	return status
}
