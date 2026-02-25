package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"tqbtui/state"
)

// App is the main TUI application model
type App struct {
	state       *state.AppState
	styles      *Styles
	keys        KeyMap
	list        *TorrentListView
	statusBar   *StatusBar
	hintsBar    *HintsBar
	width       int
	height      int
	showHints   bool
	inputMode   string    // "", "add", "search"
	inputBuffer string
	lastError   string
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewApp creates a new TUI application
func NewApp(appState *state.AppState) *App {
	ctx, cancel := context.WithCancel(context.Background())

	styles := DefaultStyles()
	keys := DefaultKeyMap()

	app := &App{
		state:     appState,
		styles:    styles,
		keys:      keys,
		list:      NewTorrentListView(styles),
		statusBar: NewStatusBar(styles),
		hintsBar:  NewHintsBar(styles, keys),
		showHints: appState.Config.UI.ShowHints,
		ctx:       ctx,
		cancel:    cancel,
	}

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return a.connectAndRefresh()
}

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
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	lines := []string{}

	// Title
	lines = append(lines, a.styles.Title.Render("tqbtui – Torrent Client TUI"))
	lines = append(lines, "")

	// Main list (with height calculation)
	listHeight := a.height - 6
	if a.showHints {
		listHeight -= 1
	}
	if a.lastError != "" {
		listHeight -= 2
	}

	listView := a.list.Render(a.width, listHeight)
	lines = append(lines, listView)

	// Error message if present
	if a.lastError != "" {
		lines = append(lines, "")
		lines = append(lines, a.styles.ListItem.Foreground(a.styles.ErrorColor).
			Render("Error: "+a.lastError))
		}

		// Status bar
		lines = append(lines, "")
		statusView := a.statusBar.Render(a.state, a.width)
		lines = append(lines, statusView)

		// Hints bar
		if a.showHints {
			hintsView := a.hintsBar.Render(a.width)
			lines = append(lines, hintsView)
		}

		return strings.Join(lines, "\n")
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
		a.inputBuffer = ""
		return a, nil

	case isKeyMatch(k, a.keys.SwitchClient):
		a.state.SwitchClient()
		a.list.ClearSelection()
		return a, a.connectAndRefresh()

	case isKeyMatch(k, a.keys.Sort):
		a.state.CycleSort()
		a.list.SetTorrents(a.state.FilteredTorrents())
		return a, nil

	case isKeyMatch(k, a.keys.Filter):
		a.state.CycleFilter()
		a.list.ClearSelection()
		a.list.SetTorrents(a.state.FilteredTorrents())
		return a, nil

	case isKeyMatch(k, a.keys.ToggleHints):
		a.showHints = !a.showHints
		return a, nil
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
		a.inputBuffer = ""
		return a, nil
	case "enter":
		if a.inputMode == "add" {
			return a, a.addTorrent(a.inputBuffer)
		}
		a.inputMode = ""
		a.inputBuffer = ""
		return a, nil
	case "backspace":
		if len(a.inputBuffer) > 0 {
			a.inputBuffer = a.inputBuffer[:len(a.inputBuffer)-1]
		}
		return a, nil
	default:
		if len(msg.String()) == 1 {
			a.inputBuffer += msg.String()
		}
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

func (a *App) addTorrent(magnetOrPath string) tea.Cmd {
	return func() tea.Msg {
		if magnetOrPath == "" {
			return errorMsg{err: fmt.Errorf("empty magnet link or path")}
		}

		if err := a.state.CurrentClient().Adapter.AddTorrent(a.ctx, magnetOrPath); err != nil {
			return errorMsg{err: err}
		}

		a.inputMode = ""
		a.inputBuffer = ""
		return a.refreshTorrents()()
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

// Shutdown cleans up resources
func (a *App) Shutdown() {
	a.cancel()
	if a.state.CurrentClient() != nil {
		a.state.CurrentClient().Adapter.Disconnect(context.Background())
	}
}
