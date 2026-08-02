package tui

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
)

const segmentFieldCount = 5

func (m *Model) beginSegmentEdit() tea.Cmd {
	index := m.builderIndex()
	if index < 0 || index >= len(m.cfg.Prompt.Order) {
		return nil
	}
	m.cursor = index
	m.segmentName = m.cfg.Prompt.Order[index]
	m.segmentDraft = m.cfg.Prompt.Segments[m.segmentName]
	m.segmentInputs = []textinput.Model{
		newSegmentInput("icon", m.segmentDraft.Icon, 32),
		newSegmentInput("fg", m.segmentDraft.FG, 16),
		newSegmentInput("bg", m.segmentDraft.BG, 16),
	}
	m.segmentField = 0
	m.segmentEditing = true
	m.refreshStyles()
	m.resize()
	m.setMessage(messageInfo, "edit fields, then press ctrl+s to apply")
	return m.focusSegmentInput()
}

func newSegmentInput(label, value string, limit int) textinput.Model {
	input := textinput.New()
	input.Prompt = label + ": "
	input.Placeholder = "none"
	input.CharLimit = limit
	input.SetValue(value)
	input.Blur()
	return input
}

func (m Model) updateSegmentEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if ok {
		switch {
		case key.Matches(keyMsg, keys.CloseForm):
			m.segmentEditing = false
			m.segmentInputs = nil
			m.setMessage(messageWarning, "segment edit cancelled")
			return m, nil
		case key.Matches(keyMsg, keys.ApplyEdit):
			m.commitSegmentEdit()
			return m, nil
		case key.Matches(keyMsg, keys.NextField), keyMsg.String() == "down":
			m.segmentField = wrapIndex(m.segmentField+1, segmentFieldCount)
			return m, m.focusSegmentInput()
		case key.Matches(keyMsg, keys.PrevField), keyMsg.String() == "up":
			m.segmentField = wrapIndex(m.segmentField-1, segmentFieldCount)
			return m, m.focusSegmentInput()
		case (keyMsg.String() == " " || keyMsg.String() == "enter") && m.segmentField < 2:
			if m.segmentField == 0 {
				m.segmentDraft.Enabled = !m.segmentDraft.Enabled
			} else {
				m.segmentDraft.Bold = !m.segmentDraft.Bold
			}
			return m, nil
		}
	}
	if m.segmentField < 2 {
		return m, nil
	}
	index := m.segmentField - 2
	var cmd tea.Cmd
	m.segmentInputs[index], cmd = m.segmentInputs[index].Update(msg)
	m.syncSegmentDraft()
	return m, cmd
}

func (m *Model) focusSegmentInput() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.segmentInputs {
		if i == m.segmentField-2 {
			cmd = m.segmentInputs[i].Focus()
		} else {
			m.segmentInputs[i].Blur()
		}
	}
	return cmd
}

func (m *Model) syncSegmentDraft() {
	if len(m.segmentInputs) != 3 {
		return
	}
	m.segmentDraft.Icon = m.segmentInputs[0].Value()
	m.segmentDraft.FG = strings.TrimSpace(m.segmentInputs[1].Value())
	m.segmentDraft.BG = strings.TrimSpace(m.segmentInputs[2].Value())
}

func (m *Model) commitSegmentEdit() {
	m.syncSegmentDraft()
	candidate := cloneConfig(m.cfg)
	candidate.Prompt.Segments[m.segmentName] = m.segmentDraft
	if err := config.Validate(candidate); err != nil {
		m.setMessage(messageError, "segment validation error: "+err.Error())
		return
	}
	m.cfg.Prompt.Segments[m.segmentName] = m.segmentDraft
	m.syncBuilderList(m.segmentName)
	m.segmentEditing = false
	m.segmentInputs = nil
	m.markDirty()
	m.setMessage(messageWarning, "segment updated; press s to save")
}

func (m Model) segmentEditorView() string {
	selected := func(field int) string {
		if m.segmentField == field {
			return "> "
		}
		return "  "
	}
	var b strings.Builder
	fmt.Fprintf(&b, "edit segment: %s\n\n", m.segmentName)
	fmt.Fprintf(&b, "%senabled: %t\n", selected(0), m.segmentDraft.Enabled)
	fmt.Fprintf(&b, "%sbold:   %t\n", selected(1), m.segmentDraft.Bold)
	for i := range m.segmentInputs {
		b.WriteString(selected(i + 2))
		b.WriteString(m.segmentInputs[i].View())
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(m.simulated(m.previewConfig()))
	return b.String()
}

func (m *Model) markDirty() {
	m.dirty = !reflect.DeepEqual(m.cfg, m.savedCfg)
}

func (m *Model) markSaved() {
	m.savedCfg = cloneConfig(m.cfg)
	m.dirty = false
}

func (m *Model) discardChanges() {
	if m.savedCfg != nil {
		m.cfg = cloneConfig(m.savedCfg)
	}
	m.refreshStyles()
	m.syncBuilderList("")
	m.dirty = false
	m.confirmDiscard = false
	m.setMessage(messageWarning, "unsaved changes discarded")
}
