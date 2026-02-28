package tui

import (
	"fmt"
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
	
	torrentCountInfo := fmt.Sprintf("%d/%d torrents",
		len(filtered), len(appState.Torrents))
	speedsInfo := fmt.Sprintf("↓%s ↑%s", downSpeedStr, upSpeedStr)

	// Speed limit status with turtle icon and actual limits
	speedLimitStatus := ""
	if appState.SpeedLimitEnabled {
		downStr := formatSpeedForBar(float64(appState.SpeedLimitDownKBs * 1024))
		upStr := formatSpeedForBar(float64(appState.SpeedLimitUpKBs * 1024))
		speedLimitStatus = fmt.Sprintf("  🐢▼%s ▲%s", downStr, upStr)
	}

	barBg := lipgloss.Color("237")

	// Color styles for individual parts (each with background applied)
	defaultColor := lipgloss.NewStyle().
		Foreground(s.styles.FgColor).
		Background(barBg)
	keyColor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")).
		Bold(true).
		Background(barBg)

	// Build left side (connection, client info, speed limit, torrent count)
	left := defaultColor.Render(fmt.Sprintf("%s  %s%s    %s", connStatus, clientInfo, speedLimitStatus, torrentCountInfo))

	// Build right side with speeds, filter and sort (right-aligned)
	filterWithValue := keyColor.Render("f") + defaultColor.Render("ilter: "+string(appState.Filter))
	sortWithValue := keyColor.Render("s") + defaultColor.Render("ort: "+string(appState.SortBy))
	right := lipgloss.JoinHorizontal(lipgloss.Left,
		defaultColor.Render(speedsInfo),
		defaultColor.Render("  "),
		filterWithValue,
		defaultColor.Render("  "),
		sortWithValue,
	)

	// Calculate middle padding to right-align the right content
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	paddingWidth := width - 2 - leftWidth - rightWidth // -2 for leading and trailing spaces
	
	// Build middle padding with background
	middlePadding := defaultColor.Render("")
	if paddingWidth > 0 {
		middlePadding = defaultColor.Render(fmt.Sprintf("%*s", paddingWidth, ""))
	}

	// Join left, padding, and right
	statusText := lipgloss.JoinHorizontal(lipgloss.Left,
		defaultColor.Render(" "),
		left,
		middlePadding,
		right,
		defaultColor.Render(" "),
	)

	// Apply bar style: width and background (background is already on text parts)
	barStyle := lipgloss.NewStyle().
		Width(width).
		Background(barBg)

	return barStyle.Render(statusText)
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
