package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	configDirMode  = 0o700
	configFileMode = 0o600
)

func Dir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "ozsh")
	}
	return ""
}

func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

func Load() (*Config, error) {
	p := Path()
	if p == "config.toml" {
		return nil, fmt.Errorf("cannot determine config directory")
	}
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to inspect config: %w", err)
		}
		if err := Save(Default()); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	var cfg Config
	if _, err := toml.DecodeFile(p, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	dir := Dir()
	if dir == "" {
		return fmt.Errorf("cannot determine config directory")
	}
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("failed to create config file: config path is not a directory")
		}
		if info.Mode().Perm()&0o200 == 0 {
			return fmt.Errorf("failed to create config file: config directory is not writable")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect config dir: %w", err)
	}
	if err := os.MkdirAll(dir, configDirMode); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.Chmod(dir, configDirMode); err != nil {
		return fmt.Errorf("failed to secure config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config.toml-*")
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(configFileMode); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to secure temporary config: %w", err)
	}
	if err := toml.NewEncoder(tmp).Encode(cfg); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to flush config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close config: %w", err)
	}
	if err := os.Rename(tmpPath, Path()); err != nil {
		return fmt.Errorf("failed to replace config atomically: %w", err)
	}
	if err := os.Chmod(Path(), configFileMode); err != nil {
		return fmt.Errorf("failed to secure config file: %w", err)
	}
	return nil
}
