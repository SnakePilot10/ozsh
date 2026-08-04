package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

type wizardCloneRunner func(context.Context, string, string) error

func (runner wizardCloneRunner) Clone(ctx context.Context, repositoryURL, destination string) error {
	return runner(ctx, repositoryURL, destination)
}

func TestPluginWizardOpensFromPluginsScreen(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardURL || !got.pluginWizard.URL.Focused() {
		t.Fatalf("wizard = %#v", got.pluginWizard)
	}
	if got.confirmApply {
		t.Fatal("plain a opened Review & Apply on Plugins")
	}
}

func TestPluginWizardInvalidURLStaysOnURLStep(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.URL.SetValue("http://example.com/demo.git")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if cmd != nil {
		t.Fatal("invalid URL returned a command")
	}
	if got.pluginWizard.Step != pluginWizardURL {
		t.Fatalf("step = %d", got.pluginWizard.Step)
	}
	if !strings.Contains(got.pluginWizard.Error, "https") {
		t.Fatalf("error = %q", got.pluginWizard.Error)
	}
}

func TestPluginWizardStartsCloneAndAcceptsCurrentResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.URL.SetValue("https://example.com/demo.git")
	model.pluginCloneRunner = wizardCloneRunner(func(_ context.Context, _, destination string) error {
		return os.WriteFile(filepath.Join(destination, "demo.plugin.zsh"), []byte("# plugin\n"), 0o600)
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cloning := updated.(Model)
	if cloning.pluginWizard.Step != pluginWizardCloning || cmd == nil {
		t.Fatalf("step=%d cmd=%v", cloning.pluginWizard.Step, cmd)
	}
	result, ok := cmd().(pluginStageResult)
	if !ok {
		t.Fatalf("command result = %T", cmd())
	}
	updated, _ = cloning.Update(result)
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardCandidates || got.pluginWizard.Stage == nil {
		t.Fatalf("wizard = %#v", got.pluginWizard)
	}
	if len(got.pluginWizard.Stage.Candidates) != 1 || got.pluginWizard.Stage.Candidates[0].Path != "demo.plugin.zsh" {
		t.Fatalf("candidates = %#v", got.pluginWizard.Stage.Candidates)
	}
	_ = got.pluginWizard.Stage.Cleanup()
}

func TestPluginWizardCloneCancellationCallsCancelAndIgnoresResult(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Step = pluginWizardCloning
	cancelled := false
	model.pluginWizard.Cancel = func() { cancelled = true }
	model.pluginWizard.RequestID = 7

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if !cancelled {
		t.Fatal("clone cancel function was not called")
	}
	if got.pluginWizard.Step != pluginWizardClosed {
		t.Fatalf("step = %d", got.pluginWizard.Step)
	}
	if got.pluginWizard.RequestID == 7 {
		t.Fatal("request ID was not invalidated")
	}
}

func TestPluginWizardStaleResultIsCleaned(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "stale", "stale.plugin.zsh")
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Step = pluginWizardCloning
	model.pluginWizard.RequestID = 3

	updated, _ := model.Update(pluginStageResult{RequestID: 2, Stage: stage})
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardCloning || got.pluginWizard.Stage != nil {
		t.Fatalf("stale result changed wizard: %#v", got.pluginWizard)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("stale stage remains: %v", err)
	}
}

func TestPluginWizardCurrentCloneErrorReturnsToURL(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Step = pluginWizardCloning
	model.pluginWizard.RequestID = 4

	updated, _ := model.Update(pluginStageResult{RequestID: 4, Err: errors.New("network unavailable")})
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardURL || !got.pluginWizard.URL.Focused() {
		t.Fatalf("wizard = %#v", got.pluginWizard)
	}
	if !strings.Contains(got.pluginWizard.Error, "network unavailable") {
		t.Fatalf("error = %q", got.pluginWizard.Error)
	}
}

