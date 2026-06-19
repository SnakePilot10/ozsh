package prompt

import (
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestGenerateDoesNotMutateCallerConfig(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"user"},
			Segments: map[string]config.SegmentConfig{
				"user": {Enabled: true, FG: "cyan"},
			},
		},
	}

	beforeSegments := len(cfg.Prompt.Segments)
	beforeSeparator := cfg.Prompt.Separator
	if _, err := Generate(cfg); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got := len(cfg.Prompt.Segments); got != beforeSegments {
		t.Fatalf("Generate() mutated segment map size = %d, want %d", got, beforeSegments)
	}
	if cfg.Prompt.Separator != beforeSeparator {
		t.Fatalf("Generate() mutated separator = %q, want %q", cfg.Prompt.Separator, beforeSeparator)
	}
}

func TestPluginSourcePathRequiresConcreteShellFile(t *testing.T) {
	cases := []struct {
		name   string
		plugin config.PluginItem
		ok     bool
	}{
		{name: "empty load", plugin: config.PluginItem{Source: "/tmp/demo"}, ok: false},
		{name: "absolute load", plugin: config.PluginItem{Source: "/tmp/demo", Load: "/tmp/plugin.zsh"}, ok: false},
		{name: "escape", plugin: config.PluginItem{Source: "/tmp/demo", Load: "../plugin.zsh"}, ok: false},
		{name: "wrong extension", plugin: config.PluginItem{Source: "/tmp/demo", Load: "README.md"}, ok: false},
		{name: "valid zsh", plugin: config.PluginItem{Source: "/tmp/demo", Load: "plugin.zsh"}, ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := pluginSourcePath(tc.plugin)
			if ok != tc.ok {
				t.Fatalf("pluginSourcePath() ok = %t, want %t", ok, tc.ok)
			}
			if ok && path != "/tmp/demo/plugin.zsh" {
				t.Fatalf("pluginSourcePath() = %q", path)
			}
		})
	}
}

func TestGenerateSkipsTrustedPluginWithoutLoadFile(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{
		Name: "legacy", Enabled: true, Trusted: true, Source: "/tmp/legacy",
	}}
	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.Contains(output, "ozsh_source_plugin") || strings.Contains(output, "/tmp/legacy") {
		t.Fatalf("Generate() sourced a plugin directory:\n%s", output)
	}
}
