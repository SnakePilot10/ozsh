package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func captureStdoutCoverage(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = original
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(data)
}

func TestCommandHelpers(t *testing.T) {
	args, verbose := parseGlobalFlags([]string{"--verbose", "preview", "-v"})
	if !verbose || len(args) != 1 || args[0] != "preview" {
		t.Fatalf("parseGlobalFlags() = %#v, %t", args, verbose)
	}
	if got := min(3, 1, 2); got != 1 {
		t.Fatalf("min() = %d, want 1", got)
	}
	if got := editDistance("aply", "apply"); got != 1 {
		t.Fatalf("editDistance() = %d, want 1", got)
	}
	if got := suggestCommand("aply"); got != "apply" {
		t.Fatalf("suggestCommand() = %q, want apply", got)
	}
	if got := suggestCommand("totally-unknown"); got != "" {
		t.Fatalf("suggestCommand(unknown) = %q, want empty", got)
	}
}

func TestThemeAndPluginPathHelpers(t *testing.T) {
	cfg := config.Default()
	preset := config.Presets["neon-red"]
	applyTheme(cfg, preset)
	if cfg.Theme.Name != "neon-red" || cfg.Prompt.Segments["user"].FG != preset.Accent {
		t.Fatalf("applyTheme() did not update theme/user: %+v", cfg.Theme)
	}
	if len(sortedThemeNames()) == 0 || len(sortedHeaderNames()) == 0 {
		t.Fatal("preset sorting returned an empty list")
	}
	if got := pluginLoadPath(config.PluginItem{Source: "/tmp/demo"}); got != "<no load file>" {
		t.Fatalf("pluginLoadPath(empty) = %q", got)
	}
	if got := pluginLoadPath(config.PluginItem{Source: "/tmp/demo", Load: "plugin.zsh"}); got != filepath.Join("/tmp/demo", "plugin.zsh") {
		t.Fatalf("pluginLoadPath(load) = %q", got)
	}
}

func TestSafeDisplayCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if out := captureStdoutCoverage(t, printUsage); !strings.Contains(out, "Usage: ozsh") || !strings.Contains(out, "plugin") {
		t.Fatalf("printUsage() output:\n%s", out)
	}
	if out := captureStdoutCoverage(t, func() { runPreview(nil) }); !strings.Contains(out, "❯") {
		t.Fatalf("runPreview() output:\n%s", out)
	}
	if out := captureStdoutCoverage(t, func() { runTheme([]string{"list"}) }); !strings.Contains(out, "cyber-cyan") {
		t.Fatalf("runTheme(list) output:\n%s", out)
	}
	if out := captureStdoutCoverage(t, func() { runHeader([]string{"list"}) }); strings.TrimSpace(out) == "" {
		t.Fatal("runHeader(list) unexpectedly returned nothing")
	}
	if out := captureStdoutCoverage(t, func() { runPlugin([]string{"list"}) }); !strings.Contains(out, "no plugins configured") {
		t.Fatalf("runPlugin(list) output:\n%s", out)
	}
	if out := captureStdoutCoverage(t, func() { runUpdate([]string{"--check"}) }); !strings.Contains(out, "no source checkout") {
		t.Fatalf("runUpdate(check) output:\n%s", out)
	}
}
