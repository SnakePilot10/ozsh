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

func TestLoadCreatesDefaultConfigIfMissing(t *testing.T) {
	home := withTempHome(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != CurrentConfigVersion {
		t.Fatalf("Load() version = %d, want %d", cfg.Version, CurrentConfigVersion)
	}
	if cfg.Prompt.Style != "simple" {
		t.Fatalf("Load() style = %q, want simple", cfg.Prompt.Style)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "ozsh", "config.toml")); err != nil {
		t.Fatalf("Load() did not create config.toml: %v", err)
	}
}

func TestLoadMigratesLegacyConfigAndBacksItUp(t *testing.T) {
	home := withTempHome(t)
	dir := filepath.Join(home, ".config", "ozsh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	legacy := `[prompt]
style = "simple"
newline = true
right_prompt = false
separator = "  "
order = ["user"]
right_order = []

[prompt.segments.user]
enabled = true
fg = "cyan"
`
	if err := os.WriteFile(Path(), []byte(legacy), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy config) error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() legacy error = %v", err)
	}
	if cfg.Version != CurrentConfigVersion {
		t.Fatalf("migrated version = %d, want %d", cfg.Version, CurrentConfigVersion)
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("ReadFile(migrated config) error = %v", err)
	}
	if !strings.Contains(string(data), "version = 2") {
		t.Fatalf("migrated config missing version:\n%s", data)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "config-*.bak"))
	if err != nil {
		t.Fatalf("Glob(config backup) error = %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("legacy backups = %d, want 1", len(backups))
	}
	backupData, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatalf("ReadFile(config backup) error = %v", err)
	}
	if string(backupData) != legacy {
		t.Fatalf("backup content changed:\n%s", backupData)
	}
}

func TestLoadRejectsFutureConfigVersion(t *testing.T) {
	withTempHome(t)
	cfg := Default()
	cfg.Version = CurrentConfigVersion + 1
	if err := Save(cfg); err == nil {
		t.Fatal("Save(future version) error = nil, want error")
	}
}

func TestLoadRejectsUnknownConfigKey(t *testing.T) {
	withTempHome(t)
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	data := "version = 1\nunknown_setting = true\n"
	if err := os.WriteFile(Path(), []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "unknown config key") {
		t.Fatalf("Load() error = %v, want unknown key error", err)
	}
}

func TestLoadExistingDoesNotCreateMissingConfig(t *testing.T) {
	withTempHome(t)
	if _, err := LoadExisting(); err == nil {
		t.Fatal("LoadExisting() error = nil for missing config")
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Fatalf("LoadExisting() created config: %v", err)
	}
}

func TestValidateRejectsNegativeVersion(t *testing.T) {
	cfg := Default()
	cfg.Version = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("Validate(negative version) error = nil")
	}
}

func TestValidateMigratesOmegaStyleToSimple(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Style = "omega"
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate(omega style) error = %v", err)
	}
	if cfg.Prompt.Style != "simple" {
		t.Fatalf("Validate(omega style) = %q, want simple", cfg.Prompt.Style)
	}
}

func TestLoadReadsExistingConfig(t *testing.T) {
	withTempHome(t)
	cfg := Default()
	cfg.Prompt.Newline = false
	cfg.Prompt.Layout = PromptLayoutOneLine
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

func TestSaveWritesReadableTOML(t *testing.T) {
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
	info, err := os.Stat(Path())
	if err != nil {
		t.Fatalf("Stat(config) error = %v", err)
	}
	if info.Mode().Perm() != configFileMode {
		t.Fatalf("config mode = %#o, want %#o", info.Mode().Perm(), configFileMode)
	}
}

func TestSaveReturnsErrorWhenConfigDirCannotBeDetermined(t *testing.T) {
	t.Setenv("HOME", "")
	if err := Save(Default()); err == nil {
		t.Fatal("Save() error = nil, want error")
	}
}

func TestSaveReturnsErrorWhenConfigDirIsNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits are not enforceable for this test environment")
	}
	home := withTempHome(t)
	dir := filepath.Join(home, ".config", "ozsh")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod(config dir) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := Save(Default()); err == nil || !strings.Contains(err.Error(), "failed to create config file") {
		t.Fatalf("Save() unwritable directory error = %v", err)
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

func TestValidateRejectsInvalidPromptStyle(t *testing.T) {
	cfg := Default()
	cfg.Prompt.Style = "../templates/simple"

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want invalid prompt style error")
	}
}

func TestValidateRejectsControlCharactersInPromptSeparator(t *testing.T) {
	for _, separator := range []string{"bad\nsep", "bad\rsep", "bad\x1bsep", "bad\x00sep"} {
		cfg := Default()
		cfg.Prompt.Separator = separator
		if err := Validate(cfg); err == nil {
			t.Fatalf("Validate() error = nil for separator %q, want control character error", separator)
		}
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
	cfg := &Config{Prompt: PromptConfig{Order: []string{"user"}, Segments: map[string]SegmentConfig{"user": {Enabled: true, FG: "cyan"}}}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Prompt.Separator != "  " || cfg.Theme.Name != "cyber-cyan" {
		t.Fatalf("Validate() did not fill expected defaults: %+v", cfg)
	}
}

func TestValidateRejectsPluginSourceOutsideManagedRoot(t *testing.T) {
	home := withTempHome(t)
	cfg := Default()
	cfg.Plugins.Items = []PluginItem{{Name: "bad", Enabled: true, Source: filepath.Join(home, "plugins", "bad"), Load: "plugin.zsh"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() error = nil, want managed source error")
	}
}

func TestValidateAcceptsManagedPlugin(t *testing.T) {
	home := withTempHome(t)
	cfg := Default()
	cfg.Plugins.Items = []PluginItem{{Name: "good", Enabled: true, Source: filepath.Join(home, ".config", "ozsh", "plugins", "good"), Load: "plugin.zsh"}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsEmptyOrUnsafePluginLoadPath(t *testing.T) {
	home := withTempHome(t)
	for _, load := range []string{"", "../init.zsh", "README.md"} {
		cfg := Default()
		cfg.Plugins.Items = []PluginItem{{Name: "bad", Enabled: true, Source: filepath.Join(home, ".config", "ozsh", "plugins", "bad"), Load: load}}
		if err := Validate(cfg); err == nil {
			t.Fatalf("Validate() error = nil for load %q", load)
		}
	}
}
