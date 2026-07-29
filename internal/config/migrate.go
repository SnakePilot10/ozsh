package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Migration struct {
	FromVersion int
	ToVersion   int
	Changes     []string
}

func PendingMigration() (*Migration, error) {
	var cfg Config
	if _, err := toml.DecodeFile(Path(), &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	if cfg.Version > CurrentConfigVersion {
		return nil, fmt.Errorf("config version %d is newer than supported version %d", cfg.Version, CurrentConfigVersion)
	}
	if cfg.Version == CurrentConfigVersion {
		return nil, nil
	}
	from := cfg.Version
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("cannot migrate invalid config: %w", err)
	}
	return &Migration{
		FromVersion: from,
		ToVersion:   CurrentConfigVersion,
		Changes: []string{
			"add transient prompt and OSC integration settings",
			"add typed segment conditions, styles, and cache TTLs",
			"add execution_time, python, and rust segment defaults",
			"add plugin provenance fields",
			"add theme portability metadata",
		},
	}, nil
}
