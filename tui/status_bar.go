package tui

import (
	"fmt"
	"github.com/pdfrg/torgo/client"
	"github.com/pdfrg/torgo/state"

	"charm.land/lipgloss/v2"
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
	return s.RenderWithFiltered(appState, width, appState.FilteredTorrents())
}

// RenderWithFiltered renders the status bar with a precomputed filtered list,
// avoiding a second filter+sort pass when the caller already has it.
func (s *StatusBar) RenderWithFiltered(appState *state.AppState, width int, filtered []client.Torrent) string {
	current := appState.CurrentClient()
	if current == nil {
		return s.styles.StatusBar.Render(fmt.Sprintf("%-"+fmt.Sprintf("%d", width)+"s", "No clients configured"))
	}

	// Connection status
	connStatus := "⚠ Offline"
	if current.Adapter.IsConnected() {
		connStatus = "✓ Online"
	}

	// Client info: name and connection details (from config)
	clientHost := ""
	clientPort := 0
	for _, cfg := range appState.Config.Clients {
		if cfg.ID == current.ID {
			clientHost = cfg.Host
			clientPort = cfg.Port
			break
		}
	}
	clientInfo := fmt.Sprintf("%s %s:%d",
		current.Name, clientHost, clientPort)

	// Torrent count and speeds (filtered list passed in by caller)
	// speeds still sum over all torrents (seeding traffic counts even when filtered out)
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

	// Get bar colors from current theme
	theme := GetCurrentTheme()
	barBg := theme.StatusBarBg
	barFg := theme.StatusBarFg
	accentColor := theme.StatusBarAccent

	// Color styles for individual parts
	defaultColor := lipgloss.NewStyle().
		Foreground(barFg).
		Background(barBg)
	keyColor := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Background(barBg)

	// Build left side (connection, client info, speed limit, torrent count)
	left := defaultColor.Render(fmt.Sprintf("%s  %s%s    %s", connStatus, clientInfo, speedLimitStatus, torrentCountInfo))

	// Build sort label with direction indicator
	sortArrow := "↓"
	if appState.SortAscending {
		sortArrow = "↑"
	}
	sortLabel := string(appState.SortBy) + " " + sortArrow

	// Build right side with speeds, filter and sort (right-aligned)
	filterWithValue := keyColor.Render("f") + defaultColor.Render("ilter: "+string(appState.Filter))
	sortWithValue := keyColor.Render("s") + defaultColor.Render("ort: "+sortLabel)
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
