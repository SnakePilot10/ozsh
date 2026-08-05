package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	applypkg "github.com/snakepilot10/ozsh/internal/apply"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

type pluginApplyResult struct {
	err error
}

func doApplyWithPlugins(cfg *config.Config, changes plugins.ChangeSet) tea.Cmd {
	configSnapshot := cloneConfig(cfg)
	changeSnapshot := changes.Clone()
	return func() tea.Msg {
		err := applypkg.Apply(applypkg.Request{
			Config:        configSnapshot,
			PluginChanges: changeSnapshot,
		})
		return pluginApplyResult{err: err}
	}
}

func (m Model) pendingPluginReview() string {
	changes := m.reviewedPluginChanges
	adds, removes := changes.Counts()

	var body strings.Builder
	body.WriteString(renderGroupLabel("Plugin changes"))
	body.WriteString("\n")
	body.WriteString(fmt.Sprintf("%d add · %d remove", adds, removes))

	for _, addition := range changes.Adds {
		body.WriteString("\n\n")
		body.WriteString(accentStyle.Render("+ " + addition.Name))
		writeChangeDetail(&body, "Repository", addition.RepositoryURL)
		writeChangeDetail(&body, "Load file", addition.Load)
		writeChangeDetail(&body, "Final path", addition.FinalDir)
	}
	for _, removal := range changes.Removes {
		body.WriteString("\n\n")
		body.WriteString(errorStyle.Render("- " + removal.Name))
		writeChangeDetail(&body, "Remove path", removal.Source)
	}
	if adds == 0 && removes == 0 {
		body.WriteString("\n")
		body.WriteString(renderHint("No plugin filesystem changes pending"))
	}
	return body.String()
}

func (m Model) appendPendingPluginReview(base string, spec layoutSpec) string {
	if m.reviewedPluginChanges.Empty() {
		return base
	}
	review := m.pendingPluginReview()
	const marker = "\n\n[t] Technical details"
	if strings.Contains(base, marker) {
		base = strings.Replace(base, marker, "\n\n"+review+marker, 1)
	} else {
		base += "\n\n" + review
	}
	return fitHeight(base, spec.contentWidth, spec.workspaceHeight)
}

func writeChangeDetail(body *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	body.WriteString("\n")
	body.WriteString(renderKeyValue(label, value))
}
