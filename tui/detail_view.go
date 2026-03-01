package tui

import (
	"fmt"
	"strings"

	"tqbtui/client"
)

// DetailView represents the torrent detail view
type DetailView struct {
	styles      *Styles
	detail      *client.TorrentDetail
	files       []client.TorrentFile
	fileTree    []*FileTreeNode // Hierarchical tree of files
	width       int
	height      int
	cursorPos   int // Current cursor position in the view (in info section or files)
	expanded    map[string]bool // Track which directories are expanded
	scrollPos   int // Vertical scroll position
	editMode    string // "", "name", "category", "tags", "comments", "location"
	editValue   string // Current edit field value
	editCursorX int    // Cursor position within edit field
	editError   string // Error message during edit
	infoScroll  int    // Scroll position in info section
}

// FileTreeNode represents a node in the file tree (either file or directory)
type FileTreeNode struct {
	Name         string
	IsDirectory  bool
	Size         int64
	Downloaded   int64
	Priority     int
	Index        int          // Original file index (for files only)
	Level        int          // Depth in tree
	Children     []*FileTreeNode
	Parent       *FileTreeNode
	Expanded     bool
	Path         string       // Full path for sorting and identification
}

// FlattenedNode is used for rendering: represents a node with its visual state
type FlattenedNode struct {
	Node     *FileTreeNode
	Depth    int
	IsLast   bool    // Last child in parent's children
	Siblings []bool  // Track which ancestors are last children (for tree lines)
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
	if len(dv.files) == 0 {
		dv.fileTree = []*FileTreeNode{}
		return
	}

	// Create a map to track directories we've seen
	dirMap := make(map[string]*FileTreeNode)

	// Create root nodes for top-level files and directories
	var rootNodes []*FileTreeNode

	for _, f := range dv.files {
		// Split the path to get directory components
		parts := strings.Split(f.Name, "/")

		if len(parts) == 1 {
			// Top-level file
			node := &FileTreeNode{
				Name:       parts[0],
				IsDirectory: false,
				Size:       f.Size,
				Downloaded: f.Downloaded,
				Priority:   f.Priority,
				Index:      f.Index,
				Level:      0,
				Path:       f.Name,
				Expanded:   false,
			}
			rootNodes = append(rootNodes, node)
		} else {
			// File in a subdirectory - build the directory tree
			currentPath := ""
			var parentNode *FileTreeNode

			for i, part := range parts[:len(parts)-1] {
				if currentPath == "" {
					currentPath = part
				} else {
					currentPath = currentPath + "/" + part
				}

				// Check if this directory already exists
				if dirNode, exists := dirMap[currentPath]; exists {
					parentNode = dirNode
				} else {
					// Create new directory node
					dirNode := &FileTreeNode{
						Name:        part,
						IsDirectory: true,
						Level:       i,
						Path:        currentPath,
						Expanded:    false,
					}

					dirMap[currentPath] = dirNode

					// Add to parent or root
					if parentNode == nil {
						rootNodes = append(rootNodes, dirNode)
					} else {
						parentNode.Children = append(parentNode.Children, dirNode)
						dirNode.Parent = parentNode
					}

					parentNode = dirNode
				}
			}

			// Add the file to its parent directory
			fileName := parts[len(parts)-1]
			fileNode := &FileTreeNode{
				Name:       fileName,
				IsDirectory: false,
				Size:       f.Size,
				Downloaded: f.Downloaded,
				Priority:   f.Priority,
				Index:      f.Index,
				Level:      len(parts) - 1,
				Path:       f.Name,
				Expanded:   false,
			}

			if parentNode != nil {
				parentNode.Children = append(parentNode.Children, fileNode)
				fileNode.Parent = parentNode
			}
		}
	}

	// Start with all directories collapsed
	dv.fileTree = rootNodes
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
	if dv.editMode == "name" {
		lines = append(lines, dv.renderEditField("Name", dv.editValue))
	} else {
		lines = append(lines, fmt.Sprintf("Name:             %s", dv.detail.Name))
	}

	// Category
	if dv.editMode == "category" {
		lines = append(lines, dv.renderEditField("Category", dv.editValue))
	} else {
		category := dv.detail.Category
		if category == "" {
			category = "(none)"
		}
		lines = append(lines, fmt.Sprintf("Category:         %s", category))
	}

	// Tags (if any)
	if dv.editMode == "tags" {
		lines = append(lines, dv.renderEditField("Tags", dv.editValue))
	} else {
		tags := "(none)"
		if len(dv.detail.Tags) > 0 {
			tags = strings.Join(dv.detail.Tags, ", ")
		}
		lines = append(lines, fmt.Sprintf("Tags:             %s", tags))
	}

	// Comments
	if dv.editMode == "comments" {
		lines = append(lines, dv.renderEditField("Comments", dv.editValue))
	} else {
		comments := dv.detail.Comments
		if comments == "" {
			comments = "(none)"
		}
		lines = append(lines, fmt.Sprintf("Comments:         %s", comments))
	}

	// Save path
	if dv.editMode == "location" {
		lines = append(lines, dv.renderEditField("Download Path", dv.editValue))
		if dv.editError != "" {
			lines = append(lines, fmt.Sprintf("  Error: %s", dv.editError))
		}
	} else {
		lines = append(lines, fmt.Sprintf("Download Path:    %s", dv.detail.SavePath))
	}

	// Size info
	totalStr := formatBytes(dv.detail.TotalSize)
	downloadedStr := formatBytes(dv.detail.Downloaded)
	progress := 0
	if dv.detail.TotalSize > 0 {
		progress = int((float64(dv.detail.Downloaded) / float64(dv.detail.TotalSize)) * 100)
	}
	lines = append(lines, fmt.Sprintf("Size:             %s / %s (%d%%)", downloadedStr, totalStr, progress))

	// Instructions/hints
	lines = append(lines, "")
	if dv.editMode != "" {
		lines = append(lines, "  [Enter] save  [Esc] cancel  |  editing: "+dv.editMode)
	} else {
		lines = append(lines, "  [e] edit field  [↑↓] navigate  [→] expand  [←←] collapse  []] expand all")
	}

	return strings.Join(lines, "\n")
}

