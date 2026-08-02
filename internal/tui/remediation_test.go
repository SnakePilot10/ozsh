package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func completeOperation(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("operation command is nil")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("operation command returned %T, want non-empty BatchMsg", cmd())
	}
	msg := batch[0]()
	updated, next := model.Update(msg)
	model = updated.(Model)
	if next != nil && model.operation != operationIdle {
		return completeOperation(t, model, next)
	}
	return model
}

func TestTabAwareCursorClampsPerScreen(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabThemes)
	if model.cursor != 0 {
		t.Fatalf("setTab() cursor = %d, want reset to zero", model.cursor)
	}
	model.cursor = 999
	model.syncCursor()
	if got, want := model.cursor, len(sortedThemeNames())-1; got != want {
		t.Fatalf("theme cursor = %d, want %d", got, want)
	}
	model.setTab(tabPlugins)
	if model.cursor != 0 {
		t.Fatalf("plugin cursor = %d, want zero with empty list", model.cursor)
	}
	model.moveCursor(1)
	if model.cursor != 0 {
		t.Fatalf("plugin cursor moved in an empty list: %d", model.cursor)
	}
}

func TestPreviewRoutesInputOnlyToFocusedField(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	beforeCwd := model.inputs[1].Value()

	updated, _ := model.Update(keyRune('x'))
	model = updated.(Model)
	if !strings.HasSuffix(model.inputs[0].Value(), "x") {
		t.Fatalf("username input was not edited: %q", model.inputs[0].Value())
	}
	if model.inputs[1].Value() != beforeCwd {
		t.Fatalf("unfocused cwd input changed: %q", model.inputs[1].Value())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('z'))
	model = updated.(Model)
	if !strings.HasSuffix(model.inputs[1].Value(), "z") {
		t.Fatalf("focused cwd input was not edited: %q", model.inputs[1].Value())
	}
}

func TestPreviewAllowsNumericExitAndDisplaysValidationError(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	model.inputFocus = 3
	model.focusPreviewInput()
	model.inputs[3].SetValue("")

	updated, _ := model.Update(keyRune('7'))
	model = updated.(Model)
	if model.previewCtx.ExitStatus != 7 {
		t.Fatalf("exit status = %d, want 7", model.previewCtx.ExitStatus)
	}

	updated, _ = model.Update(keyRune('x'))
	model = updated.(Model)
	if model.previewError == "" || !strings.Contains(model.preview(), "exit status must be an integer") {
		t.Fatalf("invalid exit status did not surface preview error: %q", model.previewError)
	}
}

func TestPluginFormSeparatesURLAndLoadFocus(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	updated, _ := model.Update(keyRune('p'))
	model = updated.(Model)

	updated, _ = model.Update(keyRune('x'))
	model = updated.(Model)
	if model.pluginURL.Value() != "x" || model.pluginLoad.Value() != "" {
		t.Fatalf("URL focus routing failed: url=%q load=%q", model.pluginURL.Value(), model.pluginLoad.Value())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('p'))
	model = updated.(Model)
	if model.pluginLoad.Value() != "p" || model.pluginURL.Value() != "x" {
		t.Fatalf("load focus routing failed: url=%q load=%q", model.pluginURL.Value(), model.pluginLoad.Value())
	}
}

func TestCloneConfigOwnsMapsSlicesAndPluginItems(t *testing.T) {
	cfg := config.Default()
	clone := cloneConfig(cfg)
	clone.Prompt.Order[0] = "time"
	clone.Prompt.Segments["user"] = config.SegmentConfig{Enabled: false}
	clone.Plugins.Items = append(clone.Plugins.Items, config.PluginItem{Name: "demo"})

	if cfg.Prompt.Order[0] == "time" {
		t.Fatal("cloneConfig shared prompt order")
	}
	if !cfg.Prompt.Segments["user"].Enabled {
		t.Fatal("cloneConfig shared segment map")
	}
	if len(cfg.Plugins.Items) != 0 {
		t.Fatal("cloneConfig shared plugin slice")
	}
}

func TestWrapIndexAndPluginHelpView(t *testing.T) {
	if wrapIndex(-1, 4) != 3 || wrapIndex(5, 4) != 1 || wrapIndex(3, 0) != 0 {
		t.Fatal("wrapIndex produced unexpected values")
	}
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	view := model.plugins()
	if !strings.Contains(view, "add plugin") || !strings.Contains(view, "press p to edit") {
		t.Fatalf("plugin view missing form guidance:\n%s", view)
	}
}

