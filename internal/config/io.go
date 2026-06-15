package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
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
	if _, err := os.Stat(p); os.IsNotExist(err) {
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	f, err := os.Create(Path())
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}
