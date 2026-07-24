package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"tqbtui/config"
	"tqbtui/state"
	"tqbtui/tui"

	tea "charm.land/bubbletea/v2"
)

var Version = "dev"

const usageText = `Usage: tqbtui [flags]

Flags:
  -h, --help            Show this help text and exit
  -v, --version         Print version and exit
  -c, --config <path>   Path to config file (default: ~/.config/tqbtui/config.toml)
  -i, --client <idx>    Client index to start with (0-based)
`

func main() {
	var (
		showHelp    bool
		showVersion bool
		configPath  string
		clientIdx   int
	)

	fs := flag.NewFlagSet("tqbtui", flag.ContinueOnError)
	fs.BoolVar(&showHelp, "h", false, "")
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.StringVar(&configPath, "c", "", "")
	fs.StringVar(&configPath, "config", "", "")
	fs.IntVar(&clientIdx, "i", -1, "")
	fs.IntVar(&clientIdx, "client", -1, "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if showHelp {
		fs.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
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

	// Override client index if provided via CLI
	if clientIdx >= 0 && clientIdx < len(appState.Clients) {
		appState.CurrentClientIdx = clientIdx
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
