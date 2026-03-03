package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"tqbtui/client"
	"tqbtui/config"
	"tqbtui/state"
)

// simpleItem is a minimal list item for categories
type simpleItem string

func (i simpleItem) FilterValue() string { return string(i) }
func (i simpleItem) Title() string       { return string(i) }
func (i simpleItem) Description() string { return "" }

// App is the main TUI application model
type App struct {
	state              *state.AppState
	styles             *Styles
	keys               KeyMap
	list               *TorrentListView
	multilineList      *MultilineTorrentListView  // Multiline view alternative
	detailView         *DetailView         // Detail view for selected torrent
	statusBar          *StatusBar
	hintsBar           *HintsBar
	width              int
	height             int
	showHints          bool
	showHelp           bool
	currentTheme       string        // "dark", "light", or "highcontrast"
	viewMode           string        // "default" or "multiline"
	screenMode         string        // "list" or "detail"
	inputMode          string        // "", "add", "search"
	torrentInput       textinput.Model
	categoryList       list.Model
	lastError          string
	inputValidationErr string        // validation error for the add dialog
	searchInput        textinput.Model
	searchFilter       *SearchFilter
	searchMode         bool          // true if in search mode
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewApp creates a new TUI application
func NewApp(appState *state.AppState) *App {
	ctx, cancel := context.WithCancel(context.Background())

	styles := DefaultStyles()
	keys := DefaultKeyMap()

	// Load the theme from config
	var themeToUse Theme
	var themeName string
	
	if appState.Config.Theme != nil && appState.Config.Theme.Colors != nil && len(appState.Config.Theme.Colors) > 0 {
		// Build omarchy theme from config colors
		themeToUse = OmarchyTheme(appState.Config.Theme.Colors)
		themeName = "omarchy"
	} else {
		// Use default dark theme
		themeToUse = DefaultTheme()
		themeName = "dark"
	}
	
	// Set the theme globally
	SetTheme(themeToUse)

	// Initialize text inputs
	ti := textinput.New()
	ti.Placeholder = "Magnet link, URL, or .torrent file path"
	ti.CharLimit = 1024

	si := textinput.New()
	si.Placeholder = "Search torrents..."
	si.CharLimit = 256

	// Initialize search filter
	searchFilter := NewSearchFilter()

	// Initialize category list with compact delegate
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	categoryList := list.New([]list.Item{}, delegate, 0, 6)
	categoryList.SetShowHelp(false)
	categoryList.SetShowStatusBar(false)
	categoryList.SetShowTitle(true)
	categoryList.Title = "Category"

	hintsBar := NewHintsBar(styles, keys)
	hintsBar.SetCurrentTheme(themeName)

	app := &App{
		state:          appState,
		styles:         styles,
		keys:           keys,
		list:           NewTorrentListView(styles),
		multilineList:  NewMultilineTorrentListView(styles),
		detailView:     nil,
		statusBar:      NewStatusBar(styles),
		hintsBar:       hintsBar,
		showHints:      appState.Config.UI.ShowHints,
		showHelp:       false,
		currentTheme:   themeName,
		viewMode:       "default",
		screenMode:     "list",
		torrentInput:   ti,
		categoryList:   categoryList,
		searchInput:    si,
		searchFilter:   searchFilter,
		searchMode:     false,
		ctx:            ctx,
		cancel:         cancel,
	}

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.connectAndRefresh(),
		a.startRefreshTicker(),
		a.startSpeedLimitTicker(),
		a.refreshSpeedLimitStatus(),
	)
}

// startRefreshTicker periodically refreshes torrents every 2 seconds
func (a *App) startRefreshTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// startSpeedLimitTicker periodically checks speed limit status every 12 seconds
func (a *App) startSpeedLimitTicker() tea.Cmd {
	return tea.Tick(12*time.Second, func(t time.Time) tea.Msg {
		return speedLimitTickMsg{}
	})
}

// tickMsg is used for torrent refresh (every 2 seconds)
type tickMsg struct{}

