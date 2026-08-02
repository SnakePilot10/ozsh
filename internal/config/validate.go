package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var validNamedColors = map[string]struct{}{
	"": {}, "black": {}, "red": {}, "green": {}, "yellow": {}, "blue": {},
	"magenta": {}, "cyan": {}, "white": {}, "default": {},
}

var (
	hexColorPattern    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	pluginNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
	themeIDPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	controlCharPattern = regexp.MustCompile(`[\x00-\x1f\x7f]`)
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Version < 0 || cfg.Version > CurrentConfigVersion {
		return fmt.Errorf("unsupported config version %d; supported version is %d", cfg.Version, CurrentConfigVersion)
	}
	if cfg.Version < CurrentConfigVersion {
		migrateConfig(cfg, cfg.Version)
	} else if cfg.Prompt.Symbol == "" {
		return fmt.Errorf("prompt symbol cannot be empty")
	}
	FillDefaults(cfg)

	if cfg.Prompt.IconMode != IconModeCompatible && cfg.Prompt.IconMode != IconModeNerd {
		return fmt.Errorf("unsupported icon mode %q", cfg.Prompt.IconMode)
	}
	if cfg.Prompt.Layout != PromptLayoutOneLine && cfg.Prompt.Layout != PromptLayoutTwoLine {
		return fmt.Errorf("unsupported prompt layout %q", cfg.Prompt.Layout)
	}
	if cfg.Prompt.Layout == PromptLayoutTwoLine && !cfg.Prompt.Newline {
		return fmt.Errorf("two-line prompt layout requires newline = true")
	}
	if cfg.Prompt.Layout == PromptLayoutOneLine && cfg.Prompt.Newline {
		return fmt.Errorf("one-line prompt layout requires newline = false")
	}
	if controlCharPattern.MatchString(cfg.Prompt.DisplayName) {
		return fmt.Errorf("display name cannot contain control characters")
	}
	if cfg.Prompt.Symbol == "" || controlCharPattern.MatchString(cfg.Prompt.Symbol) || utf8.RuneCountInString(cfg.Prompt.Symbol) > 8 {
		return fmt.Errorf("prompt symbol must contain 1 to 8 printable characters")
	}

	seen := map[string]struct{}{}
	for _, name := range cfg.Prompt.Order {
		if err := validateSegmentName(cfg, "prompt order", name); err != nil {
			return err
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("prompt order contains duplicate segment %q", name)
		}
		seen[name] = struct{}{}
	}
	for _, name := range cfg.Prompt.RightOrder {
		if err := validateSegmentName(cfg, "right prompt order", name); err != nil {
			return err
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("segment %q appears in both prompt order and right prompt order", name)
		}
		seen[name] = struct{}{}
	}

	for name, segment := range cfg.Prompt.Segments {
		for field, icon := range map[string]string{
			"icon": segment.Icon, "compatible icon": segment.CompatibleIcon, "nerd icon": segment.NerdIcon,
		} {
			if controlCharPattern.MatchString(icon) {
				return fmt.Errorf("segment %q %s cannot contain control characters", name, field)
			}
		}
		if err := validateColor(segment.FG); err != nil {
			return fmt.Errorf("segment %q fg: %w", name, err)
		}
		if err := validateColor(segment.BG); err != nil {
			return fmt.Errorf("segment %q bg: %w", name, err)
		}
	}
	if cfg.Prompt.Separator == "" {
		return fmt.Errorf("prompt separator cannot be empty")
	}
	if controlCharPattern.MatchString(cfg.Prompt.Separator) {
		return fmt.Errorf("prompt separator cannot contain control characters")
	}
	if cfg.Prompt.Style == "omega" {
		cfg.Prompt.Style = "simple"
	}
	if cfg.Prompt.Style != "simple" {
		return fmt.Errorf("unsupported prompt style %q; supported style is simple", cfg.Prompt.Style)
	}
	if err := validateTheme(cfg.Theme); err != nil {
		return err
	}
	if err := validateSelectedPlugins(cfg.Plugins.Selected); err != nil {
		return err
	}
	return validatePlugins(cfg.Plugins)
}

func FillDefaults(cfg *Config) {
	defaults := Default()
	if cfg.Version == 0 {
		cfg.Version = CurrentConfigVersion
	}
	if cfg.Prompt.Style == "" {
		cfg.Prompt.Style = defaults.Prompt.Style
	}
	if cfg.Prompt.IconMode == "" {
		cfg.Prompt.IconMode = defaults.Prompt.IconMode
	}
	if cfg.Prompt.Layout == "" {
		cfg.Prompt.Layout = defaults.Prompt.Layout
		cfg.Prompt.Newline = defaults.Prompt.Newline
	}
	if cfg.Prompt.Symbol == "" {
		cfg.Prompt.Symbol = defaults.Prompt.Symbol
	}
	if cfg.Prompt.Separator == "" {
		cfg.Prompt.Separator = defaults.Prompt.Separator
	}
	if cfg.Prompt.Order == nil {
		cfg.Prompt.Order = append([]string(nil), defaults.Prompt.Order...)
	}
	if cfg.Prompt.RightOrder == nil {
		cfg.Prompt.RightOrder = []string{}
	}
	if cfg.Prompt.Segments == nil {
		cfg.Prompt.Segments = map[string]SegmentConfig{}
	}
	for name, defaultSegment := range defaults.Prompt.Segments {
		segment, ok := cfg.Prompt.Segments[name]
		if !ok {
			cfg.Prompt.Segments[name] = defaultSegment
			continue
		}
		if segment.CompatibleIcon == "" {
			if segment.Icon != "" {
				segment.CompatibleIcon = segment.Icon
			} else {
				segment.CompatibleIcon = defaultSegment.CompatibleIcon
			}
		}
		if segment.NerdIcon == "" {
			segment.NerdIcon = defaultSegment.NerdIcon
		}
		cfg.Prompt.Segments[name] = segment
	}
	if cfg.Plugins.Engine == "" {
		cfg.Plugins.Engine = defaults.Plugins.Engine
	}
	if cfg.Plugins.Selected == nil {
		cfg.Plugins.Selected = append([]string(nil), defaults.Plugins.Selected...)
	} else {
		cfg.Plugins.Selected = append([]string(nil), cfg.Plugins.Selected...)
	}
	if cfg.Plugins.Items == nil {
		cfg.Plugins.Items = []PluginItem{}
	}
	if cfg.Theme.ID == "" {
		if cfg.Theme.Name == "cyber-cyan" {
			cfg.Theme.ID = "cyberpunk"
		} else if themeIDPattern.MatchString(cfg.Theme.Name) {
			cfg.Theme.ID = cfg.Theme.Name
		} else {
			cfg.Theme.ID = defaults.Theme.ID
		}
	}
	if cfg.Theme.Name == "" {
		cfg.Theme.Name = defaults.Theme.Name
	}
	if cfg.Theme.Accent == "" {
		cfg.Theme = defaults.Theme
	}
}

