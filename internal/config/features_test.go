package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExtendedSegmentSettings(t *testing.T) {
	cfg := Default()
	segment := cfg.Prompt.Segments["python"]
	segment.Enabled = true
	segment.When = "virtualenv"
	segment.WhenEnv = "VIRTUAL_ENV"
	segment.PaddingLeft = 2
	segment.PaddingRight = 1
	segment.CacheTTL = 30
	segment.Italic = true
	segment.Underline = true
	cfg.Prompt.Segments["python"] = segment
	cfg.Prompt.TransientPrompt = true
	cfg.Prompt.OSC7 = true
	cfg.Prompt.OSC133 = true
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() extended config error = %v", err)
	}
}

func TestValidateRejectsUnsafeExtendedSettings(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*SegmentConfig)
	}{
		{name: "condition", mut: func(s *SegmentConfig) { s.When = "run_shell" }},
		{name: "environment", mut: func(s *SegmentConfig) { s.WhenEnv = "BAD-NAME" }},
		{name: "padding", mut: func(s *SegmentConfig) { s.PaddingLeft = 21 }},
		{name: "cache", mut: func(s *SegmentConfig) { s.CacheTTL = 3601 }},
		{name: "control", mut: func(s *SegmentConfig) { s.Icon = "bad\x1b" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			segment := cfg.Prompt.Segments["python"]
			tc.mut(&segment)
			cfg.Prompt.Segments["python"] = segment
			if err := Validate(cfg); err == nil {
				t.Fatal("Validate() error = nil, want invalid extended setting")
			}
		})
	}
}

func TestThemeRoundTrip(t *testing.T) {
	theme := Presets["cyber-cyan"]
	theme.Author = "test"
	path := filepath.Join(t.TempDir(), "themes", "cyber-cyan.toml")
	if err := SaveTheme(path, theme); err != nil {
		t.Fatalf("SaveTheme() error = %v", err)
	}
	loaded, err := LoadTheme(path)
	if err != nil {
		t.Fatalf("LoadTheme() error = %v", err)
	}
	if loaded.Name != theme.Name || loaded.Tier != theme.Tier || loaded.Author != theme.Author {
		t.Fatalf("LoadTheme() = %+v, want %+v", loaded, theme)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("theme mode = %#o, want 0600", info.Mode().Perm())
	}
}

func TestLoadWrappedPresetTheme(t *testing.T) {
	path := filepath.Join("..", "..", "presets", "cyber-cyan.toml")
	theme, err := LoadTheme(path)
	if err != nil {
		t.Fatalf("LoadTheme(wrapped preset) error = %v", err)
	}
	if theme.Name != "cyber-cyan" || theme.Tier != "unicode" {
		t.Fatalf("LoadTheme(wrapped preset) = %+v", theme)
	}
}

func TestPendingMigrationDoesNotWrite(t *testing.T) {
	withTempHome(t)
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := "version = 1\n[prompt]\nstyle = \"simple\"\nseparator = \"  \"\norder = []\nright_order = []\n"
	if err := os.WriteFile(Path(), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	migration, err := PendingMigration()
	if err != nil {
		t.Fatalf("PendingMigration() error = %v", err)
	}
	if migration == nil || migration.FromVersion != 1 || migration.ToVersion != CurrentConfigVersion || len(migration.Changes) == 0 {
		t.Fatalf("PendingMigration() = %+v", migration)
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version = 1") {
		t.Fatalf("PendingMigration() modified config: %s", data)
	}
}
