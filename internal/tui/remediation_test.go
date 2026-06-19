package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
)

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestTabAwareCursorClampsPerScreen(t *testing.T) {
	model := NewModel(config.Default())
	model.cursor = 999
	model.setTab(tabThemes)
	if got, want := model.cursor, len(sortedThemeNames())-1; got != want {
		t.Fatalf("theme cursor = %d, want %d", got, want)
	}
	model.setTab(tabHeaders)
	if got, want := model.cursor, 0; got != want {
		t.Fatalf("header cursor = %d, want %d", got, want)
	}
	model.setTab(tabPlugins)
	if model.cursor != 0 {
		t.Fatalf("plugin cursor = %d, want zero with empty list", model.cursor)
	}
	model.moveCursor(1)
	if model.cursor != 0 {
		t.Fatalf("plugin cursor moved in an empty list: %d", model.cursor)
	}
}

func TestPreviewRoutesInputOnlyToFocusedField(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	beforeCwd := model.inputs[1].Value()

	updated, _ := model.Update(keyRune('x'))
	model = updated.(Model)
	if !strings.HasSuffix(model.inputs[0].Value(), "x") {
		t.Fatalf("username input was not edited: %q", model.inputs[0].Value())
	}
	if model.inputs[1].Value() != beforeCwd {
		t.Fatalf("unfocused cwd input changed: %q", model.inputs[1].Value())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('z'))
	model = updated.(Model)
	if !strings.HasSuffix(model.inputs[1].Value(), "z") {
		t.Fatalf("focused cwd input was not edited: %q", model.inputs[1].Value())
	}
}

func TestPreviewAllowsNumericExitAndDisplaysValidationError(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	model.inputFocus = 3
	model.focusPreviewInput()
	model.inputs[3].SetValue("")

	updated, _ := model.Update(keyRune('7'))
	model = updated.(Model)
	if model.previewCtx.ExitStatus != 7 {
		t.Fatalf("exit status = %d, want 7", model.previewCtx.ExitStatus)
	}

	updated, _ = model.Update(keyRune('x'))
	model = updated.(Model)
	if model.previewError == "" || !strings.Contains(model.preview(), "exit status must be an integer") {
		t.Fatalf("invalid exit status did not surface preview error: %q", model.previewError)
	}
}

func TestPluginFormSeparatesURLAndLoadFocus(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)

	updated, _ := model.Update(keyRune('x'))
	model = updated.(Model)
	if model.pluginURL.Value() != "x" || model.pluginLoad.Value() != "" {
		t.Fatalf("URL focus routing failed: url=%q load=%q", model.pluginURL.Value(), model.pluginLoad.Value())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('p'))
	model = updated.(Model)
	if model.pluginLoad.Value() != "p" || model.pluginURL.Value() != "x" {
		t.Fatalf("load focus routing failed: url=%q load=%q", model.pluginURL.Value(), model.pluginLoad.Value())
	}
}

func TestCloneConfigOwnsMapsSlicesAndPluginItems(t *testing.T) {
	cfg := config.Default()
	clone := cloneConfig(cfg)
	clone.Prompt.Order[0] = "time"
	clone.Prompt.Segments["user"] = config.SegmentConfig{Enabled: false}
	clone.Plugins.Items = append(clone.Plugins.Items, config.PluginItem{Name: "demo"})

	if cfg.Prompt.Order[0] == "time" {
		t.Fatal("cloneConfig shared prompt order")
	}
	if !cfg.Prompt.Segments["user"].Enabled {
		t.Fatal("cloneConfig shared segment map")
	}
	if len(cfg.Plugins.Items) != 0 {
		t.Fatal("cloneConfig shared plugin slice")
	}
}

func TestWrapIndexAndPluginHelpView(t *testing.T) {
	if wrapIndex(-1, 4) != 3 || wrapIndex(5, 4) != 1 || wrapIndex(3, 0) != 0 {
		t.Fatal("wrapIndex produced unexpected values")
	}
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	view := model.plugins()
	if !strings.Contains(view, "add plugin") || !strings.Contains(view, "tab switches form field") {
		t.Fatalf("plugin view missing form guidance:\n%s", view)
	}
}
