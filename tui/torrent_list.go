package tui

import (
	"fmt"
	"strings"
	"tqbtui/client"
)

// TorrentListView displays a list of torrents
type TorrentListView struct {
	torrents []client.Torrent
	selected map[string]bool
	cursor   int
	styles   *Styles
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

// Render returns the rendered list
func (t *TorrentListView) Render(width, height int) string {
	if len(t.torrents) == 0 {
		return t.styles.ListItem.Render("No torrents")
	}

	lines := []string{}

	// Calculate dynamic name width based on terminal width
	// Account for padding(0,1) in ListHeader/ListItem which adds 2 chars (1 on each side)
	// Fixed columns: checkbox(1) + space(1) + progress(9) + space(1) + down(10) + space(1) + up(10) + space(1) + seeds(5) + space(1) + leechs(6) + space(1) + status(8)
	effectiveWidth := width - 2 // Account for padding
	fixedWidth := 1 + 1 + 9 + 1 + 10 + 1 + 10 + 1 + 5 + 1 + 6 + 1 + 8
	nameWidth := effectiveWidth - fixedWidth
	if nameWidth < 10 {
		nameWidth = 10
	}

	// Header
	header := t.renderHeader(width, nameWidth)
	lines = append(lines, header)

	// Separator
	if width > 0 {
		lines = append(lines, strings.Repeat("─", width))
	}

	// Calculate visible range with scrolling
	maxItems := height - 3
	if maxItems < 1 {
		maxItems = 1
	}

	// Calculate start index for viewport
	startIdx := t.cursor - (maxItems / 2)
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx+maxItems > len(t.torrents) {
		startIdx = len(t.torrents) - maxItems
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Items
	for i := startIdx; i < startIdx+maxItems && i < len(t.torrents); i++ {
		line := t.renderTorrentRow(t.torrents[i], i == t.cursor, nameWidth)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderHeader returns the header row
func (t *TorrentListView) renderHeader(width, nameWidth int) string {
	// Format: checkbox Name | Progress | ↓Down | ↑Up | Seed | Leech | Status
	// Use checkbox placeholder (☐) to align with data rows
	return t.styles.ListHeader.Render(
		fmt.Sprintf("%s %-"+fmt.Sprintf("%d", nameWidth)+"s %9s %10s %10s %5s %6s %8s",
			"☐", "Name", "Progress", "↓Down", "↑Up", "Seed", "Leech", "Status"),
	)
}

// renderTorrentRow returns a formatted torrent row
func (t *TorrentListView) renderTorrentRow(torrent client.Torrent, selected bool, nameWidth int) string {
	checkbox := "☐"
	if t.selected[torrent.ID] {
		checkbox = "☑"
	}

	name := truncate(torrent.Name, nameWidth)
	progress := fmt.Sprintf("%3d%%", torrent.Progress)
	downSpeed := formatSpeed(torrent.SpeedDown)
	upSpeed := formatSpeed(torrent.SpeedUp)
	seeds := fmt.Sprintf("%5d", torrent.Seeds)
	leechs := fmt.Sprintf("%6d", torrent.Leechs)
	status := fmt.Sprintf("%8s", shortenStatus(string(torrent.Status)))

	row := fmt.Sprintf("%s %-"+fmt.Sprintf("%d", nameWidth)+"s %9s %10s %10s %5s %6s %8s",
		checkbox, name, progress, downSpeed, upSpeed, seeds, leechs, status)

	style := t.styles.ListItem
	if selected {
		style = t.styles.ListItemSelected
	}

	return style.Render(row)
}

// Helper functions

func truncate(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length-3]) + "..."
	}
	// Pad to exact length
	if len(runes) < length {
		return s + strings.Repeat(" ", length-len(runes))
	}
	return s
}

func rightAlign(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	padding := width - len(runes)
	return strings.Repeat(" ", padding) + s
}

func formatSpeed(speed float64) string {
	if speed == 0 {
		return "-"
	}
	if speed < 1024 {
		return fmt.Sprintf("%.0fB/s", speed)
	}
	if speed < 1024*1024 {
		return fmt.Sprintf("%.1fKB/s", speed/1024)
	}
	return fmt.Sprintf("%.1fMB/s", speed/(1024*1024))
}

func shortenStatus(status string) string {
	switch status {
	case "downloading":
		return "d/l"
	case "seeding":
		return "seed"
	case "paused":
		return "paused"
	case "error":
		return "error"
	case "queued":
		return "queue"
	default:
		return status
	}
}
