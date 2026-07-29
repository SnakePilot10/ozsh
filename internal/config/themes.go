package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func LoadTheme(path string) (ThemeConfig, error) {
	var theme ThemeConfig
	if _, err := toml.DecodeFile(path, &theme); err != nil {
		return ThemeConfig{}, fmt.Errorf("failed to decode theme: %w", err)
	}
	if theme.Name == "" {
		var wrapped struct {
			Theme ThemeConfig `toml:"theme"`
		}
		if _, err := toml.DecodeFile(path, &wrapped); err != nil {
			return ThemeConfig{}, fmt.Errorf("failed to decode wrapped theme: %w", err)
		}
		theme = wrapped.Theme
	}
	if err := ValidateTheme(theme); err != nil {
		return ThemeConfig{}, fmt.Errorf("invalid theme: %w", err)
	}
	return theme, nil
}

func SaveTheme(path string, theme ThemeConfig) error {
	if err := ValidateTheme(theme); err != nil {
		return fmt.Errorf("invalid theme: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create theme directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".theme-*.toml")
	if err != nil {
		return fmt.Errorf("failed to create theme file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := toml.NewEncoder(tmp).Encode(theme); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to encode theme: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace theme atomically: %w", err)
	}
	return os.Chmod(path, 0o600)
}
