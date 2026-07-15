package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5ff")).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b6b80"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff003c")).Bold(true)
	panelStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#09090d")).Foreground(lipgloss.Color("#e0e0e0")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#00f5ff")).Padding(1, 2)
)

var tabs = []string{"Dashboard", "Builder", "Preview", "Apply", "Doctor", "Themes", "Headers", "Plugins"}

const (
	tabDashboard = iota
	tabBuilder
	tabPreview
	tabApply
	tabDoctor
	tabThemes
	tabHeaders
	tabPlugins
)

// Model keeps one selection cursor, but clamps and resets it per tab. This
// prevents a position valid in Builder from being reused against Themes,
// Headers, or Plugins.
type Model struct {
	cfg *config.Config

	tab    int
	cursor int
	msg    string
	width  int
	height int

	confirmApply bool
	applyDiff    string
	busy         bool

	previewCtx   prompt.PreviewContext
	inputs       []textinput.Model
	inputFocus   int
	previewError string

	tuiTheme string

	pluginURL   textinput.Model
	pluginLoad  textinput.Model
	pluginFocus int
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(NewModel(cfg), tea.WithAltScreen()).Run()
	return err
}

func NewModel(cfg *config.Config) Model {
	if cfg == nil {
		cfg = config.Default()
	}
	inputs := previewInputs(prompt.DefaultPreviewContext())
	pluginURL := textinput.New()
	pluginURL.Placeholder = "https://github.com/user/plugin.git"
	pluginURL.Prompt = "url: "
	pluginURL.Focus()
	pluginLoad := textinput.New()
	pluginLoad.Placeholder = "plugin.zsh"
	pluginLoad.Prompt = "load: "

	m := Model{
		cfg:        cfg,
		previewCtx: prompt.DefaultPreviewContext(),
		inputs:     inputs,
		tuiTheme:   "dark",
		pluginURL:  pluginURL,
		pluginLoad: pluginLoad,
	}
	m.syncCursor()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case applyResult:
		m.busy = false
		m.msg = string(msg)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (msg.String() == "q" && m.tab != tabPreview && m.tab != tabPlugins) {
			return m, tea.Quit
		}
		if m.formAllowsGlobalTabNavigation(msg) {
			switch msg.String() {
			case "tab", "right":
				m.setTab((m.tab + 1) % len(tabs))
				return m, nil
			case "shift+tab", "left":
				m.setTab((m.tab + len(tabs) - 1) % len(tabs))
				return m, nil
			}
		}

		// Form tabs own printable input. This prevents global shortcut handling
		// from swallowing text, including numeric exit statuses and plugin URLs.
		if m.tab == tabPreview {
			return m.updatePreviewInputs(msg)
		}
		if m.tab == tabPlugins && m.pluginInputShouldHandle(msg) {
			return m.updatePluginInputs(msg)
		}

		switch msg.String() {
		case "1", "2", "3", "4", "5", "6", "7", "8":
			idx, _ := strconv.Atoi(msg.String())
			m.setTab(idx - 1)
		case "tab", "right":
			m.setTab((m.tab + 1) % len(tabs))
		case "shift+tab", "left":
			m.setTab((m.tab + len(tabs) - 1) % len(tabs))
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case " ":
			if m.tab == tabBuilder {
				m.toggleSegment()
			}
		case "enter":
			m.handleEnter()
		case "J":
			if m.tab == tabBuilder {
				m.moveSegment(1)
			}
		case "K":
			if m.tab == tabBuilder {
				m.moveSegment(-1)
			}
		case "h":
			if m.tab == tabBuilder {
				m.cfg.Prompt.DisableHeavySegments = !m.cfg.Prompt.DisableHeavySegments
				if m.cfg.Prompt.DisableHeavySegments {
					m.msg = "heavy segments disabled"
				} else {
					m.msg = "heavy segments enabled"
				}
			}
		case "s":
			if err := config.Save(m.cfg); err != nil {
				m.msg = "save error: " + err.Error()
			} else {
				m.msg = "config saved"
			}
		case "a":
			if m.tab == tabApply {
				before, after, err := shell.PreviewInjectBlock()
				if err != nil {
					m.msg = "apply preview error: " + err.Error()
					return m, nil
				}
				m.applyDiff = shell.DiffLines(before, after)
				m.confirmApply = true
				m.msg = "press y to apply, n or esc to cancel"
			}
		case "y":
			if m.tab == tabApply && m.confirmApply && !m.busy {
				m.busy = true
				m.confirmApply = false
				m.msg = ""
				return m, doApply(m.cfg)
			}
		case "n", "esc":
			if m.confirmApply {
				m.confirmApply = false
				m.msg = "apply cancelled"
			}
		case "c":
			if m.tab == tabThemes {
				m.saveCustomTheme()
			}
		case "t":
			if m.tab == tabThemes {
				m.nextTUITheme()
			} else if m.tab == tabPlugins {
				m.trustPluginAtCursor()
			}
		case "u":
			if m.tab == tabPlugins {
				m.untrustPluginAtCursor()
			}
		case "p":
			if m.tab == tabPlugins && m.pluginURL.Value() == "" && m.pluginLoad.Value() == "" {
				m.addPluginFromInputs()
			}
		case "?":
			m.msg = "1-8 tabs | arrows move | space toggle | J/K reorder | s save | q quit"
		}
	}
	return m, nil
}

