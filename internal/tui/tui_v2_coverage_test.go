package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/fonts"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

func TestViewCoversBusyDialogsAndMessages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupDir := shell.BackupsDir()
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(backupDir, "zshrc-test.bak")
	if err := os.WriteFile(backup, []byte("setopt promptsubst\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := NewModel(config.Default())
	base.width = 54
	base.height = 24

	tests := []struct {
		name   string
		mutate func(*Model)
		want   string
	}{
		{name: "plugins busy", mutate: func(m *Model) { m.busy, m.operation = true, "plugins" }, want: "Installing plugins"},
		{name: "font busy", mutate: func(m *Model) { m.busy, m.operation = true, "font" }, want: "Installing Nerd Font"},
		{name: "font restore busy", mutate: func(m *Model) { m.busy, m.operation = true, "font-restore" }, want: "Restoring previous"},
		{name: "backup busy", mutate: func(m *Model) { m.busy, m.operation = true, "backup" }, want: "Restoring backup"},
		{name: "apply busy", mutate: func(m *Model) { m.busy, m.operation = true, "apply" }, want: "applying configuration"},
		{name: "font list", mutate: func(m *Model) { m.fontOpen = true }, want: "Nerd Fonts"},
		{name: "font confirm", mutate: func(m *Model) { m.fontOpen, m.confirmFont = true, true }, want: "Download and install"},
		{name: "font restore confirm", mutate: func(m *Model) { m.fontOpen, m.confirmFontRestore = true, true }, want: "Restore the previous Termux font"},
		{name: "backup list", mutate: func(m *Model) { m.backupOpen, m.backupPaths = true, []string{backup} }, want: "Choose a .zshrc snapshot"},
		{name: "backup confirm", mutate: func(m *Model) { m.backupOpen, m.confirmBackup, m.backupPaths = true, true, []string{backup} }, want: "Restore this backup"},
		{name: "plugin confirm", mutate: func(m *Model) { m.confirmPlugins = true }, want: "Install selected plugins"},
		{name: "doctor", mutate: func(m *Model) { m.doctorOpen = true }, want: "press enter to fix"},
		{name: "doctor confirm", mutate: func(m *Model) { m.doctorOpen, m.confirmDoctor = true, true }, want: "Run automatic repairs"},
		{name: "success message", mutate: func(m *Model) { m.msg = "saved" }, want: "saved"},
		{name: "error message", mutate: func(m *Model) { m.msg = "save error: nope" }, want: "save error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := base
			tc.mutate(&model)
			view := model.View()
			if !strings.Contains(view, tc.want) {
				t.Fatalf("View() missing %q:\n%s", tc.want, view)
			}
		})
	}

	base.openApplyReview()
	base.showApplyTechnical = true
	if view := base.View(); !strings.Contains(view, "Planned .zshrc diff") {
		t.Fatalf("technical apply review missing diff:\n%s", view)
	}
}

func TestUpdateHandlesOperationResults(t *testing.T) {
	base := NewModel(config.Default())
	base.busy = true
	base.operation = "test"

	cases := []struct {
		name string
		msg  tea.Msg
		want string
	}{
		{name: "apply", msg: applyResult("applied"), want: "applied"},
		{name: "plugin success", msg: pluginInstallResult{cfg: config.Default()}, want: "recommended plugins installed"},
		{name: "plugin error", msg: pluginInstallResult{err: errors.New("clone failed")}, want: "plugin install error"},
		{name: "font success", msg: fontInstallResult{cfg: config.Default(), font: fonts.Manifest()[0]}, want: "installed"},
		{name: "font error", msg: fontInstallResult{err: errors.New("checksum")}, want: "font install error"},
		{name: "backup success", msg: backupRestoreResult{}, want: "backup restored"},
		{name: "backup error", msg: backupRestoreResult{err: errors.New("missing")}, want: "backup restore error"},
		{name: "font restore success", msg: fontRestoreResult{}, want: "previous Termux font restored"},
		{name: "font restore error", msg: fontRestoreResult{err: errors.New("missing")}, want: "font restore error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updated, _ := base.Update(tc.msg)
			model := updated.(Model)
			if model.busy || model.operation != "" || !strings.Contains(model.msg, tc.want) {
				t.Fatalf("result state busy=%t operation=%q msg=%q", model.busy, model.operation, model.msg)
			}
		})
	}
}

func TestHomePromptThemeAndPluginKeyPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shell.BackupsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shell.BackupsDir(), "zshrc-one.bak"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	model := NewModel(config.Default())
	updated, _ := model.Update(runeKey('d'))
	model = updated.(Model)
	if !model.doctorOpen {
		t.Fatal("d did not open Doctor")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	updated, _ = model.Update(runeKey('f'))
	model = updated.(Model)
	if !model.fontOpen {
		t.Fatal("f did not open font dialog")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	updated, _ = model.Update(runeKey('r'))
	model = updated.(Model)
	if !model.backupOpen || len(model.backupPaths) != 1 {
		t.Fatalf("r backup state open=%t paths=%v", model.backupOpen, model.backupPaths)
	}

	model.backupOpen = false
	model.setTab(tabPrompt)
	originalHeavy := model.cfg.Prompt.DisableHeavySegments
	originalMode := model.cfg.Prompt.IconMode
	originalLayout := model.cfg.Prompt.Layout
	originalSymbol := model.cfg.Prompt.Symbol
	for _, key := range []rune{'h', 'i', 'l', 'o', 'v'} {
		updated, _ = model.Update(runeKey(key))
		model = updated.(Model)
	}
	if model.cfg.Prompt.DisableHeavySegments == originalHeavy || model.cfg.Prompt.IconMode == originalMode || model.cfg.Prompt.Layout == originalLayout || model.cfg.Prompt.Symbol == originalSymbol || !model.promptAdvanced {
		t.Fatalf("prompt keys did not update settings: %+v", model.cfg.Prompt)
	}
	updated, _ = model.Update(runeKey('u'))
	model = updated.(Model)
	if !model.promptEditingName {
		t.Fatal("u did not open display-name editor")
	}
	model.promptName.SetValue("pilot")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.cfg.Prompt.DisplayName != "pilot" || model.promptEditingName {
		t.Fatalf("display name state=%q editing=%t", model.cfg.Prompt.DisplayName, model.promptEditingName)
	}

	model.setTab(tabThemes)
	updated, _ = model.Update(runeKey(']'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('['))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.cfg.Theme.ID == "" {
		t.Fatal("theme enter did not stage a theme")
	}
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.cfg.Theme.ID != "custom" {
		t.Fatalf("custom theme ID = %q", model.cfg.Theme.ID)
	}

	model.setTab(tabPlugins)
	selectedBefore := len(model.cfg.Plugins.Selected)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if len(model.cfg.Plugins.Selected) == selectedBefore {
		t.Fatal("space did not toggle recommended plugin selection")
	}
	updated, _ = model.Update(runeKey('x'))
	model = updated.(Model)
	if !model.pluginAdvanced {
		t.Fatal("x did not open advanced plugin section")
	}
}

func TestFontBackupAndApplyDialogUpdates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(home, "backup.bak")
	if err := os.WriteFile(backup, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	model := NewModel(config.Default())
	model.fontOpen = true
	updated, _ := model.updateFontDialog(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.fontCursor != 1 {
		t.Fatalf("font cursor = %d, want 1", model.fontCursor)
	}
	updated, _ = model.updateFontDialog(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.updateFontDialog(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if !model.confirmFont {
		t.Fatal("font enter did not request confirmation")
	}
	updated, cmd := model.updateFontDialog(runeKey('y'))
	model = updated.(Model)
	if !model.busy || model.operation != "font" || cmd == nil {
		t.Fatalf("font confirm busy=%t operation=%q cmd=%v", model.busy, model.operation, cmd)
	}

	model = NewModel(config.Default())
	model.fontOpen = true
	model.confirmFontRestore = true
	updated, cmd = model.updateFontDialog(runeKey('y'))
	model = updated.(Model)
	if !model.busy || model.operation != "font-restore" || cmd == nil {
		t.Fatalf("font restore busy=%t operation=%q cmd=%v", model.busy, model.operation, cmd)
	}

	model = NewModel(config.Default())
	model.backupOpen = true
	model.backupPaths = []string{backup, backup + ".2"}
	updated, _ = model.updateBackupDialog(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.backupCursor != 1 {
		t.Fatalf("wrapped backup cursor = %d, want 1", model.backupCursor)
	}
	updated, _ = model.updateBackupDialog(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if !model.confirmBackup {
		t.Fatal("backup enter did not request confirmation")
	}
	updated, cmd = model.updateBackupDialog(runeKey('y'))
	model = updated.(Model)
	if !model.busy || model.operation != "backup" || cmd == nil {
		t.Fatalf("backup confirm busy=%t operation=%q cmd=%v", model.busy, model.operation, cmd)
	}

	model = NewModel(config.Default())
	model.openApplyReview()
	updated, _ = model.updateApplyModal(runeKey('t'))
	model = updated.(Model)
	if !model.showApplyTechnical {
		t.Fatal("t did not reveal technical details")
	}
	updated, _ = model.updateApplyModal(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.confirmApply || model.reviewedConfig != nil {
		t.Fatal("esc did not clear apply review")
	}

	model.confirmApply = true
	model.reviewedConfig = nil
	updated, _ = model.updateApplyModal(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.confirmApply || !strings.Contains(model.msg, "expired") {
		t.Fatalf("expired review state confirm=%t msg=%q", model.confirmApply, model.msg)
	}
}

func TestPreviewScenarioAndInputBranches(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	updated, _ := model.updatePreviewInputs(runeKey(']'))
	model = updated.(Model)
	if model.previewScenario != 2 {
		t.Fatalf("preview scenario = %d, want 2", model.previewScenario)
	}
	updated, _ = model.updatePreviewInputs(runeKey('['))
	model = updated.(Model)
	updated, _ = model.updatePreviewInputs(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.inputFocus != len(model.inputs)-1 {
		t.Fatalf("wrapped preview focus = %d", model.inputFocus)
	}
	model.setPreviewScenario(999)
	if model.previewScenario < 0 || model.previewScenario >= len(previewScenarioIDs()) {
		t.Fatalf("wrapped scenario = %d", model.previewScenario)
	}

	for _, id := range previewScenarioIDs() {
		ctx, ok := prompt.PreviewScenario(id)
		if !ok || ctx.Username == "" {
			t.Fatalf("PreviewScenario(%q) = %+v, %t", id, ctx, ok)
		}
	}
	if _, ok := prompt.PreviewScenario("unknown"); ok {
		t.Fatal("unknown preview scenario unexpectedly exists")
	}
}

func TestTUIHelperBranches(t *testing.T) {
	model := NewModel(config.Default())
	if model.contentWidth() != 76 {
		t.Fatalf("default content width = %d", model.contentWidth())
	}
	model.width = 2
	if model.contentWidth() != 1 {
		t.Fatalf("tiny content width = %d", model.contentWidth())
	}
	if got := fitBlock("abcdef", 3); got == "abcdef" || fitBlock("x", 0) != "" {
		t.Fatalf("fitBlock results long=%q zero=%q", got, fitBlock("x", 0))
	}
	if segmentLabel("unknown") != "unknown" {
		t.Fatal("unknown segment label changed")
	}
	segment := config.SegmentConfig{Icon: "legacy", CompatibleIcon: "ascii", NerdIcon: "nerd"}
	if promptSegmentIcon(config.PromptConfig{IconMode: config.IconModeNerd}, segment) != "nerd" || promptSegmentIcon(config.PromptConfig{IconMode: config.IconModeCompatible}, segment) != "ascii" {
		t.Fatal("promptSegmentIcon chose the wrong icon")
	}
	segment.CompatibleIcon = ""
	segment.NerdIcon = ""
	if promptSegmentIcon(config.PromptConfig{}, segment) != "legacy" {
		t.Fatal("promptSegmentIcon did not use legacy fallback")
	}

	model.setTab(tabThemes)
	model.cursor = len(themecatalog.List()) - 1
	start, end := model.visibleThemeRange(len(themecatalog.List()))
	if start < 0 || end != len(themecatalog.List()) {
		t.Fatalf("visible range = %d:%d", start, end)
	}
	if swatches := themeSwatches(themecatalog.List()[0]); !strings.Contains(swatches, "●") {
		t.Fatalf("theme swatches = %q", swatches)
	}
	model.cursor = -1
	if _, ok := model.selectedTheme(); ok {
		t.Fatal("selectedTheme accepted negative cursor")
	}

	if displayNameLabel("") != "system user" || displayNameLabel("pilot") != "pilot" {
		t.Fatal("displayNameLabel returned unexpected values")
	}
	if containsString([]string{"a", "b"}, "c") || !containsString([]string{"a", "b"}, "b") {
		t.Fatal("containsString returned unexpected result")
	}
	if got := removeString([]string{"a", "b", "a"}, "a"); len(got) != 1 || got[0] != "b" {
		t.Fatalf("removeString = %#v", got)
	}
	if len(sortedThemeNames()) != 12 {
		t.Fatalf("sortedThemeNames length = %d", len(sortedThemeNames()))
	}
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