func TestPluginEditModeAcceptsNavigationAndActionRunes(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	updated, _ := model.Update(keyRune('p'))
	model = updated.(Model)

	for _, r := range "jknq?" {
		updated, _ = model.Update(keyRune(r))
		model = updated.(Model)
	}
	if got := model.pluginURL.Value(); got != "jknq?" {
		t.Fatalf("plugin URL = %q, want all action-like runes preserved", got)
	}
	if !model.pluginEditing {
		t.Fatal("plugin form left edit mode while typing")
	}
}

func TestPluginShiftTabMovesToPreviousField(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	updated, _ := model.Update(keyRune('p'))
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updated.(Model)
	if model.pluginFocus != 1 {
		t.Fatalf("shift+tab focus = %d, want previous field 1", model.pluginFocus)
	}
}

func TestPreviewLeftArrowEditsWithinFocusedInput(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPreview)
	model.inputs[0].SetValue("ab")
	model.inputs[0].SetCursor(2)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('x'))
	model = updated.(Model)

	if model.tab != tabPreview || model.inputs[0].Value() != "axb" {
		t.Fatalf("left-arrow edit tab=%d value=%q, want preview and axb", model.tab, model.inputs[0].Value())
	}
}

func TestHelpToggleAndResponsiveLayout(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	model = updated.(Model)

	if model.applyViewport.Width != 54 || model.applyViewport.Height != 11 {
		t.Fatalf("viewport size = %dx%d, want 54x11", model.applyViewport.Width, model.applyViewport.Height)
	}
	if model.builderList.Width() != 54 || model.builderList.Height() != 8 {
		t.Fatalf("builder list size = %dx%d, want 54x8", model.builderList.Width(), model.builderList.Height())
	}
	if model.themesList.Width() != 54 || model.themesList.Height() != 8 {
		t.Fatalf("themes list size = %dx%d, want 54x8", model.themesList.Width(), model.themesList.Height())
	}
	if model.pluginsList.Width() != 54 || model.pluginsList.Height() != 6 {
		t.Fatalf("plugins list size = %dx%d, want 54x6", model.pluginsList.Width(), model.pluginsList.Height())
	}
	if !strings.Contains(model.renderTabs(), "2/7") {
		t.Fatalf("compact tabs missing position: %q", model.renderTabs())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyF1})
	model = updated.(Model)
	if !model.help.ShowAll || !strings.Contains(model.View(), "move down") {
		t.Fatalf("expanded contextual help not rendered:\n%s", model.View())
	}
}

func TestApplyConfirmationViewportScrolls(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabApply
	model.confirmApply = true
	model.applyViewport.Height = 3
	model.applyViewport.SetContent(strings.Repeat("line\n", 20))

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.applyViewport.YOffset == 0 {
		t.Fatal("apply viewport did not scroll")
	}
}

func TestPluginAddRunsAsCommandAndAppliesResult(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.pluginEditing = true
	model.pluginURL.SetValue("https://example.com/demo.git")
	model.pluginLoad.SetValue("plugin.zsh")

	cmd := model.addPluginFromInputs()
	if cmd == nil || model.operation != operationPluginAdd {
		t.Fatalf("plugin add operation=%v cmd nil=%t, want asynchronous command", model.operation, cmd == nil)
	}
	if len(model.cfg.Plugins.Items) != 0 {
		t.Fatal("plugin command mutated live config before completion")
	}

	resultCfg := cloneConfig(model.cfg)
	resultCfg.Plugins.Items = append(resultCfg.Plugins.Items, config.PluginItem{Name: "demo"})
	updated, _ := model.Update(pluginAddResult{cfg: resultCfg, name: "demo"})
	model = updated.(Model)
	if model.operation != operationIdle || model.pluginEditing || len(model.cfg.Plugins.Items) != 1 {
		t.Fatalf("plugin result operation=%v editing=%t items=%d", model.operation, model.pluginEditing, len(model.cfg.Plugins.Items))
	}
}

func TestPluginAddCommandOwnsConfigurationSnapshot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := config.Default()
	cmd := addPlugin(cfg, cfg, false, "http://example.com/demo.git", "plugin.zsh")
	result := cmd().(pluginAddResult)

	if result.err == nil {
		t.Fatal("invalid plugin URL unexpectedly succeeded")
	}
	if result.cfg == cfg || len(cfg.Plugins.Items) != 0 {
		t.Fatal("plugin command shared or mutated live configuration")
	}
}

