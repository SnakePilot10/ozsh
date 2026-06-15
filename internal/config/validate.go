package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
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

	for name, segment := range cfg.Prompt.Segments {
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
	if err := validateTheme(cfg.Theme); err != nil {
		return err
	}
	if err := validateHeader(cfg.Header); err != nil {
		return err
	}
	if err := validatePlugins(cfg.Plugins); err != nil {
		return err
	}

	return nil
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
	}
	if cfg.Header.Style == "" {
		cfg.Header.Style = defaults.Header.Style
	}
	if cfg.Header.Text == "" {
		cfg.Header.Text = defaults.Header.Text
	}
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
	colors := map[string]string{
		"accent":     theme.Accent,
		"background": theme.Background,
		"muted":      theme.Muted,
		"success":    theme.Success,
		"warning":    theme.Warning,
		"error":      theme.Error,
	}
	for name, color := range colors {
		if err := validateColor(color); err != nil {
			return fmt.Errorf("theme %s: %w", name, err)
		}
	}
	return nil
}

func validateHeader(header HeaderConfig) error {
	if strings.ContainsAny(header.Text, "\r\n") {
		return fmt.Errorf("header text cannot contain newlines")
	}
	switch header.Style {
	case "figlet", "ascii", "custom":
		return nil
	default:
		return fmt.Errorf("header style must be figlet, ascii, or custom")
	}
}

func validatePlugins(plugins PluginConfig) error {
	if plugins.Engine != "manual" {
		return fmt.Errorf("plugins engine must be manual")
	}
	seen := map[string]struct{}{}
	for _, item := range plugins.Items {
		if strings.TrimSpace(item.Name) == "" {
			return fmt.Errorf("plugin name cannot be empty")
		}
		if _, ok := seen[item.Name]; ok {
			return fmt.Errorf("duplicate plugin %q", item.Name)
		}
		seen[item.Name] = struct{}{}
		if item.Source == "" {
			return fmt.Errorf("plugin %q source cannot be empty", item.Name)
		}
		if err := validateHomePath(item.Source); err != nil {
			return fmt.Errorf("plugin %q source: %w", item.Name, err)
		}
		if err := validatePluginLoad(item.Load); err != nil {
			return fmt.Errorf("plugin %q load: %w", item.Name, err)
		}
	}
	return nil
}

func validatePluginLoad(load string) error {
	if load == "" {
		return nil
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

func validateHomePath(path string) error {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absHome, absPath)
	if err != nil {
		return err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return nil
	}
	return fmt.Errorf("path must stay under HOME")
}
