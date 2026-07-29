package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestAddRejectsNonHTTPSURL(t *testing.T) {
	cfg := config.Default()
	if _, err := Add(cfg, "http://example.com/plugin.git", "plugin.zsh"); err == nil {
		t.Fatal("Add() error = nil, want https error")
	}
}

func TestAddRejectsURLQueryBeforeClone(t *testing.T) {
	cfg := config.Default()
	if _, err := Add(cfg, "https://example.com/plugin.git?token=secret", "plugin.zsh"); err == nil {
		t.Fatal("Add() error = nil, want query rejection")
	}
	if _, err := Add(cfg, "https://example.com/plugin.git?", "plugin.zsh"); err == nil {
		t.Fatal("Add() error = nil, want force-query rejection")
	}
}

func TestAddRejectsMissingLoadBeforeClone(t *testing.T) {
	cfg := config.Default()
	if _, err := Add(cfg, "https://example.com/plugin.git", ""); err == nil {
		t.Fatal("Add() error = nil, want missing load error")
	}
}

func TestRemoveAndSetEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Enabled: true, Source: pluginDir, Load: "plugin.zsh"}}

	if err := SetEnabled(cfg, "demo", false); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	if cfg.Plugins.Items[0].Enabled {
		t.Fatal("SetEnabled() left plugin enabled")
	}
	if err := Remove(cfg, "demo"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(cfg.Plugins.Items) != 0 {
		t.Fatalf("Remove() plugin count = %d, want 0", len(cfg.Plugins.Items))
	}
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Fatalf("Remove() left plugin directory behind: %v", err)
	}
}

func TestSetEnabledRejectsPathLikeName(t *testing.T) {
	cfg := config.Default()
	if err := SetEnabled(cfg, "../demo", true); err == nil {
		t.Fatal("SetEnabled() error = nil, want invalid name error")
	}
}

func TestSetTrusted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	loadPath := filepath.Join(pluginDir, "plugin.zsh")
	if err := os.WriteFile(loadPath, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(plugin.zsh) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"}}

	if err := SetTrusted(cfg, "demo", true); err != nil {
		t.Fatalf("SetTrusted() error = %v", err)
	}
	if !cfg.Plugins.Items[0].Trusted {
		t.Fatal("SetTrusted() left plugin untrusted")
	}
}

func TestSetTrustedRejectsMissingLoadFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"}}

	if err := SetTrusted(cfg, "demo", true); err == nil {
		t.Fatal("SetTrusted() error = nil, want missing load file error")
	}
	if cfg.Plugins.Items[0].Trusted {
		t.Fatal("SetTrusted() trusted plugin with missing load file")
	}
}

func TestSetTrustedRejectsSymlinkLoadFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	target := filepath.Join(pluginDir, "target.zsh")
	if err := os.WriteFile(target, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(target.zsh) error = %v", err)
	}
	if err := os.Symlink(target, filepath.Join(pluginDir, "plugin.zsh")); err != nil {
		t.Fatalf("Symlink(plugin.zsh) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"}}

	if err := SetTrusted(cfg, "demo", true); err == nil {
		t.Fatal("SetTrusted() error = nil, want symlink load file error")
	}
	if cfg.Plugins.Items[0].Trusted {
		t.Fatal("SetTrusted() trusted plugin with symlink load file")
	}
}

func TestAddRejectsUnsafeLoadPathBeforeClone(t *testing.T) {
	cfg := config.Default()
	if _, err := Add(cfg, "https://example.com/plugin.git", "../plugin.zsh"); err == nil {
		t.Fatal("Add() error = nil, want unsafe load path error")
	}
}

func TestAddRejectsNonShellLoadFileBeforeClone(t *testing.T) {
	cfg := config.Default()
	if _, err := Add(cfg, "https://example.com/plugin.git", "README.md"); err == nil {
		t.Fatal("Add() error = nil, want non-shell load file error")
	}
}

func TestAddAndSaveRollsBackCloneOnSaveFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gitDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(fake git dir) error = %v", err)
	}
	fakeGit := filepath.Join(gitDir, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nif [ \"$1\" = clone ]; then mkdir -p \"$5\"; printf '# plugin\\n' > \"$5/plugin.zsh\"; else printf '0123456789abcdef0123456789abcdef01234567\\n'; fi\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(fake git) error = %v", err)
	}
	t.Setenv("PATH", gitDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.Default()
	cfg.Version = config.CurrentConfigVersion + 1
	_, err := AddAndSave(cfg, "https://example.com/demo.git", "plugin.zsh")
	if err == nil || !strings.Contains(err.Error(), "config could not be saved") {
		t.Fatalf("AddAndSave() error = %v, want save failure", err)
	}
	if len(cfg.Plugins.Items) != 0 {
		t.Fatalf("AddAndSave() left config items = %#v", cfg.Plugins.Items)
	}
	if _, statErr := os.Stat(filepath.Join(Dir(), "demo")); !os.IsNotExist(statErr) {
		t.Fatalf("AddAndSave() left cloned plugin dir: %v", statErr)
	}
}

func TestRemoveAndSaveRollsBackOnSaveFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	cfg := config.Default()
	cfg.Version = config.CurrentConfigVersion + 1
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: pluginDir, Load: "plugin.zsh"}}

	err := RemoveAndSave(cfg, "demo")
	if err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("RemoveAndSave() error = %v, want rollback error", err)
	}
	if len(cfg.Plugins.Items) != 1 {
		t.Fatalf("RemoveAndSave() config items = %#v", cfg.Plugins.Items)
	}
	if info, statErr := os.Stat(pluginDir); statErr != nil || !info.IsDir() {
		t.Fatalf("RemoveAndSave() did not restore plugin directory: %v", statErr)
	}
}

func TestSetTrustedRejectsIntermediateSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pluginDir := filepath.Join(Dir(), "demo")
	external := filepath.Join(home, "external")
	if err := os.MkdirAll(external, 0o700); err != nil {
		t.Fatalf("MkdirAll(external) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(external, "plugin.zsh"), []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(plugin) error = %v", err)
	}
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	if err := os.Symlink(external, filepath.Join(pluginDir, "nested")); err != nil {
		t.Fatalf("Symlink(nested) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: pluginDir, Load: "nested/plugin.zsh"}}
	if err := SetTrusted(cfg, "demo", true); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SetTrusted() error = %v, want intermediate symlink error", err)
	}
}

func TestRemoveAndSaveRemovesOrphanedConfigEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pluginDir := filepath.Join(Dir(), "missing")
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "missing", Source: pluginDir, Load: "plugin.zsh"}}
	if err := RemoveAndSave(cfg, "missing"); err != nil {
		t.Fatalf("RemoveAndSave(orphan) error = %v", err)
	}
	if len(cfg.Plugins.Items) != 0 {
		t.Fatalf("RemoveAndSave(orphan) left items = %#v", cfg.Plugins.Items)
	}
}
