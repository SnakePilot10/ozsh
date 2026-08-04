package tui

import (
	"fmt"
	"strings"
)

func (m Model) pluginWizardWorkspace(spec layoutSpec) string {
	var body strings.Builder
	body.WriteString(renderSectionHeader("Add custom plugin", pluginWizardSubtitle(m.pluginWizard.Step)))
	body.WriteString("\n\n")

	switch m.pluginWizard.Step {
	case pluginWizardURL:
		body.WriteString(m.pluginWizardURLView())
	case pluginWizardCloning:
		body.WriteString(m.pluginWizardCloningView())
	case pluginWizardCandidates:
		body.WriteString(m.pluginWizardCandidatesView())
	case pluginWizardTrust:
		body.WriteString(m.pluginWizardTrustView())
	case pluginWizardSummary:
		body.WriteString(m.pluginWizardSummaryView())
	default:
		return ""
	}
	if strings.TrimSpace(m.pluginWizard.Error) != "" {
		body.WriteString("\n\n")
		body.WriteString(errorStyle.Render(m.pluginWizard.Error))
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func pluginWizardSubtitle(step pluginWizardStep) string {
	switch step {
	case pluginWizardURL:
		return "Repository URL"
	case pluginWizardCloning:
		return "Cloning repository"
	case pluginWizardCandidates:
		return "Choose load file"
	case pluginWizardTrust:
		return "Trust review"
	case pluginWizardSummary:
		return "Pending plugin"
	default:
		return "Custom plugin"
	}
}

func (m Model) pluginWizardURLView() string {
	var body strings.Builder
	body.WriteString(renderGroupLabel("Repository URL"))
	body.WriteString("\n")
	body.WriteString(m.pluginWizard.URL.View())
	body.WriteString("\n\n")
	body.WriteString(renderHint("Only credential-free HTTPS repository URLs are accepted."))
	body.WriteString("\n")
	body.WriteString(renderHint("Enter clone and inspect  ·  Esc cancel"))
	return body.String()
}

func (m Model) pluginWizardCloningView() string {
	var body strings.Builder
	body.WriteString(renderGroupLabel("Cloning repository"))
	body.WriteString("\n")
	body.WriteString("Validating the URL, creating a private staging checkout, and discovering load files.")
	body.WriteString("\n\n")
	body.WriteString(renderKeyValue("Repository", strings.TrimSpace(m.pluginWizard.URL.Value())))
	body.WriteString("\n\n")
	body.WriteString(renderHint("Esc cancel"))
	return body.String()
}

func (m Model) pluginWizardCandidatesView() string {
	var body strings.Builder
	body.WriteString(renderGroupLabel("Choose load file"))
	body.WriteString("\n")
	if m.pluginWizard.Stage == nil || len(m.pluginWizard.Stage.Candidates) == 0 {
		body.WriteString(renderHint("No candidates available"))
		return body.String()
	}
	for index, candidate := range m.pluginWizard.Stage.Candidates {
		row := candidate.Path
		if index == m.pluginWizard.Candidate {
			row = selectedRowStyle.Copy().Padding(0).Render("› " + row)
		} else {
			row = "  " + row
		}
		body.WriteString(row)
		if index+1 < len(m.pluginWizard.Stage.Candidates) {
			body.WriteByte('\n')
		}
	}
	body.WriteString("\n\n")
	body.WriteString(renderHint("up/down choose  ·  Enter continue  ·  Esc discard checkout"))
	return body.String()
}

func (m Model) pluginWizardTrustView() string {
	candidate, ok := m.selectedWizardCandidate()
	if !ok || m.pluginWizard.Stage == nil {
		return renderGroupLabel("Trust review") + "\n" + errorStyle.Render("Selected candidate is unavailable")
	}
	var body strings.Builder
	body.WriteString(renderGroupLabel("Trust review"))
	body.WriteString("\n")
	body.WriteString(errorStyle.Render("A trusted plugin executes shell code in every interactive Zsh session."))
	body.WriteString("\n\n")
	body.WriteString(renderKeyValue("Repository", m.pluginWizard.Stage.Repository.URL))
	body.WriteString("\n")
	body.WriteString(renderKeyValue("Load file", candidate.Path))
	body.WriteString("\n")
	body.WriteString(renderKeyValue("Managed path", m.pluginWizard.Stage.FinalDir))
	body.WriteString("\n\n")
	body.WriteString(renderHint("y/Enter trust and continue  ·  n/Esc back"))
	return body.String()
}

func (m Model) pluginWizardSummaryView() string {
	candidate, ok := m.selectedWizardCandidate()
	if !ok || m.pluginWizard.Stage == nil {
		return renderGroupLabel("Pending plugin") + "\n" + errorStyle.Render("Selected candidate is unavailable")
	}
	var body strings.Builder
	body.WriteString(renderGroupLabel("Pending plugin"))
	body.WriteString("\n")
	body.WriteString(accentStyle.Render(m.pluginWizard.Stage.Repository.Name))
	body.WriteString("\n\n")
	body.WriteString(renderKeyValue("Repository", m.pluginWizard.Stage.Repository.URL))
	body.WriteString("\n")
	body.WriteString(renderKeyValue("Load file", candidate.Path))
	body.WriteString("\n")
	body.WriteString(renderKeyValue("Final path", m.pluginWizard.Stage.FinalDir))
	body.WriteString("\n")
	body.WriteString(renderKeyValue("Initial state", "Trusted and enabled, pending"))
	body.WriteString("\n\n")
	body.WriteString(fmt.Sprintf("Enter queues this plugin. Nothing becomes active until %s.", accentStyle.Render("Review & Apply")))
	body.WriteString("\n")
	body.WriteString(renderHint("Enter queue plugin  ·  Esc back"))
	return body.String()
}