// speedLimitTickMsg is used for speed limit polling (every 12 seconds)
type speedLimitTickMsg struct{}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle detail view first if active
	if a.screenMode == "detail" && a.detailView != nil {
		// Update detail view and handle its commands
		_ = a.detailView.Update(msg)
		
		// Check for specific keys that should be handled by app
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			k := keyMsg.String()
			
			// ESC exits detail view (unless already handled by detail view)
			if k == "esc" && !a.detailView.State.HasChanges {
				a.screenMode = "list"
				a.detailView = nil
				// Restart refresh tickers when returning to list view
				return a, tea.Batch(
					a.startRefreshTicker(),
					a.startSpeedLimitTicker(),
				)
			}
			
			// Enter saves changes (only from Info tab with changes)
			if k == "enter" && a.detailView.CurrentTab == "info" && a.detailView.State.HasChanges {
				// Save changes via API
				return a, a.saveTorrentChanges()
			}
		}
		
		// Still in detail view, but pass through other messages (like ticks)
		// so they can keep the ui responsive
		return a, nil
	}

	// Handle search mode separately (only in list view)
	if a.searchMode {
		return a.handleSearchMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.detailView != nil {
			a.detailView.State.Width = msg.Width
			a.detailView.State.Height = msg.Height
		}
		return a, nil
	case torrentRefreshMsg:
		a.list.SetTorrents(a.state.FilteredTorrents())
		a.multilineList.SetTorrents(a.state.FilteredTorrents())
		return a, nil
	case errorMsg:
		a.lastError = msg.Error()
		return a, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return errorClearedMsg{}
		})
	case errorClearedMsg:
		a.lastError = ""
		return a, nil
	case speedLimitToggledMsg:
		// Speed limit toggled, UI will update on next render
		return a, nil
	case tickMsg:
		// Refresh torrents and re-schedule the ticker
		return a, tea.Batch(
			a.refreshTorrents(),
			a.startRefreshTicker(),
		)
	case speedLimitTickMsg:
		// Check speed limit status periodically to catch external changes (scheduler, etc.)
		return a, tea.Batch(
			a.refreshSpeedLimitStatus(),
			a.startSpeedLimitTicker(),
		)
	case savesCompleteMsg:
		// Changes saved successfully - refresh torrent list and restart tickers
		return a, tea.Batch(
			a.refreshTorrents(),
			a.startRefreshTicker(),
			a.startSpeedLimitTicker(),
		)
	case detailViewOpenedMsg:
		// Detail view opened, just trigger a redraw by returning nil command
		return a, nil
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() tea.View {
	if a.width == 0 || a.height == 0 {
		return tea.NewView("Loading...")
	}

	// Calculate heights for layout
	// Fixed overhead: title(1) + blank(1) = 2 lines (status and hints go at bottom separately)
	headerHeight := 2
	
	// Search mode height (search status + blank line)
	searchHeight := 0
	if a.searchMode || (a.searchFilter != nil && a.searchFilter.IsActive()) {
		searchHeight = 2
	}

	// Error height (blank + error)
	errorHeight := 0
	if a.lastError != "" {
		errorHeight = 2
	}

	// Status bar and hints bar at bottom
	// Note: actual output includes blank line before status, then status, then optional hints
	bottomHeight := 2 // blank + status bar
	if a.showHints {
		bottomHeight = 3 // + hints bar
	}

	// List height = total - header - search - error - bottom
	listHeight := a.height - headerHeight - searchHeight - errorHeight - bottomHeight
	if listHeight < 3 {
		listHeight = 3
	}

	// Build output
	lines := []string{}

	// Title
	titleText := "󰁇  󰁇  tqbtui – Torrent Multi-Client TUI"
	lines = append(lines, a.styles.Title.Render(titleText))
	lines = append(lines, "")

	// Show search status if in search mode (editing) or if search is active (results shown)
	if a.searchMode || (a.searchFilter != nil && a.searchFilter.IsActive()) {
		// Use search filter's query for display (not input field, which might be empty)
		query := ""
		matches := 0
		if a.searchFilter != nil {
			query = a.searchFilter.GetQuery()
			matches = a.searchFilter.GetMatchCount()
		}
		
		// Create match count text with proper pluralization
		matchText := "match"
		if matches != 1 {
			matchText = "matches"
		}
		
		// Different visual style depending on mode
		var searchStatus string
		var searchStatusStyle lipgloss.Style
		
		if a.searchMode {
			// Actively editing search - bright purple with instruction
			// Use input field value since we're typing
			inputQuery := a.searchInput.Value()
			searchStatus = fmt.Sprintf(" 🔍 SEARCH: %s  (%d %s)  [Enter to confirm, ESC to clear] ",
				inputQuery, matches, matchText)
			searchStatusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).  // Purple background
				Foreground(lipgloss.Color("255")). // White text
				Bold(true).
				Padding(0, 1)
		} else {
			// Search results active - subtle grey with instruction
			searchStatus = fmt.Sprintf(" 🔍 %s (%d %s) — / to edit, ESC to clear ",
				query, matches, matchText)
			searchStatusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).  // Darker grey background
				Foreground(lipgloss.Color("255")). // White text
				Padding(0, 1)
		}
		
		// Ensure it renders to full width
		lines = append(lines, searchStatusStyle.Width(a.width).Render(searchStatus))
		lines = append(lines, "")
	}

	// Show either list or detail view based on screenMode
	var mainView string
	if a.screenMode == "detail" && a.detailView != nil {
		// In detail view - use full available height (accounting for status/hints at bottom)
		mainView = a.detailView.View()
	} else {
		// In list view
		if a.viewMode == "multiline" {
			mainView = a.renderMultilineViewPlaceholder(a.width, listHeight)
		} else {
			mainView = a.list.Render(a.width, listHeight)
		}
	}
	lines = append(lines, mainView)
	
	// Add spacing to push status/hints to bottom when in detail view
	if a.screenMode == "detail" && a.detailView != nil {
		// Calculate remaining height and fill with blank lines
		currentHeight := len(lines)
		requiredHeight := a.height - bottomHeight
		if currentHeight < requiredHeight {
			for i := 0; i < requiredHeight-currentHeight; i++ {
				lines = append(lines, "")
			}
		}
	}

	// Error message if present
	if a.lastError != "" {
		lines = append(lines, "")
		lines = append(lines, a.styles.ListItem.Foreground(a.styles.ErrorColor()).
			Render("Error: "+a.lastError))
	}

	// Status bar and hints bar at bottom
	statusBarOutput := a.statusBar.Render(a.state, a.width)
	hintsBarOutput := ""
	if a.showHints {
		hintsBarOutput = a.hintsBar.Render(a.width)
	}

	// Build final output line by line to ensure title is at top
	finalLines := lines
	
	// Add status and hints at the end
	finalLines = append(finalLines, "")
	finalLines = append(finalLines, statusBarOutput)
	if a.showHints {
		finalLines = append(finalLines, hintsBarOutput)
	}

	// Join all lines
	output := strings.Join(finalLines, "\n")
	
	// Ensure output doesn't exceed terminal height
	// We need to keep title at top and status/hints at bottom
	outputLines := strings.Split(output, "\n")
	
	if len(outputLines) > a.height {
		// Need to trim: keep title (2 lines) + status/hints (1-3 lines) + trim middle intelligently
		// Count lines we need at bottom: blank + status + hints
		bottomLinesNeeded := 2 // blank + status
		if a.showHints {
			bottomLinesNeeded += 1 // + hints
		}
		
		// Keep title (2 lines) + middle content + bottom
		maxMiddleLines := a.height - 2 - bottomLinesNeeded
		if maxMiddleLines < 1 {
			maxMiddleLines = 1
		}
		
		// Middle content is everything between line 2 and the last bottomLinesNeeded lines
		middleStart := 2
		middleEnd := len(outputLines) - bottomLinesNeeded
		
		if middleEnd <= middleStart {
			// Not enough space, show title and status only
			outputLines = append(outputLines[:2], outputLines[len(outputLines)-bottomLinesNeeded:]...)
		} else {
			// Trim middle if needed
			middleLines := outputLines[middleStart:middleEnd]
			if len(middleLines) > maxMiddleLines {
				// For the list portion (which includes header/separator at top), preserve the top lines
				// Search status is at the beginning of middle, list viewport with headers follows
				// We want to keep the list headers visible, so trim from the bottom of the list
				middleLines = middleLines[:maxMiddleLines] // Keep first lines to preserve headers
			}
			outputLines = append(outputLines[:2], append(middleLines, outputLines[middleEnd:]...)...)
		}
	} else if len(outputLines) < a.height {
		// Pad with blank lines to fill terminal
		padding := a.height - len(outputLines)
		outputLines = append(outputLines, make([]string, padding)...)
	}
	
	output = strings.Join(outputLines, "\n")

	// Overlay modals
	if a.showHelp {
		output = a.overlayHelpDialog(output)
	}

	if a.inputMode == "add" {
		output = a.overlayAddDialog(output)
	}

	v := tea.NewView(output)
	v.AltScreen = true
	return v
}

