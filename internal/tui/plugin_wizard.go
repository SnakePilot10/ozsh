package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

type pluginWizardStep uint8

const (
	pluginWizardClosed pluginWizardStep = iota
	pluginWizardURL
	pluginWizardCloning
	pluginWizardCandidates
	pluginWizardTrust
	pluginWizardSummary
)

type pluginWizardMode uint8

const (
	pluginWizardAdd pluginWizardMode = iota
	pluginWizardChangeLoad
)

type pluginWizardModel struct {
	Step              pluginWizardStep
	Mode              pluginWizardMode
	URL               textinput.Model
	Stage             *plugins.StagedRepository
	Candidates        []plugins.Candidate
	Candidate         int
	TargetName        string
	TargetRoot        string
	TargetConfigIndex int
	Error             string
	RequestID         uint64
	Cancel            context.CancelFunc
}

type pluginStageResult struct {
	RequestID uint64
	Stage     plugins.StagedRepository
	Err       error
}

func newPluginWizardModel() pluginWizardModel {
	input := textinput.New()
	input.Prompt = "url: "
	input.Placeholder = "https://github.com/user/plugin.git"
	input.CharLimit = 512
	return pluginWizardModel{URL: input, TargetConfigIndex: -1}
}

func (m *Model) openPluginWizard() {
	if m.pluginWizard.Cancel != nil {
		m.pluginWizard.Cancel()
	}
	if m.pluginWizard.Stage != nil {
		_ = m.pluginWizard.Stage.Cleanup()
	}
	requestID := m.pluginWizard.RequestID
	m.pluginWizard = newPluginWizardModel()
	m.pluginWizard.RequestID = requestID
	m.pluginWizard.Step = pluginWizardURL
	m.pluginWizard.Mode = pluginWizardAdd
	m.pluginWizard.URL.Focus()
	m.msg = ""
}

func (m Model) updatePluginWizard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.pluginWizard.Step {
	case pluginWizardURL:
		return m.updatePluginWizardURL(msg)
	case pluginWizardCloning:
		if msg.String() == "esc" {
			if m.pluginWizard.Cancel != nil {
				m.pluginWizard.Cancel()
			}
			m.pluginWizard.RequestID++
			m.pluginWizard.Cancel = nil
			m.pluginWizard.Step = pluginWizardClosed
			m.pluginWizard.Stage = nil
			m.pluginWizard.Candidates = nil
			m.msg = "plugin clone cancelled"
		}
		return m, nil
	case pluginWizardCandidates:
		return m.updatePluginWizardCandidates(msg)
	case pluginWizardTrust:
		return m.updatePluginWizardTrust(msg)
	case pluginWizardSummary:
		return m.updatePluginWizardSummary(msg)
	default:
		return m, nil
	}
}

func (m Model) updatePluginWizardURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pluginWizard.URL.Blur()
		m.pluginWizard.Step = pluginWizardClosed
		m.pluginWizard.Error = ""
		return m, nil
	case "enter":
		rawURL := strings.TrimSpace(m.pluginWizard.URL.Value())
		repository, err := plugins.ParseRepository(rawURL)
		if err != nil {
			m.pluginWizard.Error = err.Error()
			m.pluginWizard.URL.Focus()
			return m, nil
		}
		if err := plugins.ValidateNewRepository(m.cfg, repository); err != nil {
			m.pluginWizard.Error = err.Error()
			m.pluginWizard.URL.Focus()
			return m, nil
		}
		return m.startPluginClone(rawURL)
	default:
		updated, cmd := m.pluginWizard.URL.Update(msg)
		m.pluginWizard.URL = updated
		m.pluginWizard.Error = ""
		return m, cmd
	}
}

func (m Model) startPluginClone(rawURL string) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	m.pluginWizard.RequestID++
	requestID := m.pluginWizard.RequestID
	m.pluginWizard.Cancel = cancel
	m.pluginWizard.Step = pluginWizardCloning
	m.pluginWizard.Error = ""
	m.pluginWizard.URL.Blur()
	snapshot := cloneConfig(m.cfg)
	runner := m.pluginCloneRunner
	cmd := func() tea.Msg {
		defer cancel()
		stage, err := plugins.StageRepository(ctx, snapshot, rawURL, runner)
		return pluginStageResult{RequestID: requestID, Stage: stage, Err: err}
	}
	return m, cmd
}

func (m Model) handlePluginStageResult(result pluginStageResult) (tea.Model, tea.Cmd) {
	if result.RequestID != m.pluginWizard.RequestID || m.pluginWizard.Step != pluginWizardCloning {
		if result.Stage.StagingDir != "" {
			_ = result.Stage.Cleanup()
		}
		return m, nil
	}
	m.pluginWizard.Cancel = nil
	if result.Err != nil {
		if result.Stage.StagingDir != "" {
			_ = result.Stage.Cleanup()
		}
		m.pluginWizard.Stage = nil
		m.pluginWizard.Candidates = nil
		m.pluginWizard.Step = pluginWizardURL
		m.pluginWizard.Error = result.Err.Error()
		m.pluginWizard.URL.Focus()
		return m, nil
	}
	if len(result.Stage.Candidates) == 0 {
		_ = result.Stage.Cleanup()
		m.pluginWizard.Stage = nil
		m.pluginWizard.Candidates = nil
		m.pluginWizard.Step = pluginWizardURL
		m.pluginWizard.Error = "no supported .plugin.zsh, .zsh, or .sh load files were found"
		m.pluginWizard.URL.Focus()
		return m, nil
	}
	m.pluginWizard.Stage = &result.Stage
	m.pluginWizard.Candidates = append([]plugins.Candidate(nil), result.Stage.Candidates...)
	m.pluginWizard.Candidate = 0
	m.pluginWizard.Step = pluginWizardCandidates
	m.pluginWizard.Error = ""
	return m, nil
}

