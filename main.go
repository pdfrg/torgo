package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"tqbtui/config"
	"tqbtui/state"
)

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("              tqbtui – Torrent Client TUI")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

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

	fmt.Printf("Loaded %d client(s):\n", len(appState.Clients))
	fmt.Println()

	// Display client info
	fmt.Println("┌─────┬─────────────────────────┬────────────────┬──────────────────┐")
	fmt.Println("│ Idx │ Name                    │ Type           │ Address          │")
	fmt.Println("├─────┼─────────────────────────┼────────────────┼──────────────────┤")

	for i, c := range appState.Clients {
		fmt.Printf("│ %2d  │ %-23s │ %-14s │ ?                │\n",
			i, c.Name, c.Type)
	}
	fmt.Println("└─────┴─────────────────────────┴────────────────┴──────────────────┘")
	fmt.Println()

	// Test connectivity
	fmt.Println("Testing client connections...")
	fmt.Println()
	ctx := context.Background()

	for i, c := range appState.Clients {
		fmt.Printf("[%d] %s... ", i, c.Name)
		if err := c.Adapter.Connect(ctx); err != nil {
			fmt.Printf("✗ (error: %v)\n", err)
		} else {
			fmt.Printf("✓\n")
			c.Adapter.Disconnect(ctx)
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Ready to launch TUI!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// TODO: Initialize and run TUI
	_ = appState
	os.Exit(0)
}
