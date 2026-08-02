package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func TestApplyRequiresConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	model := NewModel(config.Default())
	model.tab = 3
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)

	if !model.confirmApply {
		t.Fatal("Apply did not enter confirmation state")
	}
	if strings.Contains(readFile(t, shell.ZshrcPath()), "ozsh") {
		t.Fatal("Apply modified .zshrc before confirmation")
	}
	if !strings.Contains(model.apply(), "Press y to apply") {
		t.Fatalf("Apply view missing confirmation prompt:\n%s", model.apply())
	}
}

func TestPreviewContextIsEditable(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	model.inputs[0].SetValue("pilot")
	model.inputs[1].SetValue("~/ship")
	model.inputs[2].SetValue("feature")
	model.inputs[3].SetValue("7")
	model.syncPreviewInputs()

	view := model.preview()
	for _, want := range []string{"pilot", "~/ship", "feature", "✘ 7"} {
		if !strings.Contains(view, want) {
			t.Fatalf("preview missing %q:\n%s", want, view)
		}
	}
}

func TestThemeAndPluginControls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	pluginDir := filepath.Join(home, ".config", "ozsh", "plugins", "demo")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.zsh"), []byte("# plugin\n"), 0644); err != nil {
		t.Fatalf("WriteFile(plugin.zsh) error = %v", err)
	}
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: false, Trusted: false, Source: pluginDir, Load: "plugin.zsh"},
	}
	model := NewModel(cfg)

	model.setTab(tabThemes)
	model.cursor = 0
	model.applyThemeAtCursor()
	if model.cfg.Theme.Name == "" {
		t.Fatal("theme was not applied")
	}

	model.tab = tabPlugins
	model.pluginAdvanced = true
	model.cursor = len(model.cfg.Plugins.Selected)
	model.togglePluginAtCursor()
	model.trustPluginAtCursor()
	if !model.cfg.Plugins.Items[0].Enabled || !model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("plugin controls did not enable and trust plugin")
	}
}

func TestPluginTrustControlRejectsUnsafeLoadFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: filepath.Join(home, ".config", "ozsh", "plugins", "demo"), Load: "plugin.zsh"},
	}
	model := NewModel(cfg)
	model.tab = tabPlugins
	model.pluginAdvanced = true
	model.cursor = len(model.cfg.Plugins.Selected)

	model.trustPluginAtCursor()

	if model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("trust control trusted plugin with missing load file")
	}
	if !strings.Contains(model.msg, "trust") {
		t.Fatalf("trust control message = %q, want trust error", model.msg)
	}
}