func (m Model) updatePluginWizardCandidates(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	candidates := m.pluginWizardCandidates()
	if len(candidates) == 0 {
		if m.pluginWizard.Mode == pluginWizardChangeLoad {
			m.pluginWizard.Step = pluginWizardClosed
			m.msg = "plugin candidates are unavailable"
			return m, nil
		}
		m.pluginWizard.Step = pluginWizardURL
		m.pluginWizard.Error = "plugin candidates are unavailable"
		m.pluginWizard.URL.Focus()
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.pluginWizard.Candidate > 0 {
			m.pluginWizard.Candidate--
		}
	case "down", "j":
		if m.pluginWizard.Candidate+1 < len(candidates) {
			m.pluginWizard.Candidate++
		}
	case "enter":
		if m.pluginWizard.Mode == pluginWizardChangeLoad {
			m.pluginWizard.Step = pluginWizardSummary
		} else {
			m.pluginWizard.Step = pluginWizardTrust
		}
	case "esc":
		if m.pluginWizard.Mode == pluginWizardChangeLoad {
			m.pluginWizard = newPluginWizardModel()
			m.msg = "load-file change cancelled"
			return m, nil
		}
		if m.pluginWizard.Stage != nil {
			if err := m.pluginWizard.Stage.Cleanup(); err != nil {
				m.pluginWizard.Error = err.Error()
			}
		}
		m.pluginWizard.Stage = nil
		m.pluginWizard.Candidates = nil
		m.pluginWizard.Candidate = 0
		m.pluginWizard.Step = pluginWizardURL
		m.pluginWizard.URL.Focus()
	}
	return m, nil
}

func (m Model) updatePluginWizardTrust(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.pluginWizard.Step = pluginWizardSummary
	case "n", "esc":
		m.pluginWizard.Step = pluginWizardCandidates
	}
	return m, nil
}

func (m Model) updatePluginWizardSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		if m.pluginWizard.Mode == pluginWizardChangeLoad {
			m.pluginWizard.Step = pluginWizardCandidates
		} else {
			m.pluginWizard.Step = pluginWizardTrust
		}
		return m, nil
	case "enter", "y":
		if m.pluginWizard.Mode == pluginWizardChangeLoad {
			return m.commitPluginLoadChange()
		}
		stage := m.pluginWizard.Stage
		candidate, ok := m.selectedWizardCandidate()
		if stage == nil || !ok {
			m.pluginWizard.Error = "selected plugin candidate is unavailable"
			return m, nil
		}
		if err := m.pluginChanges.QueueAdd(m.cfg, *stage, candidate.RelativePath); err != nil {
			m.pluginWizard.Error = fmt.Sprintf("queue plugin: %v", err)
			return m, nil
		}
		m.pluginWizard.Stage = nil
		m.pluginWizard.Candidates = nil
		m.pluginWizard.Step = pluginWizardClosed
		m.pluginWizard.Error = ""
		m.pluginWizard.URL.Blur()
		m.cursor = len(m.pluginListItems()) - 1
		m.syncCursor()
		m.msg = "plugin queued; Review & Apply to activate"
	}
	return m, nil
}

func (m Model) commitPluginLoadChange() (tea.Model, tea.Cmd) {
	candidate, ok := m.selectedWizardCandidate()
	if !ok {
		m.pluginWizard.Error = "selected plugin candidate is unavailable"
		return m, nil
	}
	index := m.pluginWizard.TargetConfigIndex
	if index < 0 || index >= len(m.cfg.Plugins.Items) || m.cfg.Plugins.Items[index].Name != m.pluginWizard.TargetName {
		m.pluginWizard.Error = "target plugin configuration is unavailable"
		return m, nil
	}
	if err := plugins.ValidateCandidate(m.pluginWizard.TargetRoot, candidate.RelativePath); err != nil {
		m.pluginWizard.Error = fmt.Sprintf("validate load file: %v", err)
		return m, nil
	}
	m.cfg.Plugins.Items[index].Load = candidate.RelativePath
	m.pluginWizard = newPluginWizardModel()
	m.msg = "plugin load file updated; Review & Apply to activate"
	return m, nil
}

func (m Model) pluginWizardCandidates() []plugins.Candidate {
	if len(m.pluginWizard.Candidates) > 0 {
		return m.pluginWizard.Candidates
	}
	if m.pluginWizard.Stage != nil {
		return m.pluginWizard.Stage.Candidates
	}
	return nil
}

func (m Model) selectedWizardCandidate() (plugins.Candidate, bool) {
	candidates := m.pluginWizardCandidates()
	if m.pluginWizard.Candidate < 0 || m.pluginWizard.Candidate >= len(candidates) {
		return plugins.Candidate{}, false
	}
	return candidates[m.pluginWizard.Candidate], true
}
