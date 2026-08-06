package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
)

func TestRenderKeyValueSeparatesFullWidthLabel(t *testing.T) {
	got := plainText(renderKeyValue("Right prompt", "Enabled"))
	if got != "Right prompt Enabled" {
		t.Fatalf("renderKeyValue() = %q, want an explicit gap", got)
	}
}

func TestPreviewScenarioNavigationEscapesContextEditor(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 100, 34
	model.setTab(tabPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.previewScenario != 2 {
		t.Fatalf("down in scenario mode selected scenario %d, want 2", model.previewScenario)
	}
	if model.inputFocus != 0 {
		t.Fatalf("down in scenario mode moved input focus to %d", model.inputFocus)
	}
	if plain := plainText(model.View()); !strings.Contains(plain, "[x] Command failed") {
		t.Fatalf("selected scenario missing:\n%s", plain)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if !model.previewEditing {
		t.Fatal("enter did not open context edit mode")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	if !model.previewCustom {
		t.Fatal("typing in context edit mode did not create a custom context")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.previewEditing {
		t.Fatal("esc did not return to scenario navigation")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.previewScenario != 3 {
		t.Fatalf("down after leaving editor selected scenario %d, want 3", model.previewScenario)
	}
	if model.previewCustom {
		t.Fatal("selecting a preset after leaving editor kept custom context active")
	}
}

func TestFontProgressUpdatesBusyWorkspace(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 58, 28
	model.busy = true
	model.operation = "font"
	model.fontName = "JetBrainsMono"
	model.fontEvents = make(chan fontInstallEvent)

	updated, cmd := model.Update(fontProgressMsg{Downloaded: 25, Total: 100})
	model = updated.(Model)
	if model.fontDownloaded != 25 || model.fontTotal != 100 {
		t.Fatalf("font progress = %d/%d, want 25/100", model.fontDownloaded, model.fontTotal)
	}
	if cmd == nil {
		t.Fatal("font progress did not continue listening for install events")
	}
	plain := plainText(model.View())
	for _, expected := range []string{"JetBrainsMono", "25%", "25 B / 100 B"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("font progress view lost %q:\n%s", expected, plain)
		}
	}
}
