package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func withTempHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoad_CreatesDefaultConfigIfMissing(t *testing.T) {
	home := withTempHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Prompt.Style != "simple" {
		t.Fatalf("Load() style = %q, want simple", cfg.Prompt.Style)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "ozsh", "config.toml")); err != nil {
		t.Fatalf("Load() did not create config.toml: %v", err)
	}
}

func TestLoad_ReadsExistingConfig(t *testing.T) {
	withTempHome(t)

	cfg := Default()
	cfg.Prompt.Newline = false
	cfg.Prompt.Segments["user"] = SegmentConfig{Enabled: true, FG: "#00f5ff", Bold: true}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Prompt.Newline {
		t.Fatal("Load() Newline = true, want false")
	}
	if got := loaded.Prompt.Segments["user"].FG; got != "#00f5ff" {
		t.Fatalf("Load() user fg = %q, want #00f5ff", got)
	}
}

func TestSave_WritesReadableTOML(t *testing.T) {
	withTempHome(t)

	if err := Save(Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	if !strings.Contains(string(data), "[prompt]") || !strings.Contains(string(data), "[plugins]") {
		t.Fatalf("Save() output missing expected TOML sections:\n%s", data)
	}

	var decoded Config
	if _, err := toml.Decode(string(data), &decoded); err != nil {
		t.Fatalf("Save() wrote invalid TOML: %v\n%s", err, data)
	}
}

func TestSave_ReturnsErrorWhenConfigDirCannotBeDetermined(t *testing.T) {
	t.Setenv("HOME", "")

	if err := Save(Default()); err == nil {
		t.Fatal("Save() error = nil, want error")
	}
}

func TestSave_ReturnsErrorWhenConfigDirIsNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not enforced consistently on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write to directories without owner write permission")
	}

	home := withTempHome(t)
	dir := filepath.Join(home, ".config", "ozsh")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatalf("Chmod(config dir) error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0755)
	})

	err := Save(Default())
	if err == nil {
		t.Fatal("Save() error = nil, want permission error")
	}
	if !strings.Contains(err.Error(), "failed to create config file") {
		t.Fatalf("Save() error = %q, want config file context", err)
	}
}

func TestValidateRejectsUnknownOrderSegment(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Order = append(cfg.Prompt.Order, "missing")

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want unknown segment error")
	}
}

func TestValidateRejectsDuplicateOrderSegment(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Order = append(cfg.Prompt.Order, "user")

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want duplicate segment error")
	}
}

func TestValidateRejectsInvalidColor(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Segments["user"] = SegmentConfig{Enabled: true, FG: "not-a-color"}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want invalid color error")
	}
}

func TestValidateAcceptsNamedAndHexColors(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Segments["user"] = SegmentConfig{Enabled: true, FG: "cyan", BG: "#00f5ff"}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsDuplicateAcrossLeftAndRightOrder(t *testing.T) {
	cfg := Default()
	cfg.Prompt.RightOrder = []string{"user"}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want duplicate left/right order error")
	}
}

func TestValidateRejectsInvalidThemeColor(t *testing.T) {
	cfg := Default()
	cfg.Theme.Accent = "hot-pink-ish"

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want invalid theme color error")
	}
}

func TestValidateFillsDefaultsForOldConfig(t *testing.T) {
	cfg := &Config{
		Prompt: PromptConfig{
			Order: []string{"user"},
			Segments: map[string]SegmentConfig{
				"user": {Enabled: true, FG: "cyan"},
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Prompt.Separator != "  " {
		t.Fatalf("Validate() separator = %q, want default", cfg.Prompt.Separator)
	}
	if cfg.Theme.Name != "cyber-cyan" {
		t.Fatalf("Validate() theme = %q, want cyber-cyan", cfg.Theme.Name)
	}
}

func TestValidateRejectsPluginSourceOutsideHome(t *testing.T) {
	withTempHome(t)
	cfg := Default()
	cfg.Plugins.Items = []PluginItem{
		{Name: "bad", Enabled: true, Source: "/etc/zsh/plugin.zsh"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want source outside HOME error")
	}
}

func TestValidateRejectsUnsafePluginLoadPath(t *testing.T) {
	home := withTempHome(t)
	cfg := Default()
	cfg.Plugins.Items = []PluginItem{
		{Name: "bad", Enabled: true, Source: filepath.Join(home, ".config", "ozsh", "plugins", "bad"), Load: "../init.zsh"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want unsafe load path error")
	}
}

func TestValidateRejectsNonShellPluginLoadFile(t *testing.T) {
	home := withTempHome(t)
	cfg := Default()
	cfg.Plugins.Items = []PluginItem{
		{Name: "bad", Enabled: true, Source: filepath.Join(home, ".config", "ozsh", "plugins", "bad"), Load: "README.md"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want non-shell load file error")
	}
}
