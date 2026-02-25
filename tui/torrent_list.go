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

// SelectAll selects all torrents
func (t *TorrentListView) SelectAll() {
	for _, torrent := range t.torrents {
		t.selected[torrent.ID] = true
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

	// Header
	header := t.renderHeader(width)
	lines = append(lines, header)

	// Separator
	lines = append(lines, strings.Repeat("─", width))

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
		line := t.renderTorrentRow(t.torrents[i], i == t.cursor, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderHeader returns the header row
func (t *TorrentListView) renderHeader(width int) string {
	// Format: [S] Name | Progress | ↓Down | ↑Up | Seeds | Leechs | Status
	return t.styles.ListHeader.Render(
		fmt.Sprintf("%-3s %-30s %8s %8s %8s %5s %5s %8s",
			"[S]", "Name", "Progress", "↓Down", "↑Up", "Seeds", "Leechs", "Status"),
	)
}

// renderTorrentRow returns a formatted torrent row
func (t *TorrentListView) renderTorrentRow(torrent client.Torrent, selected bool, width int) string {
	checkbox := "☐"
	if t.selected[torrent.ID] {
		checkbox = "☑"
	}

	name := truncate(torrent.Name, 30)
	progress := fmt.Sprintf("%3d%%", torrent.Progress)
	downSpeed := formatSpeed(torrent.SpeedDown)
	upSpeed := formatSpeed(torrent.SpeedUp)
	seeds := fmt.Sprintf("%5d", torrent.Seeds)
	leechs := fmt.Sprintf("%5d", torrent.Leechs)
	status := string(torrent.Status)

	row := fmt.Sprintf("%s %s %8s %8s %8s %s %s %8s",
		checkbox, name, progress, downSpeed, upSpeed, seeds, leechs, status)

	style := t.styles.ListItem
	if selected {
		style = t.styles.ListItemSelected
	}

	return style.Render(row)
}

// Helper functions

func truncate(s string, length int) string {
	if len(s) > length {
		return s[:length-3] + "..."
	}
	return s
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
