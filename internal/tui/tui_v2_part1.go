package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5ff")).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b6b80"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff003c")).Bold(true)
	panelStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#09090d")).Foreground(lipgloss.Color("#e0e0e0")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#00f5ff")).Padding(1, 2)
)

var tabs = []string{"Home", "Prompt", "Themes", "Plugins", "Preview"}

const (
	tabHome = iota
	tabPrompt
	tabThemes
	tabPlugins
	tabPreview

	// Compatibility aliases for package callers while the old screen names are
	// removed from the UI. Apply and Doctor are actions, not destinations.
	tabDashboard = tabHome
	tabBuilder   = tabPrompt
	tabApply     = tabPreview
	tabDoctor    = tabHome
)

// Model keeps one selection cursor, but clamps and resets it per tab. This
// prevents a position valid in Builder from being reused against Themes or Plugins.
type Model struct {
	cfg *config.Config

	tab    int
	cursor int
	msg    string
	width  int
	height int

	confirmApply          bool
	confirmPlugins        bool
	confirmFont           bool
	confirmBackup         bool
	confirmDoctor         bool
	confirmFontRestore    bool
	applyDiff             string
	reviewedConfig        *config.Config
	reviewedPluginChanges plugins.ChangeSet
	showApplyTechnical    bool
	busy                  bool
	operation             string
	doctorOpen            bool
	themeVariant          int
	fontOpen              bool
	fontCursor            int
	backupOpen            bool
	backupCursor          int
	backupPaths           []string

	previewCtx        prompt.PreviewContext
	inputs            []textinput.Model
	inputFocus        int
	previewError      string
	previewScenario   int
	previewCustom     bool
	promptName        textinput.Model
	promptEditingName bool
	promptAdvanced    bool

	pluginWizard        pluginWizardModel
	pluginChanges       plugins.ChangeSet
	pluginCloneRunner   plugins.CloneRunner
	pluginRemoveConfirm bool
	pluginRemoveName    string

	// Legacy fields stay internal until the lifecycle migration removes the old
	// direct-add helpers and their compatibility tests.
	pluginURL      textinput.Model
	pluginLoad     textinput.Model
	pluginFocus    int
	pluginAdvanced bool
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	final, runErr := tea.NewProgram(NewModel(cfg), tea.WithAltScreen()).Run()
	if model, ok := final.(Model); ok {
		cleanupErr := model.pluginChanges.Cleanup()
		if runErr == nil && cleanupErr != nil {
			return cleanupErr
		}
	}
	return runErr
}

