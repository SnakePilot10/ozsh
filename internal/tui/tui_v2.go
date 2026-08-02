package tui

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	applyop "github.com/snakepilot10/ozsh/internal/apply"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

var tabs = []string{"Dashboard", "Builder", "Preview", "Apply", "Doctor", "Themes", "Plugins"}

const (
	tabDashboard = iota
	tabBuilder
	tabPreview
	tabApply
	tabDoctor
	tabThemes
	tabPlugins
)

// Model keeps one selection cursor, but clamps and resets it per tab. This
// prevents a position valid in Builder from being reused against Themes or Plugins.
type Model struct {
	cfg *config.Config

	tab          int
	cursor       int
	msg          string
	msgLevel     messageLevel
	width        int
	height       int
	styles       styles
	ascii        bool
	noColor      bool
	savedCfg     *config.Config
	configExists bool
	dirty        bool
	builderList  list.Model
	themesList   list.Model
	pluginsList  list.Model

	confirmApply  bool
	applyBase     string
	applyTarget   string
	operation     operation
	requestID     uint64
	applyViewport viewport.Model
	bodyViewport  viewport.Model
	help          help.Model
	spinner       spinner.Model
	doctorChecks  []doctorCheck
	dashboardInfo *dashboardInfo

	previewCtx   prompt.PreviewContext
	inputs       []textinput.Model
	inputFocus   int
	previewError string

	segmentEditing bool
	segmentName    string
	segmentDraft   config.SegmentConfig
	segmentInputs  []textinput.Model
	segmentField   int
	confirmDiscard bool

	pluginURL     textinput.Model
	pluginLoad    textinput.Model
	pluginFocus   int
	pluginEditing bool
	confirmTrust  *config.PluginItem
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
	cfg = cloneConfig(cfg)
	inputs := previewInputs(prompt.DefaultPreviewContext())
	pluginURL := textinput.New()
	pluginURL.Placeholder = "https://github.com/user/plugin.git"
	pluginURL.Prompt = "url: "
	pluginLoad := textinput.New()
	pluginLoad.Placeholder = "plugin.zsh"
	pluginLoad.Prompt = "load: "

	ascii := os.Getenv("OZSH_ASCII") != "" || os.Getenv("TERM") == "dumb"
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	uiStyles := newStyles(cfg.Theme, ascii, noColor)
	builderList := newBuilderList(cfg, uiStyles)
	themesList := newCatalogList(themeItems(cfg), "theme", "themes", uiStyles)
	pluginsList := newCatalogList(pluginItems(cfg), "plugin", "plugins", uiStyles)
	_, configStatErr := os.Stat(config.Path())
	helpModel := help.New()
	helpModel.Styles = uiStyles.helpStyles()
	if ascii {
		helpModel.ShortSeparator = " | "
		helpModel.FullSeparator = " | "
		helpModel.Ellipsis = "..."
	}
	applyViewport := viewport.New(80, 10)
	applyViewport.MouseWheelEnabled = false
	bodyViewport := viewport.New(80, 10)
	bodyViewport.MouseWheelEnabled = false
	spinnerModel := spinner.New()
	spinnerModel.Spinner = spinner.Dot
	if ascii {
		spinnerModel.Spinner = spinner.Line
	}
	spinnerModel.Style = uiStyles.accent
	m := Model{
		cfg:           cfg,
		previewCtx:    prompt.DefaultPreviewContext(),
		inputs:        inputs,
		pluginURL:     pluginURL,
		pluginLoad:    pluginLoad,
		help:          helpModel,
		applyViewport: applyViewport,
		bodyViewport:  bodyViewport,
		spinner:       spinnerModel,
		styles:        uiStyles,
		savedCfg:      cloneConfig(cfg),
		configExists:  configStatErr == nil,
		builderList:   builderList,
		themesList:    themesList,
		pluginsList:   pluginsList,
		ascii:         ascii,
		noColor:       noColor,
	}
	m.refreshStyles()
	m.syncCursor()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.applyViewport.Init(), m.bodyViewport.Init(), inspectDashboard)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case operationResultEnvelope:
		if msg.requestID != m.requestID || m.operation == operationIdle {
			return m, nil
		}
		return m.Update(msg.msg)
	case applyResult:
		m.operation = operationIdle
		if msg.err == nil {
			m.setMessage(messageSuccess, "applied")
			m.markSaved()
			m.configExists = true
		} else {
			if msg.configSaved {
				m.markSaved()
				m.configExists = true
			}
			m.setMessage(messageError, "apply error: "+msg.err.Error())
		}
		return m, nil
	case applyPreviewResult:
		m.operation = operationIdle
		if msg.err != nil {
			m.setMessage(messageError, "apply preview error: "+msg.err.Error())
			return m, nil
		}
		m.applyViewport.SetContent(msg.diff)
		m.applyBase = msg.base
		m.applyTarget = msg.target
		m.applyViewport.GotoTop()
		m.confirmApply = true
		m.setMessage(messageWarning, "review the diff before confirming")
		return m, nil
	case pluginAddResult:
		m.operation = operationIdle
		if msg.err != nil {
			m.setMessage(messageError, "plugin add error: "+msg.err.Error())
			return m, nil
		}
		m.cfg = msg.cfg
		m.pluginURL.SetValue("")
		m.pluginLoad.SetValue("")
		m.pluginFocus = 0
		m.pluginEditing = false
		m.pluginURL.Blur()
		m.pluginLoad.Blur()
		m.cursor = len(m.cfg.Plugins.Items) - 1
		m.pluginsList.ResetFilter()
		m.syncPluginsList(msg.name)
		m.markSaved()
		m.configExists = true
		m.setMessage(messageSuccess, "plugin added: "+msg.name)
		return m, nil
	case doctorResult:
		m.operation = operationIdle
		m.doctorChecks = append([]doctorCheck(nil), msg...)
		return m, nil
	case dashboardResult:
		if m.operation == operationDashboard {
			m.operation = operationIdle
		}
		info := dashboardInfo(msg)
		m.dashboardInfo = &info
		return m, nil
	case mutationResult:
		m.operation = operationIdle
		if msg.err != nil {
			m.setMessage(messageError, "operation failed: "+msg.err.Error())
			return m, nil
		}
		if msg.cfg == nil && msg.kind != mutationDoctorFix {
			m.setMessage(messageError, "operation failed: missing configuration result")
			return m, nil
		}
		if msg.cfg != nil {
			m.cfg = msg.cfg
			m.markSaved()
			m.configExists = true
		}
		switch msg.kind {
		case mutationTheme, mutationCustomTheme:
			m.refreshStyles()
			m.syncBuilderList("")
			m.syncThemesList(msg.selected)
		case mutationPlugin:
			m.syncPluginsList(msg.selected)
		case mutationDoctorFix:
			m.setMessage(messageSuccess, msg.message)
			return m, m.startOperation(operationDoctor, inspectDoctor)
		}
		m.setMessage(messageSuccess, msg.message)
		return m, nil
	case spinner.TickMsg:
		if m.operation == operationIdle {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if key.Matches(msg, keys.HelpForm) || (msg.String() == "?" && !m.textEntryActive()) {
			m.help.ShowAll = !m.help.ShowAll
			m.resize()
			return m, nil
		}
		if m.operation != operationIdle {
			return m, nil
		}
		if (m.help.ShowAll || (!m.textEntryActive() && m.tab != tabApply)) && (key.Matches(msg, keys.BodyUp) || key.Matches(msg, keys.BodyDown)) {
			m.prepareBodyViewport()
			if key.Matches(msg, keys.BodyUp) {
				m.bodyViewport.HalfViewUp()
			} else {
				m.bodyViewport.HalfViewDown()
			}
			return m, nil
		}
		if key.Matches(msg, keys.Quit) && m.tab != tabPreview && !(m.tab == tabPlugins && m.pluginEditing) && !m.segmentEditing && !m.activeListSettingFilter() {
			if m.dirty {
				m.setMessage(messageWarning, "unsaved changes: press s to save or return to Builder and press d")
				return m, nil
			}
			return m, tea.Quit
		}
		if m.confirmTrust != nil {
			if key.Matches(msg, keys.Confirm) {
				name := m.confirmTrust.Name
				m.confirmTrust = nil
				trusted := true
				return m, m.startOperation(operationPluginUpdate, persistPluginState(m.cfg, m.savedCfg, m.configExists, name, false, &trusted))
			}
			if key.Matches(msg, keys.Cancel) {
				m.confirmTrust = nil
				m.setMessage(messageWarning, "plugin trust cancelled")
			}
			return m, nil
		}
		if m.confirmDiscard {
			if key.Matches(msg, keys.Confirm) {
				m.discardChanges()
				return m, nil
			}
			if key.Matches(msg, keys.Cancel) {
				m.confirmDiscard = false
				m.setMessage(messageInfo, "discard cancelled")
			}
			return m, nil
		}
		if m.segmentEditing {
			return m.updateSegmentEditor(msg)
		}
		if m.tab == tabBuilder && m.builderListShouldHandle(msg) {
			return m.updateBuilderList(msg)
		}
		if m.tab == tabThemes && m.catalogListShouldHandle(m.themesList, msg) {
			return m.updateThemesList(msg)
		}
		if m.tab == tabPlugins && m.pluginEditing {
			return m.updatePluginInputs(msg)
		}
		if m.tab == tabPlugins && m.catalogListShouldHandle(m.pluginsList, msg) {
			return m.updatePluginsList(msg)
		}
		if m.tab == tabApply && m.confirmApply {
			if key.Matches(msg, keys.Confirm) {
				m.confirmApply = false
				m.setMessage(messageInfo, "")
				return m, m.startOperation(operationApply, doApply(m.cfg, m.applyBase, m.applyTarget, m.savedCfg, m.configExists))
			}
			if key.Matches(msg, keys.Cancel) {
				m.confirmApply = false
				m.setMessage(messageWarning, "apply cancelled")
				return m, nil
			}
			var cmd tea.Cmd
			m.applyViewport, cmd = m.applyViewport.Update(msg)
			return m, cmd
		}
		if m.formAllowsGlobalNavigation(msg) {
			switch {
			case key.Matches(msg, keys.NextTab):
				m.setTab((m.tab + 1) % len(tabs))
				return m, m.activeFocusCmd()
			case key.Matches(msg, keys.PrevTab):
				m.setTab((m.tab + len(tabs) - 1) % len(tabs))
				return m, m.activeFocusCmd()
			case isTabNumber(msg):
				idx, _ := strconv.Atoi(msg.String())
				m.setTab(idx - 1)
				return m, m.activeFocusCmd()
			}
		}

		// Form tabs own printable input. This prevents global shortcut handling
		// from swallowing text, including numeric exit statuses and plugin URLs.
		if m.tab == tabPreview {
			return m.updatePreviewInputs(msg)
		}
		switch {
		case isTabNumber(msg):
			idx, _ := strconv.Atoi(msg.String())
			m.setTab(idx - 1)
			return m, m.activeFocusCmd()
		case key.Matches(msg, keys.NextTab):
			m.setTab((m.tab + 1) % len(tabs))
			return m, m.activeFocusCmd()
		case key.Matches(msg, keys.PrevTab):
			m.setTab((m.tab + len(tabs) - 1) % len(tabs))
			return m, m.activeFocusCmd()
		case key.Matches(msg, keys.Up):
			m.moveCursor(-1)
		case key.Matches(msg, keys.Down):
			m.moveCursor(1)
		case msg.String() == " ":
			if m.tab == tabBuilder {
				m.toggleSegment()
			}
		case msg.String() == "enter":
			return m, m.handleEnter()
		case key.Matches(msg, keys.ReorderDown):
			if m.tab == tabBuilder {
				m.moveSegment(1)
			}
		case key.Matches(msg, keys.ReorderUp):
			if m.tab == tabBuilder {
				m.moveSegment(-1)
			}
		case key.Matches(msg, keys.Heavy):
			if m.tab == tabBuilder {
				m.cfg.Prompt.DisableHeavySegments = !m.cfg.Prompt.DisableHeavySegments
				m.markDirty()
				if m.cfg.Prompt.DisableHeavySegments {
					m.setMessage(messageInfo, "heavy segments disabled")
				} else {
					m.setMessage(messageInfo, "heavy segments enabled")
				}
			}
		case key.Matches(msg, keys.Save):
			return m, m.startOperation(operationSave, persistConfig(m.cfg, m.savedCfg, m.configExists))
		case key.Matches(msg, keys.Apply):
			if m.tab == tabApply {
				m.setMessage(messageInfo, "")
				return m, m.startOperation(operationApplyPreview, prepareApply)
			}
		case key.Matches(msg, keys.Refresh):
			if m.tab == tabDoctor {
				return m, m.startOperation(operationDoctor, inspectDoctor)
			}
		case msg.String() == "c":
			if m.tab == tabThemes {
				if !m.ensureCleanConfig("save a custom theme") {
					return m, nil
				}
				return m, m.startOperation(operationThemeSave, persistCustomTheme(m.cfg, m.savedCfg, m.configExists))
			}
		case key.Matches(msg, keys.TrustPlugin):
			if m.tab == tabPlugins {
				m.requestPluginTrust()
			}
		case key.Matches(msg, keys.UntrustPlugin):
			if m.tab == tabPlugins {
				if !m.ensureCleanConfig("change plugin trust") {
					return m, nil
				}
				name := m.selectedPluginName()
				if name != "" {
					trusted := false
					return m, m.startOperation(operationPluginUpdate, persistPluginState(m.cfg, m.savedCfg, m.configExists, name, false, &trusted))
				}
			}
		case key.Matches(msg, keys.AddPlugin):
			if m.tab == tabPlugins {
				m.pluginEditing = true
				m.pluginFocus = 0
				return m, m.focusPluginInput()
			}
		case key.Matches(msg, keys.EditSegment):
			if m.tab == tabBuilder {
				return m, m.beginSegmentEdit()
			}
		case key.Matches(msg, keys.Discard):
			if m.tab == tabBuilder && m.dirty {
				m.confirmDiscard = true
				m.setMessage(messageWarning, "discard all unsaved Builder changes?")
			}
		}
	default:
		return m.updateActiveComponent(msg)
	}
	return m, nil
}

