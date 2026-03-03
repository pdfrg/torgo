package main

import (
	"fmt"
	"log"
	"os"
	"strings"
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

	// Print available themes if custom themes were discovered
	if len(config.AvailableThemes) > 3 { // More than the 3 built-in themes (dark, light, highcontrast)
		fmt.Printf("Available themes: %s\n", strings.Join(config.AvailableThemes, ", "))
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
