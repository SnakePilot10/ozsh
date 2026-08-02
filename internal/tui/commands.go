package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/shell"

	tea "github.com/charmbracelet/bubbletea"
)

type operation uint8

const (
	operationIdle operation = iota
	operationApplyPreview
	operationApply
	operationPluginAdd
	operationDoctor
	operationDashboard
	operationSave
	operationThemeSave
	operationPluginUpdate
	operationDoctorFix
)

type operationResultEnvelope struct {
	requestID uint64
	msg       tea.Msg
}

type applyPreviewResult struct {
	diff   string
	base   string
	target string
	err    error
}

type pluginAddResult struct {
	cfg  *config.Config
	name string
	err  error
}

type doctorCheck struct {
	ok    bool
	label string
}

type doctorResult []doctorCheck

type dashboardInfo struct {
	configPath string
	omegaPath  string
	hasBlock   bool
	platform   string
	termux     bool
	backups    int
	err        error
}

type dashboardResult dashboardInfo

type mutationKind uint8

const (
	mutationSave mutationKind = iota
	mutationTheme
	mutationCustomTheme
	mutationPlugin
	mutationDoctorFix
)

type mutationResult struct {
	kind     mutationKind
	cfg      *config.Config
	selected string
	message  string
	err      error
}

func prepareApply() tea.Msg {
	preview, err := shell.PreviewInjectPlan()
	if err != nil {
		return applyPreviewResult{err: err}
	}
	return applyPreviewResult{diff: shell.DiffLines(preview.Before, preview.After), base: preview.Before, target: preview.Target}
}

func addPlugin(cfg, expected *config.Config, expectedExists bool, rawURL, load string) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		if err := ensureConfigUnchanged(expected, expectedExists); err != nil {
			return pluginAddResult{cfg: snapshot, err: err}
		}
		name, err := plugins.Add(snapshot, rawURL, load)
		if err != nil {
			return pluginAddResult{cfg: snapshot, err: err}
		}
		var added config.PluginItem
		for _, item := range snapshot.Plugins.Items {
			if item.Name == name {
				added = item
				break
			}
		}
		persisted := snapshot
		if _, statErr := os.Stat(config.Path()); statErr == nil {
			current, loadErr := config.LoadExisting()
			if loadErr != nil {
				_ = os.RemoveAll(added.Source)
				return pluginAddResult{cfg: snapshot, err: loadErr}
			}
			for _, item := range current.Plugins.Items {
				if item.Name == name {
					_ = os.RemoveAll(added.Source)
					return pluginAddResult{cfg: snapshot, err: fmt.Errorf("plugin %q was added by another process", name)}
				}
			}
			current.Plugins.Items = append(current.Plugins.Items, added)
			persisted = current
		} else if os.IsNotExist(statErr) && expectedExists {
			_ = os.RemoveAll(added.Source)
			return pluginAddResult{cfg: snapshot, err: fmt.Errorf("config.toml was removed outside this TUI; reload before saving")}
		} else if !os.IsNotExist(statErr) {
			_ = os.RemoveAll(added.Source)
			return pluginAddResult{cfg: snapshot, err: statErr}
		}
		if err := config.Save(persisted); err != nil {
			_ = os.RemoveAll(added.Source)
			return pluginAddResult{cfg: snapshot, err: err}
		}
		return pluginAddResult{cfg: persisted, name: name}
	}
}

func inspectDoctor() tea.Msg {
	return doctorResult{
		{ok: shell.HasZsh(), label: "zsh installed"},
		{ok: shell.ConfigExists(), label: "config.toml exists"},
		{ok: shell.ZshrcExists(), label: ".zshrc exists"},
		{ok: shell.HasBlock(), label: "ozsh block present"},
		{ok: shell.ZshIsDefaultShell(), label: "zsh is default shell"},
	}
}

func inspectDashboard() tea.Msg {
	backups, err := shell.Backups()
	return dashboardResult{
		configPath: config.Path(),
		omegaPath:  shell.OmegaZshPath(),
		hasBlock:   shell.HasBlock(),
		platform:   runtime.GOOS + "/" + runtime.GOARCH,
		termux:     shell.IsTermux(),
		backups:    len(backups),
		err:        err,
	}
}

