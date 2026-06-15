package tui

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
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

type Model struct {
	cfg          *config.Config
	tab          int
	cursor       int
	msg          string
	width        int
	height       int
	spin         spinner.Model
	busy         bool
	list         list.Model
	confirmApply bool
	applyDiff    string
	previewCtx   prompt.PreviewContext
	inputs       []textinput.Model
	inputFocus   int
	tuiTheme     string
	pluginURL    textinput.Model
	pluginLoad   textinput.Model
}

var tabs = []string{"Dashboard", "Builder", "Preview", "Apply", "Doctor", "Themes", "Headers", "Plugins"}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(NewModel(cfg), tea.WithAltScreen()).Run()
	return err
}

func NewModel(cfg *config.Config) Model {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = accentStyle
	inputs := previewInputs(prompt.DefaultPreviewContext())
	pluginURL := textinput.New()
	pluginURL.Placeholder = "https://github.com/user/plugin.git"
	pluginLoad := textinput.New()
	pluginLoad.Placeholder = "plugin.zsh"
	return Model{
		cfg:        cfg,
		spin:       spin,
		list:       newSegmentList(cfg),
		previewCtx: prompt.DefaultPreviewContext(),
		inputs:     inputs,
		tuiTheme:   "dark",
		pluginURL:  pluginURL,
		pluginLoad: pluginLoad,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case applyResult:
		m.busy = false
		m.msg = string(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.tab == 2 && previewInputKey(msg) {
			return m.updatePreviewInputs(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1", "2", "3", "4", "5", "6", "7", "8":
			idx, _ := strconv.Atoi(msg.String())
			if idx >= 1 && idx <= len(tabs) {
				m.tab = idx - 1
				m.cursor = 0
			}
		case "tab", "right":
			m.tab = (m.tab + 1) % len(tabs)
			m.cursor = 0
		case "shift+tab", "left":
			m.tab = (m.tab + len(tabs) - 1) % len(tabs)
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.syncCursor()
		case "down", "j":
			if m.cursor < len(m.cfg.Prompt.Order)-1 {
				m.cursor++
			}
			m.syncCursor()
		case " ", "enter":
			if m.tab == 1 {
				m.toggleSegment()
			} else if m.tab == 4 {
				m.msg = m.fixDoctor()
			} else if m.tab == 5 {
				m.applyThemeAtCursor()
			} else if m.tab == 6 {
				m.applyHeaderAtCursor()
			} else if m.tab == 7 {
				m.togglePluginAtCursor()
			}
		case "J":
			if m.tab == 1 {
				m.moveSegment(1)
			}
		case "K":
			if m.tab == 1 {
				m.moveSegment(-1)
			}
		case "s":
			m.syncPreviewInputs()
			if err := config.Save(m.cfg); err != nil {
				m.msg = err.Error()
			} else {
				m.msg = "config saved"
			}
		case "c":
			if m.tab == 5 {
				m.saveCustomTheme()
			}
		case "t":
			if m.tab == 5 {
				m.nextTUITheme()
			} else if m.tab == 7 {
				m.trustPluginAtCursor()
			}
		case "h":
			if m.tab == 1 {
				m.cfg.Prompt.DisableHeavySegments = !m.cfg.Prompt.DisableHeavySegments
				if m.cfg.Prompt.DisableHeavySegments {
					m.msg = "heavy segments disabled"
				} else {
					m.msg = "heavy segments enabled"
				}
			}
		case "u":
			if m.tab == 7 {
				m.untrustPluginAtCursor()
			}
		case "p":
			if m.tab == 7 {
				m.addPluginFromInputs()
			}
		case "a":
			if m.tab == 3 {
				before, after, err := shell.PreviewInjectBlock()
				if err != nil {
					m.msg = err.Error()
					return m, nil
				}
				m.applyDiff = shell.DiffLines(before, after)
				m.confirmApply = true
				m.msg = "press y to apply, n to cancel"
			}
		case "y":
			if m.tab == 3 && m.confirmApply {
				m.busy = true
				m.confirmApply = false
				m.msg = ""
				return m, tea.Batch(m.spin.Tick, doApply(m.cfg))
			}
		case "n", "esc":
			if m.confirmApply {
				m.confirmApply = false
				m.msg = "apply cancelled"
			}
		case "?":
			m.msg = "tab switch | q quit | space toggle | J/K reorder | s save | a apply"
		}
	}
	if m.tab == 2 {
		return m.updatePreviewInputs(msg)
	}
	if m.tab == 7 {
		return m.updatePluginInputs(msg)
	}
	if m.tab == 1 {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.cursor = m.list.Index()
		return m, cmd
	}
	return m, nil
}

func previewInputKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "tab", "shift+tab", "up", "down", "j", "k", "backspace", "delete", "left", "right", "home", "end":
		return true
	default:
		return msg.Type == tea.KeyRunes
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("ozsh"))
	b.WriteString(" ")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	switch m.tab {
	case 0:
		b.WriteString(m.dashboard())
	case 1:
		b.WriteString(m.builder())
	case 2:
		b.WriteString(m.preview())
	case 3:
		b.WriteString(m.apply())
	case 4:
		b.WriteString(m.doctor())
	case 5:
		b.WriteString(m.themes())
	case 6:
		b.WriteString(m.headers())
	case 7:
		b.WriteString(m.plugins())
	}
	if m.msg != "" {
		b.WriteString("\n\n")
		if strings.Contains(m.msg, "error") || strings.Contains(m.msg, "failed") {
			b.WriteString(errorStyle.Render(m.msg))
		} else {
			b.WriteString(mutedStyle.Render(m.msg))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("? help"))
	return panelStyle.Render(b.String())
}

func (m Model) renderTabs() string {
	rendered := make([]string, len(tabs))
	for i, tab := range tabs {
		if i == m.tab {
			rendered[i] = accentStyle.Render(tab)
		} else {
			rendered[i] = mutedStyle.Render(tab)
		}
	}
	return strings.Join(rendered, "  ")
}

func (m Model) dashboard() string {
	return fmt.Sprintf("config:   %s\nomega:    %s\nblock:    %t\nplatform: %s/%s\ntermux:   %t\nbackups:  %d\n\nactions: tab preview | tab apply + a | tab doctor | q quit",
		config.Path(),
		shell.OmegaZshPath(),
		shell.HasBlock(),
		runtime.GOOS,
		runtime.GOARCH,
		shell.IsTermux(),
		backupCount(),
	)
}

func (m Model) builder() string {
	var b strings.Builder
	b.WriteString("segments\n\n")
	m.list.SetItems(segmentItems(m.cfg))
	m.list.Select(m.cursor)
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("space toggles, J/K reorders, s saves"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("h toggles heavy segments: %t", !m.cfg.Prompt.DisableHeavySegments)))
	b.WriteString("\n\n")
	b.WriteString(m.segmentProperties())
	b.WriteString("\n\n")
	b.WriteString(prompt.Simulated(m.cfg))
	return b.String()
}

