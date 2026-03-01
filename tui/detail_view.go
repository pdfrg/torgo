package tui

import (
	"fmt"
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
	currentTab string // "info", "edit", "category", "files"
	tabOrder   []string

	// Torrent data
	detail     *client.TorrentDetail
	files      []client.TorrentFile
	categories []string

	// Shared state
	state *DetailViewState

	// Tab components
	infoTab     *InfoTabModel
	editTab     *EditTabModel
	categoryTab *CategoryTabModel
	filesTab    *FilesTabModel

	// Error message
	lastError string
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
		currentTab: "info",
		tabOrder:   []string{"info", "edit", "category", "files"},
		detail:     detail,
		files:      files,
		categories: categories,
		state:      state,
		infoTab:    infoTab,
		editTab:    editTab,
		categoryTab: categoryTab,
		filesTab:   filesTab,
	}
}

// Init initializes the detail view
func (dv *DetailView) Init() tea.Cmd {
	return tea.Batch(
		dv.infoTab.Init(),
		dv.editTab.Init(),
		dv.categoryTab.Init(),
		dv.filesTab.Init(),
	)
}

// Update handles messages
func (dv *DetailView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		dv.state.Width = msg.Width
		dv.state.Height = msg.Height
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
			if dv.currentTab == "info" {
				dv.currentTab = "edit"
			}
		case "esc":
			// Discard changes and return to info, or close if on info
			if dv.currentTab != "info" {
				dv.currentTab = "info"
			} else if dv.state.HasChanges {
				dv.state.DiscardChanges()
			}
			// Note: Closing detail view will be handled by app.go
		}
	}

	// Delegate to current tab
	switch dv.currentTab {
	case "info":
		return dv.infoTab.Update(msg, dv.state)
	case "edit":
		return dv.editTab.Update(msg, dv.state)
	case "category":
		return dv.categoryTab.Update(msg, dv.state)
	case "files":
		return dv.filesTab.Update(msg, dv.state)
	}

	return nil
}

// nextTab moves to the next tab
func (dv *DetailView) nextTab() {
	currentIndex := 0
	for i, tab := range dv.tabOrder {
		if tab == dv.currentTab {
			currentIndex = i
			break
		}
	}
	dv.currentTab = dv.tabOrder[(currentIndex+1)%len(dv.tabOrder)]
}

// prevTab moves to the previous tab
func (dv *DetailView) prevTab() {
	currentIndex := 0
	for i, tab := range dv.tabOrder {
		if tab == dv.currentTab {
			currentIndex = i
			break
		}
	}
	dv.currentTab = dv.tabOrder[(currentIndex-1+len(dv.tabOrder))%len(dv.tabOrder)]
}

// View renders the detail view
func (dv *DetailView) View() string {
	// Render tabs
	tabs := dv.renderTabs()

	// Get content from active tab
	var content string
	switch dv.currentTab {
	case "info":
		content = dv.infoTab.View(dv.state)
	case "edit":
		content = dv.editTab.View(dv.state)
	case "category":
		content = dv.categoryTab.View(dv.state)
	case "files":
		content = dv.filesTab.View(dv.state)
	default:
		content = "Unknown tab"
	}

	// Combine tabs and content
	return tabs + "\n" + content
}

