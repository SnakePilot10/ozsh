package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func TestApplyReviewSnapshotsAndListsPendingPluginChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeReviewZshrc(t)
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	if err := model.pluginChanges.QueueAdd(model.cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}

	model.openApplyReview()

	adds, removes := model.reviewedPluginChanges.Counts()
	if adds != 1 || removes != 0 {
		t.Fatalf("reviewed counts = %d add, %d remove", adds, removes)
	}
	plain := plainText(model.pendingPluginReview())
	for _, expected := range []string{"Plugin changes", "1 add", "0 remove", "demo", "demo.plugin.zsh", stage.Repository.URL, stage.FinalDir} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("review lost %q:\n%s", expected, plain)
		}
	}

	model.pluginChanges.Adds[0].Load = "mutated.zsh"
	if model.reviewedPluginChanges.Adds[0].Load != "demo.plugin.zsh" {
		t.Fatalf("review snapshot mutated with live changes: %#v", model.reviewedPluginChanges.Adds)
	}
	_ = model.pluginChanges.Cleanup()
}

func TestApplyReviewCancellationRetainsPendingPluginChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeReviewZshrc(t)
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	if err := model.pluginChanges.QueueAdd(model.cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	model.openApplyReview()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)

	if got.confirmApply {
		t.Fatal("apply confirmation remained open")
	}
	if got.pluginChanges.Empty() {
		t.Fatal("cancelling Apply discarded pending plugin changes")
	}
	if !got.reviewedPluginChanges.Empty() {
		t.Fatalf("review snapshot was not cleared: %#v", got.reviewedPluginChanges)
	}
	if _, err := os.Stat(stage.StagingDir); err != nil {
		t.Fatalf("cancelling Apply removed staging checkout: %v", err)
	}
	_ = got.pluginChanges.Cleanup()
}

func TestSuccessfulPluginApplyClearsPendingChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeReviewZshrc(t)
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	if err := model.pluginChanges.QueueAdd(model.cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	model.openApplyReview()

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	applying := updated.(Model)
	if cmd == nil || !applying.busy {
		t.Fatalf("Apply did not start: busy=%v cmd=%v", applying.busy, cmd)
	}
	result, ok := cmd().(pluginApplyResult)
	if !ok {
		t.Fatal("Apply command returned unexpected result type")
	}
	if result.err != nil {
		t.Fatalf("Apply command error = %v", result.err)
	}
	updated, _ = applying.Update(result)
	got := updated.(Model)

	if !got.pluginChanges.Empty() {
		t.Fatalf("successful Apply retained pending changes: %#v", got.pluginChanges)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("successful Apply retained staging checkout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stage.FinalDir, "demo.plugin.zsh")); err != nil {
		t.Fatalf("successful Apply did not finalize plugin: %v", err)
	}
	if !strings.Contains(got.msg, "applied") {
		t.Fatalf("success message = %q", got.msg)
	}
}

func TestFailedPluginApplyRetainsPendingChangesAndStaging(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeReviewZshrc(t)
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	if err := model.pluginChanges.QueueAdd(model.cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shell.OmegaZshPath(), 0o700); err != nil {
		t.Fatal(err)
	}
	model.openApplyReview()

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	applying := updated.(Model)
	if cmd == nil {
		t.Fatal("Apply did not return a command")
	}
	result, ok := cmd().(pluginApplyResult)
	if !ok {
		t.Fatal("Apply command returned unexpected result type")
	}
	if result.err == nil {
		t.Fatal("Apply unexpectedly succeeded")
	}
	updated, _ = applying.Update(result)
	got := updated.(Model)

	if got.pluginChanges.Empty() {
		t.Fatal("failed Apply discarded pending plugin changes")
	}
	if _, err := os.Stat(stage.StagingDir); err != nil {
		t.Fatalf("failed Apply did not restore staging checkout: %v", err)
	}
	if _, err := os.Stat(stage.FinalDir); !os.IsNotExist(err) {
		t.Fatalf("failed Apply left final plugin path: %v", err)
	}
	if !strings.Contains(got.msg, "apply error") {
		t.Fatalf("failure message = %q", got.msg)
	}
	_ = got.pluginChanges.Cleanup()
}

func TestApplyReviewListsPendingRemoval(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeReviewZshrc(t)
	cfg := lifecycleCustomPluginConfig(t, "old", true, true, "old.zsh")
	removedPath := cfg.Plugins.Items[0].Source
	model := NewModel(cfg)
	if err := model.pluginChanges.QueueRemove(model.cfg, "old"); err != nil {
		t.Fatal(err)
	}
	model.openApplyReview()

	plain := plainText(model.pendingPluginReview())
	for _, expected := range []string{"Plugin changes", "0 add", "1 remove", "old", removedPath} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("removal review lost %q:\n%s", expected, plain)
		}
	}
}

func writeReviewZshrc(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
}