func (m Model) segmentProperties() string {
	if len(m.cfg.Prompt.Order) == 0 || m.cursor >= len(m.cfg.Prompt.Order) {
		return ""
	}
	name := m.cfg.Prompt.Order[m.cursor]
	segment := m.cfg.Prompt.Segments[name]
	return fmt.Sprintf("properties: fg=%s bg=%s bold=%t icon=%q enabled=%t",
		segment.FG,
		segment.BG,
		segment.Bold,
		segment.Icon,
		segment.Enabled,
	)
}

func (m Model) preview() string {
	m.syncPreviewInputs()
	var b strings.Builder
	b.WriteString("context\n\n")
	for i := range m.inputs {
		cursor := "  "
		if i == m.inputFocus {
			cursor = "> "
		}
		b.WriteString(cursor)
		b.WriteString(m.inputs[i].View())
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(prompt.SimulatedWithContext(m.previewConfig(), m.previewCtx))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("tab moves fields, type edits, s saves config"))
	return b.String()
}

func (m Model) apply() string {
	if m.busy {
		return m.spin.View() + " generating omega.zsh and updating .zshrc"
	}
	if m.confirmApply {
		return "planned .zshrc diff\n\n" + m.applyDiff + "\n\npress y to confirm or n to cancel"
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
	if !shell.HasZsh() {
		b.WriteString("\ninstall zsh with your package manager")
	}
	if !shell.HasBlock() {
		b.WriteString("\nrun apply from the Apply tab")
	}
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
		if err := os.WriteFile(shell.ZshrcPath(), []byte{}, 0644); err != nil {
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
	for i, name := range names {
		preset := config.Presets[name]
		prefix := "  "
		if i == m.cursor%len(names) {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s  %s %s %s\n", prefix, name, preset.Accent, preset.Success, preset.Error)
	}
	b.WriteString("\n")
	b.WriteString(prompt.Simulated(m.previewThemeConfig(names[m.cursor%len(names)])))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter applies, c saves current theme as custom, t switches TUI theme"))
	return b.String()
}

func (m Model) headers() string {
	names := sortedHeaderNames()
	var b strings.Builder
	b.WriteString("headers\n\n")
	for i, name := range names {
		preset := config.HeaderPresets[name]
		prefix := "  "
		if i == m.cursor%len(names) {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s  style=%s text=%q\n", prefix, name, preset.Style, preset.Text)
	}
	selected := config.HeaderPresets[names[m.cursor%len(names)]]
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
		if i == m.cursor%max(1, len(m.cfg.Plugins.Items)) {
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
	b.WriteString("url  ")
	b.WriteString(m.pluginURL.View())
	b.WriteString("\nload ")
	b.WriteString(m.pluginLoad.View())
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter toggles enabled, t trust, u untrust, p add from form"))
	return b.String()
}

func (m *Model) toggleSegment() {
	if len(m.cfg.Prompt.Order) == 0 || m.cursor >= len(m.cfg.Prompt.Order) {
		return
	}
	name := m.cfg.Prompt.Order[m.cursor]
	segment := m.cfg.Prompt.Segments[name]
	segment.Enabled = !segment.Enabled
	m.cfg.Prompt.Segments[name] = segment
	m.list.SetItems(segmentItems(m.cfg))
}

func (m *Model) moveSegment(delta int) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.cfg.Prompt.Order) {
		return
	}
	m.cfg.Prompt.Order[m.cursor], m.cfg.Prompt.Order[next] = m.cfg.Prompt.Order[next], m.cfg.Prompt.Order[m.cursor]
	m.cursor = next
	m.list.SetItems(segmentItems(m.cfg))
	m.list.Select(m.cursor)
}

func (m *Model) syncCursor() {
	switch m.tab {
	case 1:
		m.list.Select(m.cursor)
	case 2:
		if len(m.inputs) > 0 {
			m.inputFocus = min(m.cursor, len(m.inputs)-1)
			for i := range m.inputs {
				if i == m.inputFocus {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
		}
	}
}

func (m Model) updatePreviewInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down", "j":
			m.inputFocus = (m.inputFocus + 1) % len(m.inputs)
		case "shift+tab", "up", "k":
			m.inputFocus = (m.inputFocus + len(m.inputs) - 1) % len(m.inputs)
		}
	}
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.inputFocus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	m.syncPreviewInputs()
	return m, tea.Batch(cmds...)
}

func (m Model) updatePluginInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m.pluginURL.Focus()
	var cmd tea.Cmd
	m.pluginURL, cmd = m.pluginURL.Update(msg)
	cmds = append(cmds, cmd)
	m.pluginLoad, cmd = m.pluginLoad.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) syncPreviewInputs() {
	if len(m.inputs) < 4 {
		return
	}
	m.previewCtx.Username = sanitizeInput(m.inputs[0].Value())
	m.previewCtx.Cwd = sanitizeInput(m.inputs[1].Value())
	m.previewCtx.GitBranch = sanitizeInput(m.inputs[2].Value())
	status, err := strconv.Atoi(strings.TrimSpace(m.inputs[3].Value()))
	if err == nil {
		m.previewCtx.ExitStatus = status
	}
}

func (m Model) previewConfig() *config.Config {
	if !shell.IsTermux() {
		return m.cfg
	}
	clone := *m.cfg
	clone.Prompt.DisableHeavySegments = true
	clone.Prompt.RightPrompt = false
	clone.Prompt.RightOrder = nil
	return &clone
}

func previewInputs(ctx prompt.PreviewContext) []textinput.Model {
	values := []struct {
		label string
		value string
	}{
		{"username", ctx.Username},
		{"cwd", ctx.Cwd},
		{"git", ctx.GitBranch},
		{"exit", strconv.Itoa(ctx.ExitStatus)},
	}
	inputs := make([]textinput.Model, len(values))
	for i, value := range values {
		input := textinput.New()
		input.Placeholder = value.label
		input.Prompt = value.label + ": "
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
	if len(names) == 0 {
		return
	}
	name := names[m.cursor%len(names)]
	preset := config.Presets[name]
	m.cfg.Theme = preset
	if user, ok := m.cfg.Prompt.Segments["user"]; ok {
		user.FG = preset.Accent
		m.cfg.Prompt.Segments["user"] = user
	}
	if status, ok := m.cfg.Prompt.Segments["status"]; ok {
		status.FG = preset.Error
		m.cfg.Prompt.Segments["status"] = status
	}
	m.msg = "theme applied: " + name
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
	clone := *m.cfg
	clone.Prompt.Segments = map[string]config.SegmentConfig{}
	for key, value := range m.cfg.Prompt.Segments {
		clone.Prompt.Segments[key] = value
	}
	preset := config.Presets[name]
	clone.Theme = preset
	if user, ok := clone.Prompt.Segments["user"]; ok {
		user.FG = preset.Accent
		clone.Prompt.Segments["user"] = user
	}
	if status, ok := clone.Prompt.Segments["status"]; ok {
		status.FG = preset.Error
		clone.Prompt.Segments["status"] = status
	}
	return &clone
}

func (m *Model) applyHeaderAtCursor() {
	names := sortedHeaderNames()
	if len(names) == 0 {
		return
	}
	name := names[m.cursor%len(names)]
	m.cfg.Header = config.HeaderPresets[name]
	m.msg = "header applied: " + name
}

func (m *Model) togglePluginAtCursor() {
	if len(m.cfg.Plugins.Items) == 0 {
		return
	}
	idx := m.cursor % len(m.cfg.Plugins.Items)
	m.cfg.Plugins.Items[idx].Enabled = !m.cfg.Plugins.Items[idx].Enabled
}

func (m *Model) trustPluginAtCursor() {
	if len(m.cfg.Plugins.Items) == 0 {
		return
	}
	idx := m.cursor % len(m.cfg.Plugins.Items)
	name := m.cfg.Plugins.Items[idx].Name
	if err := plugins.SetTrusted(m.cfg, name, true); err != nil {
		m.msg = err.Error()
		return
	}
	m.msg = "plugin trusted: " + name
}

func (m *Model) untrustPluginAtCursor() {
	if len(m.cfg.Plugins.Items) == 0 {
		return
	}
	idx := m.cursor % len(m.cfg.Plugins.Items)
	name := m.cfg.Plugins.Items[idx].Name
	if err := plugins.SetTrusted(m.cfg, name, false); err != nil {
		m.msg = err.Error()
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
	name, err := plugins.Add(m.cfg, url, load)
	if err != nil {
		m.msg = err.Error()
		return
	}
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
	bold := "normal"
	if i.bold {
		bold = "bold"
	}
	icon := i.icon
	if icon == "" {
		icon = "none"
	}
	return fmt.Sprintf("fg=%s bold=%s icon=%s", i.fg, bold, icon)
}

func newSegmentList(cfg *config.Config) list.Model {
	l := list.New(segmentItems(cfg), list.NewDefaultDelegate(), 0, 8)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return l
}

func segmentItems(cfg *config.Config) []list.Item {
	items := make([]list.Item, 0, len(cfg.Prompt.Order))
	for _, name := range cfg.Prompt.Order {
		segment := cfg.Prompt.Segments[name]
		items = append(items, segmentItem{
			name:    name,
			enabled: segment.Enabled,
			fg:      segment.FG,
			bold:    segment.Bold,
			icon:    segment.Icon,
		})
	}
	return items
}

type applyResult string

func doApply(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		return applyResult(apply(cfg))
	}
}

func apply(cfg *config.Config) string {
	if err := shell.EnsureOzshDir(); err != nil {
		return "setup error: " + err.Error()
	}
	generated, err := prompt.Generate(cfg)
	if err != nil {
		return "generator error: " + err.Error()
	}
	if err := os.WriteFile(shell.OmegaZshPath(), []byte(generated), 0644); err != nil {
		return "write error: " + err.Error()
	}
	if err := shell.InjectBlock(); err != nil {
		return "inject error: " + err.Error()
	}
	return "applied"
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
