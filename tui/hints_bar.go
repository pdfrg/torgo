package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
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

	// Truncate or pad to fit width
	runes := []rune(hint)
	if len(runes) > width {
		hint = string(runes[:width-3]) + "..."
	} else if len(runes) < width {
		hint = hint + strings.Repeat(" ", width-len(runes))
	}

	return h.styles.HintsBar.Render(hint)
}

// formatKeyHelp formats a single key binding with colored key and description
func formatKeyHelp(binding key.Binding) string {
	help := binding.Help()
	if help.Key == "" || help.Desc == "" {
		return ""
	}
	// Color the first character of key in bright cyan
	keyColor := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	// Color first character of description in cyan, rest in normal color
	descColor := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	
	var keyPart string
	if len(help.Key) == 1 {
		keyPart = keyColor.Render(help.Key)
	} else {
		keyPart = keyColor.Render(help.Key[:1]) + help.Key[1:]
	}
	
	// Format description with colored first letter
	var descPart string
	if len(help.Desc) > 0 {
		descPart = keyColor.Render(string(help.Desc[0])) + descColor.Render(help.Desc[1:])
	} else {
		descPart = ""
	}
	
	return keyPart + ":" + descPart
}
