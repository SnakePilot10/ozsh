package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
)

func TestFullscreenPanelUsesAvailableHeight(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 72, 32

	view := model.View()
	if got := lipgloss.Height(view); got != model.height {
		t.Fatalf("panel height = %d, want %d\n%s", got, model.height, plainText(view))
	}
	assertViewBounds(t, view, model.width, model.height)
}

func TestFooterIsAnchoredNearBottom(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 72, 32

	lines := strings.Split(plainText(model.View()), "\n")
	start := len(lines) - 5
	if start < 0 {
		start = 0
	}
	bottom := strings.Join(lines[start:], "\n")
	if !strings.Contains(strings.ToLower(bottom), "apply") || !strings.Contains(strings.ToLower(bottom), "quit") {
		t.Fatalf("footer is not anchored near the bottom:\n%s", bottom)
	}
}

func TestCompactFullscreenLayoutStaysBounded(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 58, 28
	model.setTab(tabPrompt)

	view := model.View()
	if got := lipgloss.Height(view); got != model.height {
		t.Fatalf("compact panel height = %d, want %d\n%s", got, model.height, plainText(view))
	}
	assertViewBounds(t, view, model.width, model.height)
}

func TestShortFullscreenLayoutStaysBounded(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 72, 18
	model.setTab(tabPrompt)

	view := model.View()
	assertViewBounds(t, view, model.width, model.height)
}

func assertViewBounds(t *testing.T, view string, width, height int) {
	t.Helper()
	if got := lipgloss.Height(view); got > height {
		t.Fatalf("view height = %d, exceeds %d\n%s", got, height, plainText(view))
	}
	for index, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line %d width = %d, exceeds %d: %q", index, got, width, plainText(line))
		}
	}
}
