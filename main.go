package main

import (
	"log"
	"os"
	"tqbtui/config"
	"tqbtui/state"
	"tqbtui/tui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n\nCreate ~/.config/tqbtui/config.toml (see config.example.toml)", err)
	}

	// Initialize app state with clients
	appState, err := state.NewAppState(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app state: %v", err)
	}

	// Create and run TUI
	app := tui.NewApp(appState)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}

	app.Shutdown()
	os.Exit(0)
}
