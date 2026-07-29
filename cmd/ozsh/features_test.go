package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestRunConfigMigrationDryRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := "version = 1\n[prompt]\nstyle = \"simple\"\nseparator = \"  \"\norder = []\nright_order = []\n"
	if err := os.WriteFile(config.Path(), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	output := captureStdout(t, func() { runConfig([]string{"migrate", "--dry-run"}) })
	if !strings.Contains(output, "v1 -> v2") || !strings.Contains(output, "plugin provenance") {
		t.Fatalf("dry-run output = %q", output)
	}
	data, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != legacy {
		t.Fatal("dry-run modified config")
	}
}

func TestRunThemeExportAndValidate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "theme.toml")
	exportOutput := captureStdout(t, func() { runTheme([]string{"export", path}) })
	if !strings.Contains(exportOutput, "theme exported") {
		t.Fatalf("theme export output = %q", exportOutput)
	}
	validateOutput := captureStdout(t, func() { runTheme([]string{"validate", path}) })
	if !strings.Contains(validateOutput, "theme valid: cyber-cyan") {
		t.Fatalf("theme validate output = %q", validateOutput)
	}
}

func TestThemeRequirementWarnings(t *testing.T) {
	t.Setenv("LANG", "C")
	warnings := themeRequirementWarnings(config.ThemeConfig{
		Name: "power", Requires: []string{"unicode", "nerd-font", "powerline"},
	})
	if len(warnings) != 3 {
		t.Fatalf("themeRequirementWarnings() = %v", warnings)
	}
}