// overlayAddDialog renders the add torrent dialog as a centered modal overlay on top
func (a *App) overlayAddDialog(baseOutput string) string {
	// Dimensions
	dialogWidth := 80
	dialogHeight := 10 // actual height depends on a.categoryList.SetHeight(13) ...
	// larger values can make dialog box taller, but smaller values are overridden
	if a.width < 80 {
		dialogWidth = a.width - 4
	}
	if dialogWidth < 40 {
		dialogWidth = 40
	}

	// Build dialog content
	title := "Add Torrent"
	hint := "(Ctrl+P to paste, Esc to cancel)"

	// Render torrent input - set width for v2 textinput rendering
	a.torrentInput.SetWidth(dialogWidth - 6) // Account for padding and borders
	inputSection := a.torrentInput.View()

	// Build content lines with validation error if present
	contentLines := []string{title, "", inputSection}

	// Add validation error in red if present
	if a.inputValidationErr != "" {
		errorText := a.styles.ListItem.Foreground(a.styles.ErrorColor()).Render("✗ " + a.inputValidationErr)
		contentLines = append(contentLines, errorText)
	}

	// Render category list
	categoryView := ""
	if len(a.state.Categories) > 0 {
		a.categoryList.SetWidth(dialogWidth - 4)
		a.categoryList.SetHeight(13) // This is not items, I think it's lines. 13 is min to see 5 items.
		categoryView = a.categoryList.View()
	}

	// Assemble dialog content
	if categoryView != "" {
		contentLines = append(contentLines, "", categoryView)
	}
	contentLines = append(contentLines, "", hint)

	content := strings.Join(contentLines, "\n")
	
	// Render dialog box with fixed dimensions using lipgloss
	dialogBox := a.styles.Dialog.
		Width(dialogWidth - 2). // Account for padding
		Height(dialogHeight - 2). // Account for padding
		Padding(1, 2).
		Render(content)

	// Use lipgloss.Place to center the popup properly
	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		dialogBox,
	)
}