func persistConfig(cfg, expected *config.Config, expectedExists bool) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		err := ensureConfigUnchanged(expected, expectedExists)
		if err == nil {
			err = config.Save(snapshot)
		}
		return mutationResult{kind: mutationSave, cfg: snapshot, message: "config saved", err: err}
	}
}

func persistTheme(cfg, expected *config.Config, expectedExists bool, name string, preset config.ThemeConfig) tea.Cmd {
	snapshot := cloneConfig(cfg)
	snapshot.Theme = preset
	if segment, ok := snapshot.Prompt.Segments["user"]; ok {
		segment.FG = preset.Accent
		snapshot.Prompt.Segments["user"] = segment
	}
	if segment, ok := snapshot.Prompt.Segments["status"]; ok {
		segment.FG = preset.Error
		snapshot.Prompt.Segments["status"] = segment
	}
	return func() tea.Msg {
		err := ensureConfigUnchanged(expected, expectedExists)
		if err == nil {
			err = config.Save(snapshot)
		}
		return mutationResult{kind: mutationTheme, cfg: snapshot, selected: name, message: "theme applied and saved: " + preset.Name, err: err}
	}
}

func persistCustomTheme(cfg, expected *config.Config, expectedExists bool) tea.Cmd {
	snapshot := cloneConfig(cfg)
	snapshot.Theme.Name = "custom"
	return func() tea.Msg {
		err := ensureConfigUnchanged(expected, expectedExists)
		if err == nil {
			err = config.Save(snapshot)
		}
		return mutationResult{kind: mutationCustomTheme, cfg: snapshot, selected: "custom", message: "custom theme saved", err: err}
	}
}

func persistPluginState(cfg, expected *config.Config, expectedExists bool, name string, toggleEnabled bool, trusted *bool) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		err := ensureConfigUnchanged(expected, expectedExists)
		if err != nil {
			return mutationResult{kind: mutationPlugin, cfg: snapshot, selected: name, err: err}
		}
		if trusted != nil {
			err = plugins.SetTrusted(snapshot, name, *trusted)
		} else if toggleEnabled {
			for i := range snapshot.Plugins.Items {
				if snapshot.Plugins.Items[i].Name == name {
					snapshot.Plugins.Items[i].Enabled = !snapshot.Plugins.Items[i].Enabled
					break
				}
			}
		}
		if err == nil {
			err = config.Save(snapshot)
		}
		message := "plugin state saved"
		if trusted != nil && *trusted {
			message = "plugin trusted and saved: " + name
		} else if trusted != nil {
			message = "plugin untrusted and saved: " + name
		}
		return mutationResult{kind: mutationPlugin, cfg: snapshot, selected: name, message: message, err: err}
	}
}

func ensureConfigUnchanged(expected *config.Config, expectedExists bool) error {
	if expected == nil {
		return nil
	}
	if _, err := os.Stat(config.Path()); err != nil {
		if os.IsNotExist(err) {
			if expectedExists {
				return fmt.Errorf("config.toml was removed outside this TUI; reload before saving")
			}
			return nil
		}
		return err
	}
	current, err := config.LoadExisting()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, expected) {
		return fmt.Errorf("config.toml changed outside this TUI; reload before saving")
	}
	return nil
}

func fixDoctorCommand(cfg, expected *config.Config, expectedExists bool) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		if !shell.ConfigExists() {
			if err := ensureConfigUnchanged(expected, expectedExists); err != nil {
				return mutationResult{kind: mutationDoctorFix, err: err}
			}
			if err := config.Save(snapshot); err != nil {
				return mutationResult{kind: mutationDoctorFix, err: err}
			}
		}
		if !shell.ZshrcExists() {
			_, target, resolveErr := shell.ResolveZshrcTarget()
			if resolveErr != nil {
				return mutationResult{kind: mutationDoctorFix, err: resolveErr}
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return mutationResult{kind: mutationDoctorFix, err: err}
			}
			file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return mutationResult{kind: mutationDoctorFix, err: err}
			}
			if err := file.Close(); err != nil {
				return mutationResult{kind: mutationDoctorFix, err: err}
			}
		}
		message := "no auto-fixes needed"
		if !shell.HasBlock() {
			message = "open Apply tab and confirm .zshrc change"
		}
		return mutationResult{kind: mutationDoctorFix, cfg: snapshot, message: message}
	}
}