func TestDoctorResultIsRenderedWithoutViewIO(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabDoctor
	updated, _ := model.Update(doctorResult{{ok: true, label: "zsh installed"}, {ok: false, label: "config.toml exists"}})
	model = updated.(Model)

	view := model.doctor()
	if !strings.Contains(view, "✓ zsh installed") || !strings.Contains(view, "✗ config.toml exists") {
		t.Fatalf("doctor result not rendered:\n%s", view)
	}
}

func TestDoctorTabStartsBackgroundInspection(t *testing.T) {
	model := NewModel(config.Default())
	updated, cmd := model.Update(keyRune('5'))
	model = updated.(Model)

	if model.tab != tabDoctor || model.operation != operationDoctor || cmd == nil {
		t.Fatalf("doctor activation tab=%d operation=%v cmd nil=%t", model.tab, model.operation, cmd == nil)
	}
	if !strings.Contains(model.doctor(), "checking environment") {
		t.Fatalf("doctor loading state missing:\n%s", model.doctor())
	}
}

func TestPluginAddErrorPreservesEditingState(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabPlugins
	model.pluginEditing = true
	model.operation = operationPluginAdd

	updated, _ := model.Update(pluginAddResult{err: os.ErrPermission})
	model = updated.(Model)
	if model.operation != operationIdle || !model.pluginEditing || !strings.Contains(model.msg, "permission denied") {
		t.Fatalf("plugin error operation=%v editing=%t msg=%q", model.operation, model.pluginEditing, model.msg)
	}
}

func TestPluginTrustRequiresExplicitConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pluginDir := filepath.Join(home, ".config", "ozsh", "plugins", "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(plugin) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.zsh"), []byte("# demo\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(plugin) error = %v", err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: pluginDir, Load: "plugin.zsh"}}
	model := NewModel(cfg)
	model.setTab(tabPlugins)

	updated, _ := model.Update(keyRune('t'))
	model = updated.(Model)
	if model.confirmTrust == nil || model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("trust shortcut did not pause before changing trust")
	}
	view := model.plugins()
	if !strings.Contains(view, pluginDir) || !strings.Contains(view, "plugin.zsh") || model.msgLevel != messageWarning {
		t.Fatalf("trust confirmation missing security context:\n%s", view)
	}

	updated, _ = model.Update(keyRune('n'))
	model = updated.(Model)
	if model.confirmTrust != nil || model.cfg.Plugins.Items[0].Trusted {
		t.Fatal("trust cancellation changed plugin state")
	}

	updated, _ = model.Update(keyRune('t'))
	model = updated.(Model)
	updated, cmd := model.Update(keyRune('y'))
	model = updated.(Model)
	model = completeOperation(t, model, cmd)
	if !model.cfg.Plugins.Items[0].Trusted || model.msgLevel != messageSuccess {
		t.Fatalf("confirmed trust trusted=%t level=%v msg=%q", model.cfg.Plugins.Items[0].Trusted, model.msgLevel, model.msg)
	}
}

func TestTypedMessagesAndAdaptiveStyles(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	model.pluginEditing = true
	if cmd := model.addPluginFromInputs(); cmd != nil {
		t.Fatal("empty plugin form unexpectedly returned a command")
	}
	if model.msgLevel != messageError {
		t.Fatalf("empty plugin message level = %v, want error", model.msgLevel)
	}

	foreground, ok := model.styles.accent.GetForeground().(lipgloss.AdaptiveColor)
	if !ok || foreground.Dark != model.cfg.Theme.Accent || foreground.Light == "" {
		t.Fatalf("accent color = %#v, want adaptive theme color", foreground)
	}
	model.tab = tabBuilder
	if tabs := model.renderTabs(); !strings.Contains(tabs, "[2 Builder]") {
		t.Fatalf("active tab lacks non-color marker: %q", tabs)
	}
}