// handleKeyPress handles keyboard input
func (a *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In input mode, handle differently
	if a.inputMode != "" {
		return a.handleInputMode(msg)
	}

	k := msg.String()

	// Handle detail view specific keys
	if a.screenMode == "detail" && a.detailView != nil {
		// TODO: Implement proper key handling for new detail view architecture
		// For now, ESC exits detail view
		if k == "esc" {
			a.screenMode = "list"
			a.detailView = nil
			return a, nil
		}
		// Other keys are handled by detail view's Update method (called in main Update)
		return a, nil
	}

	// Handle ESC to clear search filter if active (only in list view)
	if k == "esc" && a.screenMode == "list" && a.searchFilter.IsActive() {
		a.searchMode = false
		a.searchInput.Blur()
		a.searchInput.Reset()
		a.searchFilter.Clear()
		a.list.SetTorrents(a.state.FilteredTorrents())
		return a, nil
	}

	switch {
	case isKeyMatch(k, a.keys.Quit):
		return a, tea.Quit

	case isKeyMatch(k, a.keys.Up):
		if a.viewMode == "multiline" {
			a.multilineList.MoveCursor(-1)
		} else {
			a.list.MoveCursor(-1)
		}
		return a, nil

	case isKeyMatch(k, a.keys.Down):
		if a.viewMode == "multiline" {
			a.multilineList.MoveCursor(1)
		} else {
			a.list.MoveCursor(1)
		}
		return a, nil

	case isKeyMatch(k, a.keys.PageUp):
		if a.viewMode == "multiline" {
			a.multilineList.PageMove(-1, a.height)
		} else {
			a.list.PageMove(-1, a.height)
		}
		return a, nil

	case isKeyMatch(k, a.keys.PageDown):
		if a.viewMode == "multiline" {
			a.multilineList.PageMove(1, a.height)
		} else {
			a.list.PageMove(1, a.height)
		}
		return a, nil

	case isKeyMatch(k, a.keys.Select):
		if a.viewMode == "multiline" {
			a.multilineList.ToggleSelection()
		} else {
			a.list.ToggleSelection()
		}
		return a, nil

	case isKeyMatch(k, a.keys.SelectAll):
		if a.viewMode == "multiline" {
			a.multilineList.SelectAll()
		} else {
			a.list.SelectAll()
		}
		return a, nil

	case isKeyMatch(k, a.keys.Pause):
		return a, a.pauseSelected()

	case isKeyMatch(k, a.keys.PauseAll):
		return a, a.pauseAll()

	case isKeyMatch(k, a.keys.Resume):
		return a, a.resumeSelected()

	case isKeyMatch(k, a.keys.ResumeAll):
		return a, a.resumeAll()

	case isKeyMatch(k, a.keys.Delete):
		return a, a.deleteSelected(false)

	case isKeyMatch(k, a.keys.DeleteData):
		return a, a.deleteSelected(true)

	case isKeyMatch(k, a.keys.Details):
		// Open detail view for current torrent
		if a.screenMode == "detail" {
			// Already in detail view, ignore
			return a, nil
		}
		var torrent *client.Torrent
		if a.viewMode == "multiline" {
			torrent = a.multilineList.GetCurrentTorrent()
		} else {
			torrent = a.list.GetCurrentTorrent()
		}
		if torrent != nil {
			return a, a.openTorrentDetail(torrent.ID)
		}
		return a, nil

	case isKeyMatch(k, a.keys.AddTorrent):
		a.inputMode = "add"
		a.torrentInput.Reset()
		a.torrentInput.Focus()
		return a, a.refreshCategories()

	case isKeyMatch(k, a.keys.SwitchClient):
		a.state.SwitchClient()
		a.list.ClearSelection()
		return a, tea.Batch(
			a.connectAndRefresh(),
			a.refreshSpeedLimitStatus(),
		)

	case isKeyMatch(k, a.keys.Sort):
		a.state.CycleSort()
		a.list.SetTorrents(a.state.FilteredTorrents())
		return a, nil

	case isKeyMatch(k, a.keys.Filter):
		a.state.CycleFilter()
		a.list.ClearSelection()
		a.list.SetTorrents(a.state.FilteredTorrents())
		return a, nil

	case isKeyMatch(k, a.keys.Search):
		if a.searchMode {
			// Already in search mode, toggle off
			a.searchMode = false
			a.searchInput.Blur()
		} else if a.searchFilter.IsActive() {
			// Have active search results, enter edit mode
			a.searchMode = true
			a.searchInput.Focus()
		} else {
			// No search, enter new search mode
			a.searchMode = true
			a.searchInput.Focus()
			a.searchInput.Reset()
		}
		return a, nil

	case isKeyMatch(k, a.keys.ToggleView):
		if a.viewMode == "default" {
			a.viewMode = "multiline"
			// Sync cursor and selection from single-line to multiline view
			a.multilineList.cursor = a.list.cursor
			a.multilineList.selected = a.list.selected
			// Also sync the torrent list
			a.multilineList.SetTorrents(a.state.FilteredTorrents())
		} else {
			a.viewMode = "default"
			// Sync cursor and selection from multiline to single-line view
			a.list.cursor = a.multilineList.cursor
			a.list.selected = a.multilineList.selected
		}
		return a, nil

	case isKeyMatch(k, a.keys.ToggleTheme):
		// Cycle through available themes
		// Get the list of available themes from config
		availableThemes := getAvailableThemesForCycling(a.state.Config)
		
		// Find current position in the list
		currentIndex := -1
		for i, theme := range availableThemes {
			if theme == a.currentTheme {
				currentIndex = i
				break
			}
		}
		
		// Move to next theme (or wrap around)
		nextIndex := (currentIndex + 1) % len(availableThemes)
		nextThemeName := availableThemes[nextIndex]
		
		// Load and set the theme
		nextTheme := loadThemeByName(nextThemeName, a.state.Config)
		SetTheme(nextTheme)
		a.currentTheme = nextThemeName
		
		// Update hints bar to show new theme
		a.hintsBar.SetCurrentTheme(nextThemeName)
		
		return a, nil

	case isKeyMatch(k, a.keys.Help):
		a.showHelp = !a.showHelp
		return a, nil

	case isKeyMatch(k, a.keys.ToggleHints):
		a.showHints = !a.showHints
		return a, nil

	case isKeyMatch(k, a.keys.ToggleSpeedLimit):
		return a, a.toggleSpeedLimit()
	}

	return a, nil
}

