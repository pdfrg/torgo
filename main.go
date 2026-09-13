package main

import (
	"flag"
	"fmt"
	"github.com/pdfrg/torgo/config"
	"github.com/pdfrg/torgo/state"
	"github.com/pdfrg/torgo/tui"
	"log"
	"os"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"
)

var Version = "dev"

// resolveVersion fills Version from Go's build info when ldflags did not set it.
// This makes binaries installed via `go install` (which cannot pass ldflags)
// report their module version, while make/GoReleaser builds keep the injected tag.
func resolveVersion() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			Version = v
		}
	}
}

const usageText = `Usage: torgo [flags]

Flags:
  -h, --help            Show this help text and exit
  -v, --version         Print version and exit
  -c, --config <path>   Path to config file (default: ~/.config/torgo/config.toml)
  -i, --client <idx>    Client index to start with (0-based)
`

func main() {
	resolveVersion()

	var (
		showHelp    bool
		showVersion bool
		configPath  string
		clientIdx   int
	)

	fs := flag.NewFlagSet("torgo", flag.ContinueOnError)
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
		log.Fatalf("Failed to load config: %v\n\nCreate ~/.config/torgo/config.toml (see config.example.toml)", err)
	}

	// Print available themes if custom themes were discovered
	if len(config.AvailableThemes) > 4 { // More than the 4 built-in themes (dark, light, highcontrast, terminal)
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