func TestNumericKeysSwitchTabsAndHeavyToggle(t *testing.T) {
	model := NewModel(config.Default())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	if model.tab != 1 {
		t.Fatalf("numeric key tab = %d, want builder tab", model.tab)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model = updated.(Model)
	if !model.cfg.Prompt.DisableHeavySegments {
		t.Fatal("h key did not disable heavy segments")
	}
}

func TestBuilderToggleAndReorderSegments(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = 1
	model.cursor = 0
	first := model.cfg.Prompt.Order[0]

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if model.cfg.Prompt.Segments[first].Enabled {
		t.Fatalf("space left segment %q enabled", first)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	model = updated.(Model)
	if model.cursor != 1 {
		t.Fatalf("J cursor = %d, want 1", model.cursor)
	}
	if model.cfg.Prompt.Order[1] != first {
		t.Fatalf("J did not move %q down, order=%v", first, model.cfg.Prompt.Order)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	model = updated.(Model)
	if model.cursor != 0 || model.cfg.Prompt.Order[0] != first {
		t.Fatalf("K did not restore first segment, cursor=%d order=%v", model.cursor, model.cfg.Prompt.Order)
	}
}

func TestPreviewInputFocusCyclesAndSanitizesValues(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.inputFocus != 1 {
		t.Fatalf("down input focus = %d, want 1", model.inputFocus)
	}

	model.inputs[0].SetValue("  pilot\n")
	model.inputs[1].SetValue(" ~/repo\r")
	model.inputs[2].SetValue(" main ")
	model.inputs[3].SetValue("13")
	model.syncPreviewInputs()

	if model.previewCtx.Username != "pilot" || model.previewCtx.Cwd != "~/repo" || model.previewCtx.GitBranch != "main" || model.previewCtx.ExitStatus != 13 {
		t.Fatalf("preview context = %+v, want sanitized input values", model.previewCtx)
	}
}

func TestThemeControlsStoreCustom(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.setTab(tabThemes)

	model.saveCustomTheme()
	if model.cfg.Theme.Name != "custom" || !strings.Contains(model.msg, "custom") {
		t.Fatalf("saveCustomTheme() theme=%q msg=%q, want custom", model.cfg.Theme.Name, model.msg)
	}
}

func TestDoApplyUsesConfigurationSnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = " snapshot "
	cmd := doApply(cfg)
	cfg.Prompt.Separator = " mutated "
	if result := cmd(); result != applyResult("applied") {
		t.Fatalf("doApply result = %#v", result)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if saved.Prompt.Separator != " snapshot " {
		t.Fatalf("saved separator = %q, want snapshot", saved.Prompt.Separator)
	}
}

func TestPreviewThemeConfigDoesNotMutateBaseConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Segments["user"] = config.SegmentConfig{Enabled: true, FG: "cyan"}
	model := NewModel(cfg)

	previewCfg := model.previewThemeConfig("dracula")
	if previewCfg.Prompt.Segments["user"].FG == model.cfg.Prompt.Segments["user"].FG {
		t.Fatalf("preview theme user fg = %q, want distinct themed color", previewCfg.Prompt.Segments["user"].FG)
	}
	if model.cfg.Prompt.Segments["user"].FG != "cyan" {
		t.Fatalf("previewThemeConfig mutated base user fg = %q, want cyan", model.cfg.Prompt.Segments["user"].FG)
	}
}

func TestPluginControlsUntrustAndRejectEmptyAddForm(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pluginDir := filepath.Join(home, ".config", "ozsh", "plugins", "demo")
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: true, Source: pluginDir, Load: "plugin.zsh"},
	}
	model := NewModel(cfg)
	model.tab = tabPlugins
	model.pluginAdvanced = true
	model.cursor = len(model.cfg.Plugins.Selected)

	model.untrustPluginAtCursor()
	if model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("untrustPluginAtCursor left plugin trusted")
	}
	if !strings.Contains(model.msg, "untrusted") {
		t.Fatalf("untrust message = %q, want untrusted", model.msg)
	}

	model.addPluginFromInputs()
	if !strings.Contains(model.msg, "URL is required") {
		t.Fatalf("empty plugin form message = %q, want URL required", model.msg)
	}
}

func TestApplyWritesOmegaAndInjectsBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	cfg := config.Default()
	cfg.Prompt.Separator = " | "
	if got := apply(cfg); got != "applied" {
		t.Fatalf("apply() = %q, want applied", got)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if saved.Prompt.Separator != " | " {
		t.Fatalf("apply did not persist config separator = %q", saved.Prompt.Separator)
	}
	if !strings.Contains(readFile(t, shell.OmegaZshPath()), "ozsh_prompt()") {
		t.Fatal("apply did not write generated omega.zsh")
	}
	if !strings.Contains(readFile(t, shell.ZshrcPath()), "source \"$HOME/.config/ozsh/omega.zsh\"") {
		t.Fatal("apply did not inject ozsh block")
	}
}

func TestViewsRenderAllTabs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: filepath.Join(home, ".config", "ozsh", "plugins", "demo"), Load: "plugin.zsh"},
	}
	model := NewModel(cfg)

	for _, tc := range []struct {
		tab  int
		want string
	}{
		{tab: tabHome, want: "Welcome"},
		{tab: tabPrompt, want: "Segments"},
		{tab: tabThemes, want: "Theme gallery"},
		{tab: tabPlugins, want: "Recommended setup"},
		{tab: tabPreview, want: "Context"},
	} {
		model.tab = tc.tab
		view := model.View()
		if !strings.Contains(view, tc.want) {
			t.Fatalf("tab %d view missing %q:\n%s", tc.tab, tc.want, view)
		}
	}
}

func TestDoctorFixCreatesMissingFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	model := NewModel(config.Default())

	msg := model.fixDoctor()
	if !strings.Contains(msg, "Review & Apply") {
		t.Fatalf("fixDoctor() = %q, want apply guidance", msg)
	}
	if _, err := os.Stat(config.Path()); err != nil {
		t.Fatalf("fixDoctor did not create config: %v", err)
	}
	if _, err := os.Stat(shell.ZshrcPath()); err != nil {
		t.Fatalf("fixDoctor did not create .zshrc: %v", err)
	}
}