// isKeyMatch checks if a key string matches a keybinding (case-sensitive)
func isKeyMatch(k string, binding key.Binding) bool {
	for _, key := range binding.Keys() {
		if k == key {
			return true
		}
	}
	return false
}

// handleInputMode handles input when in special modes
func (a *App) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.inputMode = ""
		a.inputValidationErr = ""
		a.torrentInput.Blur()
		return a, nil
	case "enter":
		if a.inputMode == "add" {
			input := a.torrentInput.Value()
			
			// Validate before submitting
			validationErr := IsValidForSubmit(input)
			if validationErr != "" {
				a.inputValidationErr = validationErr
				return a, nil
			}
			
			// Get selected category from list
			var category string
			if a.categoryList.Index() >= 0 && len(a.state.Categories) > 0 {
				idx := a.categoryList.Index()
				if idx > 0 { // idx 0 is "None"
					category = a.state.Categories[idx-1]
				}
			}
			a.inputValidationErr = ""
			return a, a.addTorrent(input, category)
		}
		a.inputMode = ""
		a.inputValidationErr = ""
		a.torrentInput.Blur()
		return a, nil
	case "ctrl+c":
		a.inputMode = ""
		a.inputValidationErr = ""
		a.torrentInput.Blur()
		return a, nil
	case "ctrl+p":
		// Paste from clipboard
		if text, err := clipboard.ReadAll(); err == nil {
			// Remove trailing newlines that often come from clipboard
			newValue := a.torrentInput.Value() + strings.TrimSpace(text)
			a.torrentInput.SetValue(newValue)
			// Validate as user types
			a.inputValidationErr = ValidateInput(newValue)
		}
		return a, nil
	case "up", "down":
		// Route to category list
		var cmd tea.Cmd
		a.categoryList, cmd = a.categoryList.Update(msg)
		return a, cmd
	default:
		// Route to text input
		var cmd tea.Cmd
		a.torrentInput, cmd = a.torrentInput.Update(msg)
		// Validate as user types (except for backspace, which is always ok)
		if msg.String() != "backspace" {
			a.inputValidationErr = ValidateInput(a.torrentInput.Value())
		} else {
			// Still validate after backspace for when field becomes empty
			a.inputValidationErr = ValidateInput(a.torrentInput.Value())
		}
		return a, cmd
	}
}