func (m *Model) handleEnter() {
	switch m.tab {
	case tabBuilder:
		m.toggleSegment()
	case tabDoctor:
		m.msg = m.fixDoctor()
	case tabThemes:
		m.applyThemeAtCursor()
	case tabHeaders:
		m.applyHeaderAtCursor()
	case tabPlugins:
		if strings.TrimSpace(m.pluginURL.Value()) != "" {
			m.addPluginFromInputs()
			return
		}
		m.togglePluginAtCursor()
	}
}

func (m *Model) setTab(tab int) {
	if tab < 0 {
		tab = 0
	}
	if tab >= len(tabs) {
		tab = len(tabs) - 1
	}
	m.tab = tab
	m.cursor = 0
	m.syncCursor()
}

func (m *Model) moveCursor(delta int) {
	count := m.selectionCount()
	if count <= 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= count {
		m.cursor = count - 1
	}
	m.syncCursor()
}

func (m Model) selectionCount() int {
	switch m.tab {
	case tabBuilder:
		return len(m.cfg.Prompt.Order)
	case tabThemes:
		return len(sortedThemeNames())
	case tabHeaders:
		return len(sortedHeaderNames())
	case tabPlugins:
		return len(m.cfg.Plugins.Items)
	case tabPreview:
		return len(m.inputs)
	default:
		return 0
	}
}

func (m *Model) syncCursor() {
	count := m.selectionCount()
	if count == 0 {
		m.cursor = 0
	} else if m.cursor >= count {
		m.cursor = count - 1
	} else if m.cursor < 0 {
		m.cursor = 0
	}
	if m.tab == tabPreview {
		m.inputFocus = m.cursor
		m.focusPreviewInput()
	}
	if m.tab == tabPlugins {
		m.focusPluginInput()
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("ozsh"))
	b.WriteString(" ")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.tab {
	case tabDashboard:
		b.WriteString(m.dashboard())
	case tabBuilder:
		b.WriteString(m.builder())
	case tabPreview:
		b.WriteString(m.preview())
	case tabApply:
		b.WriteString(m.apply())
	case tabDoctor:
		b.WriteString(m.doctor())
	case tabThemes:
		b.WriteString(m.themes())
	case tabHeaders:
		b.WriteString(m.headers())
	case tabPlugins:
		b.WriteString(m.plugins())
	}

	if m.msg != "" {
		b.WriteString("\n\n")
		if strings.Contains(strings.ToLower(m.msg), "error") || strings.Contains(strings.ToLower(m.msg), "failed") {
			b.WriteString(errorStyle.Render(m.msg))
		} else {
			b.WriteString(mutedStyle.Render(m.msg))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("? help | Ctrl+C quit"))
	return panelStyle.Render(b.String())
}

func (m Model) renderTabs() string {
	items := make([]string, len(tabs))
	for i, tab := range tabs {
		if i == m.tab {
			items[i] = accentStyle.Render(tab)
		} else {
			items[i] = mutedStyle.Render(tab)
		}
	}
	return strings.Join(items, "  ")
}

func (m Model) dashboard() string {
	return fmt.Sprintf("config:   %s\nomega:    %s\nblock:    %t\nplatform: %s/%s\ntermux:   %t\nbackups:  %d\n\nactions: 1-8 switch tabs | Apply + a | Doctor + enter",
		config.Path(), shell.OmegaZshPath(), shell.HasBlock(), runtime.GOOS, runtime.GOARCH, shell.IsTermux(), backupCount())
}

func (m Model) builder() string {
	var b strings.Builder
	b.WriteString("segments\n\n")
	if len(m.cfg.Prompt.Order) == 0 {
		b.WriteString("no segments configured\n")
	} else {
		for i, name := range m.cfg.Prompt.Order {
			seg := m.cfg.Prompt.Segments[name]
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			state := "[ ]"
			if seg.Enabled {
				state = "[x]"
			}
			fmt.Fprintf(&b, "%s%s %-10s fg=%s bold=%t icon=%q\n", prefix, state, name, seg.FG, seg.Bold, seg.Icon)
		}
	}
	b.WriteString("\n")
	b.WriteString(prompt.Simulated(m.previewConfig()))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("space/enter toggle | J/K reorder | h heavy segments | s save"))
	return b.String()
}

