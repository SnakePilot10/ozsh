package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

func TestCustomPluginToggleRemainsPending(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "demo.plugin.zsh")
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())

	model.togglePluginAtCursor()

	if model.cfg.Plugins.Items[0].Enabled {
		t.Fatal("plugin remained enabled")
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("toggle saved before Apply: %v", err)
	}
	if !strings.Contains(model.msg, "pending") {
		t.Fatalf("toggle message = %q", model.msg)
	}
}

func TestInstalledPluginRemovalIsQueuedWithoutDeletingDirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "demo.plugin.zsh")
	root := cfg.Plugins.Items[0].Source
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("custom item unavailable")
	}

	if err := model.queueCustomPluginRemoval(item); err != nil {
		t.Fatalf("queueCustomPluginRemoval() error = %v", err)
	}

	if len(model.cfg.Plugins.Items) != 0 {
		t.Fatalf("pending config still contains plugin: %#v", model.cfg.Plugins.Items)
	}
	adds, removes := model.pluginChanges.Counts()
	if adds != 0 || removes != 1 {
		t.Fatalf("pending counts = %d add, %d remove", adds, removes)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("installed directory changed before Apply: %v", err)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("removal saved before Apply: %v", err)
	}
}

func TestRemovingUnappliedAddCleansStagingAndPendingEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	cfg := config.Default()
	var changes plugins.ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	model := NewModel(cfg)
	model.pluginChanges = changes
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("pending custom item unavailable")
	}

	if err := model.queueCustomPluginRemoval(item); err != nil {
		t.Fatalf("queueCustomPluginRemoval() error = %v", err)
	}

	if !model.pluginChanges.Empty() || len(model.cfg.Plugins.Items) != 0 {
		t.Fatalf("pending add was not cancelled: changes=%#v items=%#v", model.pluginChanges, model.cfg.Plugins.Items)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("staging checkout remains: %v", err)
	}
}

func TestChangeLoadFileUsesInstalledRootAndStaysPending(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "old.zsh", "demo.plugin.zsh")
	cfg.Plugins.Items[0].Load = "old.zsh"
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("custom item unavailable")
	}

	if err := model.openLoadFilePicker(item); err != nil {
		t.Fatalf("openLoadFilePicker() error = %v", err)
	}
	if model.pluginWizard.Mode != pluginWizardChangeLoad || model.pluginWizard.Step != pluginWizardCandidates {
		t.Fatalf("wizard = %#v", model.pluginWizard)
	}
	candidateIndex := lifecycleCandidateIndex(t, model.pluginWizard.Candidates, "demo.plugin.zsh")
	model.pluginWizard.Candidate = candidateIndex

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.pluginWizard.Step != pluginWizardSummary {
		t.Fatalf("step after candidate = %d", model.pluginWizard.Step)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.pluginWizard.Step != pluginWizardClosed {
		t.Fatalf("final wizard step = %d", model.pluginWizard.Step)
	}
	if model.cfg.Plugins.Items[0].Load != "demo.plugin.zsh" {
		t.Fatalf("pending load = %q", model.cfg.Plugins.Items[0].Load)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("load change saved before Apply: %v", err)
	}
}

func TestStagedPluginTrustValidatesAgainstStagingRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	cfg := config.Default()
	var changes plugins.ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	cfg.Plugins.Items[0].Trusted = false
	model := NewModel(cfg)
	model.pluginChanges = changes
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("pending custom item unavailable")
	}

	if err := model.trustCustomPlugin(item, true); err != nil {
		t.Fatalf("trustCustomPlugin() error = %v", err)
	}
	if !model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("staged plugin did not become trusted")
	}
	if _, err := os.Stat(stage.FinalDir); !os.IsNotExist(err) {
		t.Fatalf("trust unexpectedly required or created final path: %v", err)
	}
	_ = model.pluginChanges.Cleanup()
}

func TestInstalledPluginTrustValidatesAgainstFinalRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, false, "demo.plugin.zsh")
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("custom item unavailable")
	}

	if err := model.trustCustomPlugin(item, true); err != nil {
		t.Fatalf("trustCustomPlugin() error = %v", err)
	}
	if !model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("installed plugin did not become trusted")
	}
}

func TestUntrustingCustomPluginDisablesPendingActivation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "demo.plugin.zsh")
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("custom item unavailable")
	}

	if err := model.trustCustomPlugin(item, false); err != nil {
		t.Fatalf("trustCustomPlugin(false) error = %v", err)
	}
	if model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("plugin remained trusted")
	}
	if customPluginActive(model.cfg.Plugins.Items[0]) {
		t.Fatal("untrusted plugin remained active")
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("untrust saved before Apply: %v", err)
	}
}

func TestCustomPluginRemoveConfirmationQueuesOnApproval(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "demo.plugin.zsh")
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if !model.pluginRemoveConfirm || model.pluginRemoveName != "demo" {
		t.Fatalf("remove confirmation = %v name=%q", model.pluginRemoveConfirm, model.pluginRemoveName)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(Model)
	if model.pluginRemoveConfirm || model.pluginChanges.Empty() {
		t.Fatalf("remove confirmation remained or no change queued: confirm=%v changes=%#v", model.pluginRemoveConfirm, model.pluginChanges)
	}
	_, removes := model.pluginChanges.Counts()
	if removes != 1 {
		t.Fatalf("remove count = %d", removes)
	}
}

func TestRecommendedPluginCannotBeRemoved(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.cursor = 0

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := updated.(Model)
	if got.pluginRemoveConfirm {
		t.Fatal("recommended plugin opened removal confirmation")
	}
	if !strings.Contains(got.msg, "deselect") {
		t.Fatalf("message = %q", got.msg)
	}
}

func TestCustomPluginDetailsListLifecycleActions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := lifecycleCustomPluginConfig(t, "demo", true, true, "demo.plugin.zsh")
	model := NewModel(cfg)
	model.width, model.height = 100, 34
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())
	plain := plainText(model.selectedPluginPanel())

	for _, expected := range []string{
		"[space] Enable/disable",
		"[t/u] Trust/remove trust",
		"[l] Change load file",
		"[d] Remove plugin",
	} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("custom details lost %q:\n%s", expected, plain)
		}
	}
}

func lifecycleCustomPluginConfig(t *testing.T, name string, enabled, trusted bool, loads ...string) *config.Config {
	t.Helper()
	root := filepath.Join(plugins.Dir(), name)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, load := range loads {
		path := filepath.Join(root, filepath.FromSlash(load))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# plugin\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	load := ""
	if len(loads) > 0 {
		load = loads[0]
	}
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name: name, Enabled: enabled, Trusted: trusted, Source: root, Load: load,
	})
	return cfg
}

func lifecycleCandidateIndex(t *testing.T, candidates []plugins.Candidate, relativePath string) int {
	t.Helper()
	for index, candidate := range candidates {
		if candidate.RelativePath == relativePath {
			return index
		}
	}
	t.Fatalf("candidate %q not found in %#v", relativePath, candidates)
	return -1
}