// handleSearchMode handles keyboard input while in search mode
func (a *App) handleSearchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Exit search mode
			a.searchMode = false
			a.searchInput.Blur()
			a.searchInput.Reset()
			a.searchFilter.Clear()
			// Show all torrents again
			a.list.SetTorrents(a.state.FilteredTorrents())
			return a, nil

		case "enter":
			// Confirm search and exit search mode, keeping filtered results
			// User can now interact with filtered results using normal commands
			// Press / again to modify search, or ESC to clear
			a.searchMode = false
			a.searchInput.Blur()
			return a, nil

		case "ctrl+c":
			// Exit search mode
			a.searchMode = false
			a.searchInput.Blur()
			a.searchInput.Reset()
			a.searchFilter.Clear()
			a.list.SetTorrents(a.state.FilteredTorrents())
			return a, nil

		default:
			// Update search input and filter in real-time
			var cmd tea.Cmd
			a.searchInput, cmd = a.searchInput.Update(msg)
			query := a.searchInput.Value()
			a.searchFilter.SetQuery(query, a.state.FilteredTorrents())
			a.list.SetTorrents(a.searchFilter.GetResults())
			a.list.ClearSelection() // Clear selection when search changes
			return a, cmd
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	// Pass other message types through
	default:
		return a, nil
	}
}

// Command builders

func (a *App) refreshTorrents() tea.Cmd {
	return func() tea.Msg {
		if err := a.state.RefreshTorrents(a.ctx); err != nil {
			return errorMsg{err: err}
		}
		return torrentRefreshMsg{}
	}
}

func (a *App) connectAndRefresh() tea.Cmd {
	return func() tea.Msg {
		current := a.state.CurrentClient()
		if current == nil {
			return errorMsg{err: fmt.Errorf("no current client")}
		}

		// Disconnect from previous client if connected
		_ = current.Adapter.Disconnect(a.ctx)

		// Connect to new client
		if err := current.Adapter.Connect(a.ctx); err != nil {
			return errorMsg{err: fmt.Errorf("failed to connect: %v", err)}
		}

		// Refresh torrents
		if err := a.state.RefreshTorrents(a.ctx); err != nil {
			return errorMsg{err: err}
		}

		return torrentRefreshMsg{}
	}
}

func (a *App) pauseSelected() tea.Cmd {
	return func() tea.Msg {
		selected := a.list.GetSelected()
		if len(selected) == 0 {
			if current := a.list.GetCurrentTorrent(); current != nil {
				selected = []string{current.ID}
			}
		}

		for _, id := range selected {
			_ = a.state.CurrentClient().Adapter.PauseTorrent(a.ctx, id)
		}

		a.list.ClearSelection()
		return a.refreshTorrents()()
	}
}

func (a *App) pauseAll() tea.Cmd {
	return func() tea.Msg {
		_ = a.state.CurrentClient().Adapter.PauseAll(a.ctx)
		a.list.ClearSelection()
		return a.refreshTorrents()()
	}
}

func (a *App) resumeSelected() tea.Cmd {
	return func() tea.Msg {
		selected := a.list.GetSelected()
		if len(selected) == 0 {
			if current := a.list.GetCurrentTorrent(); current != nil {
				selected = []string{current.ID}
			}
		}

		for _, id := range selected {
			_ = a.state.CurrentClient().Adapter.ResumeTorrent(a.ctx, id)
		}

		a.list.ClearSelection()
		return a.refreshTorrents()()
	}
}

func (a *App) resumeAll() tea.Cmd {
	return func() tea.Msg {
		_ = a.state.CurrentClient().Adapter.ResumeAll(a.ctx)
		a.list.ClearSelection()
		return a.refreshTorrents()()
	}
}

func (a *App) deleteSelected(withData bool) tea.Cmd {
	return func() tea.Msg {
		selected := a.list.GetSelected()
		if len(selected) == 0 {
			if current := a.list.GetCurrentTorrent(); current != nil {
				selected = []string{current.ID}
			}
		}

		adapter := a.state.CurrentClient().Adapter
		for _, id := range selected {
			if withData {
				_ = adapter.RemoveTorrentWithData(a.ctx, id)
			} else {
				_ = adapter.RemoveTorrent(a.ctx, id)
			}
		}

		a.list.ClearSelection()
		return a.refreshTorrents()()
	}
}

func (a *App) addTorrent(input string, category string) tea.Cmd {
	return func() tea.Msg {
		if input == "" {
			return errorMsg{err: fmt.Errorf("empty magnet link, URL, or path")}
		}

		if err := a.state.CurrentClient().Adapter.AddTorrent(a.ctx, input, category); err != nil {
			return errorMsg{err: err}
		}

		a.inputMode = ""
		a.torrentInput.Blur()
		a.torrentInput.Reset()
		a.categoryList.ResetSelected()
		return a.refreshTorrents()()
	}
}

