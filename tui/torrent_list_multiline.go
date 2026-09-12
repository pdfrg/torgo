package tui

import (
	"fmt"
	"github.com/pdfrg/torgo/client"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// MultilineTorrentListView displays torrents in a 3-line format per entry
// Each torrent occupies 3 lines: Identity, Progress, Status/Metrics + blank line
// Rendering is virtualized: only the visible window is styled per frame.
type MultilineTorrentListView struct {
	torrents []client.Torrent
	selected map[string]bool
	cursor   int
	// yOffset is the first visible content line (line-based scroll offset,
	// mirroring the old viewport behavior including partial blocks).
	yOffset int
	styles  *Styles
	theme   Theme

	// Per-frame cached styles (rebuilt once per Render, reused by all rows).
	stNumCursor lipgloss.Style
	stNumNormal lipgloss.Style
	stSel       lipgloss.Style
	stText      lipgloss.Style
	stLabel     lipgloss.Style
	stMetrics   lipgloss.Style
	barModels   map[string]*ProgressBarBuilder
	barModelW   int
}

// NewMultilineTorrentListView creates a new multi-line torrent list view
func NewMultilineTorrentListView(styles *Styles) *MultilineTorrentListView {
	return &MultilineTorrentListView{
		torrents: []client.Torrent{},
		selected: make(map[string]bool),
		cursor:   0,
		styles:   styles,
		theme:    GetCurrentTheme(),
	}
}

// SetTorrents updates the torrent list
func (m *MultilineTorrentListView) SetTorrents(torrents []client.Torrent) {
	m.torrents = torrents
	if m.cursor >= len(torrents) {
		m.cursor = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// yOffset is clamped against content height in Render (needs height).
	if m.yOffset < 0 {
		m.yOffset = 0
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

// Update is a no-op kept for API compatibility (scrolling is cursor-driven).
func (m *MultilineTorrentListView) Update(msg interface{}) {
}

// InvalidateThemeCaches drops cached theme-dependent rendering state.
// Called after a theme change so cached progress-bar builders are rebuilt
// with the new palette. ProgressBarBuilder also tracks the live theme, so
// this is defensive belt-and-suspenders for other cached fields.
func (m *MultilineTorrentListView) InvalidateThemeCaches() {
	m.barModels = nil
	m.barModelW = 0
}

// Render returns only the visible window of torrent blocks (virtualized).
// Blocks are 4 lines each (3 content + 1 blank separator).
func (m *MultilineTorrentListView) Render(width, height int) string {
	if len(m.torrents) == 0 {
		return m.styles.ListItem.Render("No torrents")
	}

	// Bounds check for width and height
	if width < 1 || height < 1 {
		return ""
	}

	// Sync theme on every render
	m.theme = GetCurrentTheme()

	// Build per-frame shared styles once (not per row).
	m.stNumCursor = lipgloss.NewStyle().Foreground(m.theme.CursorColor)
	m.stNumNormal = lipgloss.NewStyle().Foreground(m.theme.TextNormal)
	m.stSel = lipgloss.NewStyle().Foreground(m.theme.AccentColor)
	m.stText = lipgloss.NewStyle().Foreground(m.theme.ForegroundColor)
	m.stLabel = lipgloss.NewStyle().Foreground(m.theme.TextNormal)
	m.stMetrics = lipgloss.NewStyle().Foreground(m.theme.ForegroundColor)

	// Virtual full-content geometry: each torrent is 3 content lines + 1
	// blank separator line, except the last torrent (no trailing blank).
	// Only blocks overlapping the [yOffset, yOffset+height) window are
	// styled, so a 900-torrent list renders ~height/4 rows per frame.
	// Output never exceeds height lines, so the caller's overflow trim
	// can't chop the cursor block.
	totalLines := len(m.torrents)*4 - 1
	maxOffset := totalLines - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.yOffset > maxOffset {
		m.yOffset = maxOffset
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}

	// Keep the whole cursor block (3 content lines) visible.
	cursorStart := m.cursor * 4
	cursorEnd := cursorStart + 2
	if cursorStart < m.yOffset {
		m.yOffset = cursorStart
	}
	if cursorEnd >= m.yOffset+height {
		m.yOffset = cursorEnd - height + 1
	}
	if m.yOffset > maxOffset {
		m.yOffset = maxOffset
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}

	winTop := m.yOffset
	winBottom := m.yOffset + height // exclusive

	// Build only the visible content, slicing partial edge blocks.
	lines := []string{}
	for i := 0; i < len(m.torrents) && len(lines) < height; i++ {
		base := i * 4
		// Block content lines base..base+2, separator blank at base+3
		// (no trailing blank after the last torrent).
		blockEnd := base + 3
		if i == len(m.torrents)-1 {
			blockEnd = base + 2
		}
		if blockEnd < winTop || base >= winBottom {
			continue
		}
		block := m.renderTorrentBlock(m.torrents[i], i == m.cursor, width, i+1)
		for k := 0; k < 3; k++ {
			lineNo := base + k
			if lineNo >= winTop && lineNo < winBottom {
				lines = append(lines, block[k])
			}
		}
		// Separator blank between entries (except after last torrent).
		if i < len(m.torrents)-1 {
			sepNo := base + 3
			if sepNo >= winTop && sepNo < winBottom {
				lines = append(lines, "")
			}
		}
	}

	return strings.Join(lines, "\n")
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
	if cursor {
		// Cursor row: "> " + 2-digit number = 4 chars ("> 1", "> 10", etc), styled with cursor color
		numberStr = fmt.Sprintf("> %2d", rowNum)
	} else {
		// Non-cursor row: 4-char right-aligned number = 4 chars ("   1", "  10", etc), styled with fg color
		numberStr = fmt.Sprintf("%4d", rowNum)
	}
	numberStyle := m.stNumNormal
	if cursor {
		numberStyle = m.stNumCursor
	}
	numberStyled := numberStyle.Render(numberStr)

	// Selection indicator - use accent color
	indicator := " "
	if isSelected {
		indicator = m.stSel.Render("●")
	}
	numberWithIndicator := numberStyled + indicator

	// Apply text color to name (variable data = ForegroundColor)
	textStyle := m.stText

	// Reserve space for private tracker indicator 🔒 (emoji width 2 + space = 3)
	privateSuffix := ""
	reserveForPrivate := 0
	if torrent.IsPrivate {
		reserveForPrivate = 3
		privateSuffix = " 🔒"
	}

	// Truncate name to fit width
	// Account for: number(3) + indicator(1) + space(1) = 5 chars minimum
	minNameWidth := 10
	availableForName := width - 6 - reserveForPrivate
	if availableForName < minNameWidth {
		availableForName = minNameWidth
	}
	truncatedName := TruncateString(torrent.Name, availableForName)
	styledName := textStyle.Render(truncatedName)

	// Build the line
	line := fmt.Sprintf("%s %s%s",
		numberWithIndicator,
		styledName,
		privateSuffix,
	)

	// Check visual width (ignores ANSI codes)
	if lipgloss.Width(line) > width {
		line = TruncateString(line, width)
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

	// Apply text color to suffix (variable data = ForegroundColor)
	textStyle := m.stText
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
			styledSuffix = textStyle.Render(TruncateString(suffix, availableForSuffix))
		}
	}

	// Reuse one progress model per status per frame (identical output,
	// avoids rebuilding gradient models for every row).
	barW := fixedBarWidth
	if m.barModels == nil || m.barModelW != barW {
		m.barModels = make(map[string]*ProgressBarBuilder)
		m.barModelW = barW
	}
	status := string(torrent.Status)
	pb, ok := m.barModels[status]
	if !ok {
		pb = NewProgressBarBuilder().WithWidth(barW)
		m.barModels[status] = pb
	}
	progressBar := pb.RenderProgressBar(status, float64(torrent.Progress)/100.0)

	line := indent + progressBar + styledSuffix
	return line
}

// renderStatusLine: [Icon] [Downloading]  ↓ X.XX MB/s ↑ X.XX MB/s  Ratio: X.XX  Seeds: X  Peers: Y  ETA: [Time]
func (m *MultilineTorrentListView) renderStatusLine(torrent client.Torrent, width int) string {
	indent := "      "
	labelStyle := m.stLabel
	metricsStyle := m.stMetrics

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

	seeds := FormatPeerCount(int64(torrent.Seeds), int64(torrent.TotalSeeds))
	peers := FormatPeerCount(int64(torrent.Leechs), int64(torrent.TotalLeechs))
	eta := m.calculateETA(torrent)

	// Fixed widths for vertical alignment: icon(2) status(15) speeds(25) ratio(12) seeds(12) peers(12) eta(15)
	// Pad status to fixed width BEFORE styling to preserve alignment
	paddedStatus := fmt.Sprintf("%-15s", statusLabel)
	paddedDownSpeed := fmt.Sprintf("%-12s", downSpeed)
	paddedUpSpeed := fmt.Sprintf("%-12s", upSpeed)
	paddedRatio := fmt.Sprintf("%-8s", ratioStr)
	paddedSeeds := fmt.Sprintf("%-9s", seeds)
	paddedPeers := fmt.Sprintf("%-9s", peers)

	// Apply text colors - mix of textStyle (labels) and metricsStyle (values)
	// Build with mixed styling using pre-padded values
	line := fmt.Sprintf("%s%s %s %s %s %s %s %s %s %s %s",
		indent,
		categoryIcon,
		labelStyle.Render(paddedStatus),
		labelStyle.Render("↓"), metricsStyle.Render(paddedDownSpeed),
		labelStyle.Render("↑"), metricsStyle.Render(paddedUpSpeed),
		labelStyle.Render("Ratio: ")+metricsStyle.Render(paddedRatio),
		labelStyle.Render("Seeds: ")+metricsStyle.Render(paddedSeeds),
		labelStyle.Render("Peers: ")+metricsStyle.Render(paddedPeers),
		labelStyle.Render("ETA: ")+metricsStyle.Render(eta),
	)

	// Check visual width (ignores ANSI codes)
	if lipgloss.Width(line) > width {
		return TruncateString(line, width)
	}
	return line
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
