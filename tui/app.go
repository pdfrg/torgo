package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	statusBar          *StatusBar
	hintsBar           *HintsBar
	width              int
	height             int
	showHints          bool
	showHelp           bool
	viewMode           string        // "default" or "multiline"
	inputMode          string        // "", "add", "search"
	torrentInput       textinput.Model
	categoryList       list.Model
	lastError          string
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewApp creates a new TUI application
func NewApp(appState *state.AppState) *App {
	ctx, cancel := context.WithCancel(context.Background())

	styles := DefaultStyles()
	keys := DefaultKeyMap()

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Magnet link, URL, or .torrent file path"
	ti.CharLimit = 1024

	// Initialize category list with compact delegate
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	categoryList := list.New([]list.Item{}, delegate, 0, 6)
	categoryList.SetShowHelp(false)
	categoryList.SetShowStatusBar(false)
	categoryList.SetShowTitle(true)
	categoryList.Title = "Category"

	app := &App{
		state:        appState,
		styles:       styles,
		keys:         keys,
		list:         NewTorrentListView(styles),
		statusBar:    NewStatusBar(styles),
		hintsBar:     NewHintsBar(styles, keys),
		showHints:    appState.Config.UI.ShowHints,
		showHelp:     false,
		viewMode:     "default",
		torrentInput: ti,
		categoryList: categoryList,
		ctx:          ctx,
		cancel:       cancel,
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case torrentRefreshMsg:
		a.list.SetTorrents(a.state.FilteredTorrents())
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
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Fixed overhead: title(1) + blank(1) + status(1) + hints(0-1)
	fixedHeight := 3
	if a.showHints {
		fixedHeight = 4
	}

	// Error height
	errorHeight := 0
	if a.lastError != "" {
		errorHeight = 2
	}

	// List height = total - fixed - error (no input height since dialog is modal)
	listHeight := a.height - fixedHeight - errorHeight
	if listHeight < 3 {
		listHeight = 3
	}

	// Build output
	lines := []string{}

	// Title
	titleText := "tqbtui – Torrent Client TUI"
	lines = append(lines, a.styles.Title.Render(titleText))
	lines = append(lines, "")

	// List or placeholder view
	var listView string
	if a.viewMode == "multiline" {
		listView = a.renderMultilineViewPlaceholder(a.width, listHeight)
	} else {
		listView = a.list.Render(a.width, listHeight)
	}
	lines = append(lines, listView)

	// Error message if present
	if a.lastError != "" {
		lines = append(lines, "")
		lines = append(lines, a.styles.ListItem.Foreground(a.styles.ErrorColor).
			Render("Error: "+a.lastError))
	}

	// Join content
	output := strings.Join(lines, "\n")

	// Count actual lines in output (handle multi-line list view)
	actualLineCount := strings.Count(output, "\n") + 1

	// Calculate padding to push status/hints to bottom
	statusLinesNeeded := 1
	if a.showHints {
		statusLinesNeeded = 2
	}

	paddingNeeded := a.height - actualLineCount - statusLinesNeeded
	if paddingNeeded < 0 {
		paddingNeeded = 0
	}

	// Add padding
	if paddingNeeded > 0 {
		output += strings.Repeat("\n", paddingNeeded)
	}

	// Status bar
	output += "\n" + a.statusBar.Render(a.state, a.width)

	// Hints bar
	if a.showHints {
		output += "\n" + a.hintsBar.Render(a.width)
	}

	// Overlay modals
	if a.showHelp {
		output = a.overlayHelpDialog(output)
	}

	if a.inputMode == "add" {
		output = a.overlayAddDialog(output)
	}

	return output
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

	// Render torrent input
	inputSection := a.torrentInput.View()

	// Render category list
	categoryView := ""
	if len(a.state.Categories) > 0 {
		a.categoryList.SetWidth(dialogWidth - 4)
		a.categoryList.SetHeight(13) // This is not items, I think it's lines. 13 is min to see 5 items.
		categoryView = a.categoryList.View()
	}

	// Assemble dialog content
	contentLines := []string{title, "", inputSection}
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

	switch {
	case isKeyMatch(k, a.keys.Quit):
		return a, tea.Quit

	case isKeyMatch(k, a.keys.Up):
		a.list.MoveCursor(-1)
		return a, nil

	case isKeyMatch(k, a.keys.Down):
		a.list.MoveCursor(1)
		return a, nil

	case isKeyMatch(k, a.keys.Select):
		a.list.ToggleSelection()
		return a, nil

	case isKeyMatch(k, a.keys.SelectAll):
		a.list.SelectAll()
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

	case isKeyMatch(k, a.keys.ToggleView):
		if a.viewMode == "default" {
			a.viewMode = "multiline"
		} else {
			a.viewMode = "default"
		}
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
		a.torrentInput.Blur()
		return a, nil
	case "enter":
		if a.inputMode == "add" {
			input := a.torrentInput.Value()
			if input == "" {
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
			return a, a.addTorrent(input, category)
		}
		a.inputMode = ""
		a.torrentInput.Blur()
		return a, nil
	case "ctrl+c":
		a.inputMode = ""
		a.torrentInput.Blur()
		return a, nil
	case "ctrl+p":
		// Paste from clipboard
		if text, err := clipboard.ReadAll(); err == nil {
			// Remove trailing newlines that often come from clipboard
			a.torrentInput.SetValue(a.torrentInput.Value() + strings.TrimSpace(text))
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
		return a, cmd
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

// renderMultilineViewPlaceholder renders a placeholder for multiline view
func (a *App) renderMultilineViewPlaceholder(width, height int) string {
	msg := "Multi-line view under development\nPress 'v' again to return to default single-line view"
	box := a.styles.Dialog.
		Width(width - 2).
		Height(height - 2).
		Padding(1, 2).
		Render(msg)
	return box
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

	helpBox := a.styles.Dialog.
		Width(dialogWidth - 2).
		Height(dialogHeight - 2).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		helpBox,
	)
}

// Shutdown cleans up resources
func (a *App) Shutdown() {
	a.cancel()
	if a.state.CurrentClient() != nil {
		a.state.CurrentClient().Adapter.Disconnect(context.Background())
	}
}
