package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tqbtui/client"
)

// DetailViewState holds shared state across all tabs
type DetailViewState struct {
	// Original values from when detail view opened
	OriginalValues map[string]string

	// Currently edited values
	CurrentValues map[string]string

	// Whether any value differs from original
	HasChanges bool

	// Styles
	Styles *Styles

	// Dimensions
	Width  int
	Height int
}

// GetCurrentValue returns the current or original value for a field
func (s *DetailViewState) GetCurrentValue(field string) string {
	if val, ok := s.CurrentValues[field]; ok && val != "" {
		return val
	}
	if val, ok := s.OriginalValues[field]; ok {
		return val
	}
	return ""
}

// UpdateField updates a field value and recalculates HasChanges
func (s *DetailViewState) UpdateField(field, value string) {
	s.CurrentValues[field] = value
	s.recalculateChanges()
}

// recalculateChanges checks if any field differs from original
func (s *DetailViewState) recalculateChanges() {
	s.HasChanges = false
	for field := range s.OriginalValues {
		if s.GetCurrentValue(field) != s.OriginalValues[field] {
			s.HasChanges = true
			break
		}
	}
}

// DiscardChanges resets all edited values
func (s *DetailViewState) DiscardChanges() {
	s.CurrentValues = make(map[string]string)
	s.HasChanges = false
}

// DetailView is the main coordinator for the detail view
type DetailView struct {
	// Tab management - manual cycling
	CurrentTab string // "info", "edit", "category", "files"
	TabOrder   []string

	// Torrent data
	Detail     *client.TorrentDetail
	Files      []client.TorrentFile
	Categories []string

	// Shared state
	State *DetailViewState

	// Tab components
	InfoTab     *InfoTabModel
	EditTab     *EditTabModel
	CategoryTab *CategoryTabModel
	FilesTab    *FilesTabModel

	// Error message
	LastError string
}

// NewDetailView creates a new detail view
func NewDetailView(styles *Styles, detail *client.TorrentDetail, files []client.TorrentFile, categories []string) *DetailView {
	// Initialize shared state
	originalValues := make(map[string]string)
	originalValues["name"] = detail.Name
	originalValues["tags"] = ""
	if len(detail.Tags) > 0 {
		originalValues["tags"] = strings.Join(detail.Tags, ", ")
	}
	originalValues["comments"] = detail.Comments
	originalValues["location"] = detail.SavePath
	originalValues["category"] = detail.Category

	state := &DetailViewState{
		OriginalValues: originalValues,
		CurrentValues:  make(map[string]string),
		HasChanges:     false,
		Styles:         styles,
	}

	// Create tab models
	infoTab := NewInfoTabModel(state, detail)
	editTab := NewEditTabModel(state)
	categoryTab := NewCategoryTabModel(state, categories, detail.Category)
	filesTab := NewFilesTabModel(state, files)

	return &DetailView{
		CurrentTab: "info",
		TabOrder:   []string{"info", "edit", "category", "files"},
		Detail:     detail,
		Files:      files,
		Categories: categories,
		State:      state,
		InfoTab:    infoTab,
		EditTab:    editTab,
		CategoryTab: categoryTab,
		FilesTab:   filesTab,
	}
}

// Init initializes the detail view
func (dv *DetailView) Init() tea.Cmd {
	return tea.Batch(
		dv.InfoTab.Init(),
		dv.EditTab.Init(),
		dv.CategoryTab.Init(),
		dv.FilesTab.Init(),
	)
}

// Update handles messages
func (dv *DetailView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		dv.State.Width = msg.Width
		dv.State.Height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Move to next tab
			dv.nextTab()
		case "shift+tab":
			// Move to previous tab
			dv.prevTab()
		case "e":
			// Quick jump to edit tab from info
			if dv.CurrentTab == "info" {
				dv.CurrentTab = "edit"
			}
		case "esc":
			// Discard changes and return to info, or close if on info
			if dv.CurrentTab != "info" {
				dv.CurrentTab = "info"
			} else if dv.State.HasChanges {
				dv.State.DiscardChanges()
			}
			// Note: Closing detail view will be handled by app.go
		}
	}

	// Delegate to current tab
	switch dv.CurrentTab {
	case "info":
		return dv.InfoTab.Update(msg, dv.State)
	case "edit":
		return dv.EditTab.Update(msg, dv.State)
	case "category":
		return dv.CategoryTab.Update(msg, dv.State)
	case "files":
		return dv.FilesTab.Update(msg, dv.State)
	}

	return nil
}

