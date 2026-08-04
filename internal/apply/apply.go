package apply

import (
	"errors"
	"fmt"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

// Request groups the pending configuration and filesystem changes that must be
// applied as one reversible operation.
type Request struct {
	Config        *config.Config
	PluginChanges plugins.ChangeSet
}

// ApplyConfig preserves the previous public API for callers without pending
// plugin filesystem changes.
func ApplyConfig(cfg *config.Config) error {
	return Apply(Request{Config: cfg})
}

// Apply validates and persists a configuration, writes omega.zsh, injects the
// managed .zshrc block, and finalizes plugin filesystem changes. Failures after
// mutation begins restore both files and plugin paths.
func Apply(request Request) error {
	clone := cloneConfig(request.Config)
	generated, err := prompt.Generate(clone)
	if err != nil {
		return fmt.Errorf("generate prompt: %w", err)
	}
	if _, _, err := shell.PreviewInjectBlock(); err != nil {
		return fmt.Errorf("preflight .zshrc: %w", err)
	}

	snapshots, err := captureApplySnapshots()
	if err != nil {
		return err
	}
	transaction, err := request.PluginChanges.Begin(clone)
	if err != nil {
		return fmt.Errorf("begin plugin changes: %w", err)
	}

	rollback := func(primary error) error {
		rollbackErrors := []error{primary}
		if err := transaction.Rollback(); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback plugin changes: %w", err))
		}
		for index := len(snapshots) - 1; index >= 0; index-- {
			if err := snapshots[index].Restore(); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore %s: %w", snapshots[index].path, err))
			}
		}
		return errors.Join(rollbackErrors...)
	}

	if err := config.Save(clone); err != nil {
		return rollback(fmt.Errorf("save config: %w", err))
	}
	if err := shell.WriteOmega([]byte(generated)); err != nil {
		return rollback(fmt.Errorf("write omega.zsh: %w", err))
	}
	if err := shell.InjectBlock(); err != nil {
		return rollback(fmt.Errorf("inject .zshrc: %w", err))
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit plugin changes: %w", err)
	}
	return nil
}

func captureApplySnapshots() ([]fileSnapshot, error) {
	paths := []string{config.Path(), shell.OmegaZshPath(), shell.ZshrcPath()}
	snapshots := make([]fileSnapshot, 0, len(paths))
	for _, path := range paths {
		snapshot, err := captureFile(path)
		if err != nil {
			return nil, fmt.Errorf("snapshot %s: %w", path, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
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