func (m Model) updateActiveComponent(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.segmentEditing {
		return m.updateSegmentEditor(msg)
	}
	if m.tab == tabPlugins && m.pluginEditing {
		return m.updatePluginInputs(msg)
	}
	if m.tab == tabPreview {
		return m.updatePreviewInputs(msg)
	}
	if m.tab == tabBuilder && m.builderList.SettingFilter() {
		return m.updateBuilderList(msg)
	}
	if m.tab == tabThemes && m.themesList.SettingFilter() {
		return m.updateThemesList(msg)
	}
	if m.tab == tabPlugins && m.pluginsList.SettingFilter() {
		return m.updatePluginsList(msg)
	}
	if m.tab == tabApply && m.confirmApply {
		var cmd tea.Cmd
		m.applyViewport, cmd = m.applyViewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) textEntryActive() bool {
	return m.tab == tabPreview || m.segmentEditing || (m.tab == tabPlugins && m.pluginEditing) || m.activeListSettingFilter()
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.tab {
	case tabBuilder:
		m.toggleSegment()
	case tabDoctor:
		if !m.ensureCleanConfig("fix Doctor issues") {
			return nil
		}
		return m.startOperation(operationDoctorFix, fixDoctorCommand(m.cfg, m.savedCfg, m.configExists))
	case tabThemes:
		if !m.ensureCleanConfig("apply a theme") {
			return nil
		}
		name := m.selectedThemeName()
		preset, ok := m.selectedThemePreset()
		if name != "" && ok {
			return m.startOperation(operationThemeSave, persistTheme(m.cfg, m.savedCfg, m.configExists, name, preset))
		}
	case tabPlugins:
		if !m.ensureCleanConfig("change plugin state") {
			return nil
		}
		if name := m.selectedPluginName(); name != "" {
			return m.startOperation(operationPluginUpdate, persistPluginState(m.cfg, m.savedCfg, m.configExists, name, true, nil))
		}
	}
	return nil
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
	m.pluginEditing = false
	m.confirmTrust = nil
	m.segmentEditing = false
	m.segmentInputs = nil
	m.confirmDiscard = false
	m.msg = ""
	m.bodyViewport.GotoTop()
	m.syncCursor()
}

func isTabNumber(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6", "7":
		return true
	default:
		return false
	}
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
	} else {
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
	}
	if m.tab == tabBuilder {
		m.builderList.Select(m.cursor)
		m.syncBuilderCursor()
	}
	if m.tab == tabThemes {
		m.themesList.Select(m.cursor)
		m.syncThemeCursor()
	}
	if m.tab == tabPlugins {
		m.pluginsList.Select(m.cursor)
		m.syncPluginCursor()
		if m.pluginEditing {
			m.focusPluginInput()
		} else {
			m.pluginURL.Blur()
			m.pluginLoad.Blur()
		}
	}
}

