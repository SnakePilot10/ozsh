package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func TestApplyConfigGenerateFailureChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Version = config.CurrentConfigVersion + 1

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "generate prompt") {
		t.Fatalf("ApplyConfig() error = %v, want generate error", err)
	}
	if _, statErr := os.Stat(config.Path()); !os.IsNotExist(statErr) {
		t.Fatalf("config was changed after generate failure: %v", statErr)
	}
	if _, statErr := os.Stat(shell.OmegaZshPath()); !os.IsNotExist(statErr) {
		t.Fatalf("omega was changed after generate failure: %v", statErr)
	}
	if data := readFile(t, shell.ZshrcPath()); data != "export EDITOR=vim\n" {
		t.Fatalf("zshrc changed after generate failure: %q", data)
	}
}

func TestApplyConfigWriteOmegaFailureRestoresSavedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	original := config.Default()
	original.Prompt.Separator = " before "
	if err := config.Save(original); err != nil {
		t.Fatalf("config.Save(original) error = %v", err)
	}
	before := readFile(t, config.Path())
	if err := os.MkdirAll(shell.OmegaZshPath(), 0o700); err != nil {
		t.Fatalf("MkdirAll(omega path as dir) error = %v", err)
	}
	cfg := cloneConfig(original)
	cfg.Prompt.Separator = " after "

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "write omega.zsh") {
		t.Fatalf("ApplyConfig() error = %v, want write omega failure", err)
	}
	if got := readFile(t, config.Path()); got != before {
		t.Fatalf("config not restored after omega failure:\n%s", got)
	}
	if data := readFile(t, shell.ZshrcPath()); data != "export EDITOR=vim\n" {
		t.Fatalf("zshrc changed after omega failure: %q", data)
	}
	info, statErr := os.Stat(shell.OmegaZshPath())
	if statErr != nil || !info.IsDir() {
		t.Fatalf("omega sentinel directory not restored: info=%v err=%v", info, statErr)
	}
}

func TestApplyConfigPreflightFailureChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	malformed := "export EDITOR=vim\n# >>> ozsh >>>\nstale\n"
	if err := os.WriteFile(shell.ZshrcPath(), []byte(malformed), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = " | "

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "preflight .zshrc") {
		t.Fatalf("ApplyConfig() error = %v, want preflight failure", err)
	}
	if _, statErr := os.Stat(config.Path()); !os.IsNotExist(statErr) {
		t.Fatalf("config changed after preflight failure: %v", statErr)
	}
	if _, statErr := os.Stat(shell.OmegaZshPath()); !os.IsNotExist(statErr) {
		t.Fatalf("omega changed after preflight failure: %v", statErr)
	}
	if data := readFile(t, shell.ZshrcPath()); data != malformed {
		t.Fatalf("zshrc changed after inject failure: %q", data)
	}
}

func TestApplyFinalizesPendingPluginAdd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request, stage := pendingAddRequestFixture(t, "demo", "demo.plugin.zsh")
	if err := Apply(request); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("staging remains after Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stage.FinalDir, "demo.plugin.zsh")); err != nil {
		t.Fatalf("final plugin load file missing: %v", err)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if len(saved.Plugins.Items) != 1 || saved.Plugins.Items[0].Source != stage.FinalDir || !saved.Plugins.Items[0].Trusted {
		t.Fatalf("saved plugins = %#v", saved.Plugins.Items)
	}
}

func TestApplyOmegaFailureRestoresFilesAndPluginPaths(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := config.Default()
	installed := filepath.Join(plugins.Dir(), "old")
	if err := os.MkdirAll(installed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed, "old.zsh"), []byte("# old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original.Plugins.Items = append(original.Plugins.Items, config.PluginItem{
		Name: "old", Enabled: true, Trusted: true, Source: installed, Load: "old.zsh",
	})
	if err := config.Save(original); err != nil {
		t.Fatal(err)
	}
	beforeConfig := readFile(t, config.Path())
	beforeZshrc := readFile(t, shell.ZshrcPath())
	if err := os.MkdirAll(shell.OmegaZshPath(), 0o700); err != nil {
		t.Fatal(err)
	}

	cfg := cloneConfig(original)
	stage := stagedPluginFixture(t, "demo", "demo.plugin.zsh")
	var changes plugins.ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	if err := changes.QueueRemove(cfg, "old"); err != nil {
		t.Fatal(err)
	}

	err := Apply(Request{Config: cfg, PluginChanges: changes})
	if err == nil || !strings.Contains(err.Error(), "write omega.zsh") {
		t.Fatalf("Apply() error = %v, want write omega failure", err)
	}
	if got := readFile(t, config.Path()); got != beforeConfig {
		t.Fatalf("config changed after rollback:\n%s", got)
	}
	if got := readFile(t, shell.ZshrcPath()); got != beforeZshrc {
		t.Fatalf("zshrc changed after rollback: %q", got)
	}
	if info, statErr := os.Stat(shell.OmegaZshPath()); statErr != nil || !info.IsDir() {
		t.Fatalf("omega sentinel not restored: info=%v err=%v", info, statErr)
	}
	if _, statErr := os.Stat(stage.StagingDir); statErr != nil {
		t.Fatalf("staging add not restored: %v", statErr)
	}
	if _, statErr := os.Stat(stage.FinalDir); !os.IsNotExist(statErr) {
		t.Fatalf("final add remains after rollback: %v", statErr)
	}
	if _, statErr := os.Stat(installed); statErr != nil {
		t.Fatalf("removed plugin not restored: %v", statErr)
	}
	matches, globErr := filepath.Glob(installed + ".ozsh-remove-*")
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("removal quarantine remains: %v", matches)
	}
}

func pendingAddRequestFixture(t *testing.T, name, load string) (Request, plugins.StagedRepository) {
	t.Helper()
	stage := stagedPluginFixture(t, name, load)
	cfg := config.Default()
	var changes plugins.ChangeSet
	if err := changes.QueueAdd(cfg, stage, load); err != nil {
		t.Fatalf("QueueAdd() error = %v", err)
	}
	return Request{Config: cfg, PluginChanges: changes}, stage
}

func stagedPluginFixture(t *testing.T, name, load string) plugins.StagedRepository {
	t.Helper()
	root := plugins.Dir()
	if root == "" {
		t.Fatal("plugins.Dir() returned empty path")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin root) error = %v", err)
	}
	staging, err := os.MkdirTemp(root, ".staging-")
	if err != nil {
		t.Fatalf("MkdirTemp(staging) error = %v", err)
	}
	loadPath := filepath.Join(staging, filepath.FromSlash(load))
	if err := os.MkdirAll(filepath.Dir(loadPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(load parent) error = %v", err)
	}
	if err := os.WriteFile(loadPath, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(load) error = %v", err)
	}
	candidates, err := plugins.DiscoverCandidates(staging, name, plugins.DefaultScanLimits)
	if err != nil {
		t.Fatalf("DiscoverCandidates() error = %v", err)
	}
	return plugins.StagedRepository{
		Repository: plugins.Repository{URL: "https://example.com/" + name + ".git", Name: name},
		StagingDir: staging,
		FinalDir:   filepath.Join(root, name),
		Candidates: candidates,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