func TestSegmentEditorAppliesValidatedDraftAndMarksDirty(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	name := model.cfg.Prompt.Order[0]
	original := model.cfg.Prompt.Segments[name]

	updated, _ := model.Update(keyRune('e'))
	model = updated.(Model)
	if !model.segmentEditing || model.segmentName != name {
		t.Fatalf("segment editor editing=%t name=%q", model.segmentEditing, model.segmentName)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	model.segmentField = 2
	model.focusSegmentInput()
	updated, _ = model.Update(keyRune('@'))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model = updated.(Model)

	segment := model.cfg.Prompt.Segments[name]
	if model.segmentEditing || !model.dirty || segment.Enabled == original.Enabled || segment.Icon != original.Icon+"@" {
		t.Fatalf("segment edit editing=%t dirty=%t segment=%+v original=%+v", model.segmentEditing, model.dirty, segment, original)
	}
	if !strings.Contains(model.builder(), "* unsaved") {
		t.Fatalf("builder missing dirty marker:\n%s", model.builder())
	}
}

func TestSegmentEditorRejectsInvalidColor(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	name := model.cfg.Prompt.Order[0]
	original := model.cfg.Prompt.Segments[name]
	updated, _ := model.Update(keyRune('e'))
	model = updated.(Model)
	model.segmentInputs[1].SetValue("not-a-color")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model = updated.(Model)
	if !model.segmentEditing || model.dirty || model.msgLevel != messageError {
		t.Fatalf("invalid edit editing=%t dirty=%t level=%v", model.segmentEditing, model.dirty, model.msgLevel)
	}
	if got := model.cfg.Prompt.Segments[name]; got != original {
		t.Fatalf("invalid edit changed segment: got=%+v want=%+v", got, original)
	}
}

func TestBuilderDiscardRestoresSavedSnapshot(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	name := model.cfg.Prompt.Order[0]
	original := model.cfg.Prompt.Segments[name]

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, _ = model.Update(keyRune('d'))
	model = updated.(Model)
	if !model.confirmDiscard {
		t.Fatal("discard did not request confirmation")
	}
	updated, _ = model.Update(keyRune('y'))
	model = updated.(Model)
	if model.dirty || model.confirmDiscard || model.cfg.Prompt.Segments[name] != original {
		t.Fatalf("discard dirty=%t confirm=%t segment=%+v", model.dirty, model.confirmDiscard, model.cfg.Prompt.Segments[name])
	}
}

func TestDirtyBuilderBlocksNormalQuitUntilSaved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)

	updated, cmd := model.Update(keyRune('q'))
	model = updated.(Model)
	if cmd != nil || !model.dirty || !strings.Contains(model.msg, "unsaved") {
		t.Fatalf("dirty quit cmd nil=%t dirty=%t msg=%q", cmd == nil, model.dirty, model.msg)
	}

	updated, cmd = model.Update(keyRune('s'))
	model = updated.(Model)
	model = completeOperation(t, model, cmd)
	if model.dirty || model.savedCfg == nil {
		t.Fatal("save did not clear dirty state")
	}
	_, cmd = model.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("clean builder did not allow normal quit")
	}
}

func TestBuilderFilterTargetsSelectedSegmentActions(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	model.builderList.SetFilterText("git")
	if got := model.selectedBuilderName(); got != "git" {
		t.Fatalf("filtered selection = %q, want git", got)
	}
	original := model.cfg.Prompt.Segments["git"]

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if model.cfg.Prompt.Segments["git"].Enabled == original.Enabled {
		t.Fatal("toggle did not affect filtered git segment")
	}
	if model.selectedBuilderName() != "git" || model.builderList.FilterValue() != "git" {
		t.Fatalf("toggle lost filtered selection: selected=%q filter=%q", model.selectedBuilderName(), model.builderList.FilterValue())
	}

	updated, _ = model.Update(keyRune('e'))
	model = updated.(Model)
	if !model.segmentEditing || model.segmentName != "git" {
		t.Fatalf("editor targeted name=%q editing=%t, want git", model.segmentName, model.segmentEditing)
	}
}

func TestBuilderFilterOwnsPrintableQuitKey(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(keyRune('/'))
	model = updated.(Model)
	if !model.builderList.SettingFilter() {
		t.Fatal("slash did not activate Builder filter")
	}

	updated, cmd := model.Update(keyRune('q'))
	model = updated.(Model)
	if model.builderList.FilterValue() != "q" {
		t.Fatalf("filter q value=%q, want text input", model.builderList.FilterValue())
	}
	if cmd != nil {
		if _, quitting := cmd().(tea.QuitMsg); quitting {
			t.Fatal("filter q unexpectedly quit")
		}
	}
}

