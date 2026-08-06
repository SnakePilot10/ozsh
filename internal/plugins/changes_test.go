package plugins

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func stagedFixture(t *testing.T, name, load string) StagedRepository {
	t.Helper()
	root := Dir()
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
	if err := os.Chmod(staging, 0o700); err != nil {
		t.Fatalf("Chmod(staging) error = %v", err)
	}
	writeCandidate(t, staging, load)
	candidates, err := DiscoverCandidates(staging, name, DefaultScanLimits)
	if err != nil {
		t.Fatalf("DiscoverCandidates() error = %v", err)
	}
	return StagedRepository{
		Repository: Repository{URL: "https://example.com/" + name + ".git", Name: name},
		StagingDir: staging,
		FinalDir:   filepath.Join(root, name),
		Candidates: candidates,
	}
}

func TestChangeSetQueueAddCreatesTrustedPendingItem(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := stagedFixture(t, "demo", "demo.plugin.zsh")
	cfg := config.Default()
	var changes ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatalf("QueueAdd() error = %v", err)
	}
	item := cfg.Plugins.Items[len(cfg.Plugins.Items)-1]
	if item.Name != "demo" || item.Source != stage.FinalDir || !item.Enabled || !item.Trusted {
		t.Fatalf("pending item = %#v", item)
	}
	if item.Load != "demo.plugin.zsh" {
		t.Fatalf("pending load = %q", item.Load)
	}
	adds, removes := changes.Counts()
	if adds != 1 || removes != 0 || changes.Empty() {
		t.Fatalf("counts = (%d, %d), empty=%v", adds, removes, changes.Empty())
	}
	if got, ok := changes.RepositoryURLFor("demo"); !ok || got != stage.Repository.URL {
		t.Fatalf("RepositoryURLFor() = %q, %v", got, ok)
	}
	if got := changes.RootFor("demo", stage.FinalDir); got != stage.StagingDir {
		t.Fatalf("RootFor() = %q, want %q", got, stage.StagingDir)
	}
}

func TestQueueRemoveCancelsUnappliedAdd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stage := stagedFixture(t, "demo", "demo.plugin.zsh")
	cfg := config.Default()
	var changes ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	if err := changes.QueueRemove(cfg, "demo"); err != nil {
		t.Fatal(err)
	}
	if !changes.Empty() || len(cfg.Plugins.Items) != 0 {
		t.Fatalf("changes=%#v items=%#v", changes, cfg.Plugins.Items)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("staging directory remains: %v", err)
	}
}

func TestTransactionRollbackRestoresAddAndRemoval(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := config.Default()

	installed := filepath.Join(Dir(), "old")
	if err := os.MkdirAll(installed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed, "old.zsh"), []byte("# old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name: "old", Enabled: true, Trusted: true, Source: installed, Load: "old.zsh",
	})

	stage := stagedFixture(t, "demo", "demo.plugin.zsh")
	var changes ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	if err := changes.QueueRemove(cfg, "old"); err != nil {
		t.Fatal(err)
	}

	transaction, err := changes.Begin(cfg)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if _, err := os.Stat(stage.FinalDir); err != nil {
		t.Fatalf("final add missing after Begin: %v", err)
	}
	if _, err := os.Stat(installed); !os.IsNotExist(err) {
		t.Fatalf("removal source still exists after Begin: %v", err)
	}

	if err := transaction.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if _, err := os.Stat(stage.StagingDir); err != nil {
		t.Fatalf("staging add not restored: %v", err)
	}
	if _, err := os.Stat(stage.FinalDir); !os.IsNotExist(err) {
		t.Fatalf("final add remains after rollback: %v", err)
	}
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("removed plugin not restored: %v", err)
	}
	if err := transaction.Commit(); err == nil || !strings.Contains(err.Error(), "transaction already completed") {
		t.Fatalf("Commit() after Rollback error = %v", err)
	}
}

func TestTransactionCommitFinalizesChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := config.Default()
	installed := filepath.Join(Dir(), "old")
	if err := os.MkdirAll(installed, 0o700); err != nil {
		t.Fatal(err)
	}
	writeCandidate(t, installed, "old.zsh")
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name: "old", Enabled: true, Trusted: true, Source: installed, Load: "old.zsh",
	})
	stage := stagedFixture(t, "demo", "demo.plugin.zsh")
	var changes ChangeSet
	if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
		t.Fatal(err)
	}
	if err := changes.QueueRemove(cfg, "old"); err != nil {
		t.Fatal(err)
	}
	transaction, err := changes.Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if _, err := os.Stat(stage.FinalDir); err != nil {
		t.Fatalf("final add missing: %v", err)
	}
	if _, err := os.Stat(installed); !os.IsNotExist(err) {
		t.Fatalf("removed plugin remains: %v", err)
	}
	matches, err := filepath.Glob(installed + ".ozsh-remove-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("quarantine remains: %v", matches)
	}
	if err := transaction.Rollback(); err == nil || !strings.Contains(err.Error(), "transaction already completed") {
		t.Fatalf("Rollback() after Commit error = %v", err)
	}
}

