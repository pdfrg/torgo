package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the application
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding

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
	Details    key.Binding  // View torrent details

	// Client & view
	AddTorrent   key.Binding
	SwitchClient key.Binding
	Sort         key.Binding
	Filter       key.Binding
	ToggleView   key.Binding
	ToggleHints  key.Binding
	ToggleSpeedLimit key.Binding
	Search       key.Binding

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
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp/^U", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn/^D", "page down"),
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
			key.WithHelp("P", "all"),
		),
		Resume: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resume"),
		),
		ResumeAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "all"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "del"),
		),
		DeleteData: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "w/data"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),

		// Client & view
		AddTorrent: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		SwitchClient: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "client"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		ToggleView: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "view"),
		),
		ToggleHints: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "hints"),
		),
		ToggleSpeedLimit: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "limit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
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
		k.Delete, k.DeleteData, k.Details, k.AddTorrent,
		k.SwitchClient, k.Search, k.ToggleSpeedLimit, k.ToggleHints, k.ToggleView, k.Help, k.Quit,
	}
}

// FullHelp returns full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Select, k.SelectAll},
		{k.Pause, k.PauseAll, k.Resume, k.ResumeAll},
		{k.Delete, k.DeleteData, k.Details, k.AddTorrent},
		{k.SwitchClient, k.Sort, k.Filter, k.Search, k.ToggleView, k.ToggleSpeedLimit},
		{k.ToggleHints, k.Help, k.Quit},
	}
}

// GetFullHelpText returns the full descriptive text for a key
func GetFullHelpText(keyName string) string {
	fullHelpMap := map[string]string{
		"↑/k":     "move cursor up",
		"↓/j":     "move cursor down",
		"PgUp/^U": "page up",
		"PgDn/^D": "page down",
		"P":       "pause all",
		"R":       "resume all",
		"x":       "delete",
		"X":       "delete with data",
		"enter":   "view details",
		"a":       "add torrent",
		"c":       "cycle clients",
		"v":       "cycle views",
		"h":       "toggle hints bar",
		"l":       "toggle speed limit",
		"space":   "toggle select",
		"A":       "select all",
		"p":       "pause",
		"r":       "resume",
		"s":       "sort",
		"f":       "filter",
		"/":       "search",
		"?":       "help",
		"q":       "quit",
	}
	if full, ok := fullHelpMap[keyName]; ok {
		return full
	}
	return keyName
}