func TestBuilderFilteredReorderPreservesSelection(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	model.builderList.SetFilterText("git")
	originalIndex := model.builderIndex()

	updated, _ := model.Update(keyRune('J'))
	model = updated.(Model)
	if got := model.builderIndex(); got != originalIndex+1 {
		t.Fatalf("filtered reorder index = %d, want %d", got, originalIndex+1)
	}
	if model.selectedBuilderName() != "git" || model.builderList.FilterValue() != "git" {
		t.Fatalf("reorder lost selection=%q filter=%q", model.selectedBuilderName(), model.builderList.FilterValue())
	}
}

func TestThemesFilterAppliesSelectedPreset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.setTab(tabThemes)
	model.themesList.SetFilterText("neon-red")
	if got := model.selectedThemeName(); got != "neon-red" {
		t.Fatalf("filtered theme selection = %q, want neon-red", got)
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	model = completeOperation(t, model, cmd)
	if model.cfg.Theme.Name != "neon-red" || model.themesList.FilterValue() != "neon-red" {
		t.Fatalf("theme=%q filter=%q, want filtered neon-red", model.cfg.Theme.Name, model.themesList.FilterValue())
	}
	if !strings.Contains(model.themes(), "[active] neon-red") {
		t.Fatalf("active filtered theme not rendered:\n%s", model.themes())
	}
}

func TestPluginsFilterTargetsToggleAndTrust(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pluginsRoot := filepath.Join(home, ".config", "ozsh", "plugins")
	items := []config.PluginItem{
		{Name: "alpha", Enabled: true, Source: filepath.Join(pluginsRoot, "alpha"), Load: "alpha.zsh"},
		{Name: "target", Enabled: false, Source: filepath.Join(pluginsRoot, "target"), Load: "target.zsh"},
	}
	for _, item := range items {
		if err := os.MkdirAll(item.Source, 0o700); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", item.Name, err)
		}
		if err := os.WriteFile(filepath.Join(item.Source, item.Load), []byte("# plugin\n"), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", item.Name, err)
		}
	}
	cfg := config.Default()
	cfg.Plugins.Items = items
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.pluginsList.SetFilterText("target")
	if got := model.selectedPluginName(); got != "target" {
		t.Fatalf("filtered plugin selection = %q, want target", got)
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	model = completeOperation(t, model, cmd)
	if !model.cfg.Plugins.Items[1].Enabled || !model.cfg.Plugins.Items[0].Enabled {
		t.Fatalf("filtered toggle changed wrong plugin: %+v", model.cfg.Plugins.Items)
	}
	if model.selectedPluginName() != "target" || model.pluginsList.FilterValue() != "target" {
		t.Fatalf("toggle lost selected=%q filter=%q", model.selectedPluginName(), model.pluginsList.FilterValue())
	}

	updated, _ = model.Update(keyRune('t'))
	model = updated.(Model)
	if model.confirmTrust == nil || model.confirmTrust.Name != "target" {
		t.Fatalf("filtered trust targeted %#v, want target", model.confirmTrust)
	}
}

func TestInteractiveListFilterUpdatesSynchronously(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(keyRune('/'))
	model = updated.(Model)
	for _, r := range "git" {
		updated, _ = model.Update(keyRune(r))
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)

	visible := model.builderList.VisibleItems()
	if len(visible) != 1 || model.selectedBuilderName() != "git" || model.builderList.SettingFilter() {
		t.Fatalf("interactive filter visible=%d selected=%q setting=%t", len(visible), model.selectedBuilderName(), model.builderList.SettingFilter())
	}
}

func TestNormalQuitIsIgnoredDuringOperation(t *testing.T) {
	model := NewModel(config.Default())
	model.operation = operationApply
	updated, cmd := model.Update(keyRune('q'))
	model = updated.(Model)
	if cmd != nil || model.operation != operationApply {
		t.Fatalf("busy quit cmd nil=%t operation=%v", cmd == nil, model.operation)
	}
}

func TestStaleOperationResultIsIgnored(t *testing.T) {
	model := NewModel(config.Default())
	model.operation = operationDoctor
	model.requestID = 2
	updated, _ := model.Update(operationResultEnvelope{requestID: 1, msg: applyResult{}})
	model = updated.(Model)
	if model.operation != operationDoctor || model.requestID != 2 {
		t.Fatalf("stale result changed operation=%v request=%d", model.operation, model.requestID)
	}
}