// nextTab moves to the next tab
func (dv *DetailView) nextTab() {
	currentIndex := 0
	for i, tab := range dv.TabOrder {
		if tab == dv.CurrentTab {
			currentIndex = i
			break
		}
	}
	dv.CurrentTab = dv.TabOrder[(currentIndex+1)%len(dv.TabOrder)]
}

// prevTab moves to the previous tab
func (dv *DetailView) prevTab() {
	currentIndex := 0
	for i, tab := range dv.TabOrder {
		if tab == dv.CurrentTab {
			currentIndex = i
			break
		}
	}
	dv.CurrentTab = dv.TabOrder[(currentIndex-1+len(dv.TabOrder))%len(dv.TabOrder)]
}

// View renders the detail view
func (dv *DetailView) View() string {
	// Render tabs
	tabs := dv.renderTabs()

	// Get content from active tab
	var content string
	switch dv.CurrentTab {
	case "info":
		content = dv.InfoTab.View(dv.State, dv.FilesTab)
	case "edit":
		content = dv.EditTab.View(dv.State)
	case "category":
		content = dv.CategoryTab.View(dv.State)
	case "files":
		content = dv.FilesTab.View(dv.State)
	default:
		content = "Unknown tab"
	}

	// Combine tabs and content
	return tabs + "\n" + content
}

// tabBorderWithBottom creates a custom tab border with specified bottom characters
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

// renderTabs renders the tab bar with active tab highlighted using lipgloss borders
func (dv *DetailView) renderTabs() string {
	tabLabels := map[string]string{
		"info":     "Info",
		"edit":     "Edit",
		"category": "Category",
		"files":    "Files",
	}

	// Create tab border styles
	inactiveTabBorder := tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder := tabBorderWithBottom("┘", " ", "└")

	var renderedTabs []string

	for i, tabName := range dv.TabOrder {
		label := tabLabels[tabName]
		isActive := tabName == dv.CurrentTab
		isFirst := i == 0
		isLast := i == len(dv.TabOrder)-1

		// Choose base style
		var tabStyle lipgloss.Style
		if isActive {
			tabStyle = lipgloss.NewStyle().
				Border(activeTabBorder, true).
				BorderForeground(dv.State.Styles.SelectColor).
				Padding(0, 1)
		} else {
			tabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(dv.State.Styles.HintColor).
				Padding(0, 1)
		}

		// Adjust corners for first and last tabs
		border, _, _, _, _ := tabStyle.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		}
		if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		tabStyle = tabStyle.Border(border)

		renderedTabs = append(renderedTabs, tabStyle.Render(label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// ==============================================================================
// INFO TAB
// ==============================================================================

type InfoTabModel struct {
	detail *client.TorrentDetail
}

func NewInfoTabModel(state *DetailViewState, detail *client.TorrentDetail) *InfoTabModel {
	return &InfoTabModel{
		detail: detail,
	}
}

func (m *InfoTabModel) Init() tea.Cmd {
	return nil
}

func (m *InfoTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	// Info tab is read-only, just handle tab navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if state.HasChanges {
				// Trigger save - will be handled by app.go
				return nil
			}
		}
	}
	return nil
}