func (m Model) preview() string {
	m.syncPreviewInputs()
	var b strings.Builder
	b.WriteString("context\n\n")
	for i := range m.inputs {
		prefix := "  "
		if i == m.inputFocus {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(m.inputs[i].View())
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(prompt.SimulatedWithContext(m.previewConfig(), m.previewCtx))
	if m.previewError != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.previewError))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("up/down switches fields | tab/shift+tab/left/right changes tabs | type edits only the focused field"))
	return b.String()
}

func (m Model) apply() string {
	if m.busy {
		return "applying configuration…"
	}
	if m.confirmApply {
		return "planned .zshrc diff\n\n" + m.applyDiff + "\n\npress y to confirm or n/esc to cancel"
	}
	before, after, err := shell.PreviewInjectBlock()
	if err != nil {
		return "preview error: " + err.Error()
	}
	return "planned .zshrc diff\n\n" + shell.DiffLines(before, after) + "\n\npress a to review and confirm"
}

func (m Model) doctor() string {
	var b strings.Builder
	writeCheck(&b, shell.HasZsh(), "zsh installed")
	writeCheck(&b, shell.ConfigExists(), "config.toml exists")
	writeCheck(&b, shell.ZshrcExists(), ".zshrc exists")
	writeCheck(&b, shell.HasBlock(), "ozsh block present")
	writeCheck(&b, shell.ZshIsDefaultShell(), "zsh is default shell")
	b.WriteString("\n\npress enter to fix auto-resolvable issues")
	return b.String()
}

func (m *Model) fixDoctor() string {
	if !shell.ConfigExists() {
		if err := config.Save(m.cfg); err != nil {
			return "config fix error: " + err.Error()
		}
	}
	if !shell.ZshrcExists() {
		if err := os.MkdirAll(filepath.Dir(shell.ZshrcPath()), 0o700); err != nil {
			return "zshrc fix error: " + err.Error()
		}
		if err := os.WriteFile(shell.ZshrcPath(), []byte{}, 0o600); err != nil {
			return "zshrc fix error: " + err.Error()
		}
	}
	if !shell.HasBlock() {
		return "open Apply tab and confirm .zshrc change"
	}
	return "no auto-fixes needed"
}

func (m Model) themes() string {
	names := sortedThemeNames()
	var b strings.Builder
	b.WriteString("theme gallery\n\n")
	if len(names) == 0 {
		return "theme gallery\n\nno presets available"
	}
	for i, name := range names {
		preset := config.Presets[name]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s  accent=%s success=%s error=%s\n", prefix, name, preset.Accent, preset.Success, preset.Error)
	}
	b.WriteString("\n")
	b.WriteString(prompt.Simulated(m.previewThemeConfig(names[m.cursor])))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter applies | c stores current theme as custom | t cycles TUI label"))
	return b.String()
}

func (m Model) headers() string {
	names := sortedHeaderNames()
	var b strings.Builder
	b.WriteString("headers\n\n")
	if len(names) == 0 {
		return "headers\n\nno presets available"
	}
	for i, name := range names {
		preset := config.HeaderPresets[name]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s  style=%s text=%q\n", prefix, name, preset.Style, preset.Text)
	}
	selected := config.HeaderPresets[names[m.cursor]]
	b.WriteString("\npreview\n")
	b.WriteString(selected.Text)
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter applies selected header"))
	return b.String()
}

