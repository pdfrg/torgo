package tui

import (
	"fmt"
	"github.com/pdfrg/torgo/client"
	"strings"

	"charm.land/lipgloss/v2"
)

// TorrentListView displays a list of torrents (virtualized, sticky header)
type TorrentListView struct {
	torrents []client.Torrent
	selected map[string]bool
	cursor   int
	// scrollFirst is the index of the first visible torrent (header is sticky).
	scrollFirst int
	styles      *Styles

	// Per-frame cached styles (rebuilt once per Render, reused by all rows).
	stNumCursor lipgloss.Style
	stNumNormal lipgloss.Style
	stSel       lipgloss.Style
	stField     lipgloss.Style
	stSep       lipgloss.Style
}

// NewTorrentListView creates a new torrent list view
func NewTorrentListView(styles *Styles) *TorrentListView {
	return &TorrentListView{
		torrents: []client.Torrent{},
		selected: make(map[string]bool),
		cursor:   0,
		styles:   styles,
	}
}

// SetTorrents updates the torrent list
func (t *TorrentListView) SetTorrents(torrents []client.Torrent) {
	t.torrents = torrents
	if t.cursor >= len(torrents) {
		t.cursor = 0
	}
	if t.scrollFirst >= len(torrents) {
		t.scrollFirst = 0
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.scrollFirst < 0 {
		t.scrollFirst = 0
	}
}

// MoveCursor moves the selection up or down
func (t *TorrentListView) MoveCursor(direction int) {
	t.cursor += direction
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor >= len(t.torrents) {
		t.cursor = len(t.torrents) - 1
	}
}

// PageMove moves the cursor by a full page (for Page Up/Down)
// direction: -1 for page up, +1 for page down
// pageSize is the number of visible lines (typically viewport height)
func (t *TorrentListView) PageMove(direction int, pageSize int) {
	if pageSize <= 0 {
		pageSize = 10 // Sensible default
	}
	t.cursor += direction * (pageSize - 1) // -1 to keep one line of context
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor >= len(t.torrents) {
		t.cursor = len(t.torrents) - 1
	}
}

// ToggleSelection toggles the selected state of the current torrent
func (t *TorrentListView) ToggleSelection() {
	if t.cursor < len(t.torrents) {
		id := t.torrents[t.cursor].ID
		t.selected[id] = !t.selected[id]
	}
}

// SelectAll toggles all torrents - if all selected, deselect all; otherwise select all
func (t *TorrentListView) SelectAll() {
	// Check if all are selected
	allSelected := true
	for _, torrent := range t.torrents {
		if !t.selected[torrent.ID] {
			allSelected = false
			break
		}
	}

	// If all selected, deselect all; otherwise select all
	if allSelected {
		t.ClearSelection()
	} else {
		for _, torrent := range t.torrents {
			t.selected[torrent.ID] = true
		}
	}
}

// ClearSelection clears all selections
func (t *TorrentListView) ClearSelection() {
	t.selected = make(map[string]bool)
}

// GetSelected returns the IDs of selected torrents
func (t *TorrentListView) GetSelected() []string {
	ids := []string{}
	for id, sel := range t.selected {
		if sel {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetCurrentTorrent returns the currently selected torrent
func (t *TorrentListView) GetCurrentTorrent() *client.Torrent {
	if t.cursor >= 0 && t.cursor < len(t.torrents) {
		return &t.torrents[t.cursor]
	}
	return nil
}

// Render returns the rendered list (virtualized: header + visible rows only)
func (t *TorrentListView) Render(width, height int) string {
	if len(t.torrents) == 0 {
		return t.styles.ListItem.Render("No torrents")
	}

	// Calculate dynamic name width based on terminal width
	// Account for padding(0,1) in ListHeader which adds 2 chars (1 on each side)
	// Fixed columns: number(3) + indicator(1) + space(1) + size(7) + space(1) + progress(5) + space(1) + down(7) + space(1) + up(7) + space(1) + seeds(9) + space(1) + leechs(9) + space(1) + status(8)
	effectiveWidth := width - 2 // Account for padding
	fixedWidth := 3 + 1 + 1 + 7 + 1 + 5 + 1 + 7 + 1 + 7 + 1 + 9 + 1 + 9 + 1 + 8
	nameWidth := effectiveWidth - fixedWidth
	if nameWidth < 10 {
		nameWidth = 10
	}

	// Per-frame shared styles (not per row).
	t.stNumCursor = lipgloss.NewStyle().Foreground(GetCurrentTheme().CursorColor)
	t.stNumNormal = lipgloss.NewStyle().Foreground(GetCurrentTheme().TextNormal)
	t.stSel = lipgloss.NewStyle().Foreground(GetCurrentTheme().AccentColor)
	t.stField = lipgloss.NewStyle().Foreground(GetCurrentTheme().ForegroundColor)
	t.stSep = lipgloss.NewStyle().Foreground(GetCurrentTheme().AccentColor)

	// Build the visible content: sticky header + separator + window of rows.
	lines := []string{}

	// Header
	header := t.renderHeader(width, nameWidth)
	lines = append(lines, header)

	// Separator with accent color
	if width > 0 {
		separator := t.stSep.Render(strings.Repeat("─", width))
		lines = append(lines, separator)
	}

	// Window the visible rows so a 900-torrent list only styles ~height rows.
	visibleRows := height - 2 // header + separator
	if visibleRows < 1 {
		visibleRows = 1
	}
	if visibleRows > len(t.torrents) {
		visibleRows = len(t.torrents)
	}
	if t.cursor < t.scrollFirst {
		t.scrollFirst = t.cursor
	} else if t.cursor >= t.scrollFirst+visibleRows {
		t.scrollFirst = t.cursor - visibleRows + 1
	}
	maxFirst := len(t.torrents) - visibleRows
	if maxFirst < 0 {
		maxFirst = 0
	}
	if t.scrollFirst > maxFirst {
		t.scrollFirst = maxFirst
	}
	if t.scrollFirst < 0 {
		t.scrollFirst = 0
	}
	end := t.scrollFirst + visibleRows
	if end > len(t.torrents) {
		end = len(t.torrents)
	}

	for i := t.scrollFirst; i < end; i++ {
		line := t.renderTorrentRow(t.torrents[i], i == t.cursor, nameWidth, i+1)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Update is a no-op kept for API compatibility (scrolling is cursor-driven).
func (t *TorrentListView) Update(msg interface{}) {
}

// renderHeader returns the header row
func (t *TorrentListView) renderHeader(width, nameWidth int) string {
	// Format: # Name | Size | Prog | ↓Down | ↑Up | Seed | Leech | Status
	// Use "#" as column header for sequential numbering (right-aligned in 2 chars)
	// No space between name and size to match row format
	headerStyle := lipgloss.NewStyle().
		Foreground(GetCurrentTheme().CursorColor).
		Bold(true).
		Padding(0, 1)

	return headerStyle.Render(
		fmt.Sprintf(" %2s  %-"+fmt.Sprintf("%d", nameWidth)+"s%7s %5s %7s %7s %9s %9s %8s",
			"#", "Name", "Size", "Prog", "↓Down", "↑Up", "Seed", "Leech", "Status"),
	)
}

// renderTorrentRow returns a formatted torrent row
func (t *TorrentListView) renderTorrentRow(torrent client.Torrent, cursor bool, nameWidth int, rowNum int) string {
	// Format the row number with optional selection indicator and cursor prompt
	// Cursor position: "> N" in cursor color with dot indicator (●)
	// Non-cursor: "  N" in text normal with dot indicator (●)
	// Selected: add dot indicator (●)
	isSelected := t.selected[torrent.ID]

	// Build number string with cursor prompt or padding
	// Both formats are 4 chars to prevent title shift when going from single to double digits
	var numberStr string
	if cursor {
		// Cursor row: "> " + 2-digit number = 4 chars ("> 1", "> 10", etc), styled with cursor color
		numberStr = fmt.Sprintf("> %2d", rowNum)
	} else {
		// Non-cursor row: 4-char right-aligned number = 4 chars ("   1", "  10", etc), styled with text normal
		numberStr = fmt.Sprintf("%4d", rowNum)
	}
	numberStyle := t.stNumNormal
	if cursor {
		numberStyle = t.stNumCursor
	}

	numberStyled := numberStyle.Render(numberStr)

	// Add dot indicator for selected items (styled with accent color)
	indicator := " "
	if isSelected {
		indicator = t.stSel.Render("●")
	}
	numberWithIndicator := numberStyled + indicator

	// Reserve space for private tracker indicator 🔒 (emoji width 2 + space = 3)
	privateSuffix := ""
	effectiveNameWidth := nameWidth
	if torrent.IsPrivate {
		effectiveNameWidth = nameWidth - 3
		privateSuffix = " 🔒"
	}

	// Pad the name to exact width (using rune-aware width)
	paddedName := truncate(torrent.Name, effectiveNameWidth)

	// Calculate progress bar fill using rune length for unicode-aware width
	nameRunes := []rune(paddedName)
	filledWidth := (len(nameRunes) * int(torrent.Progress)) / 100
	if filledWidth > len(nameRunes) {
		filledWidth = len(nameRunes)
	}

	// Get solid status color for oneline view with smart contrast text
	statusColor := GetCurrentTheme().GetStatusColorForOneline(string(torrent.Status))
	statusColorHex := GetCurrentTheme().GetStatusColorHexForOneline(string(torrent.Status))
	textColor := GetCurrentTheme().GetContrastTextColorForBg(statusColorHex)

	// Render the name with progress bar background
	filledPart := string(nameRunes[:filledWidth])
	unfilledPart := string(nameRunes[filledWidth:])

	filledStyle := lipgloss.NewStyle().Background(statusColor).Foreground(textColor)
	unfilledStyle := t.stField

	// Render the styled name parts
	renderedName := filledStyle.Render(filledPart) + unfilledStyle.Render(unfilledPart) + privateSuffix

	// Note: lipgloss.Width() on styled text returns the visual width (excluding ANSI codes)
	// We need to account for the original name width in our format string
	nameColWidth := lipgloss.Width(renderedName)

	// Create a style for individual fields without padding or highlighting
	// Only the number gets the highlight styling based on selection state
	fieldStyle := t.stField

	// Format field values with proper alignment BEFORE applying style
	// Match the header format exactly: %3s %-nameWidths %7s %5s %7s %7s %9s %9s %8s
	sizeStr := fmt.Sprintf("%7s", formatSize(torrent.Size))
	progressStr := fmt.Sprintf("%5s", fmt.Sprintf("%d%%", torrent.Progress))
	downSpeedStr := fmt.Sprintf("%7s", formatSpeed(torrent.SpeedDown))
	upSpeedStr := fmt.Sprintf("%7s", formatSpeed(torrent.SpeedUp))
	seedsStr := fmt.Sprintf("%9s", FormatPeerCount(int64(torrent.Seeds), int64(torrent.TotalSeeds)))
	leechsStr := fmt.Sprintf("%9s", FormatPeerCount(int64(torrent.Leechs), int64(torrent.TotalLeechs)))
	statusStr := fmt.Sprintf("%8s", shortenStatus(string(torrent.Status)))

	// Apply fieldStyle (without padding) to the formatted strings for consistent text color
	size := fieldStyle.Render(sizeStr)
	progress := fieldStyle.Render(progressStr)
	downSpeed := fieldStyle.Render(downSpeedStr)
	upSpeed := fieldStyle.Render(upSpeedStr)
	seeds := fieldStyle.Render(seedsStr)
	leechs := fieldStyle.Render(leechsStr)
	status := fieldStyle.Render(statusStr)

	// Build format string with dynamic padding to account for styled name
	// Account for the difference between rendered width and byte length due to ANSI codes
	paddingAfterName := nameWidth - nameColWidth
	if paddingAfterName < 0 {
		paddingAfterName = 0
	}

	// Build row to match header format: %4s %-nameWidths %7s %5s %7s %7s %9s %9s %8s
	// The renderedName already includes ANSI codes, so we use padding to account for visual width
	// numberWithIndicator is 4 chars (3 for number + 1 for indicator)
	row := fmt.Sprintf("%s %s%*s%s %s %s %s %s %s %s",
		numberWithIndicator, renderedName, paddingAfterName, "", size, progress, downSpeed, upSpeed, seeds, leechs, status)

	return row
}

// Helper functions

func truncate(s string, length int) string {
	// Use visual width to handle wide characters (emojis, CJK, etc.)
	visualWidth := lipgloss.Width(s)

	if visualWidth > length {
		// Truncate character by character, checking visual width
		runes := []rune(s)
		for i := len(runes) - 1; i >= 0; i-- {
			testStr := string(runes[:i])
			if lipgloss.Width(testStr) <= length-3 {
				return testStr + "..."
			}
		}
		return "..."
	}

	// Pad to exact length using spaces
	if visualWidth < length {
		return s + strings.Repeat(" ", length-visualWidth)
	}

	return s
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	if bytes == 0 {
		return "0"
	}
	if bytes >= TB {
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	}
	if bytes >= GB {
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.0fM", float64(bytes)/float64(MB))
	}
	if bytes >= KB {
		return fmt.Sprintf("%.0fK", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%dB", bytes)
}

func formatSpeed(speed float64) string {
	if speed == 0 {
		return "-"
	}
	if speed < 1024 {
		return fmt.Sprintf("%.0fB", speed)
	}
	if speed < 1024*1024 {
		return fmt.Sprintf("%.1fK", speed/1024)
	}
	return fmt.Sprintf("%.1fM", speed/(1024*1024))
}

func shortenStatus(status string) string {
	switch status {
	case "downloading":
		return "d/l"
	case "queuedDL":
		return "queue"
	case "stalledDL":
		return "stall"
	case "paused":
		return "paused"
	case "seeding":
		return "seed"
	case "completed":
		return "done"
	case "error":
		return "error"
	default:
		return status
	}
}
