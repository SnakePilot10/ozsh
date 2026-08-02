package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

func (m Model) homeWorkspace(spec layoutSpec) string {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if shell.IsTermux() {
		platform = "Termux · " + platform
	}

	var summary strings.Builder
	summary.WriteString(renderGroupLabel("System summary"))
	summary.WriteString("\n")
	summary.WriteString(renderKeyValue("Platform", platform))
	summary.WriteString("\n")
	summary.WriteString(renderKeyValue("Theme", m.cfg.Theme.Name))
	summary.WriteString("\n")
	summary.WriteString(renderKeyValue("Icons", string(m.cfg.Prompt.IconMode)))
	summary.WriteString("\n")
	summary.WriteString(renderKeyValue("Plugins", fmt.Sprintf("%d selected", len(m.cfg.Plugins.Selected))))
	summary.WriteString("\n")
	summary.WriteString(renderKeyValue("Backups", fmt.Sprintf("%d available", backupCount())))

	var actions strings.Builder
	actions.WriteString(renderGroupLabel("Quick actions"))
	actions.WriteString("\n")
	actions.WriteString("[d]  Run Doctor\n")
	actions.WriteString("[f]  Manage Nerd Font\n")
	actions.WriteString("[r]  Restore Backup\n")
	actions.WriteString("[a]  Review & Apply")

	var body strings.Builder
	body.WriteString(renderSectionHeader("Home", "System status and safe configuration actions"))
	body.WriteString("\n\n")
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 55 / 100
		right := spec.contentWidth - gutter - left
		body.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			workspaceBoxStyle.Copy().Width(innerBoxWidth(left)).Render(summary.String()),
			strings.Repeat(" ", gutter),
			workspaceBoxStyle.Copy().Width(innerBoxWidth(right)).Render(actions.String()),
		))
	} else {
		body.WriteString(summary.String())
		body.WriteString("\n\n")
		body.WriteString(actions.String())
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func (m Model) themesWorkspace(spec layoutSpec) string {
	presets := themecatalog.List()
	var body strings.Builder
	body.WriteString(renderSectionHeader("Themes", fmt.Sprintf("%d families · %d selectable presets", len(themecatalog.Families()), len(presets))))
	body.WriteString("\n\n")

	listRows := 8
	if !spec.wide {
		listRows = 5
	}
	library := m.themeLibraryPanel(presets, listRows)
	details := m.themeDetailPanel(spec.contentWidth)
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 42 / 100
		right := spec.contentWidth - gutter - left
		library = workspaceBoxStyle.Copy().Width(innerBoxWidth(left)).Render(m.themeLibraryPanel(presets, listRows))
		details = workspaceBoxStyle.Copy().Width(innerBoxWidth(right)).Render(m.themeDetailPanel(right))
		body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, library, strings.Repeat(" ", gutter), details))
	} else {
		body.WriteString(library)
		body.WriteString("\n")
		body.WriteString(details)
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func (m Model) themeLibraryPanel(presets []themecatalog.Preset, rows int) string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Theme library"))
	b.WriteString("\n")
	if len(presets) == 0 {
		b.WriteString(renderHint("No presets available"))
		return b.String()
	}
	if rows < 1 {
		rows = 1
	}
	start := m.cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > len(presets) {
		start = len(presets) - rows
		if start < 0 {
			start = 0
		}
	}
	end := start + rows
	if end > len(presets) {
		end = len(presets)
	}
	if start > 0 {
		b.WriteString(renderHint("↑ more"))
		b.WriteByte('\n')
	}
	for i := start; i < end; i++ {
		preset := presets[i]
		applied := " "
		if m.cfg.Theme.ID == preset.ID && m.cfg.Theme.Variant == preset.Variant {
			applied = "✓"
		}
		row := fmt.Sprintf("%s %s", applied, preset.Name)
		if i == m.cursor {
			row = selectedRowStyle.Render("› " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		if i+1 < end {
			b.WriteByte('\n')
		}
	}
	if end < len(presets) {
		b.WriteByte('\n')
		b.WriteString(renderHint("↓ more"))
	}
	return b.String()
}

func (m Model) themeDetailPanel(width int) string {
	selected, ok := m.selectedTheme()
	if !ok {
		return renderGroupLabel("Description") + "\n" + renderHint("Select a theme to inspect it")
	}
	previewWidth := width
	if previewWidth < 20 {
		previewWidth = 20
	}
	var b strings.Builder
	b.WriteString(renderGroupLabel("Description"))
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(selected.Name))
	if selected.Variant != "" {
		b.WriteString("  ")
		b.WriteString(renderVariantBadge(selected.Variant))
	}
	b.WriteString("\n")
	b.WriteString(selected.Description)
	b.WriteString("\n\n")
	b.WriteString(renderGroupLabel("Palette"))
	b.WriteString("\n")
	b.WriteString(themeSwatches(selected))
	b.WriteString("\n\n")
	b.WriteString(renderPreviewBox("Live preview", prompt.Simulated(themecatalog.Apply(m.cfg, selected)), previewWidth))
	return b.String()
}

func (m Model) pluginsWorkspace(spec layoutSpec) string {
	catalog := plugins.Catalog()
	var body strings.Builder
	body.WriteString(renderSectionHeader("Plugins", "Curated completion and editing helpers"))
	body.WriteString("\n\n")
	list := m.pluginLibraryPanel(catalog)
	details := m.selectedPluginPanel(catalog)
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 52 / 100
		right := spec.contentWidth - gutter - left
		body.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			workspaceBoxStyle.Copy().Width(innerBoxWidth(left)).Render(list),
			strings.Repeat(" ", gutter),
			workspaceBoxStyle.Copy().Width(innerBoxWidth(right)).Render(details),
		))
	} else {
		body.WriteString(list)
		body.WriteString("\n\n")
		body.WriteString(details)
	}
	if m.pluginAdvanced {
		body.WriteString("\n\n")
		body.WriteString(renderGroupLabel("Advanced repository"))
		body.WriteString("\n")
		body.WriteString(m.pluginURL.View())
		body.WriteString("\n")
		body.WriteString(m.pluginLoad.View())
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func (m Model) pluginLibraryPanel(catalog []plugins.Definition) string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Recommended plugins"))
	b.WriteString("\n")
	for i, definition := range catalog {
		status := plugins.StatusFor(m.cfg, definition)
		check := "[ ]"
		if status.Selected {
			check = "[x]"
		}
		state := pluginStateLabel(status)
		row := fmt.Sprintf("%s %-21s %s", check, definition.Name, state)
		if i == m.cursor {
			row = selectedRowStyle.Render("› " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		if i+1 < len(catalog) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) selectedPluginPanel(catalog []plugins.Definition) string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Selected plugin"))
	b.WriteString("\n")
	if len(catalog) == 0 {
		b.WriteString(renderHint("No curated plugins available"))
		return b.String()
	}
	index := m.cursor
	if index < 0 {
		index = 0
	}
	if index >= len(catalog) {
		index = len(catalog) - 1
	}
	definition := catalog[index]
	status := plugins.StatusFor(m.cfg, definition)
	b.WriteString(accentStyle.Render(definition.Name))
	b.WriteString("\n")
	b.WriteString(definition.Description)
	b.WriteString("\n\n")
	b.WriteString(renderKeyValue("Selected", yesNo(status.Selected)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Installed", yesNo(status.Installed)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Trusted", yesNo(status.Trusted)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Healthy", yesNo(status.Healthy)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Active", yesNo(status.Active)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Load", definition.Load))
	return b.String()
}

func pluginStateLabel(status plugins.Status) string {
	switch {
	case status.Active:
		return "Active"
	case status.Installed && !status.Healthy:
		return "Needs attention"
	case status.Installed && !status.Trusted:
		return "Not trusted"
	case status.Installed:
		return "Disabled"
	default:
		return "Not installed"
	}
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func (m Model) previewWorkspace(spec layoutSpec) string {
	m.syncPreviewInputs()
	var body strings.Builder
	body.WriteString(renderSectionHeader("Preview", "Test prompt states before applying"))
	body.WriteString("\n\n")

	scenarios := m.previewScenariosPanel()
	context := m.previewContextPanel()
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 44 / 100
		right := spec.contentWidth - gutter - left
		body.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			workspaceBoxStyle.Copy().Width(innerBoxWidth(left)).Render(scenarios),
			strings.Repeat(" ", gutter),
			workspaceBoxStyle.Copy().Width(innerBoxWidth(right)).Render(context),
		))
	} else {
		body.WriteString(scenarios)
		body.WriteString("\n")
		body.WriteString(context)
	}
	body.WriteString("\n\n")
	body.WriteString(renderPreviewBox("Live preview", prompt.SimulatedWithContext(m.previewConfig(), m.previewCtx), spec.contentWidth))
	if m.previewError != "" {
		body.WriteString("\n")
		body.WriteString(errorStyle.Render(m.previewError))
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func (m Model) previewScenariosPanel() string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Scenarios"))
	b.WriteString("\n")
	for i, label := range previewScenarioLabels() {
		marker := "[ ]"
		if i == m.previewScenario && !m.previewCustom {
			marker = "[x]"
		}
		fmt.Fprintf(&b, "%s %s", marker, label)
		if i+1 < len(previewScenarioLabels()) {
			b.WriteByte('\n')
		}
	}
	if m.previewCustom {
		b.WriteString("\n[x] Custom context")
	}
	return b.String()
}

func (m Model) previewContextPanel() string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Context"))
	b.WriteString("\n")
	for i := range m.inputs {
		prefix := "  "
		if i == m.inputFocus {
			prefix = "› "
		}
		b.WriteString(prefix)
		b.WriteString(m.inputs[i].View())
		if i+1 < len(m.inputs) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
