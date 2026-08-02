package apply

import (
	"fmt"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

// ApplyConfig validates and persists cfg, writes omega.zsh, then injects the
// managed .zshrc block. Each file write is atomic, but the three-file operation
// is intentionally not presented as a transaction.
func ApplyConfig(cfg *config.Config) error {
	clone := cloneConfig(cfg)
	generated, err := prompt.Generate(clone)
	if err != nil {
		return fmt.Errorf("generate prompt: %w", err)
	}
	if _, _, err := shell.PreviewInjectBlock(); err != nil {
		return fmt.Errorf("preflight .zshrc: %w", err)
	}
	if err := config.Save(clone); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if err := shell.WriteOmega([]byte(generated)); err != nil {
		return fmt.Errorf("config saved, but omega.zsh could not be updated: %w", err)
	}
	if err := shell.InjectBlock(); err != nil {
		return fmt.Errorf("config and omega.zsh saved, but .zshrc could not be updated: %w", err)
	}
	return nil
}

func cloneConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return config.Default()
	}
	clone := *cfg
	clone.Prompt.Order = append([]string(nil), cfg.Prompt.Order...)
	clone.Prompt.RightOrder = append([]string(nil), cfg.Prompt.RightOrder...)
	clone.Prompt.Segments = make(map[string]config.SegmentConfig, len(cfg.Prompt.Segments))
	for key, value := range cfg.Prompt.Segments {
		clone.Prompt.Segments[key] = value
	}
	clone.Plugins.Selected = append([]string(nil), cfg.Plugins.Selected...)
	clone.Plugins.Items = append([]config.PluginItem(nil), cfg.Plugins.Items...)
	return &clone
}
