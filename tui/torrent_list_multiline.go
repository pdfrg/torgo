package tui

import (
	"fmt"
	"strings"
	"time"
	"tqbtui/client"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

// MultilineTorrentListView displays torrents in a 4-line format per entry
// Each torrent occupies 4 lines: Identity, Progress, Status, Metrics + blank line
type MultilineTorrentListView struct {
	torrents []client.Torrent
	selected map[string]bool
	cursor   int
	styles   *Styles
	theme    Theme
	viewport viewport.Model
}

// NewMultilineTorrentListView creates a new multi-line torrent list view
func NewMultilineTorrentListView(styles *Styles) *MultilineTorrentListView {
	return &MultilineTorrentListView{
		torrents: []client.Torrent{},
		selected: make(map[string]bool),
		cursor:   0,
		styles:   styles,
		theme:    CurrentTheme,
	}
}

// SetTorrents updates the torrent list
func (m *MultilineTorrentListView) SetTorrents(torrents []client.Torrent) {
	m.torrents = torrents
	if m.cursor >= len(torrents) {
		m.cursor = 0
	}
}

// MoveCursor moves the selection up or down
func (m *MultilineTorrentListView) MoveCursor(direction int) {
	m.cursor += direction
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.torrents) {
		m.cursor = len(m.torrents) - 1
	}
}

// PageMove moves the cursor by a full page (for Page Up/Down)
// In multiline view, one entry is 3 lines (plus 1 blank line between entries = 4 total)
func (m *MultilineTorrentListView) PageMove(direction int, pageSize int) {
	if pageSize <= 0 {
		pageSize = 10
	}
	// Adjust: in multiline view, each torrent is 4 lines (3 content + 1 blank), so divide pageSize by 4
	torrentPageSize := pageSize / 4
	if torrentPageSize < 1 {
		torrentPageSize = 1
	}
	m.cursor += direction * (torrentPageSize - 1)
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.torrents) {
		m.cursor = len(m.torrents) - 1
	}
}

// ToggleSelection toggles the selected state of the current torrent
func (m *MultilineTorrentListView) ToggleSelection() {
	if m.cursor < len(m.torrents) {
		id := m.torrents[m.cursor].ID
		m.selected[id] = !m.selected[id]
	}
}

// SelectAll toggles all torrents
func (m *MultilineTorrentListView) SelectAll() {
	allSelected := true
	for _, torrent := range m.torrents {
		if !m.selected[torrent.ID] {
			allSelected = false
			break
		}
	}

	if allSelected {
		m.ClearSelection()
	} else {
		for _, torrent := range m.torrents {
			m.selected[torrent.ID] = true
		}
	}
}

// ClearSelection clears all selections
func (m *MultilineTorrentListView) ClearSelection() {
	m.selected = make(map[string]bool)
}

