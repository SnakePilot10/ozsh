package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

func promptSegmentIcon(promptConfig config.PromptConfig, segment config.SegmentConfig) string {
	if promptConfig.IconMode == config.IconModeNerd && segment.NerdIcon != "" {
		return segment.NerdIcon
	}
	if segment.CompatibleIcon != "" {
		return segment.CompatibleIcon
	}
	return segment.Icon
}

func segmentLabel(name string) string {
	labels := map[string]string{
		"user": "User", "cwd": "Directory", "git": "Git", "status": "Status",
		"time": "Time", "host": "Host", "venv": "Python", "node": "Node.js",
		"go": "Go", "battery": "Battery", "jobs": "Jobs",
	}
	if label, ok := labels[name]; ok {
		return label
	}
	return name
}

func (m Model) formAllowsGlobalNavigation(msg tea.KeyMsg) bool {
	switch m.tab {
	case tabPreview:
		if m.previewEditing {
			return false
		}
		switch msg.String() {
		case "tab", "shift+tab", "left", "right":
			return true
		}
	case tabPlugins:
		switch msg.String() {
		case "left", "right":
			return true
		case "1", "2", "3", "4", "5":
			return !m.pluginAdvanced || (m.pluginURL.Value() == "" && m.pluginLoad.Value() == "")
		}
	}
	return false
}

func (m *Model) focusPreviewInput() {
	if !m.previewEditing {
		m.inputFocus = wrapIndex(m.previewScenario-1, len(m.inputs))
	}
	for i := range m.inputs {
		if m.previewEditing && i == m.inputFocus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m Model) pluginInputShouldHandle(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "tab", "shift+tab":
		return true
	case "backspace", "delete", "ctrl+h", "ctrl+w", "ctrl+u", "ctrl+k", "home", "end":
		return true
	case "up", "down", "j", "k", "enter":
		return false
	}
	if msg.Type != tea.KeyRunes {
		return false
	}

	if m.pluginURL.Value() == "" && m.pluginLoad.Value() == "" {
		switch msg.String() {
		case "p", "t", "u":
			return false
		}
	}
	return true
}

func (m Model) updatePluginInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab":
			m.pluginFocus = (m.pluginFocus + 1) % 2
			m.focusPluginInput()
			return m, nil
		case "shift+tab":
			m.pluginFocus = (m.pluginFocus + 1) % 2
			m.focusPluginInput()
			return m, nil
		}
	}
	m.focusPluginInput()
	var cmd tea.Cmd
	if m.pluginFocus == 0 {
		m.pluginURL, cmd = m.pluginURL.Update(msg)
	} else {
		m.pluginLoad, cmd = m.pluginLoad.Update(msg)
	}
	return m, cmd
}

func (m *Model) focusPluginInput() {
	if m.pluginFocus == 0 {
		m.pluginURL.Focus()
		m.pluginLoad.Blur()
		return
	}
	m.pluginURL.Blur()
	m.pluginLoad.Focus()
}

func (m *Model) syncPreviewInputs() {
	if len(m.inputs) < 4 {
		return
	}
	m.previewCtx.Username = sanitizeInput(m.inputs[0].Value())
	m.previewCtx.Cwd = sanitizeInput(m.inputs[1].Value())
	m.previewCtx.GitBranch = sanitizeInput(m.inputs[2].Value())
	statusRaw := strings.TrimSpace(m.inputs[3].Value())
	if statusRaw == "" {
		m.previewError = ""
		m.previewCtx.ExitStatus = 0
		return
	}
	status, err := strconv.Atoi(statusRaw)
	if err != nil {
		m.previewError = "exit status must be an integer"
		return
	}
	m.previewError = ""
	m.previewCtx.ExitStatus = status
}

func (m Model) previewConfig() *config.Config {
	cfg := cloneConfig(m.cfg)
	if shell.IsTermux() {
		cfg.Prompt.DisableHeavySegments = true
		cfg.Prompt.RightPrompt = false
		cfg.Prompt.RightOrder = []string{}
	}
	return cfg
}

func previewInputs(ctx prompt.PreviewContext) []textinput.Model {
	values := []struct{ label, value string }{
		{"username", ctx.Username},
		{"cwd", ctx.Cwd},
		{"git", ctx.GitBranch},
		{"exit", strconv.Itoa(ctx.ExitStatus)},
	}
	inputs := make([]textinput.Model, len(values))
	for i, value := range values {
		input := textinput.New()
		input.Prompt = value.label + ": "
		input.Placeholder = value.label
		input.SetValue(value.value)
		inputs[i] = input
	}
	return inputs
}

func sanitizeInput(value string) string {
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	return strings.TrimSpace(value)
}

func (m *Model) applyThemeAtCursor() {
	preset, ok := m.selectedTheme()
	if !ok {
		return
	}
	m.cfg = themecatalog.Apply(m.cfg, preset)
	m.msg = "theme selected: " + preset.Name + " · use Review & Apply to save"
}

func (m *Model) saveCustomTheme() {
	m.cfg.Theme.ID = "custom"
	m.cfg.Theme.Variant = ""
	m.cfg.Theme.Name = "custom"
	m.msg = "custom theme staged · use Review & Apply to save"
}

func (m Model) previewThemeConfig(name string) *config.Config {
	variant := ""
	if name == "circuit" {
		variant = "blue"
	}
	preset, ok := themecatalog.Get(name, variant)
	if !ok {
		return cloneConfig(m.cfg)
	}
	return themecatalog.Apply(m.cfg, preset)
}

func (m Model) selectedTheme() (themecatalog.Preset, bool) {
	presets := themecatalog.List()
	if m.cursor < 0 || m.cursor >= len(presets) {
		return themecatalog.Preset{}, false
	}
	return presets[m.cursor], true
}
