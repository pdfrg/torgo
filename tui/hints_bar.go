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

	// Account for padding(0, 1) in HintsBar style which adds 2 chars (1 on each side)
	// So we need to reduce available width by 2 for the padding
	availableWidth := width - 2
	
	// Truncate or pad to fit available width
	runes := []rune(hint)
	if len(runes) > availableWidth {
		hint = string(runes[:availableWidth-3]) + "..."
	} else if len(runes) < availableWidth {
		hint = hint + strings.Repeat(" ", availableWidth-len(runes))
	}

	return h.styles.HintsBar.Render(hint)
}

// formatKeyHelp formats a single key binding with colored key
func formatKeyHelp(binding key.Binding) string {
	help := binding.Help()
	if help.Key == "" || help.Desc == "" {
		return ""
	}
	// Color the first character of key in bright cyan
	keyColor := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	
	var keyPart string
	if len(help.Key) == 1 {
		keyPart = keyColor.Render(help.Key)
	} else {
		keyPart = keyColor.Render(help.Key[:1]) + help.Key[1:]
	}
	
	return keyPart + ":" + help.Desc
}
