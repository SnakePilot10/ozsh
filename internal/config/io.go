package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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

	cfg, legacy, err := decodeFile(p)
	if err != nil {
		return nil, err
	}
	if legacy {
		if err := backupConfig(p); err != nil {
			return nil, fmt.Errorf("failed to back up legacy config: %w", err)
		}
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to migrate config: %w", err)
		}
	}
	return cfg, nil
}

// LoadExisting validates an existing config without creating or migrating it.
// Diagnostic commands use this to remain read-only.
func LoadExisting() (*Config, error) {
	p := Path()
	if p == "config.toml" {
		return nil, fmt.Errorf("cannot determine config directory")
	}
	if _, err := os.Stat(p); err != nil {
		return nil, fmt.Errorf("failed to inspect config: %w", err)
	}
	cfg, _, err := decodeFile(p)
	return cfg, err
}

func decodeFile(path string) (*Config, bool, error) {
	var cfg Config
	metadata, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode config: %w", err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return nil, false, fmt.Errorf("unknown config key %q", undecoded[0].String())
	}
	sourceVersion := cfg.Version
	legacy := sourceVersion < CurrentConfigVersion
	if err := Validate(&cfg); err != nil {
		return nil, false, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, legacy, nil
}

func Save(cfg *Config) error {
	if cfg != nil && cfg.Version == 0 {
		cfg.Version = CurrentConfigVersion
	}
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

func backupConfig(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	now := time.Now()
	backupPath := filepath.Join(filepath.Dir(path), fmt.Sprintf("config-%s-%d.bak", now.Format("20060102-150405"), now.Nanosecond()))
	dst, err := os.OpenFile(backupPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, configFileMode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Chmod(backupPath, configFileMode)
}