func TestPluginInputsUpdateURLField(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabPlugins
	model.pluginAdvanced = true
	model.focusPluginInput()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)

	if model.pluginURL.Value() != "x" {
		t.Fatalf("plugin URL input = %q, want x", model.pluginURL.Value())
	}
}

func TestPluginInputsAllowBackspaceInURLAndLoad(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabPlugins
	model.pluginAdvanced = true
	model.focusPluginInput()
	model.pluginURL.SetValue("https://example.com/x")
	model.pluginURL.SetCursor(len(model.pluginURL.Value()))

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(Model)
	if model.pluginURL.Value() != "https://example.com/" {
		t.Fatalf("plugin URL after backspace = %q", model.pluginURL.Value())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model.pluginLoad.SetValue("plugin.zsh")
	model.pluginLoad.SetCursor(len(model.pluginLoad.Value()))
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(Model)
	if model.pluginLoad.Value() != "plugin.zs" {
		t.Fatalf("plugin load after backspace = %q", model.pluginLoad.Value())
	}
}

func TestTabNavigationWorksFromPreviewAndBuilder(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if model.tab != tabHome {
		t.Fatalf("preview tab navigation = %d, want home tab", model.tab)
	}

	model.setTab(tabBuilder)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if model.tab != tabThemes {
		t.Fatalf("builder tab navigation = %d, want themes tab", model.tab)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(Model)
	if model.tab != tabPlugins {
		t.Fatalf("themes right navigation = %d, want plugins tab", model.tab)
	}
}

func TestPluginNavigationWorksWhileKeepingNumericURLInput(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(Model)
	if model.tab != tabPreview {
		t.Fatalf("plugin right navigation = %d, want preview tab", model.tab)
	}

	model.setTab(tabPlugins)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	if model.tab != tabPrompt || model.pluginURL.Value() != "" {
		t.Fatalf("empty plugin numeric navigation tab=%d url=%q, want prompt tab and empty url", model.tab, model.pluginURL.Value())
	}

	model.setTab(tabPlugins)
	model.pluginAdvanced = true
	model.focusPluginInput()
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	if model.tab != tabPlugins || model.pluginURL.Value() != "h2" {
		t.Fatalf("active plugin URL numeric input tab=%d url=%q, want plugins tab and h2", model.tab, model.pluginURL.Value())
	}
}

func TestSegmentItemFormatting(t *testing.T) {
	item := segmentItem{name: "git", enabled: true, fg: "magenta", bold: true}

	if item.FilterValue() != "git" {
		t.Fatalf("FilterValue() = %q, want git", item.FilterValue())
	}
	if got := item.Title(); got != "[x] git" {
		t.Fatalf("Title() = %q, want enabled title", got)
	}
	if got := item.Description(); !strings.Contains(got, "fg=magenta") || !strings.Contains(got, "bold=bold") || !strings.Contains(got, "icon=none") {
		t.Fatalf("Description() = %q, want fg/bold/icon details", got)
	}
}

func TestWriteCheckRendersPassAndFail(t *testing.T) {
	var b strings.Builder

	writeCheck(&b, true, "zsh installed")
	writeCheck(&b, false, "config exists")

	got := b.String()
	if !strings.Contains(got, "✓ zsh installed") || !strings.Contains(got, "✗ config exists") {
		t.Fatalf("writeCheck output = %q, want pass and fail lines", got)
	}
}

func TestTermuxPreviewConfigDisablesHeavySegments(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "1")
	model := NewModel(config.Default())
	model.cfg.Prompt.RightPrompt = true
	model.cfg.Prompt.RightOrder = []string{"time"}

	previewCfg := model.previewConfig()
	if !previewCfg.Prompt.DisableHeavySegments {
		t.Fatal("Termux preview config did not disable heavy segments")
	}
	if previewCfg.Prompt.RightPrompt || len(previewCfg.Prompt.RightOrder) != 0 {
		t.Fatal("Termux preview config did not simplify right prompt")
	}
}

func TestSanitizeInputRemovesControlLines(t *testing.T) {
	if got := sanitizeInput("  hello\nworld\r  "); got != "helloworld" {
		t.Fatalf("sanitizeInput() = %q, want helloworld", got)
	}
}

func TestModelViewUsesLessThan50MB(t *testing.T) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	model := NewModel(config.Default())
	_ = model.View()
	runtime.ReadMemStats(&after)

	used := after.Alloc - before.Alloc
	if used > 50*1024*1024 {
		t.Fatalf("TUI view allocated %d bytes, want < 50MB", used)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
