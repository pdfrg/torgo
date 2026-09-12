package tui

import (
	"image/color"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

// ProgressBarBuilder creates progress bars with status-based gradients.
//
// By default a builder follows the live theme (GetCurrentTheme) so cached
// builders never render a stale palette after a theme change. Call WithTheme
// to pin a specific theme instead.
type ProgressBarBuilder struct {
	theme    Theme
	explicit bool // true once WithTheme pins an explicit theme
	width    int
}

// NewProgressBarBuilder creates a builder that tracks the current theme
func NewProgressBarBuilder() *ProgressBarBuilder {
	return &ProgressBarBuilder{
		theme: GetCurrentTheme(),
		width: 35, // default width
	}
}

// WithWidth sets the width of progress bars created by this builder
func (pb *ProgressBarBuilder) WithWidth(w int) *ProgressBarBuilder {
	pb.width = w
	return pb
}

// WithTheme pins a custom theme for this builder, opting out of live
// current-theme tracking.
func (pb *ProgressBarBuilder) WithTheme(t Theme) *ProgressBarBuilder {
	pb.theme = t
	pb.explicit = true
	return pb
}

// activeTheme returns the theme to render with: the pinned theme when
// WithTheme was used, otherwise the current global theme.
func (pb *ProgressBarBuilder) activeTheme() Theme {
	if pb.explicit {
		return pb.theme
	}
	return GetCurrentTheme()
}

// BuildForStatus creates a progress bar with gradient colors for a given status
func (pb *ProgressBarBuilder) BuildForStatus(status string) progress.Model {
	theme := pb.activeTheme()
	colors := theme.GetStatusGradient(status)
	p := progress.New(
		progress.WithColors(colors[0], colors[1]),
		progress.WithFillCharacters('▌', '░'), // Half block for better gradient blending + light shade for texture
	)
	// Set empty color - use theme's ProgressBarEmptyColor if available, fallback to dark gray
	if theme.ProgressBarEmptyColor != nil {
		p.EmptyColor = theme.ProgressBarEmptyColor
	} else {
		p.EmptyColor = lipgloss.Color("#333333")
	}
	// Style the percentage text with TextNormal (label-like, distinct from adjacent data values)
	p.PercentageStyle = lipgloss.NewStyle().Foreground(theme.TextNormal)
	p.SetWidth(pb.width)
	return p
}

// BuildForStatusQuick is a convenience function that uses the current theme
func BuildForStatus(status string) progress.Model {
	return NewProgressBarBuilder().BuildForStatus(status)
}

// BuildCustom creates a progress bar with custom gradient colors
func (pb *ProgressBarBuilder) BuildCustom(color1, color2 color.Color) progress.Model {
	theme := pb.activeTheme()
	p := progress.New(
		progress.WithColors(color1, color2),
		progress.WithFillCharacters('▌', '░'), // Half block for better gradient blending + light shade for texture
	)
	// Set empty color - use theme's ProgressBarEmptyColor if available, fallback to dark gray
	if theme.ProgressBarEmptyColor != nil {
		p.EmptyColor = theme.ProgressBarEmptyColor
	} else {
		p.EmptyColor = lipgloss.Color("#333333")
	}
	// Style the percentage text with TextNormal (label-like, distinct from adjacent data values)
	p.PercentageStyle = lipgloss.NewStyle().Foreground(theme.TextNormal)
	p.SetWidth(pb.width)
	return p
}

// BuildSolid creates a solid-color progress bar
func (pb *ProgressBarBuilder) BuildSolid(color color.Color) progress.Model {
	theme := pb.activeTheme()
	p := progress.New(
		progress.WithColors(color),
	)
	p.Full = progress.DefaultFullCharFullBlock // Use full block for solid fills
	// A single stop bypasses gradient blending, preserving ANSI palette
	// indices so the terminal resolves them from its live palette.
	if theme.ProgressBarEmptyColor != nil {
		p.EmptyColor = theme.ProgressBarEmptyColor
	}
	p.PercentageStyle = lipgloss.NewStyle().Foreground(theme.TextNormal)
	p.SetWidth(pb.width)
	return p
}

// RenderProgressBar renders a progress bar at a given percentage with a status label
func (pb *ProgressBarBuilder) RenderProgressBar(status string, percent float64) string {
	theme := pb.activeTheme()
	if theme.SolidProgressBars {
		p := pb.BuildSolid(theme.GetStatusGradient(status)[0])
		return p.ViewAs(percent)
	}
	p := pb.BuildForStatus(status)
	return p.ViewAs(percent)
}
