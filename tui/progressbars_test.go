package tui

import "testing"

// TestProgressBarBuilderFollowsCurrentTheme guards against cached builders
// rendering a stale palette after a theme change (the original bug).
func TestProgressBarBuilderFollowsCurrentTheme(t *testing.T) {
	original := GetCurrentTheme()
	t.Cleanup(func() { SetTheme(original) })

	SetTheme(DefaultTheme())
	pb := NewProgressBarBuilder()

	// TerminalTheme uses solid bars, so it is an easy observable switch.
	SetTheme(TerminalTheme())
	if !pb.activeTheme().SolidProgressBars {
		t.Fatal("builder did not follow the new current theme")
	}

	// A builder pinned with WithTheme must ignore later theme changes.
	pinned := NewProgressBarBuilder().WithTheme(DefaultTheme())
	if pinned.activeTheme().SolidProgressBars {
		t.Fatal("WithTheme should pin the supplied theme")
	}
}

// TestMultilineInvalidateThemeCaches ensures the explicit cache drop works.
func TestMultilineInvalidateThemeCaches(t *testing.T) {
	m := NewMultilineTorrentListView(DefaultStyles())
	m.barModels = map[string]*ProgressBarBuilder{"downloading": NewProgressBarBuilder()}
	m.barModelW = 90

	m.InvalidateThemeCaches()

	if m.barModels != nil {
		t.Fatal("barModels was not cleared")
	}
	if m.barModelW != 0 {
		t.Fatal("barModelW was not reset")
	}
}