// renderEditField renders a field in edit mode
func (dv *DetailView) renderEditField(label, value string) string {
	// Pad label to consistent width
	padding := 18 - len(label) - 1 // -1 for the colon
	padStr := strings.Repeat(" ", padding)
	
	// Build the cursor line
	cursorLine := value
	if dv.editCursorX < len(value) {
		// Cursor is within the text
		cursorLine = value[:dv.editCursorX] + "|" + value[dv.editCursorX:]
	} else {
		// Cursor is at the end
		cursorLine = value + "|"
	}

	// Add visual indicator that this field is being edited
	result := fmt.Sprintf("%s:%s%s", label, padStr, cursorLine)
	return "▸ " + result // Triangle indicator shows this is active
}

// renderFilesSection renders the file tree
func (dv *DetailView) renderFilesSection(width, maxLines int) string {
	var lines []string

	lines = append(lines, "Files:")

	if len(dv.fileTree) == 0 {
		lines = append(lines, "  (no files)")
		return strings.Join(lines, "\n")
	}

	// Flatten the tree to a list of visible nodes
	flattened := dv.flattenTree()

	// Apply scroll offset
	startIdx := dv.scrollPos
	endIdx := startIdx + maxLines - 1
	if endIdx >= len(flattened) {
		endIdx = len(flattened)
	}

	// Render visible nodes with cursor indicator
	for i := startIdx; i < endIdx && i < len(flattened); i++ {
		fnode := flattened[i]
		line := dv.renderTreeNode(fnode, width-4)
		
		// Add cursor indicator for current position
		if i == dv.cursorPos {
			lines = append(lines, "> "+line)
		} else {
			lines = append(lines, "  "+line)
		}
	}

	// Show count if there are more items
	if len(flattened) > endIdx {
		remaining := len(flattened) - endIdx
		lines = append(lines, fmt.Sprintf("  ... (%d more items)", remaining))
	}

	return strings.Join(lines, "\n")
}