func TestDirtyStateReflectsActualDifference(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if !model.dirty {
		t.Fatal("first toggle did not mark model dirty")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if model.dirty {
		t.Fatal("reverting the toggle left a false dirty state")
	}
}

func TestDirtyBuilderBlocksAutoPersistingTheme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	originalTheme := model.cfg.Theme.Name
	model.setTab(tabThemes)
	model.themesList.SetFilterText("neon-red")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.cfg.Theme.Name != originalTheme || !model.dirty || model.msgLevel != messageWarning {
		t.Fatalf("dirty theme action theme=%q dirty=%t level=%v", model.cfg.Theme.Name, model.dirty, model.msgLevel)
	}
}

func TestApplyRejectsZshrcChangedAfterPreview(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	preview := prepareApply().(applyPreviewResult)
	if preview.err != nil {
		t.Fatalf("prepareApply error = %v", preview.err)
	}
	if err := os.WriteFile(zshrc, []byte("export EDITOR=nano\n"), 0o600); err != nil {
		t.Fatalf("mutate .zshrc error = %v", err)
	}
	result := doApply(config.Default(), preview.base, preview.target, config.Default(), false)().(applyResult)
	if result.err == nil || !strings.Contains(result.err.Error(), "changed after review") {
		t.Fatalf("stale apply error = %v", result.err)
	}
	if got := readFile(t, zshrc); got != "export EDITOR=nano\n" {
		t.Fatalf("stale apply changed .zshrc: %q", got)
	}
}

func TestNewModelOwnsConfiguration(t *testing.T) {
	cfg := config.Default()
	model := NewModel(cfg)
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	name := cfg.Prompt.Order[0]
	if model.cfg.Prompt.Segments[name].Enabled == cfg.Prompt.Segments[name].Enabled {
		t.Fatal("model mutation unexpectedly changed caller configuration")
	}
}

func TestViewFitsTerminalMatrix(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, size := range []struct{ width, height int }{{120, 40}, {80, 24}, {60, 20}, {40, 12}} {
		for tab := range tabs {
			model := NewModel(config.Default())
			model.setTab(tab)
			updated, _ := model.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			model = updated.(Model)
			view := model.View()
			if got := lipgloss.Width(view); got > size.width {
				t.Fatalf("tab %d at %dx%d width=%d", tab, size.width, size.height, got)
			}
			if got := lipgloss.Height(view); got > size.height {
				t.Fatalf("tab %d at %dx%d height=%d", tab, size.width, size.height, got)
			}
		}
	}
}

func TestASCIIAndNoColorView(t *testing.T) {
	t.Setenv("OZSH_ASCII", "1")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	model := NewModel(config.Default())
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	model = updated.(Model)
	view := model.View()
	if strings.Contains(view, "\x1b[") || strings.ContainsAny(view, "╭╮╰╯❯‹›✓✗") {
		t.Fatalf("ASCII/no-color view contains styled or Unicode output: %q", view)
	}
	if !strings.Contains(view, "+") {
		t.Fatalf("ASCII view missing ASCII border: %q", view)
	}
}

func TestPluginFormCancelClearsDraft(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabPlugins)
	updated, _ := model.Update(keyRune('p'))
	model = updated.(Model)
	updated, _ = model.Update(keyRune('x'))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.pluginEditing || model.pluginURL.Value() != "" || model.pluginLoad.Value() != "" {
		t.Fatalf("cancelled plugin draft editing=%t url=%q load=%q", model.pluginEditing, model.pluginURL.Value(), model.pluginLoad.Value())
	}
}

func TestPartialApplySynchronizesSavedSnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	omegaPath := filepath.Join(home, ".config", "ozsh", "omega.zsh")
	if err := os.MkdirAll(omegaPath, 0o700); err != nil {
		t.Fatalf("MkdirAll(omega target) error = %v", err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = " partial "
	model := NewModel(cfg)
	model.savedCfg = config.Default()
	model.markDirty()
	preview, err := shell.PreviewInjectPlan()
	if err != nil {
		t.Fatalf("PreviewInjectBlock error = %v", err)
	}
	result := doApply(model.cfg, preview.Before, preview.Target, model.savedCfg, false)().(applyResult)
	if result.err == nil || !result.configSaved {
		t.Fatalf("partial result err=%v configSaved=%t", result.err, result.configSaved)
	}
	updated, _ := model.Update(result)
	model = updated.(Model)
	if model.dirty || model.msgLevel != messageError {
		t.Fatalf("partial apply dirty=%t level=%v msg=%q", model.dirty, model.msgLevel, model.msg)
	}
	saved, err := config.LoadExisting()
	if err != nil || saved.Prompt.Separator != " partial " {
		t.Fatalf("saved partial config separator=%q err=%v", saved.Prompt.Separator, err)
	}
}

func TestBuilderSaveFailureKeepsDirtySnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o700); err != nil {
		t.Fatalf("MkdirAll(.config) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "ozsh"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(config blocker) error = %v", err)
	}
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	snapshot := cloneConfig(model.savedCfg)
	updated, cmd := model.Update(keyRune('s'))
	model = updated.(Model)
	model = completeOperation(t, model, cmd)
	if !model.dirty || model.msgLevel != messageError || !reflect.DeepEqual(model.savedCfg, snapshot) {
		t.Fatalf("failed save dirty=%t level=%v snapshot changed=%t", model.dirty, model.msgLevel, !reflect.DeepEqual(model.savedCfg, snapshot))
	}
}

func TestApplyConfirmationControlsStayVisibleInTinyTerminal(t *testing.T) {
	model := NewModel(config.Default())
	model.tab = tabApply
	model.confirmApply = true
	model.applyViewport.SetContent(strings.Repeat("changed line\n", 20))
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	model = updated.(Model)
	view := model.View()
	if !strings.Contains(view, "y confirm") || !strings.Contains(view, "n/esc cancel") {
		t.Fatalf("tiny Apply view hides confirmation controls:\n%s", view)
	}
	if lipgloss.Height(view) > 12 || lipgloss.Width(view) > 40 {
		t.Fatalf("tiny Apply view dimensions = %dx%d", lipgloss.Width(view), lipgloss.Height(view))
	}
}

func TestExpandedHelpScrollsInTextAndApplyViews(t *testing.T) {
	for _, tab := range []int{tabPreview, tabApply} {
		model := NewModel(config.Default())
		model.tab = tab
		updated, _ := model.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
		model = updated.(Model)
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyF1})
		model = updated.(Model)
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		model = updated.(Model)
		if model.bodyViewport.YOffset == 0 && model.bodyViewport.TotalLineCount() > model.bodyViewport.Height {
			t.Fatalf("tab %d expanded help overflows without scrolling", tab)
		}
	}
}

func TestASCIIViewCoversPreviewAndHelp(t *testing.T) {
	t.Setenv("OZSH_ASCII", "1")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	model := NewModel(config.Default())
	model.tab = tabPreview
	model.previewCtx.ExitStatus = 7
	model.inputs[3].SetValue("7")
	model.syncPreviewInputs()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyF1})
	model = updated.(Model)
	view := model.View()
	for _, r := range view {
		if r > 0x7e {
			t.Fatalf("ASCII preview/help contains non-ASCII rune %q", r)
		}
	}
}

func TestCompleteApplyFlowRunsCommandsAndMarksSaved(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	model := NewModel(config.Default())
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	model.setTab(tabApply)
	updated, cmd := model.Update(keyRune('a'))
	model = completeOperation(t, updated.(Model), cmd)
	if !model.confirmApply {
		t.Fatal("Apply preview did not request confirmation")
	}
	updated, cmd = model.Update(keyRune('y'))
	model = completeOperation(t, updated.(Model), cmd)
	if model.operation != operationIdle || model.dirty || model.msgLevel != messageSuccess {
		t.Fatalf("Apply completion operation=%v dirty=%t level=%v msg=%q", model.operation, model.dirty, model.msgLevel, model.msg)
	}
	if !strings.Contains(readFile(t, shell.ZshrcPath()), "# >>> ozsh >>>") {
		t.Fatal("complete Apply flow did not inject managed block")
	}
}

