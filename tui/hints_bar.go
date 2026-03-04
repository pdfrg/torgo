package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// HintsBar displays keybinding hints
type HintsBar struct {
	styles      *Styles
	keys        KeyMap
	currentTheme string
}

// NewHintsBar creates a new hints bar
func NewHintsBar(styles *Styles, keys KeyMap) *HintsBar {
	return &HintsBar{
		styles:       styles,
		keys:         keys,
		currentTheme: "dark",
	}
}

// SetCurrentTheme updates the current theme displayed in the hints bar
func (h *HintsBar) SetCurrentTheme(theme string) {
	h.currentTheme = theme
}

// abbreviateThemeName returns a short name for the theme
func abbreviateThemeName(theme string) string {
	switch theme {
	case "dark":
		return "dark"
	case "light":
		return "lite"
	case "highcontrast":
		return "HC"
	case "omarchy":
		return "omarchy"
	case "custom":
		return "custom"
	case "default":
		return "default"
	default:
		return theme
	}
}

// Render returns the rendered hints bar
func (h *HintsBar) Render(width int) string {
	hints := h.keys.ShortHelp()

	// Get bar colors from theme
	theme := CurrentTheme
	barBg := theme.StatusBarBg
	barFg := theme.StatusBarFg
	accentColor := theme.StatusBarAccent

	// Color styles for the hint parts
	keyColor := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Background(barBg)
	descColor := lipgloss.NewStyle().
		Foreground(barFg).
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
	
	// Add theme indicator at the end
	themeAbbr := abbreviateThemeName(h.currentTheme)
	themeHint := keyColor.Render("t") + descColor.Render(":theme ("+themeAbbr+")")
	hintText = descColor.Render(" ") + hintText + descColor.Render("  ") + themeHint + descColor.Render(" ")

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
