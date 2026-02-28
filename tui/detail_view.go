package tui

import (
	"fmt"
	"strings"

	"tqbtui/client"
)

// DetailView represents the torrent detail view
type DetailView struct {
	styles     *Styles
	detail     *client.TorrentDetail
	files      []client.TorrentFile
	fileTree   []FileTreeNode // Hierarchical tree of files
	width      int
	height     int
	cursorPos  int // Current cursor position in the view
	expanded   map[string]bool // Track which directories are expanded
	scrollPos  int // Vertical scroll position
}

// FileTreeNode represents a node in the file tree (either file or directory)
type FileTreeNode struct {
	Name         string
	IsDirectory  bool
	Size         int64
	Downloaded   int64
	Priority     int
	Index        int // Original file index (for files only)
	Level        int // Depth in tree
	Children     []FileTreeNode
	Parent       *FileTreeNode
	Expanded     bool
}

// NewDetailView creates a new detail view
func NewDetailView(styles *Styles, detail *client.TorrentDetail, files []client.TorrentFile) *DetailView {
	dv := &DetailView{
		styles:    styles,
		detail:    detail,
		files:     files,
		expanded:  make(map[string]bool),
		cursorPos: 0,
		scrollPos: 0,
	}

	// Build file tree
	dv.buildFileTree()

	return dv
}

// buildFileTree builds a hierarchical tree from the flat file list
func (dv *DetailView) buildFileTree() {
	// For now, create a simple flat list
	// TODO: Build proper directory tree with expansion support
	dv.fileTree = make([]FileTreeNode, len(dv.files))
	for i, f := range dv.files {
		dv.fileTree[i] = FileTreeNode{
			Name:       f.Name,
			IsDirectory: false,
			Size:       f.Size,
			Downloaded: f.Downloaded,
			Priority:   f.Priority,
			Index:      f.Index,
			Level:      0,
		}
	}
}

// Render renders the detail view
func (dv *DetailView) Render(width, height int) string {
	dv.width = width
	dv.height = height

	if dv.detail == nil {
		return "Loading..."
	}

	var lines []string

	// Title with torrent name
	titleLine := fmt.Sprintf("  %s", dv.detail.Name)
	lines = append(lines, dv.styles.Title.Width(width).Render(titleLine))
	lines = append(lines, strings.Repeat("─", width))

	// Info section
	lines = append(lines, "")
	lines = append(lines, dv.renderInfoSection())

	// Files section
	lines = append(lines, "")
	lines = append(lines, dv.renderFilesSection(width, height-len(lines)-2))

	// Pad with empty lines to fill the available height
	output := strings.Join(lines, "\n")
	outputLines := strings.Split(output, "\n")
	for len(outputLines) < height {
		outputLines = append(outputLines, "")
	}

	return strings.Join(outputLines, "\n")
}

// renderInfoSection renders the basic info fields
func (dv *DetailView) renderInfoSection() string {
	var lines []string

	// Name
	lines = append(lines, fmt.Sprintf("Name:             %s", dv.detail.Name))

	// Category
	category := dv.detail.Category
	if category == "" {
		category = "(none)"
	}
	lines = append(lines, fmt.Sprintf("Category:         %s", category))

	// Tags (if any)
	tags := "(none)"
	if len(dv.detail.Tags) > 0 {
		tags = strings.Join(dv.detail.Tags, ", ")
	}
	lines = append(lines, fmt.Sprintf("Tags:             %s", tags))

	// Comments
	comments := dv.detail.Comments
	if comments == "" {
		comments = "(none)"
	}
	lines = append(lines, fmt.Sprintf("Comments:         %s", comments))

	// Save path
	lines = append(lines, fmt.Sprintf("Download Path:    %s", dv.detail.SavePath))

	// Size info
	totalStr := formatBytes(dv.detail.TotalSize)
	downloadedStr := formatBytes(dv.detail.Downloaded)
	progress := 0
	if dv.detail.TotalSize > 0 {
		progress = int((float64(dv.detail.Downloaded) / float64(dv.detail.TotalSize)) * 100)
	}
	lines = append(lines, fmt.Sprintf("Size:             %s / %s (%d%%)", downloadedStr, totalStr, progress))

	return strings.Join(lines, "\n")
}

// renderFilesSection renders the file tree
func (dv *DetailView) renderFilesSection(width, maxLines int) string {
	var lines []string

	lines = append(lines, "Files:")

	if len(dv.fileTree) == 0 {
		lines = append(lines, "  (no files)")
		return strings.Join(lines, "\n")
	}

	// Render file tree (simple flat list for now)
	for i, node := range dv.fileTree {
		if i >= maxLines-1 {
			lines = append(lines, "  ... (more files)")
			break
		}

		fileLine := dv.renderFileNode(node, width-4)
		lines = append(lines, "  "+fileLine)
	}

	return strings.Join(lines, "\n")
}

// renderFileNode renders a single file or directory node
func (dv *DetailView) renderFileNode(node FileTreeNode, width int) string {
	// Priority indicator
	priority := "[ ]"
	if node.Priority != 0 {
		priority = "[x]"
	}

	// Size
	sizeStr := formatBytes(node.Size)

	// Progress
	downloaded := node.Downloaded
	progress := 0
	if node.Size > 0 {
		progress = int((float64(downloaded) / float64(node.Size)) * 100)
	}

	// Build the line (simplified format)
	// [x] filename                                   12.3 MB [75%]
	name := node.Name
	if len(name) > width-30 {
		name = name[:width-30] + "…"
	}

	line := fmt.Sprintf("%s %-*s %8s [%3d%%]", priority, width-30, name, sizeStr, progress)
	return line
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// MoveCursor moves the cursor up or down
func (dv *DetailView) MoveCursor(delta int) {
	newPos := dv.cursorPos + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= len(dv.fileTree) {
		newPos = len(dv.fileTree) - 1
	}
	dv.cursorPos = newPos
}

// GetCurrentFile returns the currently selected file (or nil)
func (dv *DetailView) GetCurrentFile() *client.TorrentFile {
	if dv.cursorPos >= 0 && dv.cursorPos < len(dv.files) {
		return &dv.files[dv.cursorPos]
	}
	return nil
}
