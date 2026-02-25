package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application
type KeyMap struct {
	// Navigation
	Up   key.Binding
	Down key.Binding

	// Selection
	Select    key.Binding
	SelectAll key.Binding

	// Torrent control
	Pause      key.Binding
	PauseAll   key.Binding
	Resume     key.Binding
	ResumeAll  key.Binding
	Delete     key.Binding
	DeleteData key.Binding

	// Client & view
	AddTorrent   key.Binding
	SwitchClient key.Binding
	Sort         key.Binding
	Filter       key.Binding
	ToggleHints  key.Binding
	ToggleSpeedLimit key.Binding

	// General
	Quit key.Binding
	Help key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),

		// Selection
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "select all"),
		),

		// Torrent control
		Pause: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pause"),
		),
		PauseAll: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "pause all"),
		),
		Resume: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resume"),
		),
		ResumeAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "resume all"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		DeleteData: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "delete with data"),
		),

		// Client & view
		AddTorrent: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add torrent"),
		),
		SwitchClient: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "cycle client"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		ToggleHints: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "toggle hints"),
		),
		ToggleSpeedLimit: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "toggle speed limit"),
		),

		// General
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Pause, k.PauseAll, k.Resume, k.ResumeAll,
		k.Delete, k.DeleteData, k.AddTorrent,
		k.SwitchClient, k.ToggleSpeedLimit, k.ToggleHints, k.Quit,
	}
}

// FullHelp returns full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.SelectAll},
		{k.Pause, k.PauseAll, k.Resume, k.ResumeAll},
		{k.Delete, k.DeleteData, k.AddTorrent},
		{k.SwitchClient, k.Sort, k.Filter, k.ToggleSpeedLimit},
		{k.ToggleHints, k.Help, k.Quit},
	}
}