func (m *InfoTabModel) View(state *DetailViewState, filesTab *FilesTabModel) string {
	if m.detail == nil {
		return "No torrent selected"
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(state.Styles.SelectColor).
		Bold(true)

	var content strings.Builder
	content.WriteString("\n")
	
	// Name (always shows current edited value if changed)
	content.WriteString(labelStyle.Render("Name:") + " ")
	if name := state.GetCurrentValue("name"); name != m.detail.Name {
		content.WriteString(m.detail.Name + " → " + name)
	} else {
		content.WriteString(m.detail.Name)
	}
	content.WriteString("\n\n")
	
	// Category
	content.WriteString(labelStyle.Render("Category:") + " ")
	if category := state.GetCurrentValue("category"); category != m.detail.Category {
		content.WriteString(m.detail.Category + " → " + category)
	} else {
		content.WriteString(m.detail.Category)
		if m.detail.Category == "" {
			content.WriteString("(none)")
		}
	}
	content.WriteString("\n\n")
	
	// Tags
	content.WriteString(labelStyle.Render("Tags:") + " ")
	originalTags := state.OriginalValues["tags"]
	if tags := state.GetCurrentValue("tags"); tags != originalTags {
		content.WriteString(originalTags + " → " + tags)
	} else {
		if originalTags == "" {
			content.WriteString("(none)")
		} else {
			content.WriteString(originalTags)
		}
	}
	content.WriteString("\n\n")
	
	// Comments
	content.WriteString(labelStyle.Render("Comments:") + " ")
	if m.detail.Comments == "" {
		content.WriteString("(none)")
	} else {
		content.WriteString(m.detail.Comments)
	}
	content.WriteString("\n\n")
	
	// Size info and progress
	content.WriteString(labelStyle.Render("Total Size:") + " " + formatBytes(m.detail.TotalSize) + "\n")
	
	// Calculate and show progress percentage
	var progress int64
	if m.detail.TotalSize > 0 {
		progress = (m.detail.Downloaded * 100) / m.detail.TotalSize
	}
	downloaded := formatBytes(m.detail.Downloaded)
	remaining := formatBytes(m.detail.TotalSize - m.detail.Downloaded)
	
	content.WriteString(labelStyle.Render("Progress:") + " " + fmt.Sprintf("%d%%", progress) + " (" + downloaded + " / " + formatBytes(m.detail.TotalSize) + ")\n")
	content.WriteString(labelStyle.Render("Remaining:") + " " + remaining + "\n")
	
	// Save Path
	content.WriteString("\n" + labelStyle.Render("Save Path:") + "\n")
	if location := state.GetCurrentValue("location"); location != m.detail.SavePath {
		content.WriteString("  " + m.detail.SavePath + "\n")
		content.WriteString("  → " + location + "\n")
	} else {
		content.WriteString("  " + m.detail.SavePath + "\n")
	}
	
	// Check for any changes (including file priority changes)
	hasFieldChanges := state.HasChanges
	hasFileChanges := filesTab != nil && filesTab.HasFileChanges()
	hasAnyChanges := hasFieldChanges || hasFileChanges
	
	// Change indicator
	if hasAnyChanges {
		var changeMsg string
		if hasFileChanges && !hasFieldChanges {
			changeMsg = "✎ File priorities changed"
		} else if hasFieldChanges && !hasFileChanges {
			changeMsg = "✎ Changes detected"
		} else {
			changeMsg = "✎ Changes detected (fields and files)"
		}
		
		content.WriteString("\n" + lipgloss.NewStyle().
			Foreground(state.Styles.ErrorColor).
			Bold(true).
			Render(changeMsg) + " - press Enter to save or Esc to discard\n")
	} else {
		content.WriteString("\nPress 'e' to edit, Tab to switch tabs\n")
	}

	return content.String()
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	value := float64(b) / float64(div)
	
	switch exp {
	case 1:
		return fmt.Sprintf("%.1f KB", value)
	case 2:
		// Use integer format for MB if it's clean, otherwise .1f
		if value >= 100 {
			return fmt.Sprintf("%.0f MB", value)
		}
		return fmt.Sprintf("%.1f MB", value)
	case 3:
		// Use integer format for GB if it's clean
		if value >= 10 {
			return fmt.Sprintf("%.1f GB", value)
		}
		return fmt.Sprintf("%.2f GB", value)
	default:
		return fmt.Sprintf("%.2f TB", value)
	}
}

// ==============================================================================
// EDIT TAB
// ==============================================================================

type EditTabModel struct {
	fields      []EditFormField
	focusIndex  int
}

type EditFormField struct {
	Name     string
	Label    string
	Original string
	Input    textinput.Model
}

func NewEditTabModel(state *DetailViewState) *EditTabModel {
	fields := []EditFormField{
		{
			Name:     "name",
			Label:    "Name",
			Original: state.OriginalValues["name"],
			Input:    createStyledTextInput("name", state.OriginalValues["name"]),
		},
		{
			Name:     "tags",
			Label:    "Tags",
			Original: state.OriginalValues["tags"],
			Input:    createStyledTextInput("tags", state.OriginalValues["tags"]),
		},
		{
			Name:     "comments",
			Label:    "Comments",
			Original: state.OriginalValues["comments"],
			Input:    createStyledTextInput("comments", state.OriginalValues["comments"]),
		},
		{
			Name:     "location",
			Label:    "Download Path",
			Original: state.OriginalValues["location"],
			Input:    createStyledTextInput("location", state.OriginalValues["location"]),
		},
	}

	// Focus first input
	if len(fields) > 0 {
		fields[0].Input.Focus()
	}

	return &EditTabModel{
		fields:     fields,
		focusIndex: 0,
	}
}

func createStyledTextInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Placeholder = placeholder
	return ti
}