func TestPluginWizardTransitionsQueueTrustedPendingAdd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Step = pluginWizardCandidates
	model.pluginWizard.Stage = &stage

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	trust := updated.(Model)
	if trust.pluginWizard.Step != pluginWizardTrust {
		t.Fatalf("step after candidate = %d", trust.pluginWizard.Step)
	}
	updated, _ = trust.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	summary := updated.(Model)
	if summary.pluginWizard.Step != pluginWizardSummary {
		t.Fatalf("step after trust = %d", summary.pluginWizard.Step)
	}
	updated, _ = summary.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardClosed {
		t.Fatalf("final step = %d", got.pluginWizard.Step)
	}
	adds, removes := got.pluginChanges.Counts()
	if adds != 1 || removes != 0 {
		t.Fatalf("pending counts = %d add, %d remove", adds, removes)
	}
	if len(got.cfg.Plugins.Items) != 1 {
		t.Fatalf("plugins = %#v", got.cfg.Plugins.Items)
	}
	item := got.cfg.Plugins.Items[0]
	if item.Name != "demo" || !item.Trusted || !item.Enabled || item.Load != "demo.plugin.zsh" {
		t.Fatalf("queued plugin = %#v", item)
	}
	if !strings.Contains(got.msg, "Review & Apply") {
		t.Fatalf("message = %q", got.msg)
	}
	if _, err := os.Stat(stage.StagingDir); err != nil {
		t.Fatalf("queued stage was cleaned too early: %v", err)
	}
	_ = got.pluginChanges.Cleanup()
}

func TestPluginWizardCandidatesEscapeCleansStageAndReturnsURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Step = pluginWizardCandidates
	model.pluginWizard.Stage = &stage

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.pluginWizard.Step != pluginWizardURL || !got.pluginWizard.URL.Focused() {
		t.Fatalf("wizard = %#v", got.pluginWizard)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("stage remains after escape: %v", err)
	}
}

func TestPluginWizardViewsExposeSecurityAndPendingDetails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := wizardStageFixture(t, "demo", "demo.plugin.zsh")
	model := NewModel(config.Default())
	model.width, model.height = 58, 28
	model.setTab(tabPlugins)
	model.openPluginWizard()
	model.pluginWizard.Stage = &stage

	checks := []struct {
		step pluginWizardStep
		want []string
	}{
		{pluginWizardURL, []string{"Add custom plugin", "Repository URL"}},
		{pluginWizardCloning, []string{"Cloning repository", "Esc cancel"}},
		{pluginWizardCandidates, []string{"Choose load file", "demo.plugin.zsh"}},
		{pluginWizardTrust, []string{"Trust review", "executes shell code", stage.FinalDir}},
		{pluginWizardSummary, []string{"Pending plugin", "demo.plugin.zsh", "Review & Apply"}},
	}
	for _, check := range checks {
		model.pluginWizard.Step = check.step
		plain := plainText(model.View())
		for _, expected := range check.want {
			if !strings.Contains(plain, expected) {
				t.Fatalf("step %d lost %q:\n%s", check.step, expected, plain)
			}
		}
		assertViewBounds(t, model.View(), model.width, model.height)
	}
	_ = stage.Cleanup()
}

func wizardStageFixture(t *testing.T, name, load string) plugins.StagedRepository {
	t.Helper()
	root := plugins.Dir()
	if root == "" {
		t.Fatal("plugins.Dir() returned empty path")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	staging, err := os.MkdirTemp(root, ".staging-")
	if err != nil {
		t.Fatal(err)
	}
	loadPath := filepath.Join(staging, filepath.FromSlash(load))
	if err := os.MkdirAll(filepath.Dir(loadPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loadPath, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidates, err := plugins.DiscoverCandidates(staging, name, plugins.DefaultScanLimits)
	if err != nil {
		t.Fatal(err)
	}
	return plugins.StagedRepository{
		Repository: plugins.Repository{URL: "https://example.com/" + name + ".git", Name: name},
		StagingDir: staging,
		FinalDir:   filepath.Join(root, name),
		Candidates: candidates,
	}
}
