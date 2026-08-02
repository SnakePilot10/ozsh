package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestV2UsesFiveFocusedScreens(t *testing.T) {
	want := []string{"Home", "Prompt", "Themes", "Plugins", "Preview"}
	if len(tabs) != len(want) {
		t.Fatalf("tabs = %#v, want %#v", tabs, want)
	}
	for i := range want {
		if tabs[i] != want[i] {
			t.Fatalf("tabs[%d] = %q, want %q", i, tabs[i], want[i])
		}
	}
}

func TestV2NarrowViewKeepsPrimaryNavigation(t *testing.T) {
	model := NewModel(config.Default())
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 54, Height: 28})
	view := updated.(Model).View()
	for _, label := range []string{"Home", "Prompt", "Themes", "Plugins", "Preview"} {
		if !strings.Contains(view, label) {
			t.Fatalf("narrow view missing %q:\n%s", label, view)
		}
	}
}

func TestReviewApplyCapturesIndependentSnapshot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.openApplyReview()
	if !model.confirmApply || model.reviewedConfig == nil {
		t.Fatal("openApplyReview() did not create a review snapshot")
	}
	model.cfg.Prompt.Symbol = "$"
	if model.reviewedConfig.Prompt.Symbol == "$" {
		t.Fatal("review snapshot aliases the editable configuration")
	}
}