func NewModel(cfg *config.Config) Model {
	if cfg == nil {
		cfg = config.Default()
	}
	initialPreview, _ := prompt.PreviewScenario("git-dirty")
	inputs := previewInputs(initialPreview)
	pluginURL := textinput.New()
	pluginURL.Placeholder = "https://github.com/user/plugin.git"
	pluginURL.Prompt = "url: "
	pluginURL.Focus()
	pluginLoad := textinput.New()
	pluginLoad.Placeholder = "plugin.zsh"
	pluginLoad.Prompt = "load: "
	promptName := textinput.New()
	promptName.Prompt = "Display name: "
	promptName.Placeholder = "empty uses the system user"
	promptName.SetValue(cfg.Prompt.DisplayName)

	m := Model{
		cfg:               cfg,
		previewCtx:        initialPreview,
		inputs:            inputs,
		previewScenario:   1,
		promptName:        promptName,
		pluginWizard:      newPluginWizardModel(),
		pluginCloneRunner: plugins.ExecCloneRunner{},
		pluginURL:         pluginURL,
		pluginLoad:        pluginLoad,
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
	case pluginStageResult:
		return m.handlePluginStageResult(msg)
	case applyResult:
		m.busy = false
		m.operation = ""
		m.reviewedConfig = nil
		m.reviewedPluginChanges = plugins.ChangeSet{}
		m.showApplyTechnical = false
		m.msg = string(msg)
		return m, nil
	case pluginApplyResult:
		m.busy = false
		m.operation = ""
		m.reviewedConfig = nil
		m.reviewedPluginChanges = plugins.ChangeSet{}
		m.showApplyTechnical = false
		if msg.err != nil {
			m.msg = "apply error: " + msg.err.Error()
			return m, nil
		}
		m.pluginChanges = plugins.ChangeSet{}
		m.msg = "applied"
		return m, nil
	case pluginInstallResult:
		m.busy = false
		m.operation = ""
		if msg.err != nil {
			m.msg = "plugin install error: " + msg.err.Error()
			return m, nil
		}
		m.cfg = msg.cfg
		m.msg = "recommended plugins installed and active"
		return m, nil
	case fontInstallResult:
		m.busy = false
		m.operation = ""
		if msg.err != nil {
			m.msg = "font install error: " + msg.err.Error()
			return m, nil
		}
		m.cfg = msg.cfg
		if shell.IsTermux() {
			m.msg = msg.font.Name + " installed; Nerd icons are active"
		} else {
			m.msg = msg.font.Name + " installed; select it in your terminal settings"
		}
		return m, nil
	case backupRestoreResult:
		m.busy = false
		m.operation = ""
		if msg.err != nil {
			m.msg = "backup restore error: " + msg.err.Error()
		} else {
			m.msg = "backup restored to .zshrc"
		}
		return m, nil
	case fontRestoreResult:
		m.busy = false
		m.operation = ""
		if msg.err != nil {
			m.msg = "font restore error: " + msg.err.Error()
		} else {
			m.msg = "previous Termux font restored"
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (msg.String() == "q" && !m.promptEditingName && m.tab != tabPreview && m.tab != tabPlugins) {
			return m, tea.Quit
		}
		if m.pluginRemoveConfirm {
			return m.updatePluginRemoveConfirmation(msg)
		}
		if m.pluginWizard.Step != pluginWizardClosed {
			return m.updatePluginWizard(msg)
		}
		if m.busy {
			return m, nil
		}
		if m.fontOpen {
			return m.updateFontDialog(msg)
		}
		if m.backupOpen {
			return m.updateBackupDialog(msg)
		}
		if m.confirmApply {
			return m.updateApplyModal(msg)
		}
		if msg.String() == "ctrl+a" {
			m.openApplyReview()
			return m, nil
		}
		if m.doctorOpen {
			switch msg.String() {
			case "enter":
				m.confirmDoctor = true
			case "y":
				if m.confirmDoctor {
					m.msg = m.fixDoctor()
					m.confirmDoctor = false
				}
			case "n":
				m.confirmDoctor = false
			case "esc", "q":
				if m.confirmDoctor {
					m.confirmDoctor = false
				} else {
					m.doctorOpen = false
					m.msg = ""
				}
			}
			return m, nil
		}
		if m.tab == tabPrompt && m.promptEditingName {
			return m.updatePromptName(msg)
		}
		if m.formAllowsGlobalNavigation(msg) {
			switch msg.String() {
			case "tab", "right":
				m.setTab((m.tab + 1) % len(tabs))
				return m, nil
			case "shift+tab", "left":
				m.setTab((m.tab + len(tabs) - 1) % len(tabs))
				return m, nil
			case "1", "2", "3", "4", "5":
				idx, _ := strconv.Atoi(msg.String())
				m.setTab(idx - 1)
				return m, nil
			}
		}

		if m.tab == tabPreview {
			return m.updatePreviewInputs(msg)
		}
		if m.tab == tabPlugins && m.pluginAdvanced && m.pluginInputShouldHandle(msg) {
			return m.updatePluginInputs(msg)
		}

		switch msg.String() {
		case "1", "2", "3", "4", "5":
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
			if m.tab == tabPrompt {
				m.toggleSegment()
			} else if m.tab == tabPlugins {
				m.togglePluginAtCursor()
			}
		case "enter":
			m.handleEnter()
		case "J":
			if m.tab == tabPrompt {
				m.moveSegment(1)
			}
		case "K":
			if m.tab == tabPrompt {
				m.moveSegment(-1)
			}
		case "h":
			if m.tab == tabPrompt {
				m.cfg.Prompt.DisableHeavySegments = !m.cfg.Prompt.DisableHeavySegments
				if m.cfg.Prompt.DisableHeavySegments {
					m.msg = "heavy segments disabled"
				} else {
					m.msg = "heavy segments enabled"
				}
			}
		case "a":
			if m.tab == tabPlugins {
				m.openPluginWizard()
			} else {
				m.openApplyReview()
			}
		case "y":
			if m.confirmApply && !m.busy {
				m.busy = true
				m.operation = "apply"
				m.confirmApply = false
				m.msg = ""
				return m, doApply(m.cfg)
			}
			if m.confirmPlugins && !m.busy {
				m.busy = true
				m.operation = "plugins"
				m.confirmPlugins = false
				m.msg = ""
				return m, doInstallPlugins(m.cfg)
			}
		case "n", "esc":
			if m.confirmApply {
				m.confirmApply = false
				m.msg = "apply cancelled"
			} else if m.doctorOpen {
				m.doctorOpen = false
				m.msg = ""
			} else if m.confirmPlugins {
				m.confirmPlugins = false
				m.msg = "plugin install cancelled"
			}
		case "d":
			if m.tab == tabHome {
				m.doctorOpen = true
				m.confirmDoctor = false
				m.confirmApply = false
			} else if m.tab == tabPlugins {
				m.openPluginRemovalAtCursor()
			}
		case "f":
			if m.tab == tabHome {
				m.fontOpen = true
				m.fontCursor = 0
				m.confirmFont = false
				m.confirmFontRestore = false
				m.confirmApply = false
			}
		case "r":
			if m.tab == tabHome {
				backups, err := shell.Backups()
				if err != nil {
					m.msg = "backup list error: " + err.Error()
				} else if len(backups) == 0 {
					m.msg = "no backups available"
				} else {
					m.backupPaths = backups
					m.backupCursor = len(backups) - 1
					m.backupOpen = true
					m.confirmBackup = false
				}
			}
		case "c":
			if m.tab == tabThemes {
				m.saveCustomTheme()
			}
		case "[":
			if m.tab == tabThemes {
				m.cycleThemeVariant(-1)
			}
		case "]":
			if m.tab == tabThemes {
				m.cycleThemeVariant(1)
			}
		case "t":
			if m.tab == tabPlugins {
				m.trustPluginAtCursor()
			}
		case "u":
			if m.tab == tabPrompt {
				m.promptName.SetValue(m.cfg.Prompt.DisplayName)
				m.promptName.SetCursor(len(m.promptName.Value()))
				m.promptName.Focus()
				m.promptEditingName = true
			} else if m.tab == tabPlugins {
				m.untrustPluginAtCursor()
			}
		case "p":
			if m.tab == tabPlugins && m.pluginAdvanced && m.pluginURL.Value() == "" && m.pluginLoad.Value() == "" {
				m.addPluginFromInputs()
			}
		case "i":
			if m.tab == tabPrompt {
				if m.cfg.Prompt.IconMode == config.IconModeCompatible {
					m.cfg.Prompt.IconMode = config.IconModeNerd
				} else {
					m.cfg.Prompt.IconMode = config.IconModeCompatible
				}
				m.msg = "icon mode updated"
			} else if m.tab == tabPlugins && m.missingSelectedPlugins() > 0 {
				m.confirmPlugins = true
				m.confirmApply = false
			}
		case "x":
			if m.tab == tabPlugins && m.pluginURL.Value() == "" && m.pluginLoad.Value() == "" {
				m.pluginAdvanced = !m.pluginAdvanced
				m.cursor = 0
				m.syncCursor()
			}
		case "l":
			if m.tab == tabPrompt {
				if m.cfg.Prompt.Layout == config.PromptLayoutTwoLine {
					m.cfg.Prompt.Layout = config.PromptLayoutOneLine
					m.cfg.Prompt.Newline = false
				} else {
					m.cfg.Prompt.Layout = config.PromptLayoutTwoLine
					m.cfg.Prompt.Newline = true
				}
			} else if m.tab == tabPlugins {
				item, ok := m.selectedPluginListItem()
				if !ok || item.Kind != pluginItemCustom {
					m.msg = "select a custom plugin to change its load file"
				} else if err := m.openLoadFilePicker(item); err != nil {
					m.msg = "plugin load-file error: " + err.Error()
				}
			}
		case "o":
			if m.tab == tabPrompt {
				m.cyclePromptSymbol()
			}
		case "v":
			if m.tab == tabPrompt {
				m.promptAdvanced = !m.promptAdvanced
			}
		case "?":
			m.msg = "1-5 screens | arrows move | Ctrl+A review & apply | d doctor/remove | q quit"
		}
	}
	return m, nil
}

func (m *Model) handleEnter() {
	switch m.tab {
	case tabPrompt:
		m.toggleSegment()
	case tabThemes:
		m.applyThemeAtCursor()
	case tabPlugins:
		if strings.TrimSpace(m.pluginURL.Value()) != "" {
			m.addPluginFromInputs()
			return
		}
		m.togglePluginAtCursor()
	}
}

func (m *Model) openApplyReview() {
	before, after, err := shell.PreviewInjectBlock()
	if err != nil {
		m.msg = "apply preview error: " + err.Error()
		return
	}
	m.applyDiff = shell.DiffLines(before, after)
	m.reviewedConfig = cloneConfig(m.cfg)
	m.reviewedPluginChanges = m.pluginChanges.Clone()
	m.showApplyTechnical = false
	m.confirmApply = true
	m.confirmPlugins = false
	m.doctorOpen = false
	m.msg = "press y to apply, n or esc to cancel"
}

func (m Model) updateApplyModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.reviewedConfig == nil {
			m.confirmApply = false
			m.reviewedPluginChanges = plugins.ChangeSet{}
			m.msg = "apply review expired; open it again"
			return m, nil
		}
		snapshot := cloneConfig(m.reviewedConfig)
		changeSnapshot := m.reviewedPluginChanges.Clone()
		m.cfg = cloneConfig(snapshot)
		m.busy = true
		m.operation = "apply"
		m.confirmApply = false
		m.msg = ""
		return m, doApplyWithPlugins(snapshot, changeSnapshot)
	case "n", "esc":
		m.confirmApply = false
		m.reviewedConfig = nil
		m.reviewedPluginChanges = plugins.ChangeSet{}
		m.showApplyTechnical = false
		m.msg = "apply cancelled"
	case "t":
		m.showApplyTechnical = !m.showApplyTechnical
	}
	return m, nil
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
	m.themeVariant = 0
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
	if m.tab == tabThemes {
		m.themeVariant = 0
	}
	m.syncCursor()
}

func (m Model) selectionCount() int {
	switch m.tab {
	case tabPrompt:
		return len(m.cfg.Prompt.Order)
	case tabThemes:
		return len(themecatalog.List())
	case tabPlugins:
		return len(m.pluginListItems())
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