func TestExternalConfigChangeBlocksTUISave(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := config.Default()
	if err := config.Save(base); err != nil {
		t.Fatalf("Save(base) error = %v", err)
	}
	model := NewModel(base)
	external := cloneConfig(base)
	external.Prompt.Separator = " external "
	if err := config.Save(external); err != nil {
		t.Fatalf("Save(external) error = %v", err)
	}
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, cmd := model.Update(keyRune('s'))
	model = completeOperation(t, updated.(Model), cmd)
	if !model.dirty || model.msgLevel != messageError || !strings.Contains(model.msg, "changed outside") {
		t.Fatalf("conflict dirty=%t level=%v msg=%q", model.dirty, model.msgLevel, model.msg)
	}
	current, err := config.LoadExisting()
	if err != nil || current.Prompt.Separator != " external " {
		t.Fatalf("external config overwritten separator=%q err=%v", current.Prompt.Separator, err)
	}
}

func TestApplyRejectsChangedZshrcSymlinkTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetA := filepath.Join(home, "zshrc-a")
	targetB := filepath.Join(home, "zshrc-b")
	content := []byte("export EDITOR=vim\n")
	for _, path := range []string{targetA, targetB} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
	logical := filepath.Join(home, ".zshrc")
	if err := os.Symlink(targetA, logical); err != nil {
		t.Fatalf("Symlink(A) error = %v", err)
	}
	preview := prepareApply().(applyPreviewResult)
	if preview.err != nil {
		t.Fatalf("prepareApply error = %v", preview.err)
	}
	if err := os.Remove(logical); err != nil {
		t.Fatalf("Remove(symlink) error = %v", err)
	}
	if err := os.Symlink(targetB, logical); err != nil {
		t.Fatalf("Symlink(B) error = %v", err)
	}
	result := doApply(config.Default(), preview.base, preview.target, config.Default(), false)().(applyResult)
	if result.err == nil || !strings.Contains(result.err.Error(), "target changed") {
		t.Fatalf("changed target apply error = %v", result.err)
	}
	if got := readFile(t, targetB); got != string(content) {
		t.Fatalf("changed target was modified: %q", got)
	}
}

func TestDoctorRejectsBrokenZshrcSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	outside := filepath.Join(t.TempDir(), "outside-zshrc")
	if err := os.Symlink(outside, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatalf("Symlink(broken) error = %v", err)
	}
	result := fixDoctorCommand(config.Default(), config.Default(), false)().(mutationResult)
	if result.err == nil || !strings.Contains(result.err.Error(), "resolve") {
		t.Fatalf("broken symlink doctor error = %v", result.err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("Doctor created broken symlink target: %v", err)
	}
}

func TestExternalConfigDeletionBlocksTUISave(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := config.Default()
	if err := config.Save(base); err != nil {
		t.Fatalf("Save(base) error = %v", err)
	}
	model := NewModel(base)
	if !model.configExists {
		t.Fatal("model did not record existing config")
	}
	if err := os.Remove(config.Path()); err != nil {
		t.Fatalf("Remove(config) error = %v", err)
	}
	model.setTab(tabBuilder)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, cmd := model.Update(keyRune('s'))
	model = completeOperation(t, updated.(Model), cmd)
	if model.msgLevel != messageError || !strings.Contains(model.msg, "removed outside") {
		t.Fatalf("deleted config level=%v msg=%q", model.msgLevel, model.msg)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("deleted config was recreated: %v", err)
	}
}

func TestExternalConfigDeletionBlocksPluginAddBeforeClone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := config.Default()
	if err := config.Save(base); err != nil {
		t.Fatalf("Save(base) error = %v", err)
	}
	if err := os.Remove(config.Path()); err != nil {
		t.Fatalf("Remove(config) error = %v", err)
	}
	result := addPlugin(base, base, true, "https://example.com/demo.git", "plugin.zsh")().(pluginAddResult)
	if result.err == nil || !strings.Contains(result.err.Error(), "removed outside") {
		t.Fatalf("deleted config plugin add error = %v", result.err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "ozsh", "plugins", "demo")); !os.IsNotExist(err) {
		t.Fatalf("plugin clone started despite config conflict: %v", err)
	}
}

func TestExternalConfigDeletionBlocksDoctorRepair(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := config.Default()
	if err := config.Save(base); err != nil {
		t.Fatalf("Save(base) error = %v", err)
	}
	if err := os.Remove(config.Path()); err != nil {
		t.Fatalf("Remove(config) error = %v", err)
	}
	result := fixDoctorCommand(base, base, true)().(mutationResult)
	if result.err == nil || !strings.Contains(result.err.Error(), "removed outside") {
		t.Fatalf("deleted config Doctor error = %v", result.err)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("Doctor recreated deleted config: %v", err)
	}
}