func TestChangeSetCloneIsIndependent(t *testing.T) {
	changes := ChangeSet{
		Adds:    []PendingAdd{{Name: "demo", Load: "demo.zsh"}},
		Removes: []PendingRemove{{Name: "old", Source: "/tmp/old"}},
	}
	clone := changes.Clone()
	clone.Adds[0].Load = "changed.zsh"
	clone.Removes[0].Source = "/tmp/changed"
	if changes.Adds[0].Load != "demo.zsh" || changes.Removes[0].Source != "/tmp/old" {
		t.Fatalf("Clone() shared backing arrays: %#v", changes)
	}
}

func TestQueueAddRejectsConflictsAndUnsafePaths(t *testing.T) {
	t.Run("duplicate add", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		cfg := config.Default()
		stage := stagedFixture(t, "demo", "demo.plugin.zsh")
		var changes ChangeSet
		if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
			t.Fatal(err)
		}
		other := stagedFixture(t, "demo", "demo.plugin.zsh")
		if err := changes.QueueAdd(cfg, other, "demo.plugin.zsh"); err == nil {
			t.Fatal("second QueueAdd() error = nil")
		}
		_ = other.Cleanup()
	})

	t.Run("pending removal name", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		cfg := config.Default()
		installed := filepath.Join(Dir(), "demo")
		if err := os.MkdirAll(installed, 0o700); err != nil {
			t.Fatal(err)
		}
		writeCandidate(t, installed, "demo.zsh")
		cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{Name: "demo", Source: installed, Load: "demo.zsh"})
		var changes ChangeSet
		if err := changes.QueueRemove(cfg, "demo"); err != nil {
			t.Fatal(err)
		}
		stage := stagedFixture(t, "demo", "demo.plugin.zsh")
		if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err == nil {
			t.Fatal("QueueAdd() during pending removal error = nil")
		}
		_ = stage.Cleanup()
	})

	t.Run("staging outside managed root", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		outside := t.TempDir()
		writeCandidate(t, outside, "demo.plugin.zsh")
		stage := StagedRepository{
			Repository: Repository{URL: "https://example.com/demo.git", Name: "demo"},
			StagingDir: outside,
			FinalDir:   filepath.Join(Dir(), "demo"),
		}
		var changes ChangeSet
		if err := changes.QueueAdd(config.Default(), stage, "demo.plugin.zsh"); err == nil {
			t.Fatal("QueueAdd() with external staging error = nil")
		}
	})

	t.Run("incorrect final path", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		stage := stagedFixture(t, "demo", "demo.plugin.zsh")
		stage.FinalDir = filepath.Join(Dir(), "other")
		var changes ChangeSet
		if err := changes.QueueAdd(config.Default(), stage, "demo.plugin.zsh"); err == nil {
			t.Fatal("QueueAdd() with incorrect final path error = nil")
		}
		_ = stage.Cleanup()
	})

	t.Run("symlink load target", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		stage := stagedFixture(t, "demo", "real.zsh")
		if err := os.Symlink(filepath.Join(stage.StagingDir, "real.zsh"), filepath.Join(stage.StagingDir, "demo.plugin.zsh")); err != nil {
			t.Fatal(err)
		}
		var changes ChangeSet
		if err := changes.QueueAdd(config.Default(), stage, "demo.plugin.zsh"); err == nil {
			t.Fatal("QueueAdd() with symlink load error = nil")
		}
		_ = stage.Cleanup()
	})
}

func TestQueueRemoveRejectsUnmanagedSource(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	outside := t.TempDir()
	cfg := config.Default()
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name: "demo", Source: outside, Load: "demo.zsh", Enabled: true, Trusted: true,
	})
	var changes ChangeSet
	if err := changes.QueueRemove(cfg, "demo"); err == nil {
		t.Fatal("QueueRemove() with unmanaged source error = nil")
	}
}

func TestChangeSetCleanupJoinsErrorsAndRemovesValidStages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	valid := stagedFixture(t, "demo", "demo.plugin.zsh")
	changes := ChangeSet{Adds: []PendingAdd{
		{Name: "demo", StagingDir: valid.StagingDir},
		{Name: "unsafe", StagingDir: t.TempDir()},
	}}
	err := changes.Cleanup()
	if err == nil {
		t.Fatal("Cleanup() error = nil")
	}
	if _, statErr := os.Stat(valid.StagingDir); !os.IsNotExist(statErr) {
		t.Fatalf("valid staging remains: %v", statErr)
	}
	if !errors.Is(err, err) { // Ensure a joined error remains a normal non-nil error.
		t.Fatalf("unexpected cleanup error: %v", err)
	}
}
