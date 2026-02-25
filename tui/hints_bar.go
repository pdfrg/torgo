package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// HintsBar displays keybinding hints
type HintsBar struct {
	styles *Styles
	keys   KeyMap
}

// NewHintsBar creates a new hints bar
func NewHintsBar(styles *Styles, keys KeyMap) *HintsBar {
	return &HintsBar{
		styles: styles,
		keys:   keys,
	}
}

// Render returns the rendered hints bar
func (h *HintsBar) Render(width int) string {
	hints := h.keys.ShortHelp()

	parts := []string{}
	for _, binding := range hints {
		part := formatKeyHelp(binding)
		if part != "" {
			parts = append(parts, part)
		}
	}

	hint := strings.Join(parts, "  ")

	// Truncate if too long
	if len(hint) > width {
		hint = hint[:width-3] + "..."
	}

	return h.styles.HintsBar.Render(hint)
}

// formatKeyHelp formats a single key binding
func formatKeyHelp(binding key.Binding) string {
	help := binding.Help()
	if help.Key == "" || help.Desc == "" {
		return ""
	}
	return help.Key + ":" + help.Desc
}
