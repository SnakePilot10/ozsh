package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMigratesVersionOneConfigToVersionTwo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}

	v1 := `version = 1

[prompt]
style = "simple"
newline = false
right_prompt = false
disable_heavy_segments = false
separator = "  "
order = ["user", "cwd", "git", "status"]
right_order = []

[prompt.segments.user]
enabled = true
icon = "@"
fg = "cyan"
bg = ""
bold = true

[plugins]
engine = "manual"
items = []

[theme]
name = "cyber-cyan"
accent = "#00f5ff"
background = "#09090d"
muted = "#6b6b80"
success = "#00ff9f"
warning = "#ffe600"
error = "#ff003c"
`
	if err := os.WriteFile(Path(), []byte(v1), 0o600); err != nil {
		t.Fatalf("WriteFile(v1 config) error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != 2 {
		t.Fatalf("version = %d, want 2", cfg.Version)
	}
	if cfg.Prompt.IconMode != IconModeCompatible {
		t.Fatalf("icon mode = %q, want %q", cfg.Prompt.IconMode, IconModeCompatible)
	}
	if cfg.Prompt.Layout != PromptLayoutOneLine {
		t.Fatalf("layout = %q, want %q", cfg.Prompt.Layout, PromptLayoutOneLine)
	}
	if cfg.Prompt.Symbol != "❯" {
		t.Fatalf("symbol = %q, want ❯", cfg.Prompt.Symbol)
	}
	if cfg.Prompt.Segments["user"].CompatibleIcon != "@" {
		t.Fatalf("compatible icon = %q, want @", cfg.Prompt.Segments["user"].CompatibleIcon)
	}
	if cfg.Theme.ID != "cyberpunk" {
		t.Fatalf("theme ID = %q, want cyberpunk", cfg.Theme.ID)
	}
	wantSelected := []string{"zsh-autosuggestions", "fzf-tab", "zsh-syntax-highlighting"}
	if !reflect.DeepEqual(cfg.Plugins.Selected, wantSelected) {
		t.Fatalf("selected plugins = %#v, want %#v", cfg.Plugins.Selected, wantSelected)
	}

	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("ReadFile(migrated config) error = %v", err)
	}
	if !strings.Contains(string(data), "version = 2") {
		t.Fatalf("migrated config missing version 2:\n%s", data)
	}
	backups, err := filepath.Glob(filepath.Join(Dir(), "config-*.bak"))
	if err != nil {
		t.Fatalf("Glob(backups) error = %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
}

func TestDefaultUsesSafeV2PresentationSettings(t *testing.T) {
	cfg := Default()
	if cfg.Version != 2 {
		t.Fatalf("default version = %d, want 2", cfg.Version)
	}
	if cfg.Prompt.IconMode != IconModeCompatible || cfg.Prompt.Layout != PromptLayoutTwoLine || cfg.Prompt.Symbol != "❯" {
		t.Fatalf("default prompt presentation = %+v", cfg.Prompt)
	}
	if cfg.Theme.ID == "" || len(cfg.Plugins.Selected) != 3 {
		t.Fatalf("default catalog settings theme=%q selected=%v", cfg.Theme.ID, cfg.Plugins.Selected)
	}
}

func TestValidateRejectsInvalidV2EnumsAndDuplicateSelections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "icon mode", mutate: func(cfg *Config) { cfg.Prompt.IconMode = "emoji" }},
		{name: "layout", mutate: func(cfg *Config) { cfg.Prompt.Layout = "stacked" }},
		{name: "empty symbol", mutate: func(cfg *Config) { cfg.Prompt.Symbol = "" }},
		{name: "duplicate selected plugin", mutate: func(cfg *Config) { cfg.Plugins.Selected = append(cfg.Plugins.Selected, cfg.Plugins.Selected[0]) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mutate(cfg)
			if err := Validate(cfg); err == nil {
				t.Fatalf("Validate() error = nil for %s", tc.name)
			}
		})
	}
}

func TestFillDefaultsDoesNotAliasSelectedPlugins(t *testing.T) {
	first := &Config{}
	second := &Config{}
	FillDefaults(first)
	FillDefaults(second)
	first.Plugins.Selected[0] = "changed"
	if second.Plugins.Selected[0] == "changed" {
		t.Fatal("FillDefaults aliased selected plugin slices")
	}
}
