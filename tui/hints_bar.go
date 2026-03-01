package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
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

	barBg := lipgloss.Color("237")

	// Color styles for the hint parts (each with background applied)
	keyColor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")).
		Bold(true).
		Background(barBg)
	descColor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(barBg)

	// Build hint parts with their colors and background
	parts := []string{}
	for _, binding := range hints {
		help := binding.Help()
		if help.Key == "" || help.Desc == "" {
			continue
		}
		// Render key in cyan, description in default color (both with background)
		part := keyColor.Render(help.Key) + descColor.Render(":"+help.Desc)
		parts = append(parts, part)
	}

	// Join parts with space separators (also with background)
	var spacedParts []string
	for i, p := range parts {
		spacedParts = append(spacedParts, p)
		if i < len(parts)-1 {
			spacedParts = append(spacedParts, descColor.Render("  "))
		}
	}
	hintText := lipgloss.JoinHorizontal(lipgloss.Left, spacedParts...)
	hintText = descColor.Render(" ") + hintText + descColor.Render(" ")

	// Apply bar style: width and background (background is already on text parts)
	barStyle := lipgloss.NewStyle().
		Width(width).
		Background(barBg)

	return barStyle.Render(hintText)
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
