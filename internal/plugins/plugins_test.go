package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestAddRejectsNonHTTPSURL(t *testing.T) {
	cfg := config.Default()

	if _, err := Add(cfg, "http://example.com/plugin.git", "plugin.zsh"); err == nil {
		t.Fatal("Add() error = nil, want https error")
	}
}

func TestRemoveAndSetEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Source: "/tmp/demo"},
	}

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
}

func TestSetEnabledRejectsPathLikeName(t *testing.T) {
	cfg := config.Default()

	if err := SetEnabled(cfg, "../demo", true); err == nil {
		t.Fatal("SetEnabled() error = nil, want invalid name error")
	}
}

func TestSetTrusted(t *testing.T) {
	pluginDir := t.TempDir()
	loadPath := filepath.Join(pluginDir, "plugin.zsh")
	if err := os.WriteFile(loadPath, []byte("# plugin\n"), 0644); err != nil {
		t.Fatalf("WriteFile(plugin.zsh) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"},
	}

	if err := SetTrusted(cfg, "demo", true); err != nil {
		t.Fatalf("SetTrusted() error = %v", err)
	}
	if !cfg.Plugins.Items[0].Trusted {
		t.Fatal("SetTrusted() left plugin untrusted")
	}
}

func TestSetTrustedRejectsMissingLoadFile(t *testing.T) {
	pluginDir := t.TempDir()
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"},
	}

	if err := SetTrusted(cfg, "demo", true); err == nil {
		t.Fatal("SetTrusted() error = nil, want missing load file error")
	}
	if cfg.Plugins.Items[0].Trusted {
		t.Fatal("SetTrusted() trusted plugin with missing load file")
	}
}

func TestSetTrustedRejectsSymlinkLoadFile(t *testing.T) {
	pluginDir := t.TempDir()
	target := filepath.Join(pluginDir, "target.zsh")
	if err := os.WriteFile(target, []byte("# plugin\n"), 0644); err != nil {
		t.Fatalf("WriteFile(target.zsh) error = %v", err)
	}
	if err := os.Symlink(target, filepath.Join(pluginDir, "plugin.zsh")); err != nil {
		t.Fatalf("Symlink(plugin.zsh) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"},
	}

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
