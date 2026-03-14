package tui

import (
	"image/color"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

// ProgressBarBuilder creates progress bars with status-based gradients
type ProgressBarBuilder struct {
	theme Theme
	width int
}

// NewProgressBarBuilder creates a builder with the current theme
func NewProgressBarBuilder() *ProgressBarBuilder {
	return &ProgressBarBuilder{
		theme: CurrentTheme,
		width: 35, // default width
	}
}

// WithWidth sets the width of progress bars created by this builder
func (pb *ProgressBarBuilder) WithWidth(w int) *ProgressBarBuilder {
	pb.width = w
	return pb
}

// WithTheme sets a custom theme for this builder
func (pb *ProgressBarBuilder) WithTheme(t Theme) *ProgressBarBuilder {
	pb.theme = t
	return pb
}

// BuildForStatus creates a progress bar with gradient colors for a given status
func (pb *ProgressBarBuilder) BuildForStatus(status string) progress.Model {
	colors := pb.theme.GetStatusGradient(status)
	p := progress.New(
		progress.WithColors(colors[0], colors[1]),
		progress.WithFillCharacters('▌', '░'), // Half block for better gradient blending + light shade for texture
	)
	// Set empty color - use theme's ProgressBarEmptyColor if available, fallback to dark gray
	if pb.theme.ProgressBarEmptyColor != nil {
		p.EmptyColor = pb.theme.ProgressBarEmptyColor
	} else {
		p.EmptyColor = lipgloss.Color("#333333")
	}
	// Style the percentage text with TextNormal (label-like, distinct from adjacent data values)
	p.PercentageStyle = lipgloss.NewStyle().Foreground(pb.theme.TextNormal)
	p.SetWidth(pb.width)
	return p
}

// BuildForStatusQuick is a convenience function that uses the current theme
func BuildForStatus(status string) progress.Model {
	return NewProgressBarBuilder().BuildForStatus(status)
}

// BuildCustom creates a progress bar with custom gradient colors
func (pb *ProgressBarBuilder) BuildCustom(color1, color2 color.Color) progress.Model {
	p := progress.New(
		progress.WithColors(color1, color2),
		progress.WithFillCharacters('▌', '░'), // Half block for better gradient blending + light shade for texture
	)
	// Set empty color - use theme's ProgressBarEmptyColor if available, fallback to dark gray
	if pb.theme.ProgressBarEmptyColor != nil {
		p.EmptyColor = pb.theme.ProgressBarEmptyColor
	} else {
		p.EmptyColor = lipgloss.Color("#333333")
	}
	// Style the percentage text with TextNormal (label-like, distinct from adjacent data values)
	p.PercentageStyle = lipgloss.NewStyle().Foreground(pb.theme.TextNormal)
	p.SetWidth(pb.width)
	return p
}

// BuildSolid creates a solid-color progress bar
func (pb *ProgressBarBuilder) BuildSolid(color color.Color) progress.Model {
	p := progress.New(
		progress.WithColors(color),
	)
	p.Full = progress.DefaultFullCharFullBlock // Use full block for solid fills
	p.SetWidth(pb.width)
	return p
}

// RenderProgressBar renders a progress bar at a given percentage with a status label
func (pb *ProgressBarBuilder) RenderProgressBar(status string, percent float64) string {
	p := pb.BuildForStatus(status)
	return p.ViewAs(percent)
}