func (m Model) plugins() string {
	var b strings.Builder
	b.WriteString("plugins\n\n")
	if len(m.cfg.Plugins.Items) == 0 {
		b.WriteString("no plugins configured\n")
	}
	for i, item := range m.cfg.Plugins.Items {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		state := "disabled"
		if item.Enabled {
			state = "enabled"
		}
		trust := "untrusted"
		if item.Trusted {
			trust = "trusted"
		}
		fmt.Fprintf(&b, "%s%s  %s  %s\n", prefix, item.Name, state, trust)
	}
	b.WriteString("\nadd plugin\n")
	b.WriteString(m.pluginURL.View())
	b.WriteByte('\n')
	b.WriteString(m.pluginLoad.View())
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("tab switches form field | enter adds when URL is present, otherwise toggles selected | t trust | u untrust"))
	return b.String()
}

func (m *Model) toggleSegment() {
	if m.tab != tabBuilder || m.cursor < 0 || m.cursor >= len(m.cfg.Prompt.Order) {
		return
	}
	name := m.cfg.Prompt.Order[m.cursor]
	seg := m.cfg.Prompt.Segments[name]
	seg.Enabled = !seg.Enabled
	m.cfg.Prompt.Segments[name] = seg
}

func (m *Model) moveSegment(delta int) {
	if m.tab != tabBuilder {
		return
	}
	next := m.cursor + delta
	if m.cursor < 0 || next < 0 || next >= len(m.cfg.Prompt.Order) {
		return
	}
	m.cfg.Prompt.Order[m.cursor], m.cfg.Prompt.Order[next] = m.cfg.Prompt.Order[next], m.cfg.Prompt.Order[m.cursor]
	m.cursor = next
}

func (m Model) updatePreviewInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "down":
			m.inputFocus = wrapIndex(m.inputFocus+1, len(m.inputs))
			m.cursor = m.inputFocus
			m.focusPreviewInput()
			return m, nil
		case "up":
			m.inputFocus = wrapIndex(m.inputFocus-1, len(m.inputs))
			m.cursor = m.inputFocus
			m.focusPreviewInput()
			return m, nil
		}
	}
	if len(m.inputs) == 0 {
		return m, nil
	}
	m.focusPreviewInput()
	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	m.syncPreviewInputs()
	return m, cmd
}

func (m Model) formAllowsGlobalTabNavigation(msg tea.KeyMsg) bool {
	if m.tab != tabPreview {
		return false
	}
	switch msg.String() {
	case "tab", "shift+tab", "left", "right":
		return true
	}
	return false
}