// GetSelected returns the IDs of selected torrents
func (m *MultilineTorrentListView) GetSelected() []string {
	ids := []string{}
	for id, sel := range m.selected {
		if sel {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetCurrentTorrent returns the currently selected torrent
func (m *MultilineTorrentListView) GetCurrentTorrent() *client.Torrent {
	if m.cursor >= 0 && m.cursor < len(m.torrents) {
		return &m.torrents[m.cursor]
	}
	return nil
}

// Update handles keyboard input and viewport scrolling
func (m *MultilineTorrentListView) Update(msg interface{}) {
	// Pass scroll messages to viewport
	m.viewport.Update(msg)
}

// Render returns the rendered list with viewport-based scrolling
func (m *MultilineTorrentListView) Render(width, height int) string {
	if len(m.torrents) == 0 {
		return m.styles.ListItem.Render("No torrents")
	}

	// Bounds check for width and height
	if width < 1 || height < 1 {
		return ""
	}

	// Sync theme on every render
	m.theme = CurrentTheme

	// Build the full content
	lines := []string{}

	// Render each torrent as a 3-line block
	for i := 0; i < len(m.torrents); i++ {
		block := m.renderTorrentBlock(m.torrents[i], i == m.cursor, width, i+1)
		lines = append(lines, block...)
		// Add blank line between entries (except after last)
		if i < len(m.torrents)-1 {
			lines = append(lines, "")
		}
	}

	content := strings.Join(lines, "\n")

	// Only update viewport dimensions if they actually changed
	if m.viewport.Width() != width {
		m.viewport.SetWidth(width)
	}
	if m.viewport.Height() != height {
		m.viewport.SetHeight(height)
	}

	// Set content in viewport
	m.viewport.SetContent(content)

	// Ensure cursor is visible in viewport
	// In multiline mode, each torrent takes 4 lines (3 lines + 1 blank)
	cursorLineStart := m.cursor * 4 // Each torrent block is 4 lines (3 content + 1 blank)
	cursorLineEnd := cursorLineStart + 3
	contentHeight := strings.Count(content, "\n") + 1

	visibleTop := m.viewport.YOffset()
	visibleBottom := visibleTop + m.viewport.Height()

	// Only adjust scroll if needed
	if cursorLineEnd >= visibleBottom {
		// Cursor is below visible bottom, scroll down
		newOffset := cursorLineEnd - m.viewport.Height() + 1
		if newOffset < 0 {
			newOffset = 0
		}
		m.viewport.SetYOffset(newOffset)
	} else if cursorLineStart < visibleTop {
		// Cursor is above visible top
		m.viewport.SetYOffset(cursorLineStart)
	}

	// Final safety check for offset bounds
	maxOffset := contentHeight - m.viewport.Height()
	if maxOffset < 0 {
		maxOffset = 0
	}
	currentYOffset := m.viewport.YOffset()
	if currentYOffset > maxOffset {
		m.viewport.SetYOffset(maxOffset)
	}
	if currentYOffset < 0 {
		m.viewport.SetYOffset(0)
	}

	return m.viewport.View()
}

// renderTorrentBlock returns 3 lines representing a single torrent
// Line 1: [Selection] [#] Name
// Line 2: [Progress Bar] (Downloaded / Total)
// Line 3: Status: [Label]  ↓ [Speed] ↑ [Speed]  Ratio: [X.XX]  Seeds: [X]  Peers: [Y]  ETA: [Time]
func (m *MultilineTorrentListView) renderTorrentBlock(torrent client.Torrent, cursor bool, width, rowNum int) []string {
	block := []string{}

	// Line 1: Identity
	line1 := m.renderIdentityLine(torrent, cursor, width, rowNum)
	block = append(block, line1)

	// Line 2: Progress
	line2 := m.renderProgressLine(torrent, width)
	block = append(block, line2)

	// Line 3: Status & Metrics (merged)
	line3 := m.renderStatusLine(torrent, width)
	block = append(block, line3)

	return block
}

// renderIdentityLine: [>/  ][#][●/ ] Name
func (m *MultilineTorrentListView) renderIdentityLine(torrent client.Torrent, cursor bool, width, rowNum int) string {
	isSelected := m.selected[torrent.ID]

	// Row number styling with cursor prompt - use cursor color when cursor is on this row
	// Both formats are 4 chars to prevent title shift when going from single to double digits
	var numberStr string
	var numberStyle lipgloss.Style
	if cursor {
		// Cursor row: "> " + 2-digit number = 4 chars ("> 1", "> 10", etc), styled with cursor color
		numberStr = fmt.Sprintf("> %2d", rowNum)
		numberStyle = lipgloss.NewStyle().Foreground(m.theme.CursorColor)
	} else {
		// Non-cursor row: 4-char right-aligned number = 4 chars ("   1", "  10", etc), styled with fg color
		numberStr = fmt.Sprintf("%4d", rowNum)
		numberStyle = lipgloss.NewStyle().Foreground(m.styles.FgColor())
	}
	numberStyled := numberStyle.Render(numberStr)

	// Selection indicator - use accent color
	indicator := " "
	if isSelected {
		selectionStyle := lipgloss.NewStyle().Foreground(m.theme.AccentColor)
		indicator = selectionStyle.Render("●")
	}
	numberWithIndicator := numberStyled + indicator

	// Apply text color to name and metadata
	textStyle := lipgloss.NewStyle().Foreground(m.styles.FgColor())

	// Truncate name to fit width
	// Account for: number(3) + indicator(1) + space(1) = 5 chars minimum
	minNameWidth := 10
	availableForName := width - 6
	if availableForName < minNameWidth {
		availableForName = minNameWidth
	}
	truncatedName := truncateString(torrent.Name, availableForName)
	styledName := textStyle.Render(truncatedName)

	// Build the line
	line := fmt.Sprintf("%s %s",
		numberWithIndicator,
		styledName,
	)

	// Check visual width (ignores ANSI codes)
	if lipgloss.Width(line) > width {
		line = truncateString(line, width)
	}
	return line
}

// renderProgressLine: [████████░░] (X.XX GB / Y.YY GB)
func (m *MultilineTorrentListView) renderProgressLine(torrent client.Torrent, width int) string {
	// Indent to align with identity line text (add one more space)
	indent := "      "
	fixedBarWidth := 90 // Fixed width bar so all lines align
	
	// Build suffix with file sizes (progress bar renders its own percentage)
	downloaded := FormatBytes(torrent.Downloaded)
	total := FormatBytes(torrent.Size)
	suffix := fmt.Sprintf(" (%s / %s)", downloaded, total)

	// Apply text color to suffix
	textStyle := lipgloss.NewStyle().Foreground(m.styles.FgColor())
	styledSuffix := textStyle.Render(suffix)

	// Use visual width for calculations
	indentLen := lipgloss.Width(indent)
	suffixLen := lipgloss.Width(suffix)
	
	// Check if everything fits
	totalNeeded := indentLen + fixedBarWidth + suffixLen
	if totalNeeded > width {
		// Reduce suffix if needed
		availableForSuffix := width - indentLen - fixedBarWidth
		if availableForSuffix < 5 {
			// Not enough space, just show indent and bar
			styledSuffix = ""
		} else if availableForSuffix < suffixLen {
			// Truncate suffix
			styledSuffix = textStyle.Render(truncateString(suffix, availableForSuffix))
		}
	}

	// Build progress bar with fixed width
	pb := NewProgressBarBuilder().WithWidth(fixedBarWidth)
	progressBar := pb.RenderProgressBar(string(torrent.Status), float64(torrent.Progress)/100.0)

	line := indent + progressBar + styledSuffix
	return line
}

// renderStatusLine: [Icon] [Downloading]  ↓ X.XX MB/s ↑ X.XX MB/s  Ratio: X.XX  Seeds: X  Peers: Y  ETA: [Time]
func (m *MultilineTorrentListView) renderStatusLine(torrent client.Torrent, width int) string {
	indent := "      "
	textStyle := lipgloss.NewStyle().Foreground(m.styles.FgColor())
	metricsStyle := lipgloss.NewStyle().Foreground(m.theme.ForegroundColor)
	
	// Get category icon for its own column
	categoryIcon := GetCategoryIcon(torrent.Category)
	
	statusLabel := m.formatStatus(string(torrent.Status))
	downSpeed := FormatSpeed(int64(torrent.SpeedDown))
	upSpeed := FormatSpeed(int64(torrent.SpeedUp))
	
	// Calculate ratio from uploaded/downloaded
	var ratio float64
	if torrent.Downloaded > 0 {
		ratio = float64(torrent.Uploaded) / float64(torrent.Downloaded)
	} else if torrent.Uploaded > 0 {
		ratio = float64(torrent.Uploaded)
	} else {
		ratio = 0
	}
	ratioStr := FormatRatio(ratio)

	seeds := FormatPeerCount(int64(torrent.Seeds))
	peers := FormatPeerCount(int64(torrent.Leechs))
	eta := m.calculateETA(torrent)

	// Fixed widths for vertical alignment: icon(2) status(15) speeds(25) ratio(12) seeds(12) peers(12) eta(15)
	// Pad status to fixed width BEFORE styling to preserve alignment
	paddedStatus := fmt.Sprintf("%-15s", statusLabel)
	paddedDownSpeed := fmt.Sprintf("%-12s", downSpeed)
	paddedUpSpeed := fmt.Sprintf("%-12s", upSpeed)
	paddedRatio := fmt.Sprintf("%-8s", ratioStr)
	paddedSeeds := fmt.Sprintf("%-5s", seeds)
	paddedPeers := fmt.Sprintf("%-5s", peers)

	// Apply text colors - mix of textStyle (labels) and metricsStyle (values)
	// Build with mixed styling using pre-padded values
	line := fmt.Sprintf("%s%s %s ↓ %s ↑ %s %s %s %s %s",
		indent,
		categoryIcon,
		textStyle.Render(paddedStatus),
		metricsStyle.Render(paddedDownSpeed),
		metricsStyle.Render(paddedUpSpeed),
		textStyle.Render("Ratio: ") + metricsStyle.Render(paddedRatio),
		textStyle.Render("Seeds: ") + metricsStyle.Render(paddedSeeds),
		textStyle.Render("Peers: ") + metricsStyle.Render(paddedPeers),
		textStyle.Render("ETA: ") + metricsStyle.Render(eta),
	)

	// Check visual width (ignores ANSI codes)
	if lipgloss.Width(line) > width {
		return truncateString(line, width)
	}
	return line
}

// renderMetricsLine is no longer used - metrics merged into status line
func (m *MultilineTorrentListView) renderMetricsLine(torrent client.Torrent, width int) string {
	return ""
}

// Helper functions

// formatStatus converts torrent status to a readable label
func (m *MultilineTorrentListView) formatStatus(status string) string {
	switch status {
	case "downloading":
		return "Downloading"
	case "seeding":
		return "Seeding"
	case "paused":
		return "Paused"
	case "completed":
		return "Completed"
	case "error":
		return "Error"
	case "queueing", "queuedDL", "queuedForChecking":
		return "Queueing"
	case "stalledDL", "stalledUP":
		return "Stalled"
	case "allocating":
		return "Allocating"
	case "metaDL":
		return "Meta DL"
	case "forcedDL", "forcedUP":
		return "Forced"
	default:
		return status
	}
}

// calculateETA returns formatted ETA from the API
func (m *MultilineTorrentListView) calculateETA(torrent client.Torrent) string {
	// Completed torrents show "Done"
	if torrent.Status == client.StatusCompleted {
		return "Done"
	}

	// 8640000 seconds (100 days) is qBittorrent's sentinel for infinite/unknown ETA
	// This happens for paused torrents, unlimited seeding, or other unknown conditions
	if torrent.ETA == 8640000 {
		return "∞"
	}

	// ETA is in seconds from the API
	duration := time.Duration(torrent.ETA) * time.Second
	return FormatDuration(duration)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "…"
	}
	return s[:maxLen-1] + "…"
}
