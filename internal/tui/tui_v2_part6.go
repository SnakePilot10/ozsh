package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	applyop "github.com/snakepilot10/ozsh/internal/apply"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/fonts"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/shell"
	"os"
	"strings"
	"time"
)

type applyResult string

type pluginInstallResult struct {
	cfg *config.Config
	err error
}

type fontInstallResult struct {
	cfg  *config.Config
	font fonts.Font
	err  error
}

type backupRestoreResult struct{ err error }

type fontRestoreResult struct{ err error }

func doApply(cfg *config.Config) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg { return applyResult(apply(snapshot)) }
}

func doInstallPlugins(cfg *config.Config) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		err := plugins.InstallRecommended(context.Background(), snapshot, snapshot.Plugins.Selected)
		return pluginInstallResult{cfg: snapshot, err: err}
	}
}

func doInstallFont(cfg *config.Config, font fonts.Font) tea.Cmd {
	snapshot := cloneConfig(cfg)
	home := os.Getenv("HOME")
	termux := shell.IsTermux()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		manager := fonts.NewManager(home, termux)
		if err := manager.Install(ctx, font, nil); err != nil {
			return fontInstallResult{cfg: snapshot, font: font, err: err}
		}
		snapshot.Prompt.IconMode = config.IconModeNerd
		if err := config.Save(snapshot); err != nil {
			return fontInstallResult{cfg: snapshot, font: font, err: fmt.Errorf("font installed but icon mode could not be saved: %w", err)}
		}
		return fontInstallResult{cfg: snapshot, font: font}
	}
}

func doRestoreBackup(path string) tea.Cmd {
	return func() tea.Msg { return backupRestoreResult{err: shell.RestoreBackup(path)} }
}

func doRestoreTermuxFont() tea.Cmd {
	home := os.Getenv("HOME")
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return fontRestoreResult{err: fonts.NewManager(home, true).RestoreTermux(ctx)}
	}
}

func apply(cfg *config.Config) string {
	if err := applyop.ApplyConfig(cfg); err != nil {
		return "apply error: " + err.Error()
	}
	return "applied"
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
	clone.Plugins.Selected = append([]string(nil), cfg.Plugins.Selected...)
	return &clone
}

func wrapIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	index %= length
	if index < 0 {
		index += length
	}
	return index
}

func writeCheck(b *strings.Builder, ok bool, label string) {
	if ok {
		fmt.Fprintf(b, "✓ %s\n", label)
		return
	}
	fmt.Fprintf(b, "✗ %s\n", label)
}

func backupCount() int {
	backups, err := shell.Backups()
	if err != nil {
		return 0
	}
	return len(backups)
}