// flattenTree converts the tree to a flat list of visible nodes (respecting expanded state)
func (dv *DetailView) flattenTree() []*FlattenedNode {
	var result []*FlattenedNode

	for _, rootNode := range dv.fileTree {
		dv.flattenNode(rootNode, 0, nil, &result)
	}

	return result
}

// flattenNode recursively flattens a node and its children
func (dv *DetailView) flattenNode(node *FileTreeNode, depth int, siblings []bool, result *[]*FlattenedNode) {
	isLast := false
	if node.Parent != nil {
		isLast = len(node.Parent.Children) > 0 && node.Parent.Children[len(node.Parent.Children)-1] == node
	}

	fnode := &FlattenedNode{
		Node:     node,
		Depth:    depth,
		IsLast:   isLast,
		Siblings: append([]bool{}, siblings...),
	}

	*result = append(*result, fnode)

	// Recursively add children if expanded
	if node.IsDirectory && node.Expanded {
		for _, child := range node.Children {
			newSiblings := append(siblings, isLast)
			dv.flattenNode(child, depth+1, newSiblings, result)
		}
	}
}

// renderTreeNode renders a single node with tree formatting
func (dv *DetailView) renderTreeNode(fnode *FlattenedNode, width int) string {
	node := fnode.Node

	// Build tree prefix (├─, └─, │, etc.)
	var prefix string
	for i := 0; i < fnode.Depth; i++ {
		if i < len(fnode.Siblings) && fnode.Siblings[i] {
			prefix += "  "
		} else {
			prefix += "│ "
		}
	}

	// Add the branch character
	if fnode.Depth > 0 {
		if fnode.IsLast {
			prefix += "└─"
		} else {
			prefix += "├─"
		}
	}

	// Directory/file indicator and expand marker
	var indicator string
	if node.IsDirectory {
		if node.Expanded {
			indicator = "[-]"
		} else {
			indicator = "[+]"
		}
	} else {
		// File priority indicator
		if node.Priority != 0 {
			indicator = "[x]"
		} else {
			indicator = "[ ]"
		}
	}

	// Size and progress
	sizeStr := formatBytes(node.Size)
	downloaded := node.Downloaded
	progress := 0
	if node.Size > 0 {
		progress = int((float64(downloaded) / float64(node.Size)) * 100)
	}

	// Truncate name if needed
	maxNameWidth := width - len(prefix) - len(indicator) - len(sizeStr) - 10
	name := node.Name
	if len(name) > maxNameWidth {
		name = name[:maxNameWidth-1] + "…"
	}

	// Build the line
	// Format: [+] name                                   12.3 MB [75%]
	line := fmt.Sprintf("%s%s %-*s %8s [%3d%%]", prefix, indicator, maxNameWidth, name, sizeStr, progress)
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

// MoveCursor moves the cursor up or down in the flattened tree
func (dv *DetailView) MoveCursor(delta int) {
	flattened := dv.flattenTree()

	newPos := dv.cursorPos + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= len(flattened) {
		newPos = len(flattened) - 1
	}
	dv.cursorPos = newPos

	// Keep cursor visible in viewport
	dv.ensureCursorVisible(len(flattened))
}

// ToggleExpanded toggles the expanded state of the current node if it's a directory
func (dv *DetailView) ToggleExpanded() {
	flattened := dv.flattenTree()
	if dv.cursorPos >= 0 && dv.cursorPos < len(flattened) {
		node := flattened[dv.cursorPos].Node
		if node.IsDirectory {
			node.Expanded = !node.Expanded
		}
	}
}

// ExpandAll expands all directories
func (dv *DetailView) ExpandAll() {
	for _, root := range dv.fileTree {
		dv.expandAllRecursive(root)
	}
}

// expandAllRecursive recursively expands all directories
func (dv *DetailView) expandAllRecursive(node *FileTreeNode) {
	if node.IsDirectory {
		node.Expanded = true
		for _, child := range node.Children {
			dv.expandAllRecursive(child)
		}
	}
}

// CollapseAll collapses all directories
func (dv *DetailView) CollapseAll() {
	for _, root := range dv.fileTree {
		dv.collapseAllRecursive(root)
	}
}

// collapseAllRecursive recursively collapses all directories
func (dv *DetailView) collapseAllRecursive(node *FileTreeNode) {
	if node.IsDirectory {
		node.Expanded = false
		for _, child := range node.Children {
			dv.collapseAllRecursive(child)
		}
	}
}

// ensureCursorVisible adjusts scroll position to keep cursor visible
func (dv *DetailView) ensureCursorVisible(treeSize int) {
	// Simple scrolling: keep cursor roughly centered if possible
	viewportHeight := 10 // Estimated viewport height for files section
	
	if dv.cursorPos < dv.scrollPos {
		// Cursor moved above viewport
		dv.scrollPos = dv.cursorPos
	} else if dv.cursorPos >= dv.scrollPos+viewportHeight {
		// Cursor moved below viewport
		dv.scrollPos = dv.cursorPos - viewportHeight + 1
	}

	// Bounds check
	if dv.scrollPos < 0 {
		dv.scrollPos = 0
	}
	if dv.scrollPos > treeSize-1 {
		dv.scrollPos = treeSize - 1
	}
}

// GetCurrentFile returns the currently selected file (or nil if directory is selected)
func (dv *DetailView) GetCurrentFile() *client.TorrentFile {
	flattened := dv.flattenTree()
	if dv.cursorPos >= 0 && dv.cursorPos < len(flattened) {
		node := flattened[dv.cursorPos].Node
		if !node.IsDirectory && node.Index >= 0 && node.Index < len(dv.files) {
			return &dv.files[node.Index]
		}
	}
	return nil
}

// EditDefaultField starts editing the appropriate field based on what makes sense
func (dv *DetailView) EditDefaultField() {
	// For now, always edit name as the primary field
	// In future, could be context-aware based on cursor position
	dv.StartEdit("name")
}

// StartEdit starts editing a specific field
func (dv *DetailView) StartEdit(field string) {
	dv.editMode = field
	dv.editError = ""
	dv.editCursorX = 0

	// Initialize edit value based on field
	switch field {
	case "name":
		dv.editValue = dv.detail.Name
		dv.editCursorX = len(dv.editValue)
	case "category":
		dv.editValue = dv.detail.Category
		dv.editCursorX = len(dv.editValue)
	case "tags":
		dv.editValue = strings.Join(dv.detail.Tags, ", ")
		dv.editCursorX = len(dv.editValue)
	case "comments":
		dv.editValue = dv.detail.Comments
		dv.editCursorX = len(dv.editValue)
	case "location":
		dv.editValue = dv.detail.SavePath
		dv.editCursorX = len(dv.editValue)
	}
}

// CancelEdit cancels the current edit
func (dv *DetailView) CancelEdit() {
	dv.editMode = ""
	dv.editValue = ""
	dv.editError = ""
	dv.editCursorX = 0
}

// HandleEditKey handles keyboard input while in edit mode
func (dv *DetailView) HandleEditKey(key string) bool {
	if dv.editMode == "" {
		return false
	}

	switch key {
	case "enter":
		// Save edit (will be handled by app)
		return true

	case "esc":
		dv.CancelEdit()
		return true

	case "backspace":
		if dv.editCursorX > 0 {
			dv.editValue = dv.editValue[:dv.editCursorX-1] + dv.editValue[dv.editCursorX:]
			dv.editCursorX--
		}
		return true

	case "delete":
		if dv.editCursorX < len(dv.editValue) {
			dv.editValue = dv.editValue[:dv.editCursorX] + dv.editValue[dv.editCursorX+1:]
		}
		return true

	case "home":
		dv.editCursorX = 0
		return true

	case "end":
		dv.editCursorX = len(dv.editValue)
		return true

	case "left":
		if dv.editCursorX > 0 {
			dv.editCursorX--
		}
		return true

	case "right":
		if dv.editCursorX < len(dv.editValue) {
			dv.editCursorX++
		}
		return true

	default:
		// Handle regular character input
		if len(key) == 1 && key >= " " && key <= "~" {
			dv.editValue = dv.editValue[:dv.editCursorX] + key + dv.editValue[dv.editCursorX:]
			dv.editCursorX++
			return true
		}
	}

	return false
}