func (m *Model) activeFocusCmd() tea.Cmd {
	if m.tab == tabDashboard {
		return m.startOperation(operationDashboard, inspectDashboard)
	}
	if m.tab == tabPreview {
		return m.focusPreviewInput()
	}
	if m.tab == tabBuilder && m.segmentEditing {
		return m.focusSegmentInput()
	}
	if m.tab == tabDoctor {
		return m.startOperation(operationDoctor, inspectDoctor)
	}
	if m.tab == tabPlugins && m.pluginEditing {
		return m.focusPluginInput()
	}
	return nil
}

func (m *Model) startOperation(op operation, cmd tea.Cmd) tea.Cmd {
	m.operation = op
	m.requestID++
	requestID := m.requestID
	wrapped := func() tea.Msg {
		return operationResultEnvelope{requestID: requestID, msg: cmd()}
	}
	return tea.Batch(wrapped, m.spinner.Tick)
}

func (m *Model) setMessage(level messageLevel, text string) {
	m.msgLevel = level
	m.msg = text
	if m.width > 0 && m.height > 0 {
		m.resize()
	}
}

func (m *Model) ensureCleanConfig(action string) bool {
	if !m.dirty {
		return true
	}
	m.setMessage(messageWarning, "save or discard Builder changes before you "+action)
	return false
}

