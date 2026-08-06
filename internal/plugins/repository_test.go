package plugins

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestParseRepositoryAcceptsHTTPSGitURL(t *testing.T) {
	got, err := ParseRepository("  https://github.com/example/demo.git  ")
	if err != nil {
		t.Fatalf("ParseRepository() error = %v", err)
	}
	want := Repository{URL: "https://github.com/example/demo.git", Name: "demo"}
	if got != want {
		t.Fatalf("ParseRepository() = %#v, want %#v", got, want)
	}
}

func TestParseRepositoryRejectsUnsafeForms(t *testing.T) {
	cases := []string{
		"",
		"http://example.com/demo.git",
		"https://user:secret@example.com/demo.git",
		"https://example.com/demo.git?token=secret",
		"https://example.com/demo.git?",
		"https://example.com/demo.git#readme",
		"https://example.com/",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseRepository(raw); err == nil {
				t.Fatalf("ParseRepository(%q) error = nil", raw)
			}
		})
	}
}

func TestValidateNewRepositoryRejectsConfiguredDuplicate(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{Name: "demo"})
	err := ValidateNewRepository(cfg, Repository{URL: "https://example.com/demo.git", Name: "demo"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("ValidateNewRepository() error = %v, want duplicate error", err)
	}
}

func TestValidateLoadPathAcceptsNestedShellFile(t *testing.T) {
	for _, load := range []string{"demo.plugin.zsh", "lib/plugin.zsh", "scripts/plugin.sh"} {
		if err := ValidateLoadPath(load); err != nil {
			t.Fatalf("ValidateLoadPath(%q) error = %v", load, err)
		}
	}
}

func TestValidateLoadPathRejectsUnsafeValues(t *testing.T) {
	cases := []string{"", ".", "..", "../plugin.zsh", "lib/../../plugin.zsh", "/tmp/plugin.zsh", "README.md"}
	if filepath.Separator != '/' {
		cases = append(cases, `C:\\plugin.zsh`)
	}
	for _, load := range cases {
		t.Run(load, func(t *testing.T) {
			if err := ValidateLoadPath(load); err == nil {
				t.Fatalf("ValidateLoadPath(%q) error = nil", load)
			}
		})
	}
}