func (a *App) toggleSpeedLimit() tea.Cmd {
	return func() tea.Msg {
		if err := a.state.ToggleSpeedLimit(a.ctx); err != nil {
			return errorMsg{err: err}
		}
		return speedLimitToggledMsg{}
	}
}

func (a *App) refreshCategories() tea.Cmd {
	return func() tea.Msg {
		categories, err := a.state.CurrentClient().Adapter.GetCategories(a.ctx)
		if err != nil {
			// Categories fetch failed, continue with empty list
			a.state.Categories = []string{}
		} else {
			a.state.Categories = categories
		}
		
		// Build category list items: "None" first, then actual categories
		items := []list.Item{}
		items = append(items, simpleItem("None"))
		for _, cat := range a.state.Categories {
			items = append(items, simpleItem(cat))
		}
		a.categoryList.SetItems(items)
		a.categoryList.ResetSelected()
		
		return nil
	}
}

func (a *App) refreshSpeedLimitStatus() tea.Cmd {
	return func() tea.Msg {
		if err := a.state.RefreshSpeedLimitStatus(a.ctx); err != nil {
			// Speed limit fetch failed, but don't break the app
			// Just log it and return a message to trigger update
			// This allows the UI to still refresh even if speed limit fetch fails
			return speedLimitToggledMsg{}
		}
		return speedLimitToggledMsg{}
	}
}

// Message types

type torrentRefreshMsg struct{}

type errorMsg struct {
	err error
}

func (e errorMsg) Error() string {
	return e.err.Error()
}

type errorClearedMsg struct{}

type speedLimitToggledMsg struct{}

// renderMultilineViewPlaceholder renders the multiline view
func (a *App) renderMultilineViewPlaceholder(width, height int) string {
	return a.multilineList.Render(width, height)
}

// overlayHelpDialog renders a help popup with all keybindings
func (a *App) overlayHelpDialog(baseOutput string) string {
	dialogWidth := 55
	dialogHeight := 35

	if a.width < 55 {
		dialogWidth = a.width - 4
	}
	if dialogWidth < 40 {
		dialogWidth = 40
	}

	// Flatten all bindings from FullHelp into a single list
	help := a.keys.FullHelp()
	var allBindings []struct{ key, desc string }

	for _, row := range help {
		for _, binding := range row {
			k, d := binding.Help().Key, binding.Help().Desc
			if k != "" && d != "" {
				allBindings = append(allBindings, struct{ key, desc string }{k, d})
			}
		}
	}

	// Calculate max key width for alignment
	maxKeyWidth := 0
	for _, b := range allBindings {
		if len(b.key) > maxKeyWidth {
			maxKeyWidth = len(b.key)
		}
	}

	// Build help content with 2-column layout (key | desc)
	var lines []string
	lines = append(lines, "Help - Keybindings")
	lines = append(lines, "")

	for _, binding := range allBindings {
		fullDesc := GetFullHelpText(binding.key)
		line := fmt.Sprintf("%-*s  %s", maxKeyWidth, binding.key, fullDesc)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, "(Press '?' to close)")

	content := strings.Join(lines, "\n")

	// Calculate actual content height to avoid extra padding
	contentHeight := strings.Count(content, "\n") + 1
	// Add some breathing room but don't exceed dialog height
	boxHeight := contentHeight + 2 // +2 for top/bottom padding
	if boxHeight > dialogHeight-2 {
		boxHeight = dialogHeight - 2
	}

	helpBox := a.styles.Dialog.
		Width(dialogWidth - 2).
		Height(boxHeight).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		helpBox,
	)
}

// openTorrentDetail opens the detail view for a specific torrent
func (a *App) openTorrentDetail(id string) tea.Cmd {
	return func() tea.Msg {
		// Fetch torrent detail
		detail, err := a.state.CurrentClient().Adapter.GetTorrentDetail(a.ctx, id)
		if err != nil {
			return errorMsg{err: err}
		}

		// Fetch torrent files
		files, err := a.state.CurrentClient().Adapter.GetTorrentFiles(a.ctx, id)
		if err != nil {
			return errorMsg{err: err}
		}

		// Fetch fresh categories list
		categories, err := a.state.CurrentClient().Adapter.GetCategories(a.ctx)
		if err != nil {
			// If fetch fails, use cached categories
			categories = a.state.Categories
		} else {
			// Update cache with fresh data
			a.state.Categories = categories
		}

		// Create detail view with categories
		clientHost := a.state.CurrentClient().ID
		// Try to get actual hostname from config if available
		if len(a.state.Config.Clients) > 0 && a.state.CurrentClientIdx < len(a.state.Config.Clients) {
			clientHost = a.state.Config.Clients[a.state.CurrentClientIdx].Host
		}
		a.detailView = NewDetailViewWithHost(a.styles, detail, files, categories, clientHost)
		a.screenMode = "detail"

		// Return a message to trigger an immediate update and redraw
		return detailViewOpenedMsg{}
	}
}