func (m *Model) refreshStyles() {
	m.styles = newStyles(m.cfg.Theme, m.ascii, m.noColor)
	m.help.Styles = m.styles.helpStyles()
	m.spinner.Style = m.styles.accent
	styleInput := func(input *textinput.Model) {
		input.PromptStyle = m.styles.accent
		input.PlaceholderStyle = m.styles.muted
		input.Cursor.Style = m.styles.accent
	}
	for i := range m.inputs {
		styleInput(&m.inputs[i])
	}
	styleInput(&m.pluginURL)
	styleInput(&m.pluginLoad)
	for i := range m.segmentInputs {
		styleInput(&m.segmentInputs[i])
	}
	styleList(&m.builderList, m.styles)
	styleList(&m.themesList, m.styles)
	styleList(&m.pluginsList, m.styles)
}

func (m *Model) resize() {
	contentWidth := max(12, m.width-m.styles.panel.GetHorizontalFrameSize())
	helpView := m.help.View(m.contextualHelp())
	messageView := ""
	if m.msg != "" {
		messageView = m.msg
	}
	bodyHeight := m.availableBodyHeight(helpView, messageView)
	for i := range m.inputs {
		m.inputs[i].Width = max(8, contentWidth-14)
	}
	m.pluginURL.Width = max(8, contentWidth-8)
	m.pluginLoad.Width = max(8, contentWidth-8)
	for i := range m.segmentInputs {
		m.segmentInputs[i].Width = max(8, contentWidth-10)
	}
	m.builderList.SetSize(contentWidth, max(5, bodyHeight/2))
	m.themesList.SetSize(contentWidth, max(5, bodyHeight/2))
	m.pluginsList.SetSize(contentWidth, max(5, (bodyHeight-3)/2))
	m.applyViewport.Width = contentWidth
	m.applyViewport.Height = max(3, bodyHeight-5)
	m.bodyViewport.Width = contentWidth
	m.bodyViewport.Height = bodyHeight
	m.help.Width = contentWidth
}