func migrateConfig(cfg *Config, sourceVersion int) {
	if sourceVersion <= 1 {
		if cfg.Prompt.IconMode == "" {
			cfg.Prompt.IconMode = IconModeCompatible
		}
		if cfg.Prompt.Layout == "" {
			if cfg.Prompt.Newline {
				cfg.Prompt.Layout = PromptLayoutTwoLine
			} else {
				cfg.Prompt.Layout = PromptLayoutOneLine
			}
		}
		if cfg.Prompt.Symbol == "" {
			cfg.Prompt.Symbol = "❯"
		}
		for name, segment := range cfg.Prompt.Segments {
			if segment.CompatibleIcon == "" && segment.Icon != "" {
				segment.CompatibleIcon = segment.Icon
			}
			cfg.Prompt.Segments[name] = segment
		}
		if cfg.Theme.ID == "" {
			if cfg.Theme.Name == "cyber-cyan" || cfg.Theme.Name == "" {
				cfg.Theme.ID = "cyberpunk"
			} else {
				cfg.Theme.ID = cfg.Theme.Name
			}
		}
		if cfg.Plugins.Selected == nil {
			cfg.Plugins.Selected = append([]string(nil), defaultSelectedPlugins...)
		}
	}
	cfg.Version = CurrentConfigVersion
}

func validateSegmentName(cfg *Config, field, name string) error {
	if _, ok := cfg.Prompt.Segments[name]; !ok {
		return fmt.Errorf("%s references unknown segment %q", field, name)
	}
	return nil
}

func validateColor(color string) error {
	if _, ok := validNamedColors[color]; ok {
		return nil
	}
	if hexColorPattern.MatchString(color) {
		return nil
	}
	return fmt.Errorf("invalid color %q", color)
}

func validateTheme(theme ThemeConfig) error {
	if !themeIDPattern.MatchString(theme.ID) && theme.ID != "custom" {
		return fmt.Errorf("invalid theme id %q", theme.ID)
	}
	if controlCharPattern.MatchString(theme.Variant) || controlCharPattern.MatchString(theme.Name) {
		return fmt.Errorf("theme name and variant cannot contain control characters")
	}
	colors := map[string]string{
		"accent": theme.Accent, "background": theme.Background, "muted": theme.Muted,
		"success": theme.Success, "warning": theme.Warning, "error": theme.Error,
	}
	for name, color := range colors {
		if err := validateColor(color); err != nil {
			return fmt.Errorf("theme %s: %w", name, err)
		}
	}
	return nil
}

func validateSelectedPlugins(selected []string) error {
	seen := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		if !pluginNamePattern.MatchString(id) {
			return fmt.Errorf("invalid selected plugin %q", id)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate selected plugin %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validatePlugins(plugins PluginConfig) error {
	if plugins.Engine != "manual" {
		return fmt.Errorf("plugins engine must be manual")
	}
	seen := map[string]struct{}{}
	for _, item := range plugins.Items {
		if !pluginNamePattern.MatchString(item.Name) {
			return fmt.Errorf("invalid plugin name %q", item.Name)
		}
		if _, ok := seen[item.Name]; ok {
			return fmt.Errorf("duplicate plugin %q", item.Name)
		}
		seen[item.Name] = struct{}{}
		if err := validateManagedPluginSource(item); err != nil {
			return fmt.Errorf("plugin %q source: %w", item.Name, err)
		}
		if err := validatePluginLoad(item.Load); err != nil {
			return fmt.Errorf("plugin %q load: %w", item.Name, err)
		}
	}
	return nil
}

func validatePluginLoad(load string) error {
	if strings.TrimSpace(load) == "" {
		return fmt.Errorf("load file is required")
	}
	if filepath.IsAbs(load) {
		return fmt.Errorf("path must be relative to plugin source")
	}
	clean := filepath.Clean(load)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("path must stay inside plugin source")
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext != ".zsh" && ext != ".sh" {
		return fmt.Errorf("file must be .zsh or .sh")
	}
	return nil
}

func validateManagedPluginSource(item PluginItem) error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("cannot determine HOME")
	}
	root := filepath.Join(home, ".config", "ozsh", "plugins")
	expected := filepath.Clean(filepath.Join(root, item.Name))
	actual := filepath.Clean(item.Source)
	if actual != expected {
		return fmt.Errorf("source must be %s", expected)
	}
	return nil
}