func (m *EditTabModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *EditTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			// Next field
			m.fields[m.focusIndex].Input.Blur()
			m.focusIndex = (m.focusIndex + 1) % len(m.fields)
			m.fields[m.focusIndex].Input.Focus()
			return nil
		case "shift+tab", "up":
			// Previous field
			m.fields[m.focusIndex].Input.Blur()
			m.focusIndex = (m.focusIndex - 1 + len(m.fields)) % len(m.fields)
			m.fields[m.focusIndex].Input.Focus()
			return nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.fields[m.focusIndex].Input, cmd = m.fields[m.focusIndex].Input.Update(msg)

	// Update state with current values from all fields
	for i, field := range m.fields {
		value := m.fields[i].Input.Value()
		state.UpdateField(field.Name, value)
	}

	return cmd
}

func (m *EditTabModel) View(state *DetailViewState) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(state.Styles.SelectColor).
		Bold(true)

	focusStyle := lipgloss.NewStyle().
		Foreground(state.Styles.BgColor).
		Background(state.Styles.SelectColor).
		Padding(0, 1)

	var content strings.Builder
	content.WriteString("\n")
	
	for i, field := range m.fields {
		isFocused := i == m.focusIndex
		
		// Field label
		label := labelStyle.Render(field.Label + ":")
		if isFocused {
			label = focusStyle.Render(" " + field.Label + " ")
		}
		
		content.WriteString(label + "\n")
		content.WriteString("  " + field.Input.View() + "\n")
		
		// Show original value as hint
		if field.Original != "" && field.Input.Value() != field.Original {
			hintStyle := lipgloss.NewStyle().Foreground(state.Styles.HintColor)
			content.WriteString("  " + hintStyle.Render("(originally: "+field.Original+")") + "\n")
		}
		
		if i < len(m.fields)-1 {
			content.WriteString("\n")
		}
	}
	
	content.WriteString("\n" + lipgloss.NewStyle().
		Foreground(state.Styles.HintColor).
		Render("↑/↓ or Tab/Shift+Tab to navigate  •  Enter to save  •  Esc to cancel\n"))
	
	return content.String()
}

// ==============================================================================
// CATEGORY TAB
// ==============================================================================

type CategoryTabModel struct {
	list list.Model
}

func NewCategoryTabModel(state *DetailViewState, categories []string, currentCategory string) *CategoryTabModel {
	items := []list.Item{}
	items = append(items, simpleItem("(none)"))
	for _, cat := range categories {
		items = append(items, simpleItem(cat))
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)

	// Width and height will be set dynamically in View based on available space
	// Height of 15 to show all items with room to spare
	l := list.New(items, delegate, 50, 15)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	// Select current category
	if currentCategory != "" {
		for i, item := range items {
			if string(item.(simpleItem)) == currentCategory {
				l.Select(i)
				break
			}
		}
	}

	return &CategoryTabModel{
		list: l,
	}
}

func (m *CategoryTabModel) Init() tea.Cmd {
	return nil
}

func (m *CategoryTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	// Handle list updates
	m.list, _ = m.list.Update(msg)

	// Update state with selected category whenever selection changes
	if item := m.list.SelectedItem(); item != nil {
		catValue := string(item.(simpleItem))
		if catValue != "(none)" {
			state.UpdateField("category", catValue)
		} else {
			state.UpdateField("category", "")
		}
	}

	return nil
}

func (m *CategoryTabModel) View(state *DetailViewState) string {
	var content strings.Builder
	content.WriteString("\n")
	
	labelStyle := lipgloss.NewStyle().
		Foreground(state.Styles.SelectColor).
		Bold(true)
	
	content.WriteString(labelStyle.Render("Select a category:") + "\n\n")
	
	// Get list view with proper dimensions
	listView := m.list.View()
	content.WriteString(listView)
	
	content.WriteString("\n" + lipgloss.NewStyle().
		Foreground(state.Styles.HintColor).
		Render("↑/↓ to navigate  •  Enter to select  •  Esc to cancel\n"))
	
	return content.String()
}