// saveTorrentChanges saves all pending torrent changes via API
func (a *App) saveTorrentChanges() tea.Cmd {
	return func() tea.Msg {
		if a.detailView == nil || a.detailView.Detail == nil {
			return errorMsg{err: fmt.Errorf("no torrent selected")}
		}

		currentClient := a.state.CurrentClient()
		adapter := currentClient.Adapter
		torrentID := a.detailView.Detail.ID
		state := a.detailView.State
		clientType := currentClient.Type

		// Check for name changes
		if newName := state.GetCurrentValue("name"); newName != state.OriginalValues["name"] && newName != "" {
			if err := adapter.SetTorrentName(a.ctx, torrentID, newName); err != nil {
				return errorMsg{err: fmt.Errorf("failed to save name: %w", err)}
			}
		}

		// Check for location changes
		newLocation := state.GetCurrentValue("location")
		locationChanged := newLocation != state.OriginalValues["location"] && newLocation != ""
		
		if locationChanged {
			if err := adapter.SetSavePath(a.ctx, torrentID, newLocation); err != nil {
				return errorMsg{err: fmt.Errorf("failed to save location: %w", err)}
			}

			// For Transmission, derive category/label from directory name
			if clientType == "transmission" {
				dirName := GetSubdirectoryFromPath(newLocation)
				if dirName != "" && dirName != "/" {
					if err := adapter.SetCategory(a.ctx, torrentID, dirName); err != nil {
						return errorMsg{err: fmt.Errorf("failed to save category: %w", err)}
					}
				}
			}
		}

		// Check for category changes (explicit)
		newCategory := state.GetCurrentValue("category")
		categoryChanged := newCategory != state.OriginalValues["category"]
		
		if categoryChanged && !locationChanged {
			// Only set category if location wasn't already changed
			if err := adapter.SetCategory(a.ctx, torrentID, newCategory); err != nil {
				return errorMsg{err: fmt.Errorf("failed to save category: %w", err)}
			}
		}

		// Check for tag changes
		if newTags := state.GetCurrentValue("tags"); newTags != state.OriginalValues["tags"] {
			tagList := []string{}
			if newTags != "" {
				// Parse comma-separated tags
				for _, tag := range strings.Split(newTags, ",") {
					tagList = append(tagList, strings.TrimSpace(tag))
				}
			}
			if err := adapter.SetTags(a.ctx, torrentID, tagList); err != nil {
				return errorMsg{err: fmt.Errorf("failed to save tags: %w", err)}
			}
		}

		// Check for file priority changes
		if a.detailView.FilesTab != nil && a.detailView.FilesTab.HasFileChanges() {
			selectedIndices := a.detailView.FilesTab.GetSelectedFileIndices()
			if err := adapter.SetFilePriorities(a.ctx, torrentID, selectedIndices); err != nil {
				return errorMsg{err: fmt.Errorf("failed to save file priorities: %w", err)}
			}
		}

		// All changes saved successfully
		// Clear the detail view and return to list
		a.screenMode = "list"
		a.detailView = nil

		return savesCompleteMsg{}
	}
}

// savesCompleteMsg is sent when torrent changes are saved
type savesCompleteMsg struct{}

// detailViewOpenedMsg is sent when detail view opens to trigger immediate redraw
type detailViewOpenedMsg struct{}

// syncListViews synchronizes torrents to both single-line and multi-line views
func (a *App) syncListViews(torrents []client.Torrent) {
	a.list.SetTorrents(torrents)
	a.multilineList.SetTorrents(torrents)
}

// Shutdown cleans up resources
func (a *App) Shutdown() {
	a.cancel()
	if a.state.CurrentClient() != nil {
		a.state.CurrentClient().Adapter.Disconnect(context.Background())
	}
}

// getAvailableThemesForCycling returns the list of available themes to cycle through
func getAvailableThemesForCycling(cfg *config.Config) []string {
	return config.AvailableThemes
}

// loadThemeByName loads a theme by its name
func loadThemeByName(themeName string, cfg *config.Config) Theme {
	switch themeName {
	case "dark":
		return DefaultTheme()
	case "light":
		return LightTheme()
	case "highcontrast":
		return HighContrastTheme()
	case "omarchy":
		if cfg.LoadedColors != nil && len(cfg.LoadedColors["omarchy"]) > 0 {
			return OmarchyTheme(cfg.LoadedColors["omarchy"])
		}
		return DefaultTheme()
	case "custom":
		if cfg.LoadedColors != nil && len(cfg.LoadedColors["custom"]) > 0 {
			return OmarchyTheme(cfg.LoadedColors["custom"])
		}
		return DefaultTheme()
	default:
		// Fallback to dark theme
		return DefaultTheme()
	}
}
