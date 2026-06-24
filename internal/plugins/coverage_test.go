package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestValidationRejectsUnsafeNamesAndLoads(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../demo", "demo/name", " has-space"} {
		if err := validateName(name); err == nil {
			t.Fatalf("validateName(%q) error = nil", name)
		}
	}
	for _, load := range []string{"", "/tmp/plugin.zsh", "../plugin.zsh", ".", "README.md"} {
		if err := validateLoad(load); err == nil {
			t.Fatalf("validateLoad(%q) error = nil", load)
		}
	}
}

func TestAddRejectsCredentialURLFragmentAndDuplicate(t *testing.T) {
	cfg := config.Default()
	for _, rawURL := range []string{
		"https://token@example.com/demo.git",
		"https://example.com/demo.git#branch",
	} {
		if _, err := Add(cfg, rawURL, "plugin.zsh"); err == nil {
			t.Fatalf("Add(%q) error = nil", rawURL)
		}
	}
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: "/tmp/demo", Load: "plugin.zsh"}}
	if _, err := Add(cfg, "https://example.com/demo.git", "plugin.zsh"); err == nil {
		t.Fatal("Add() duplicate error = nil")
	}
}

func TestAddRejectsExistingDestinationWithoutCloning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(Dir(), "demo"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if _, err := Add(config.Default(), "https://example.com/demo.git", "plugin.zsh"); err == nil {
		t.Fatal("Add() existing destination error = nil")
	}
}

func TestAddRequiresHomeForManagedDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := Add(config.Default(), "https://example.com/demo.git", "plugin.zsh"); err == nil {
		t.Fatal("Add() missing HOME error = nil")
	}
}

func TestPluginMutationErrorsAndUnsafeRemove(t *testing.T) {
	cfg := config.Default()
	if err := SetEnabled(cfg, "missing", true); err == nil {
		t.Fatal("SetEnabled(missing) error = nil")
	}
	if err := SetTrusted(cfg, "missing", true); err == nil {
		t.Fatal("SetTrusted(missing) error = nil")
	}
	if err := Remove(cfg, "missing"); err == nil {
		t.Fatal("Remove(missing) error = nil")
	}

	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: "/tmp/not-managed", Load: "plugin.zsh"}}
	if err := Remove(cfg, "demo"); err == nil {
		t.Fatal("Remove(unsafe source) error = nil")
	}
}

func TestTrustRejectsSymlinkedPluginRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatalf("MkdirAll(outside) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "plugin.zsh"), []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	managed := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatalf("MkdirAll(managed parent) error = %v", err)
	}
	if err := os.Symlink(outside, managed); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: managed, Load: "plugin.zsh"}}
	if err := SetTrusted(cfg, "demo", true); err == nil {
		t.Fatal("SetTrusted(symlink root) error = nil")
	}
}