// renderTabs renders the tab bar with active tab highlighted
func (dv *DetailView) renderTabs() string {
	tabLabels := map[string]string{
		"info":     "Info",
		"edit":     "Edit",
		"category": "Category",
		"files":    "Files",
	}

	// Build top line and middle line separately
	var topLine, middleLine strings.Builder
	totalWidth := 0

	for i, tabName := range dv.tabOrder {
		label := tabLabels[tabName]
		isActive := tabName == dv.currentTab

		paddedLabel := " " + label + " "
		tabWidth := len(paddedLabel) + 2 // +2 for the box borders
		topBorder := "┌" + strings.Repeat("─", len(paddedLabel)) + "┐"
		midBorder := "│" + paddedLabel + "│"

		if isActive {
			// Active tab in select color
			activeStyle := lipgloss.NewStyle().Foreground(dv.state.Styles.SelectColor)
			topLine.WriteString(activeStyle.Render(topBorder))
			middleLine.WriteString(activeStyle.Render("│") + activeStyle.Render(paddedLabel) + activeStyle.Render("│"))
		} else {
			// Inactive tab in hint color
			inactiveStyle := lipgloss.NewStyle().Foreground(dv.state.Styles.HintColor)
			topLine.WriteString(inactiveStyle.Render(topBorder))
			middleLine.WriteString(inactiveStyle.Render(midBorder))
		}

		totalWidth += tabWidth

		// Add spacing between tabs
		if i < len(dv.tabOrder)-1 {
			topLine.WriteString(" ")
			middleLine.WriteString(" ")
			totalWidth += 1
		}
	}

	// Bottom border - continuous line
	bottomBorder := strings.Repeat("─", totalWidth)

	return topLine.String() + "\n" + middleLine.String() + "\n" + bottomBorder
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

func (m *InfoTabModel) View(state *DetailViewState) string {
	if m.detail == nil {
		return "No torrent selected"
	}

	var content strings.Builder
	content.WriteString("\n")
	content.WriteString("Name: " + m.detail.Name + "\n")
	content.WriteString("Category: " + m.detail.Category + "\n")
	content.WriteString("Save Path: " + m.detail.SavePath + "\n")
	content.WriteString("Total Size: " + formatBytes(m.detail.TotalSize) + "\n")
	content.WriteString("Downloaded: " + formatBytes(m.detail.Downloaded) + "\n")
	
	if state.HasChanges {
		content.WriteString("\n[Changes detected - press Enter to save, Esc to discard]\n")
	}

	return content.String()
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return string(rune(b)) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	switch exp {
	case 1:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(unit))
	case 2:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(unit*unit))
	case 3:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(unit*unit*unit))
	default:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(unit*unit*unit*unit))
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
		case "tab":
			// Next field
			m.fields[m.focusIndex].Input.Blur()
			m.focusIndex = (m.focusIndex + 1) % len(m.fields)
			m.fields[m.focusIndex].Input.Focus()
			return nil
		case "shift+tab":
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

	// Update state with current value
	for _, field := range m.fields {
		state.UpdateField(field.Name, field.Input.Value())
	}

	return cmd
}

func (m *EditTabModel) View(state *DetailViewState) string {
	var content strings.Builder
	content.WriteString("\n")
	
	for i, field := range m.fields {
		content.WriteString(field.Label + ": " + field.Input.View() + "\n")
		if i < len(m.fields)-1 {
			content.WriteString("\n")
		}
	}
	
	content.WriteString("\n[Use Tab/Shift+Tab to navigate fields]\n")
	
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

	l := list.New(items, delegate, 0, 6)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Title = "Category"

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
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// Update state with selected category
	if item := m.list.SelectedItem(); item != nil {
		catValue := string(item.(simpleItem))
		if catValue != "(none)" {
			state.UpdateField("category", catValue)
		} else {
			state.UpdateField("category", "")
		}
	}

	return cmd
}

func (m *CategoryTabModel) View(state *DetailViewState) string {
	// TODO: Implement category tab view
	return "Category Tab - TODO\n" + m.list.View()
}

// ==============================================================================
// FILES TAB
// ==============================================================================

type FilesTabModel struct {
	files []client.TorrentFile
	// TODO: Add tree-bubble component here
}

func NewFilesTabModel(state *DetailViewState, files []client.TorrentFile) *FilesTabModel {
	return &FilesTabModel{
		files: files,
	}
}

func (m *FilesTabModel) Init() tea.Cmd {
	return nil
}

func (m *FilesTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	// TODO: Implement files tab update with tree-bubble
	return nil
}

func (m *FilesTabModel) View(state *DetailViewState) string {
	// TODO: Implement files tab view with tree-bubble
	return "Files Tab - TODO"
}