// ==============================================================================
// FILES TAB
// ==============================================================================

type FilesTabModel struct {
	files          []client.TorrentFile
	selectedFiles  map[int]bool // Track which file indices are selected for download
	cursorIndex    int           // Currently focused file (for navigation)
}

// FileNode represents a file or folder in the tree
type FileNode struct {
	Index    int            // Original file index from TorrentFile
	Name     string         // Display name (filename only, not full path)
	Size     int64          // File size
	IsFolder bool           // Whether this is a folder/directory
	Children []*FileNode    // Child nodes if folder
	Depth    int            // Indentation depth
}

func NewFilesTabModel(state *DetailViewState, files []client.TorrentFile) *FilesTabModel {
	// Initialize selectedFiles based on current priority status
	// Priority 0 = do not download, otherwise = download
	selectedFiles := make(map[int]bool)
	for _, f := range files {
		selectedFiles[f.Index] = f.Priority != 0
	}

	return &FilesTabModel{
		files:         files,
		selectedFiles: selectedFiles,
		cursorIndex:   0,
	}
}

func (m *FilesTabModel) Init() tea.Cmd {
	return nil
}

// GetSelectedFileIndices returns the list of file indices selected for download
func (m *FilesTabModel) GetSelectedFileIndices() []int {
	var selected []int
	for fileIdx, isSelected := range m.selectedFiles {
		if isSelected {
			selected = append(selected, fileIdx)
		}
	}
	// Sort for consistent ordering
	sort.Ints(selected)
	return selected
}

// HasFileChanges checks if file selections have changed from original state
func (m *FilesTabModel) HasFileChanges() bool {
	for _, f := range m.files {
		originalSelected := f.Priority != 0
		currentSelected := m.selectedFiles[f.Index]
		if originalSelected != currentSelected {
			return true
		}
	}
	return false
}

func (m *FilesTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down":
			// Move cursor down
			if m.cursorIndex < len(m.files)-1 {
				m.cursorIndex++
			}
		case "up":
			// Move cursor up
			if m.cursorIndex > 0 {
				m.cursorIndex--
			}
		case " ", "enter":
			// Toggle current file selection (space is represented as " ")
			if m.cursorIndex >= 0 && m.cursorIndex < len(m.files) {
				fileIdx := m.files[m.cursorIndex].Index
				m.selectedFiles[fileIdx] = !m.selectedFiles[fileIdx]
			}
		}
	}
	return nil
}

func (m *FilesTabModel) View(state *DetailViewState) string {
	var content strings.Builder
	content.WriteString("\n")
	
	labelStyle := lipgloss.NewStyle().
		Foreground(state.Styles.SelectColor).
		Bold(true)
	
	hintStyle := lipgloss.NewStyle().Foreground(state.Styles.HintColor)
	selectedStyle := lipgloss.NewStyle().Foreground(state.Styles.SelectColor)
	
	content.WriteString(labelStyle.Render("Files") + " (" + fmt.Sprintf("%d", len(m.files)) + " total)\n\n")
	
	if len(m.files) == 0 {
		content.WriteString("No files in this torrent\n")
	} else {
		// Show all files with checkboxes
		for i, f := range m.files {
			isSelected := m.selectedFiles[f.Index]
			isCursor := i == m.cursorIndex
			
			// Checkbox
			checkbox := "☐"
			if isSelected {
				checkbox = "☑"
			}
			
			// Build line with cursor indicator
			prefix := "  "
			if isCursor {
				prefix = "> "
			}
			
			fileName := f.Name
			fileSize := formatBytes(f.Size)
			
			line := fmt.Sprintf("%s%s %s (%s)\n", prefix, checkbox, fileName, fileSize)
			
			// Highlight current line
			if isCursor {
				line = selectedStyle.Render(line)
			}
			
			content.WriteString(line)
		}
	}
	
	content.WriteString("\n" + hintStyle.Render("↑/↓ to navigate  •  Space/Enter to toggle download  •  Tab to return to info\n"))
	
	return content.String()
}