func (m *Model) focusPreviewInput() {
	for i := range m.inputs {
		if i == m.inputFocus {
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
	case "backspace", "delete", "left", "right", "ctrl+h", "ctrl+w", "ctrl+u", "ctrl+k", "home", "end":
		return true
	case "up", "down", "j", "k", "enter":
		return false
	}
	if msg.Type != tea.KeyRunes {
		return false
	}
	// Keep legacy quick actions when the form is empty, but once the user
	// begins a URL or load path every rune belongs to the focused input.
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
	clone := cloneConfig(m.cfg)
	if shell.IsTermux() {
		clone.Prompt.DisableHeavySegments = true
		clone.Prompt.RightPrompt = false
		clone.Prompt.RightOrder = nil
	}
	return clone
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
		if i == 0 {
			input.Focus()
		}
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
	names := sortedThemeNames()
	if len(names) == 0 || m.cursor < 0 || m.cursor >= len(names) {
		return
	}
	preset := config.Presets[names[m.cursor]]
	m.cfg.Theme = preset
	if seg, ok := m.cfg.Prompt.Segments["user"]; ok {
		seg.FG = preset.Accent
		m.cfg.Prompt.Segments["user"] = seg
	}
	if seg, ok := m.cfg.Prompt.Segments["status"]; ok {
		seg.FG = preset.Error
		m.cfg.Prompt.Segments["status"] = seg
	}
	m.msg = "theme applied: " + preset.Name
}

func (m *Model) saveCustomTheme() {
	m.cfg.Theme.Name = "custom"
	m.msg = "custom theme stored in config"
}

func (m *Model) nextTUITheme() {
	switch m.tuiTheme {
	case "dark":
		m.tuiTheme = "light"
	case "light":
		m.tuiTheme = "terminal"
	default:
		m.tuiTheme = "dark"
	}
	m.msg = "TUI theme: " + m.tuiTheme
}

func (m Model) previewThemeConfig(name string) *config.Config {
	clone := cloneConfig(m.cfg)
	preset, ok := config.Presets[name]
	if !ok {
		return clone
	}
	clone.Theme = preset
	if seg, ok := clone.Prompt.Segments["user"]; ok {
		seg.FG = preset.Accent
		clone.Prompt.Segments["user"] = seg
	}
	if seg, ok := clone.Prompt.Segments["status"]; ok {
		seg.FG = preset.Error
		clone.Prompt.Segments["status"] = seg
	}
	return clone
}

func (m *Model) applyHeaderAtCursor() {
	names := sortedHeaderNames()
	if len(names) == 0 || m.cursor < 0 || m.cursor >= len(names) {
		return
	}
	m.cfg.Header = config.HeaderPresets[names[m.cursor]]
	m.msg = "header applied: " + names[m.cursor]
}

func (m *Model) togglePluginAtCursor() {
	if m.cursor < 0 || m.cursor >= len(m.cfg.Plugins.Items) {
		return
	}
	m.cfg.Plugins.Items[m.cursor].Enabled = !m.cfg.Plugins.Items[m.cursor].Enabled
}

func (m *Model) trustPluginAtCursor() {
	if m.cursor < 0 || m.cursor >= len(m.cfg.Plugins.Items) {
		return
	}
	name := m.cfg.Plugins.Items[m.cursor].Name
	if err := plugins.SetTrusted(m.cfg, name, true); err != nil {
		m.msg = "plugin trust error: " + err.Error()
		return
	}
	m.msg = "plugin trusted: " + name
}

func (m *Model) untrustPluginAtCursor() {
	if m.cursor < 0 || m.cursor >= len(m.cfg.Plugins.Items) {
		return
	}
	name := m.cfg.Plugins.Items[m.cursor].Name
	if err := plugins.SetTrusted(m.cfg, name, false); err != nil {
		m.msg = "plugin untrust error: " + err.Error()
		return
	}
	m.msg = "plugin untrusted: " + name
}

func (m *Model) addPluginFromInputs() {
	url := sanitizeInput(m.pluginURL.Value())
	load := sanitizeInput(m.pluginLoad.Value())
	if url == "" {
		m.msg = "plugin URL is required"
		return
	}
	if load == "" {
		m.msg = "plugin load file is required"
		return
	}
	name, err := plugins.Add(m.cfg, url, load)
	if err != nil {
		m.msg = "plugin add error: " + err.Error()
		return
	}
	m.pluginURL.SetValue("")
	m.pluginLoad.SetValue("")
	m.pluginFocus = 0
	m.focusPluginInput()
	m.cursor = len(m.cfg.Plugins.Items) - 1
	m.msg = "plugin added: " + name
}

func sortedThemeNames() []string {
	names := make([]string, 0, len(config.Presets))
	for name := range config.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedHeaderNames() []string {
	names := make([]string, 0, len(config.HeaderPresets))
	for name := range config.HeaderPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Kept as a small public-to-package value object for existing callers and tests.
type segmentItem struct {
	name    string
	enabled bool
	fg      string
	bold    bool
	icon    string
}

func (i segmentItem) FilterValue() string { return i.name }
func (i segmentItem) Title() string {
	check := "[ ]"
	if i.enabled {
		check = "[x]"
	}
	return fmt.Sprintf("%s %s", check, i.name)
}
func (i segmentItem) Description() string {
	weight := "normal"
	if i.bold {
		weight = "bold"
	}
	icon := i.icon
	if icon == "" {
		icon = "none"
	}
	return fmt.Sprintf("fg=%s bold=%s icon=%s", i.fg, weight, icon)
}

type applyResult string

func doApply(cfg *config.Config) tea.Cmd {
	return func() tea.Msg { return applyResult(apply(cfg)) }
}

func apply(cfg *config.Config) string {
	if err := shell.EnsureOzshDir(); err != nil {
		return "setup error: " + err.Error()
	}
	generated, err := prompt.Generate(cloneConfig(cfg))
	if err != nil {
		return "generator error: " + err.Error()
	}
	if err := atomicWriteFile(shell.OmegaZshPath(), []byte(generated), 0o600); err != nil {
		return "write error: " + err.Error()
	}
	if err := shell.InjectBlock(); err != nil {
		return "inject error: " + err.Error()
	}
	return "applied"
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ozsh-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
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