func (m Model) contextualHelp() contextualKeyMap {
	if m.operation != operationIdle {
		short := []key.Binding{keys.Help, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	if m.confirmDiscard {
		short := []key.Binding{keys.Confirm, keys.Cancel, keys.Help, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	if m.segmentEditing {
		short := []key.Binding{keys.NextField, keys.PrevField, keys.Toggle, keys.ApplyEdit, keys.CloseForm, keys.HelpForm, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	if m.activeListSettingFilter() {
		var model list.Model
		switch m.tab {
		case tabBuilder:
			model = m.builderList
		case tabThemes:
			model = m.themesList
		case tabPlugins:
			model = m.pluginsList
		}
		short := []key.Binding{model.KeyMap.AcceptWhileFiltering, model.KeyMap.CancelWhileFiltering, keys.HelpForm, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	if m.confirmTrust != nil {
		short := []key.Binding{keys.Confirm, keys.Cancel, keys.Help, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	navigation := []key.Binding{keys.NextTab, keys.PrevTab, keys.Help, keys.Quit}
	if m.tab == tabPreview {
		short := []key.Binding{keys.Up, keys.Down, keys.NextTabForm, keys.PrevTabForm, keys.HelpForm, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}
	if m.tab == tabPlugins && m.pluginEditing {
		short := []key.Binding{keys.NextField, keys.PrevField, keys.Submit, keys.CloseForm, keys.HelpForm, keys.ForceQuit}
		return contextualKeyMap{short: short, full: [][]key.Binding{short}}
	}

	var actions []key.Binding
	switch m.tab {
	case tabBuilder:
		actions = []key.Binding{keys.Up, keys.Down, keys.Filter, keys.PageUp, keys.PageDown, keys.Toggle, keys.EditSegment, keys.ReorderDown, keys.ReorderUp, keys.Heavy, keys.Save}
		if m.dirty {
			actions = append(actions, keys.Discard)
		}
	case tabApply:
		if m.confirmApply {
			actions = []key.Binding{keys.Confirm, keys.Cancel, keys.Up, keys.Down, keys.PageUp, keys.PageDown}
		} else {
			actions = []key.Binding{keys.Apply}
		}
	case tabDoctor:
		actions = []key.Binding{keys.Fix, keys.Refresh}
	case tabThemes:
		actions = []key.Binding{keys.Up, keys.Down, keys.Filter, keys.PageUp, keys.PageDown, keys.ApplyTheme, keys.CustomTheme}
	case tabPlugins:
		actions = []key.Binding{keys.Up, keys.Down, keys.Filter, keys.PageUp, keys.PageDown, keys.TogglePlugin, keys.AddPlugin, keys.TrustPlugin, keys.UntrustPlugin}
	}
	short := []key.Binding{keys.NextTab, keys.PrevTab, keys.Help, keys.Quit}
	if len(actions) > 0 {
		short = []key.Binding{actions[0], keys.NextTab, keys.Help, keys.Quit}
	}
	fullActions := append(append([]key.Binding{}, actions...), keys.BodyUp, keys.BodyDown)
	return contextualKeyMap{short: short, full: [][]key.Binding{fullActions, navigation}}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.styles.accent.Render("ozsh"))
	if m.dirty {
		b.WriteString(" ")
		b.WriteString(m.styles.warning.Render("* unsaved"))
	}
	b.WriteString(" ")
	b.WriteString(m.renderTabs())
	helpView := m.help.View(m.contextualHelp())
	messageView := ""
	if m.msg != "" {
		messageView = m.styles.message(m.msgLevel).Render(singleLine(m.msg, m.contentWidth()))
	}
	if m.confirmApply {
		messageView = m.styles.warning.Render(singleLine("y confirm | n/esc cancel | writes config.toml, omega.zsh and .zshrc", m.contentWidth()))
	}
	body := m.body()
	if m.help.ShowAll {
		body = helpView
		helpView = m.styles.muted.Render("F1/? close help")
	}
	viewport := m.bodyViewport
	viewport.Width = m.contentWidth()
	viewport.Height = m.availableBodyHeight(helpView, messageView)
	viewport.SetContent(body)
	b.WriteString("\n")
	b.WriteString(viewport.View())
	if messageView != "" {
		b.WriteString("\n")
		b.WriteString(messageView)
	}
	b.WriteString("\n")
	b.WriteString(helpView)
	style := m.styles.panel
	if m.width > 0 {
		style = style.Width(max(1, m.width-style.GetHorizontalFrameSize()))
	}
	rendered := style.Render(b.String())
	if m.ascii {
		return asciiOnly(rendered)
	}
	return rendered
}

func (m Model) body() string {
	if m.operation == operationSave {
		return m.spinner.View() + " saving configuration"
	}
	switch m.tab {
	case tabDashboard:
		return m.dashboard()
	case tabBuilder:
		return m.builder()
	case tabPreview:
		return m.preview()
	case tabApply:
		return m.apply()
	case tabDoctor:
		return m.doctor()
	case tabThemes:
		return m.themes()
	case tabPlugins:
		return m.plugins()
	}
	return ""
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return max(1, m.width-m.styles.panel.GetHorizontalFrameSize())
}

func (m Model) availableBodyHeight(helpView, messageView string) int {
	if m.height <= 0 {
		return max(3, strings.Count(m.body(), "\n")+1)
	}
	fixed := m.styles.panel.GetVerticalFrameSize() + 3 + lineCount(helpView)
	if messageView != "" {
		fixed += lineCount(messageView)
	}
	return max(1, m.height-fixed)
}

func (m *Model) prepareBodyViewport() {
	helpView := m.help.View(m.contextualHelp())
	body := m.body()
	if m.help.ShowAll {
		body = helpView
		helpView = m.styles.muted.Render("F1/? close help")
	}
	messageView := ""
	if m.msg != "" {
		messageView = m.msg
	}
	m.bodyViewport.Width = m.contentWidth()
	m.bodyViewport.Height = m.availableBodyHeight(helpView, messageView)
	m.bodyViewport.SetContent(body)
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func singleLine(value string, width int) string {
	value = strings.Join(strings.Fields(value), " ")
	if width <= 1 || len(value) <= width {
		return value
	}
	return value[:max(1, width-3)] + "..."
}

func asciiOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r == '\n' || r == '\t' || (r >= 0x20 && r <= 0x7e) || r == 0x1b {
			b.WriteRune(r)
		} else {
			b.WriteByte('?')
		}
	}
	return b.String()
}

func (m Model) renderTabs() string {
	if m.width > 0 && m.width < 88 {
		left, right := "‹", "›"
		if m.ascii {
			left, right = "<", ">"
		}
		return fmt.Sprintf("%s %d/%d %s %s", left, m.tab+1, len(tabs), m.styles.accent.Render(tabs[m.tab]), right)
	}
	items := make([]string, len(tabs))
	for i, tab := range tabs {
		label := fmt.Sprintf("%d %s", i+1, tab)
		if i == m.tab {
			items[i] = m.styles.accent.Render("[" + label + "]")
		} else {
			items[i] = m.styles.muted.Render(label)
		}
	}
	return strings.Join(items, "  ")
}

func (m Model) dashboard() string {
	if m.operation == operationDashboard {
		return m.spinner.View() + " refreshing dashboard"
	}
	if m.dashboardInfo == nil {
		return "loading dashboard"
	}
	info := m.dashboardInfo
	view := fmt.Sprintf("config:   %s\nomega:    %s\nblock:    %t\nplatform: %s\ntermux:   %t\nbackups:  %d",
		info.configPath, info.omegaPath, info.hasBlock, info.platform, info.termux, info.backups)
	if info.err != nil {
		view += "\nbackup error: " + info.err.Error()
	}
	return view
}

func (m Model) builder() string {
	if m.confirmDiscard {
		return "discard unsaved Builder changes?\n\nThis restores the last successfully saved configuration."
	}
	if m.segmentEditing {
		return m.segmentEditorView()
	}
	if len(m.cfg.Prompt.Order) == 0 {
		return "segments\n\nNo segments in prompt.order. Restore defaults or add segment names in config.toml."
	}
	var b strings.Builder
	b.WriteString("segments")
	if m.dirty {
		b.WriteString("  * unsaved")
	}
	b.WriteString("\n\n")
	b.WriteString(m.builderList.View())
	b.WriteString("\n")
	b.WriteString(m.simulated(m.previewConfig()))
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
	b.WriteString(m.simulatedWithContext(m.previewConfig(), m.previewCtx))
	if m.previewError != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.error.Render(m.previewError))
	}
	return b.String()
}

func (m Model) apply() string {
	switch m.operation {
	case operationApplyPreview:
		return m.spinner.View() + " preparing apply preview"
	case operationApply:
		return m.spinner.View() + " applying configuration"
	}
	if m.confirmApply {
		position := fmt.Sprintf("scroll %d%%", int(m.applyViewport.ScrollPercent()*100))
		if m.applyViewport.AtTop() {
			position = "top"
		} else if m.applyViewport.AtBottom() {
			position = "bottom"
		}
		return "planned .zshrc change (" + position + ")\n" + m.applyViewport.View() + "\napply also replaces config.toml and omega.zsh\npress y to confirm or n/esc to cancel"
	}
	return "apply configuration\n\npress a to prepare the .zshrc diff and review all changes"
}

func (m Model) doctor() string {
	if m.operation == operationDoctor || m.operation == operationDoctorFix {
		return m.spinner.View() + " checking environment"
	}
	if len(m.doctorChecks) == 0 {
		return "diagnostics not loaded\n\npress r to refresh"
	}
	var b strings.Builder
	for _, check := range m.doctorChecks {
		m.writeCheck(&b, check.ok, check.label)
	}
	b.WriteString("\n\npress enter to fix auto-resolvable issues")
	return b.String()
}

func (m Model) writeCheck(b *strings.Builder, ok bool, label string) {
	if m.ascii {
		if ok {
			fmt.Fprintf(b, "[OK] %s\n", label)
		} else {
			fmt.Fprintf(b, "[X]  %s\n", label)
		}
		return
	}
	writeCheck(b, ok, label)
}

func (m Model) themes() string {
	if m.operation == operationThemeSave {
		return m.spinner.View() + " saving theme"
	}
	var b strings.Builder
	b.WriteString("theme gallery\n\n")
	if len(m.themesList.Items()) == 0 {
		return "theme gallery\n\nno presets available"
	}
	b.WriteString(m.themesList.View())
	b.WriteString("\n")
	if name := m.selectedThemeName(); name != "" {
		b.WriteString(m.simulated(m.previewThemeConfig(name)))
	}
	return b.String()
}

func (m Model) plugins() string {
	if m.confirmTrust != nil {
		item := m.confirmTrust
		return fmt.Sprintf("trust third-party plugin?\n\nname:   %s\nsource: %s\nload:   %s\n\nTrust permits generated shell configuration to source this code.\nReview the repository before continuing.", item.Name, item.Source, item.Load)
	}
	var b strings.Builder
	b.WriteString("plugins\n\n")
	if m.operation == operationPluginAdd || m.operation == operationPluginUpdate {
		b.WriteString(m.spinner.View())
		if m.operation == operationPluginAdd {
			b.WriteString(" cloning and saving plugin\n\n")
		} else {
			b.WriteString(" saving plugin state\n\n")
		}
	}
	if len(m.cfg.Plugins.Items) == 0 {
		b.WriteString("No plugins configured. Press p to add one.\n")
	} else {
		b.WriteString(m.pluginsList.View())
	}
	b.WriteString("\nadd plugin\n")
	b.WriteString(m.pluginURL.View())
	b.WriteByte('\n')
	b.WriteString(m.pluginLoad.View())
	if !m.pluginEditing {
		b.WriteString("\n")
		b.WriteString(m.styles.muted.Render("press p to edit the add form"))
	}
	return b.String()
}

func (m *Model) toggleSegment() {
	index := m.builderIndex()
	if m.tab != tabBuilder || index < 0 || index >= len(m.cfg.Prompt.Order) {
		return
	}
	name := m.cfg.Prompt.Order[index]
	seg := m.cfg.Prompt.Segments[name]
	seg.Enabled = !seg.Enabled
	m.cfg.Prompt.Segments[name] = seg
	m.cursor = index
	m.syncBuilderList(name)
	m.markDirty()
}

func (m *Model) moveSegment(delta int) {
	if m.tab != tabBuilder {
		return
	}
	index := m.builderIndex()
	next := index + delta
	if index < 0 || next < 0 || next >= len(m.cfg.Prompt.Order) {
		return
	}
	name := m.cfg.Prompt.Order[index]
	m.cfg.Prompt.Order[index], m.cfg.Prompt.Order[next] = m.cfg.Prompt.Order[next], m.cfg.Prompt.Order[index]
	m.cursor = next
	m.syncBuilderList(name)
	m.markDirty()
}

func (m Model) updatePreviewInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "down":
			m.inputFocus = wrapIndex(m.inputFocus+1, len(m.inputs))
			m.cursor = m.inputFocus
			return m, m.focusPreviewInput()
		case "up":
			m.inputFocus = wrapIndex(m.inputFocus-1, len(m.inputs))
			m.cursor = m.inputFocus
			return m, m.focusPreviewInput()
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

func (m Model) formAllowsGlobalNavigation(msg tea.KeyMsg) bool {
	switch m.tab {
	case tabPreview:
		switch msg.String() {
		case "tab", "shift+tab":
			return true
		}
	case tabPlugins:
		return !m.pluginEditing
	}
	return false
}

func (m *Model) focusPreviewInput() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.inputs {
		if i == m.inputFocus {
			cmd = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return cmd
}

func (m Model) updatePluginInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.CloseForm):
			m.pluginEditing = false
			m.pluginURL.SetValue("")
			m.pluginLoad.SetValue("")
			m.pluginFocus = 0
			m.pluginURL.Blur()
			m.pluginLoad.Blur()
			return m, nil
		case key.Matches(keyMsg, keys.Submit):
			return m, m.addPluginFromInputs()
		case key.Matches(keyMsg, keys.NextField):
			m.pluginFocus = (m.pluginFocus + 1) % 2
			return m, m.focusPluginInput()
		case key.Matches(keyMsg, keys.PrevField):
			m.pluginFocus = wrapIndex(m.pluginFocus-1, 2)
			return m, m.focusPluginInput()
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

func (m *Model) focusPluginInput() tea.Cmd {
	if m.pluginFocus == 0 {
		cmd := m.pluginURL.Focus()
		m.pluginLoad.Blur()
		return cmd
	}
	m.pluginURL.Blur()
	return m.pluginLoad.Focus()
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
	if m.segmentEditing && m.segmentName != "" {
		clone.Prompt.Segments[m.segmentName] = m.segmentDraft
	}
	return clone
}

func (m Model) simulated(cfg *config.Config) string {
	marker := "❯"
	if m.ascii {
		marker = ">"
	}
	if m.noColor {
		return prompt.SimulatedPlain(cfg, marker)
	}
	return prompt.Simulated(cfg)
}

func (m Model) simulatedWithContext(cfg *config.Config, ctx prompt.PreviewContext) string {
	marker := "❯"
	if m.ascii {
		marker = ">"
	}
	if m.noColor {
		return prompt.SimulatedWithContextPlain(cfg, ctx, marker)
	}
	return prompt.SimulatedWithContext(cfg, ctx)
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

func (m Model) previewThemeConfig(name string) *config.Config {
	clone := cloneConfig(m.cfg)
	preset, ok := config.Presets[name]
	if name == "custom" {
		preset, ok = m.cfg.Theme, true
	}
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

func (m *Model) requestPluginTrust() {
	if !m.ensureCleanConfig("change plugin trust") {
		return
	}
	index := m.pluginIndex()
	if index < 0 || index >= len(m.cfg.Plugins.Items) {
		return
	}
	item := m.cfg.Plugins.Items[index]
	if item.Trusted {
		m.setMessage(messageInfo, "plugin already trusted: "+item.Name)
		return
	}
	m.confirmTrust = &item
	m.setMessage(messageWarning, "confirm trust only after reviewing the plugin source")
}

func (m *Model) addPluginFromInputs() tea.Cmd {
	if !m.ensureCleanConfig("add a plugin") {
		return nil
	}
	url := sanitizeInput(m.pluginURL.Value())
	load := sanitizeInput(m.pluginLoad.Value())
	if url == "" {
		m.setMessage(messageError, "plugin URL is required")
		return nil
	}
	if load == "" {
		m.setMessage(messageError, "plugin load file is required")
		return nil
	}
	m.setMessage(messageInfo, "")
	return m.startOperation(operationPluginAdd, addPlugin(m.cfg, m.savedCfg, m.configExists, url, load))
}

func sortedThemeNames() []string {
	names := make([]string, 0, len(config.Presets))
	for name := range config.Presets {
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
	bg      string
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
	bg := i.bg
	if bg == "" {
		bg = "none"
	}
	return fmt.Sprintf("fg=%s bg=%s bold=%s icon=%s", i.fg, bg, weight, icon)
}

type applyResult struct {
	err         error
	configSaved bool
}

func doApply(cfg *config.Config, expectedZshrc, expectedTarget string, expectedConfig *config.Config, configExisted bool) tea.Cmd {
	snapshot := cloneConfig(cfg)
	return func() tea.Msg {
		if err := ensureConfigUnchanged(expectedConfig, configExisted); err != nil {
			return applyResult{err: err}
		}
		preview, err := shell.PreviewInjectPlan()
		if err != nil {
			return applyResult{err: fmt.Errorf("refresh .zshrc preview: %w", err)}
		}
		if preview.Target != expectedTarget {
			return applyResult{err: fmt.Errorf(".zshrc target changed after review; prepare and confirm a new preview")}
		}
		if preview.Before != expectedZshrc {
			return applyResult{err: fmt.Errorf(".zshrc changed after review; prepare and confirm a new preview")}
		}
		err = applyop.ApplyConfigExpected(snapshot, expectedZshrc, expectedTarget)
		var partial *applyop.PartialError
		return applyResult{err: err, configSaved: errors.As(err, &partial) && partial.ConfigSaved}
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
