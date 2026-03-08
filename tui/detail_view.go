package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	AvailableHeight int // Height available for tab content (accounting for header/footer overhead)
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
	return NewDetailViewWithHost(styles, detail, files, categories, "localhost")
}

// NewDetailViewWithHost creates a new detail view with client host info
func NewDetailViewWithHost(styles *Styles, detail *client.TorrentDetail, files []client.TorrentFile, categories []string, clientHost string) *DetailView {
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
	editTab := NewEditTabModelWithHost(state, clientHost)
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
	case subdirectoriesMsg:
		// Pass subdirectory message to edit tab
		return dv.EditTab.Update(msg, dv.State)
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

// SetAvailableHeight sets the available height for tab content from app.go
// This is called before View() to pass the calculated height
func (dv *DetailView) SetAvailableHeight(height int) {
	dv.State.AvailableHeight = height
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

		// Choose base style - use theme colors
		// Note: Both active and inactive tabs use accent color for the bottom border line
		// to create visual continuity
		var tabStyle lipgloss.Style
		if isActive {
			tabStyle = lipgloss.NewStyle().
				Border(activeTabBorder, true).
				BorderForeground(CurrentTheme.DetailTabActiveBorder).
				Padding(0, 1)
		} else {
			tabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(CurrentTheme.DetailTabActiveBorder).  // Use accent for the bottom line too
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
		tabStyle = tabStyle.Border(border, true)

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
	// Info tab is read-only - all key handling is done by app.go
	// Don't consume any keys here
	return nil
}

func (m *InfoTabModel) View(state *DetailViewState, filesTab *FilesTabModel) string {
	if m.detail == nil {
		return "No torrent selected"
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(CurrentTheme.DetailCursorColor).
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
			Foreground(state.Styles.ErrorColor()).
			Bold(true).
			Render(changeMsg) + " - press Enter to save or Esc to discard\n")
	} else {
		hintStyle := lipgloss.NewStyle().Foreground(CurrentTheme.TextMuted)
		content.WriteString("\n" + hintStyle.Render("Press 'e' to edit, Tab to switch tabs\n"))
	}

	return content.String()
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	
	// Calculate exponent (1=KB, 2=MB, 3=GB, 4=TB, etc.)
	exp := 0
	div := int64(1)
	for bytes := b; bytes >= unit; bytes /= unit {
		div *= unit
		exp++
		if exp >= 4 {
			// Cap at TB
			break
		}
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
	fields                []EditFormField
	focusIndex            int
	subdirHelper          *SubdirectoryHelper
	currentSubdirectories []string
	selectedSubdirIndex   int
	showSubdirectories    bool
	lastLocationValue     string // Track location field changes for debouncing
	debounceTimer         *time.Timer
	pendingPath           string // Path being fetched
}

// subdirectoriesMsg is sent when subdirectories have been fetched
type subdirectoriesMsg struct {
	path          string
	subdirectories []string
}

type EditFormField struct {
	Name     string
	Label    string
	Original string
	Input    textinput.Model
}

func NewEditTabModel(state *DetailViewState) *EditTabModel {
	return NewEditTabModelWithHost(state, "localhost")
}

func NewEditTabModelWithHost(state *DetailViewState, clientHost string) *EditTabModel {
	fields := []EditFormField{
		{
			Name:     "name",
			Label:    "Name",
			Original: state.OriginalValues["name"],
			Input:    createStyledTextInput("enter torrent name", state.OriginalValues["name"]),
		},
		{
			Name:     "tags",
			Label:    "Tags",
			Original: state.OriginalValues["tags"],
			Input:    createStyledTextInput("<no tag>", state.OriginalValues["tags"]),
		},
		{
			Name:     "comments",
			Label:    "Comments",
			Original: state.OriginalValues["comments"],
			Input:    createStyledTextInput("<no comment>", state.OriginalValues["comments"]),
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

	// Only enable subdirectory helper for local clients
	var subdirHelper *SubdirectoryHelper
	if IsLocalhost(clientHost) {
		subdirHelper = NewSubdirectoryHelper()
	}

	return &EditTabModel{
		fields:                fields,
		focusIndex:            0,
		subdirHelper:          subdirHelper,
		currentSubdirectories: []string{},
		selectedSubdirIndex:   0,
		showSubdirectories:    false,
		lastLocationValue:     fields[3].Input.Value(), // location is at index 3
		debounceTimer:         nil,
		pendingPath:           "",
	}
}

func createStyledTextInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Placeholder = placeholder
	ti.SetWidth(60)  // Set width to accommodate placeholder text like "<no comment>"
	
	// Apply theme-aware styles to text input
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(CurrentTheme.ForegroundColor),
			Placeholder: lipgloss.NewStyle().Foreground(CurrentTheme.TextMuted),
			Suggestion:  lipgloss.NewStyle().Foreground(CurrentTheme.TextMuted),
			Prompt:      lipgloss.NewStyle().Foreground(CurrentTheme.DetailCursorColor),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(CurrentTheme.TextNormal),
			Placeholder: lipgloss.NewStyle().Foreground(CurrentTheme.TextMuted),
			Suggestion:  lipgloss.NewStyle().Foreground(CurrentTheme.TextMuted),
			Prompt:      lipgloss.NewStyle().Foreground(CurrentTheme.DetailCursorColor),
		},
		Cursor: textinput.CursorStyle{
			Color: CurrentTheme.DetailCursorColor,
		},
	}
	ti.SetStyles(styles)
	
	return ti
}

func (m *EditTabModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *EditTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	// Handle subdirectories fetched message
	if subMsg, ok := msg.(subdirectoriesMsg); ok {
		// Only update if this is for the path we're currently editing
		if subMsg.path == m.pendingPath {
			m.currentSubdirectories = subMsg.subdirectories
			m.showSubdirectories = len(subMsg.subdirectories) > 0
			m.selectedSubdirIndex = 0
			m.pendingPath = ""
		}
		return nil
	}

	// Handle subdirectory selection if list is shown
	if m.showSubdirectories {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "down":
				if m.selectedSubdirIndex < len(m.currentSubdirectories)-1 {
					m.selectedSubdirIndex++
				}
				return nil
			case "up":
				if m.selectedSubdirIndex > 0 {
					m.selectedSubdirIndex--
				}
				return nil
			case "enter":
				// Select the subdirectory
				if m.selectedSubdirIndex < len(m.currentSubdirectories) {
					subdir := m.currentSubdirectories[m.selectedSubdirIndex]
					basePath := GetBasePathForListing(m.lastLocationValue)
					newPath := basePath + "/" + subdir
					m.fields[3].Input.SetValue(newPath)
					m.showSubdirectories = false
					m.currentSubdirectories = []string{}
					m.selectedSubdirIndex = 0
					state.UpdateField("location", newPath)
				}
				return nil
			case "esc":
				// Close subdirectory list
				m.showSubdirectories = false
				m.currentSubdirectories = []string{}
				m.selectedSubdirIndex = 0
				return nil
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if !m.showSubdirectories {
				// Next field
				m.fields[m.focusIndex].Input.Blur()
				m.focusIndex = (m.focusIndex + 1) % len(m.fields)
				m.fields[m.focusIndex].Input.Focus()
				m.showSubdirectories = false
				m.currentSubdirectories = []string{}
			}
			return nil
		case "shift+tab", "up":
			if !m.showSubdirectories {
				// Previous field
				m.fields[m.focusIndex].Input.Blur()
				m.focusIndex = (m.focusIndex - 1 + len(m.fields)) % len(m.fields)
				m.fields[m.focusIndex].Input.Focus()
				m.showSubdirectories = false
				m.currentSubdirectories = []string{}
			}
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

	// Handle subdirectory debouncing for location field (only if subdirHelper is enabled)
	if m.subdirHelper != nil && m.focusIndex == 3 { // location field
		currentValue := m.fields[3].Input.Value()
		if currentValue != m.lastLocationValue {
			m.lastLocationValue = currentValue

			// Check if path ends with slash
			if IsPathWithTrailingSlash(currentValue) {
				// Set pending path and return command to fetch subdirectories
				m.pendingPath = currentValue
				return m.fetchSubdirectories(currentValue)
			} else {
				// No trailing slash, hide subdirectories
				m.showSubdirectories = false
				m.currentSubdirectories = []string{}
			}
		}
	}

	return cmd
}

// fetchSubdirectories returns a command that fetches subdirectories asynchronously
func (m *EditTabModel) fetchSubdirectories(path string) tea.Cmd {
	return func() tea.Msg {
		// Wait for debounce
		time.Sleep(250 * time.Millisecond)
		
		// Fetch subdirectories
		subdirs, _ := m.subdirHelper.FetchSubdirectories(path)
		
		return subdirectoriesMsg{
			path:           path,
			subdirectories: subdirs,
		}
	}
}

func (m *EditTabModel) View(state *DetailViewState) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(CurrentTheme.DetailCursorColor).
		Bold(true)

	hintStyle := lipgloss.NewStyle().Foreground(state.Styles.HintColor())
	selectedStyle := lipgloss.NewStyle().Foreground(state.Styles.SelectColor())

	var content strings.Builder
	content.WriteString("\n")
	
	for i, field := range m.fields {
		isFocused := i == m.focusIndex
		
		// Field label - always use consistent styling, no special focus appearance
		label := labelStyle.Render(field.Label + ":")
		
		content.WriteString(label + "\n")
		content.WriteString("  " + field.Input.View() + "\n")
		
		// Show original value as hint
		if field.Original != "" && field.Input.Value() != field.Original {
			content.WriteString("  " + hintStyle.Render("(originally: "+field.Original+")") + "\n")
		}
		
		// Show subdirectory list for location field if active
		if isFocused && i == 3 && m.showSubdirectories && len(m.currentSubdirectories) > 0 {
			content.WriteString("\n  Available subdirectories:\n")
			for j, subdir := range m.currentSubdirectories {
				isSelected := j == m.selectedSubdirIndex
				prefix := "    • "
				if isSelected {
					prefix = "  > "
					content.WriteString(selectedStyle.Render(prefix + subdir) + "\n")
				} else {
					content.WriteString(prefix + subdir + "\n")
				}
			}
			content.WriteString("  " + hintStyle.Render("(↑/↓ to select, Enter to choose, Esc to close)") + "\n")
		}
		
		if i < len(m.fields)-1 {
			content.WriteString("\n")
		}
	}
	
	hint := "↑/↓ or Tab/Shift+Tab to navigate  •  Enter to save  •  Esc to cancel"
	if m.showSubdirectories && len(m.currentSubdirectories) > 0 {
		hint = "↑/↓ to select subdirectory  •  Enter to choose  •  Esc to close"
	}
	content.WriteString("\n" + hintStyle.Render(hint) + "\n")
	
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
	
	// Customize list item styles - keep defaults, only change selected color
	styles := list.NewDefaultItemStyles(true)  // dark theme defaults
	// Override selected title and its left border ("|") to foreground color
	selectedStyle := styles.SelectedTitle
	selectedStyle = selectedStyle.
		Foreground(CurrentTheme.ForegroundColor).
		BorderLeftForeground(CurrentTheme.ForegroundColor)
	styles.SelectedTitle = selectedStyle
	
	delegate.Styles = styles

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
		Foreground(CurrentTheme.DetailCursorColor).
		Bold(true)
	
	content.WriteString(labelStyle.Render("Select a category:") + "\n\n")
	
	// Get list view with proper dimensions
	listView := m.list.View()
	content.WriteString(listView)
	
	content.WriteString("\n" + lipgloss.NewStyle().
		Foreground(state.Styles.HintColor()).
		Render("↑/↓ to navigate  •  Enter to select  •  Esc to cancel\n"))
	
	return content.String()
}

// ==============================================================================
// FILES TAB
// ==============================================================================

type FilesTabModel struct {
	files          []client.TorrentFile
	selectedFiles  map[int]bool // Track which file indices are selected for download
	expandedFolders map[string]bool // Track which folder paths are expanded
	cursorIndex    int           // Currently focused item in flat tree view
	flatTree       []*TreeItem   // Flattened tree view for navigation
	viewportStart  int           // First visible item index
	viewportHeight int           // Number of lines available for display
}

// FileNode represents a file or folder in the tree
type FileNode struct {
	Index      int            // Original file index from TorrentFile (-1 for folders)
	Name       string         // Display name (filename only, not full path)
	Size       int64          // File size in bytes
	Downloaded int64          // Bytes downloaded (0 for folders, calculated as sum of children)
	IsFolder   bool           // Whether this is a folder/directory
	Children   []*FileNode    // Child nodes if folder
	Depth      int            // Indentation depth
	Path       string         // Full path for folder (e.g. "dir/subdir")
}

// TreeItem represents an item visible in the flattened tree
type TreeItem struct {
	Node     *FileNode
	IsExpanded bool
}

func NewFilesTabModel(state *DetailViewState, files []client.TorrentFile) *FilesTabModel {
	// Initialize selectedFiles based on current priority status
	// Priority 0 = do not download, otherwise = download
	selectedFiles := make(map[int]bool)
	for _, f := range files {
		selectedFiles[f.Index] = f.Priority != 0
	}

	model := &FilesTabModel{
		files:           files,
		selectedFiles:   selectedFiles,
		expandedFolders: make(map[string]bool),
		cursorIndex:     0,
	}
	
	// Build and flatten the tree
	model.rebuildTree()
	
	// Auto-expand single top-level folder if there's only one and it has <= 20 files
	model.autoExpandSingleFolder()
	
	return model
}

// autoExpandSingleFolder automatically expands a single top-level folder if it has <= 20 files
func (m *FilesTabModel) autoExpandSingleFolder() {
	if len(m.files) == 0 {
		return
	}
	
	// Count top-level folders in the file tree
	// We need to check the first level of the tree
	root := m.buildFileTree()
	
	// Count folders and files at root level
	folderCount := 0
	var singleFolder *FileNode
	for _, child := range root.Children {
		if child.IsFolder {
			folderCount++
			singleFolder = child
		}
	}
	
	// If there's exactly one top-level folder with <= 20 files, auto-expand it
	if folderCount == 1 && singleFolder != nil && countFilesInFolder(singleFolder) <= 20 {
		m.expandedFolders[singleFolder.Path] = true
		m.rebuildTree() // Rebuild to apply the expansion
	}
}

// countFilesInFolder recursively counts all files in a folder
func countFilesInFolder(node *FileNode) int {
	if !node.IsFolder {
		return 1
	}
	
	count := 0
	for _, child := range node.Children {
		count += countFilesInFolder(child)
	}
	return count
}

// rebuildTree reconstructs the file tree and flattens it for rendering
func (m *FilesTabModel) rebuildTree() {
	if len(m.files) == 0 {
		m.flatTree = []*TreeItem{}
		return
	}
	
	// Build tree structure
	root := m.buildFileTree()
	
	// Calculate progress for all folders
	root.calculateFolderProgress()
	
	// Flatten tree for navigation
	m.flatTree = []*TreeItem{}
	m.flattenTree(root, &m.flatTree)
}

// buildFileTree constructs a hierarchical tree from flat file list
func (m *FilesTabModel) buildFileTree() *FileNode {
	root := &FileNode{
		Index:    -1,
		Name:     "root",
		IsFolder: true,
		Children: []*FileNode{},
		Depth:    0,
		Path:     "",
	}
	
	// Insert each file into tree
	for _, file := range m.files {
		parts := strings.Split(file.Name, "/")
		
		// Navigate/create path to file
		current := root
		var pathParts []string
		
		for i, part := range parts {
			if part == "" {
				continue
			}
			
			pathParts = append(pathParts, part)
			isLastPart := i == len(parts)-1
			
			if !isLastPart {
				// This is a directory, find or create it
				var found *FileNode
				for _, child := range current.Children {
					if child.Name == part && child.IsFolder {
						found = child
						break
					}
				}
				if found == nil {
					found = &FileNode{
						Index:    -1,
						Name:     part,
						IsFolder: true,
						Children: []*FileNode{},
						Depth:    current.Depth + 1,
						Path:     strings.Join(pathParts, "/"),
					}
					current.Children = append(current.Children, found)
				}
				current = found
			} else {
				// This is a file
				fileNode := &FileNode{
					Index:      file.Index,
					Name:       part,
					Size:       file.Size,
					Downloaded: file.Downloaded,
					IsFolder:   false,
					Children:   []*FileNode{},
					Depth:      current.Depth + 1,
					Path:       strings.Join(pathParts, "/"),
				}
				current.Children = append(current.Children, fileNode)
			}
		}
	}
	
	return root
}

// getProgressPercent returns progress as a percentage (0-100)
func (n *FileNode) getProgressPercent() int {
	if n.Size == 0 {
		return 0
	}
	percent := (n.Downloaded * 100) / n.Size
	if percent > 100 {
		percent = 100
	}
	return int(percent)
}

// calculateFolderProgress calculates total size and downloaded for a folder and all children
func (n *FileNode) calculateFolderProgress() (totalSize, totalDownloaded int64) {
	if !n.IsFolder {
		return n.Size, n.Downloaded
	}
	
	for _, child := range n.Children {
		childSize, childDownloaded := child.calculateFolderProgress()
		totalSize += childSize
		totalDownloaded += childDownloaded
	}
	
	// Store the totals back in the folder node for display
	n.Size = totalSize
	n.Downloaded = totalDownloaded
	
	return totalSize, totalDownloaded
}

// flattenTree flattens the tree structure into a list for rendering
func (m *FilesTabModel) flattenTree(node *FileNode, result *[]*TreeItem) {
	if node.Index == -1 && node.Path == "" {
		// Root node, just process children
		for _, child := range node.Children {
			m.flattenTree(child, result)
		}
		return
	}
	
	// Add current node
	expanded := m.expandedFolders[node.Path]
	*result = append(*result, &TreeItem{
		Node:       node,
		IsExpanded: expanded,
	})
	
	// Add children if expanded (or if it's a file)
	if !node.IsFolder || expanded {
		for _, child := range node.Children {
			m.flattenTree(child, result)
		}
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

// toggleFolder toggles a folder's expansion state and all its files
func (m *FilesTabModel) toggleFolder(folderPath string, node *FileNode) {
	expanded := m.expandedFolders[folderPath]
	m.expandedFolders[folderPath] = !expanded
	
	// If collapsing, deselect all files in folder
	// If expanding, don't change selection (user may want to toggle files individually)
	if expanded {
		m.deselectFolder(node)
	}
	
	// Rebuild the flattened tree
	m.rebuildTree()
}

// deselectFolder recursively deselects all files in a folder
func (m *FilesTabModel) deselectFolder(node *FileNode) {
	for _, child := range node.Children {
		if child.IsFolder {
			m.deselectFolder(child)
		} else {
			m.selectedFiles[child.Index] = false
		}
	}
}

// selectFolder recursively selects all files in a folder
func (m *FilesTabModel) selectFolder(node *FileNode, selected bool) {
	for _, child := range node.Children {
		if child.IsFolder {
			m.selectFolder(child, selected)
		} else {
			m.selectedFiles[child.Index] = selected
		}
	}
}

// updateViewport adjusts viewport to ensure cursor is visible
func (m *FilesTabModel) updateViewport(availableHeight int) {
	m.viewportHeight = availableHeight
	
	// Ensure cursor is visible in viewport
	if m.cursorIndex < m.viewportStart {
		// Cursor moved above viewport
		m.viewportStart = m.cursorIndex
	} else if m.cursorIndex >= m.viewportStart+m.viewportHeight {
		// Cursor moved below viewport
		m.viewportStart = m.cursorIndex - m.viewportHeight + 1
	}
	
	// Ensure viewport doesn't show past the end
	if m.viewportStart+m.viewportHeight > len(m.flatTree) {
		m.viewportStart = len(m.flatTree) - m.viewportHeight
	}
	
	// Never show negative viewport
	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

func (m *FilesTabModel) Update(msg tea.Msg, state *DetailViewState) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down":
			// Move cursor down
			if m.cursorIndex < len(m.flatTree)-1 {
				m.cursorIndex++
			}
		case "up":
			// Move cursor up
			if m.cursorIndex > 0 {
				m.cursorIndex--
			}
		case "right", "l":
			// Expand folder at cursor
			if m.cursorIndex >= 0 && m.cursorIndex < len(m.flatTree) {
				item := m.flatTree[m.cursorIndex]
				if item.Node.IsFolder && !item.IsExpanded {
					m.expandedFolders[item.Node.Path] = true
					m.rebuildTree()
				}
			}
		case "left", "h":
			// Collapse folder at cursor
			if m.cursorIndex >= 0 && m.cursorIndex < len(m.flatTree) {
				item := m.flatTree[m.cursorIndex]
				if item.Node.IsFolder && item.IsExpanded {
					m.expandedFolders[item.Node.Path] = false
					m.rebuildTree()
				}
			}
		case "space", " ":
			// Toggle current item (space only - enter is reserved for saving changes)
			if m.cursorIndex >= 0 && m.cursorIndex < len(m.flatTree) {
				item := m.flatTree[m.cursorIndex]
				if item.Node.IsFolder {
					// Toggle folder expansion
					m.toggleFolder(item.Node.Path, item.Node)
				} else {
					// Toggle file selection
					m.selectedFiles[item.Node.Index] = !m.selectedFiles[item.Node.Index]
					// Note: HasChanges tracking is handled separately by HasFileChanges()
				}
			}
		}
	}
	return nil
}

func (m *FilesTabModel) View(state *DetailViewState) string {
	var content strings.Builder
	content.WriteString("\n")
	
	labelStyle := lipgloss.NewStyle().
		Foreground(CurrentTheme.DetailCursorColor).
		Bold(true)
	
	hintStyle := lipgloss.NewStyle().Foreground(state.Styles.HintColor())
	selectedStyle := lipgloss.NewStyle().Foreground(CurrentTheme.DetailCursorColor)
	
	content.WriteString(labelStyle.Render("Files") + " (" + fmt.Sprintf("%d", len(m.files)) + " total)\n\n")
	
	hasPositionIndicator := false
	if len(m.files) == 0 {
		content.WriteString("No files in this torrent\n")
	} else if len(m.flatTree) == 0 {
		content.WriteString("No files to display\n")
	} else {
		// Use available height from state (calculated by DetailView)
		availableHeight := state.AvailableHeight
		if availableHeight < 3 {
			availableHeight = 3
		}
		
		// Account for overhead:
		// - Initial blank line + "Files (N total)" label + blank = 3 lines
		// - Position indicator (when more files than viewport): blank + position text = 2 lines
		// - Main hints section: blank + hints text = 2 lines
		// - So file viewport gets: availableHeight - 3 (header) - 2 (position) - 2 (hints)
		fileViewportHeight := availableHeight - 7
		if fileViewportHeight < 3 {
			fileViewportHeight = 3
		}
		
		// Update viewport to show cursor
		m.updateViewport(fileViewportHeight)
		
		// Render only visible portion of tree
		for i := m.viewportStart; i < m.viewportStart+m.viewportHeight && i < len(m.flatTree); i++ {
			item := m.flatTree[i]
			isCursor := i == m.cursorIndex
			node := item.Node
			
			// Indentation
			indent := strings.Repeat("  ", node.Depth)
			
			var line string
			if node.IsFolder {
				// Folder display
				chevron := "▸"
				if item.IsExpanded {
					chevron = "▼"
				}
				fileCount := len(node.Children)
				folderSize := formatBytes(node.Size)
				progress := node.getProgressPercent()
				line = fmt.Sprintf("%s%s %s 📁 (%d items, %s, %d%%)", indent, chevron, node.Name, fileCount, folderSize, progress)
			} else {
				// File display with checkbox
				isSelected := m.selectedFiles[node.Index]
				checkbox := "☐"
				if isSelected {
					checkbox = "☑"
				}
				fileSize := formatBytes(node.Size)
				progress := node.getProgressPercent()
				line = fmt.Sprintf("%s%s %s (%s, %d%%)", indent, checkbox, node.Name, fileSize, progress)
			}
			
			// Add cursor indicator
			prefix := "  "
			if isCursor {
				prefix = "> "
			}
			
			// Apply styling to the content (not including newline)
			if isCursor {
				line = selectedStyle.Render(prefix + line)
			} else {
				line = prefix + line
			}
			
			content.WriteString(line + "\n")
		}
		
		// Show position in list if there are more items than viewport
		if len(m.flatTree) > m.viewportHeight {
			end := m.viewportStart + m.viewportHeight
			if end > len(m.flatTree) {
				end = len(m.flatTree)
			}
			positionStr := fmt.Sprintf(" (%d-%d of %d)", m.viewportStart+1, end, len(m.flatTree))
			content.WriteString("\n" + hintStyle.Render(positionStr) + "\n")
			hasPositionIndicator = true
		}
	}
	
	// Main hints - don't add leading blank if position indicator already present
	if hasPositionIndicator {
		content.WriteString(hintStyle.Render("↑/↓ to navigate  •  ←/→ to collapse/expand  •  Space to toggle  •  Tab to return\n"))
	} else {
		content.WriteString("\n" + hintStyle.Render("↑/↓ to navigate  •  ←/→ to collapse/expand  •  Space to toggle  •  Tab to return\n"))
	}
	
	return content.String()
}
