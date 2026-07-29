package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var validNamedColors = map[string]struct{}{
	"":        {},
	"black":   {},
	"red":     {},
	"green":   {},
	"yellow":  {},
	"blue":    {},
	"magenta": {},
	"cyan":    {},
	"white":   {},
	"default": {},
}

var (
	hexColorPattern    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	pluginNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
	promptStylePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	envNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	controlCharPattern = regexp.MustCompile(`[\x00-\x1f\x7f]`)
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Version < 0 {
		return fmt.Errorf("unsupported config version %d; supported version is %d", cfg.Version, CurrentConfigVersion)
	}
	if cfg.Version < CurrentConfigVersion {
		cfg.Version = CurrentConfigVersion
	}
	if cfg.Version > CurrentConfigVersion {
		return fmt.Errorf("unsupported config version %d; supported version is %d", cfg.Version, CurrentConfigVersion)
	}
	FillDefaults(cfg)

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
	transientSeen := map[string]struct{}{}
	for _, name := range cfg.Prompt.TransientOrder {
		if err := validateSegmentName(cfg, "transient prompt order", name); err != nil {
			return err
		}
		if _, ok := transientSeen[name]; ok {
			return fmt.Errorf("transient prompt order contains duplicate segment %q", name)
		}
		transientSeen[name] = struct{}{}
	}

	for name, segment := range cfg.Prompt.Segments {
		if controlCharPattern.MatchString(segment.Icon) {
			return fmt.Errorf("segment %q icon cannot contain control characters", name)
		}
		if err := validateColor(segment.FG); err != nil {
			return fmt.Errorf("segment %q fg: %w", name, err)
		}
		if err := validateColor(segment.BG); err != nil {
			return fmt.Errorf("segment %q bg: %w", name, err)
		}
		if segment.PaddingLeft < 0 || segment.PaddingLeft > 20 || segment.PaddingRight < 0 || segment.PaddingRight > 20 {
			return fmt.Errorf("segment %q padding must be between 0 and 20", name)
		}
		if segment.CacheTTL < 0 || segment.CacheTTL > 3600 {
			return fmt.Errorf("segment %q cache TTL must be between 0 and 3600 seconds", name)
		}
		if err := validateCondition(segment.When); err != nil {
			return fmt.Errorf("segment %q when: %w", name, err)
		}
		if segment.WhenEnv != "" && !envNamePattern.MatchString(segment.WhenEnv) {
			return fmt.Errorf("segment %q when_env must be a valid environment variable name", name)
		}
		for field, value := range map[string]string{"icon": segment.Icon, "leading_symbol": segment.LeadingSymbol, "trailing_symbol": segment.TrailingSymbol} {
			if controlCharPattern.MatchString(value) {
				return fmt.Errorf("segment %q %s cannot contain control characters", name, field)
			}
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
	return validatePlugins(cfg.Plugins)
}

func FillDefaults(cfg *Config) {
	defaults := Default()
	if cfg.Prompt.Style == "" {
		cfg.Prompt.Style = defaults.Prompt.Style
	}
	if cfg.Prompt.Separator == "" {
		cfg.Prompt.Separator = defaults.Prompt.Separator
	}
	if cfg.Prompt.Segments == nil {
		cfg.Prompt.Segments = map[string]SegmentConfig{}
	}
	if cfg.Prompt.TransientOrder == nil {
		cfg.Prompt.TransientOrder = append([]string(nil), defaults.Prompt.TransientOrder...)
	}
	for name, segment := range defaults.Prompt.Segments {
		if _, ok := cfg.Prompt.Segments[name]; !ok {
			cfg.Prompt.Segments[name] = segment
		}
	}
	if cfg.Plugins.Engine == "" {
		cfg.Plugins.Engine = defaults.Plugins.Engine
	}
	if cfg.Theme.Name == "" {
		cfg.Theme = defaults.Theme
	} else {
		if cfg.Theme.Tier == "" {
			cfg.Theme.Tier = "ascii"
		}
		if cfg.Theme.Requires == nil {
			cfg.Theme.Requires = []string{}
		}
	}
}

func ValidateTheme(theme ThemeConfig) error {
	return validateTheme(theme)
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
	validTiers := map[string]bool{"ascii": true, "unicode": true, "nerd-font": true, "powerline": true}
	if !promptStylePattern.MatchString(theme.Name) {
		return fmt.Errorf("theme name must contain only letters, numbers, underscores, or hyphens")
	}
	if !validTiers[theme.Tier] {
		return fmt.Errorf("theme tier must be ascii, unicode, nerd-font, or powerline")
	}
	seenRequirements := map[string]bool{}
	for _, requirement := range theme.Requires {
		if !validTiers[requirement] || requirement == "ascii" {
			return fmt.Errorf("theme requirement %q is not supported", requirement)
		}
		if seenRequirements[requirement] {
			return fmt.Errorf("theme requirement %q is duplicated", requirement)
		}
		seenRequirements[requirement] = true
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

func validateCondition(condition string) error {
	switch condition {
	case "", "always", "git_repository", "virtualenv", "command_success", "command_failure":
		return nil
	default:
		return fmt.Errorf("unsupported condition %q", condition)
	}
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
		if item.Repository != "" {
			repository, err := url.Parse(item.Repository)
			if err != nil || repository.Scheme != "https" || repository.Host == "" || repository.User != nil {
				return fmt.Errorf("plugin %q repository must be an https URL without credentials", item.Name)
			}
		}
		if item.Revision != "" && !regexp.MustCompile(`^[0-9a-fA-F]{40}$`).MatchString(item.Revision) {
			return fmt.Errorf("plugin %q revision must be a full git commit hash", item.Name)
		}
		if item.InstalledAt != "" {
			if _, err := time.Parse(time.RFC3339, item.InstalledAt); err != nil {
				return fmt.Errorf("plugin %q installed_at must use RFC3339", item.Name)
			}
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
