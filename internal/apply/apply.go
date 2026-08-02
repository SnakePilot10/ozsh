package apply

import (
	"fmt"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

type PartialError struct {
	Stage       string
	ConfigSaved bool
	OmegaSaved  bool
	Err         error
}

func (e *PartialError) Error() string {
	return fmt.Sprintf("%s: %v", e.Stage, e.Err)
}

func (e *PartialError) Unwrap() error { return e.Err }

// ApplyConfig validates and persists cfg, writes omega.zsh, then injects the
// managed .zshrc block. Each file write is atomic, but the three-file operation
// is intentionally not presented as a transaction.
func ApplyConfig(cfg *config.Config) error {
	preview, err := shell.PreviewInjectPlan()
	if err != nil {
		return fmt.Errorf("preflight .zshrc: %w", err)
	}
	return ApplyConfigExpected(cfg, preview.Before, preview.Target)
}

func ApplyConfigExpected(cfg *config.Config, expectedZshrc, expectedTarget string) error {
	clone := cloneConfig(cfg)
	generated, err := prompt.Generate(clone)
	if err != nil {
		return fmt.Errorf("generate prompt: %w", err)
	}
	preview, err := shell.PreviewInjectPlan()
	if err != nil {
		return fmt.Errorf("preflight .zshrc: %w", err)
	}
	if preview.Target != expectedTarget {
		return fmt.Errorf("preflight .zshrc: target changed after review")
	}
	if preview.Before != expectedZshrc {
		return fmt.Errorf("preflight .zshrc: content changed after review")
	}
	if err := config.Save(clone); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if err := shell.WriteOmega([]byte(generated)); err != nil {
		return &PartialError{Stage: "config saved, but omega.zsh could not be updated", ConfigSaved: true, Err: err}
	}
	if err := shell.InjectBlockPlanIfUnchanged(expectedZshrc, expectedTarget); err != nil {
		return &PartialError{Stage: "config and omega.zsh saved, but .zshrc could not be updated", ConfigSaved: true, OmegaSaved: true, Err: err}
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
	clone.Plugins.Items = append([]config.PluginItem(nil), cfg.Plugins.Items...)
	return &clone
}
