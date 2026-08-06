package tui

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
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
	body.WriteString(renderSectionHeader("Home", "Welcome · System status and safe configuration actions"))
	body.WriteString("\n\n")
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 55 / 100
		right := spec.contentWidth - gutter - left
		body.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			totalWidthStyle(workspaceBoxStyle.Copy(), left).Render(summary.String()),
			strings.Repeat(" ", gutter),
			totalWidthStyle(workspaceBoxStyle.Copy(), right).Render(actions.String()),
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
	body.WriteString(renderSectionHeader(
		"Themes",
		fmt.Sprintf("Theme gallery · %d families · %d selectable presets", len(themecatalog.Families()), len(presets)),
	))
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
		detailsWidth := right - workspaceBoxStyle.GetHorizontalFrameSize()
		if detailsWidth < 12 {
			detailsWidth = 12
		}
		library = totalWidthStyle(workspaceBoxStyle.Copy(), left).Render(m.themeLibraryPanel(presets, listRows))
		details = totalWidthStyle(workspaceBoxStyle.Copy(), right).Render(m.themeDetailPanel(detailsWidth))
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
	if width < 20 {
		width = 20
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
	b.WriteString(themePaletteLines(selected))
	b.WriteString("\n\n")
	b.WriteString(renderPreviewBox("Live preview", prompt.Simulated(themecatalog.Apply(m.cfg, selected)), width))
	return b.String()
}

func themePaletteLines(preset themecatalog.Preset) string {
	colors := []string{preset.Theme.Accent, preset.Theme.Success, preset.Theme.Warning, preset.Theme.Error}
	dots := make([]string, 0, len(colors))
	for _, color := range colors {
		dots = append(dots, lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●"))
	}
	return strings.Join(dots, "  ") + "\n" +
		renderHint(colors[0]+"  ·  "+colors[1]) + "\n" +
		renderHint(colors[2]+"  ·  "+colors[3])
}

func (m Model) pluginsWorkspace(spec layoutSpec) string {
	var body strings.Builder
	body.WriteString(renderSectionHeader("Plugins", "Recommended setup · Custom Zsh extensions"))
	body.WriteString("\n\n")
	list := m.pluginLibraryPanel()
	details := m.selectedPluginPanel()
	if spec.wide {
		gutter := 2
		left := (spec.contentWidth - gutter) * 52 / 100
		right := spec.contentWidth - gutter - left
		body.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			totalWidthStyle(workspaceBoxStyle.Copy(), left).Render(list),
			strings.Repeat(" ", gutter),
			totalWidthStyle(workspaceBoxStyle.Copy(), right).Render(details),
		))
	} else {
		body.WriteString(list)
		body.WriteString("\n\n")
		body.WriteString(details)
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func (m Model) pluginLibraryPanel() string {
	items := m.pluginListItems()
	var b strings.Builder
	b.WriteString(renderGroupLabel("Recommended plugins"))
	b.WriteString("\n")
	for index, item := range items {
		if item.Kind != pluginItemRecommended {
			continue
		}
		status := plugins.StatusFor(m.cfg, item.Definition)
		check := "[ ]"
		if status.Selected {
			check = "[x]"
		}
		row := fmt.Sprintf("%s %s %s", check, item.Definition.Name, pluginStateBadge(status))
		b.WriteString(m.renderPluginListRow(index, row))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(renderGroupLabel("Custom plugins"))
	b.WriteString("\n")
	customCount := 0
	for index, item := range items {
		if item.Kind != pluginItemCustom {
			continue
		}
		customCount++
		configured := m.cfg.Plugins.Items[item.ConfigIndex]
		check := "[ ]"
		if configured.Enabled {
			check = "[x]"
		}
		row := fmt.Sprintf("%s %s %s", check, configured.Name, customPluginStateBadge(configured, item.Pending))
		b.WriteString(m.renderPluginListRow(index, row))
		b.WriteByte('\n')
	}
	if customCount == 0 {
		b.WriteString("  ")
		b.WriteString(renderHint("No custom plugins yet"))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(accentStyle.Render("[a] Add custom plugin"))
	return strings.TrimSuffix(b.String(), "\n")
}

func (m Model) renderPluginListRow(index int, row string) string {
	if index == m.cursor {
		return selectedRowStyle.Copy().Padding(0).Render("› " + row)
	}
	return "  " + row
}

func pluginStateBadge(status plugins.Status) string {
	label := "[missing]"
	style := mutedStyle
	switch {
	case status.Active:
		label = "[active]"
		style = stateOnStyle
	case status.Installed && !status.Healthy:
		label = "[attention]"
		style = errorStyle
	case status.Installed && !status.Trusted:
		label = "[untrusted]"
		style = errorStyle
	case status.Installed:
		label = "[disabled]"
	}
	return style.Render(label)
}

func customPluginStateBadge(item config.PluginItem, pending bool) string {
	if pending {
		return accentStyle.Render("[pending]")
	}
	label := "[attention]"
	style := errorStyle
	switch {
	case !item.Trusted:
		label = "[untrusted]"
	case !item.Enabled:
		label = "[disabled]"
		style = mutedStyle
	case customPluginActive(item):
		label = "[active]"
		style = stateOnStyle
	}
	return style.Render(label)
}

func customPluginActive(item config.PluginItem) bool {
	if !item.Enabled || !item.Trusted {
		return false
	}
	return plugins.ValidateCandidate(filepath.Clean(item.Source), filepath.Clean(item.Load)) == nil
}

func (m Model) selectedPluginPanel() string {
	var b strings.Builder
	b.WriteString(renderGroupLabel("Selected plugin"))
	b.WriteString("\n")
	selected, ok := m.selectedPluginListItem()
	if !ok {
		b.WriteString(renderHint("No plugins available"))
		return b.String()
	}
	if selected.Kind == pluginItemCustom {
		m.writeSelectedCustomPluginPanel(&b, selected)
		return b.String()
	}
	m.writeSelectedRecommendedPluginPanel(&b, selected.Definition)
	return b.String()
}

func (m Model) writeSelectedRecommendedPluginPanel(b *strings.Builder, definition plugins.Definition) {
	status := plugins.StatusFor(m.cfg, definition)
	b.WriteString(accentStyle.Render(definition.Name))
	b.WriteString("\n")
	b.WriteString(definition.Description)
	b.WriteString("\n\n")
	b.WriteString(renderKeyValue("State", pluginStateLabel(status)))
	b.WriteString("\n")
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
	b.WriteString(renderKeyValue("Load file", definition.Load))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Repository", definition.URL))
}

func (m Model) writeSelectedCustomPluginPanel(b *strings.Builder, selected pluginListItem) {
	if selected.ConfigIndex < 0 || selected.ConfigIndex >= len(m.cfg.Plugins.Items) {
		b.WriteString(renderHint("Custom plugin configuration is unavailable"))
		return
	}
	item := m.cfg.Plugins.Items[selected.ConfigIndex]
	active := !selected.Pending && customPluginActive(item)
	state := customPluginStateLabel(item)
	if selected.Pending {
		state = "Pending"
	}
	b.WriteString(accentStyle.Render(item.Name))
	b.WriteString("\n")
	b.WriteString("Custom plugin managed by ozsh.")
	b.WriteString("\n\n")
	b.WriteString(renderKeyValue("State", state))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Managed path", item.Source))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Load file", item.Load))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Trusted", yesNo(item.Trusted)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Enabled", yesNo(item.Enabled)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Active", yesNo(active)))
	b.WriteString("\n\n")
	b.WriteString(renderGroupLabel("Actions"))
	b.WriteString("\n")
	b.WriteString("[space] Enable/disable")
	b.WriteString("\n")
	b.WriteString("[t/u] Trust/remove trust")
	b.WriteString("\n")
	b.WriteString("[l] Change load file")
	b.WriteString("\n")
	b.WriteString("[d] Remove plugin")
}

func customPluginStateLabel(item config.PluginItem) string {
	switch {
	case !item.Trusted:
		return "Untrusted"
	case !item.Enabled:
		return "Disabled"
	case customPluginActive(item):
		return "Active"
	default:
		return "Attention"
	}
}

func pluginStateLabel(status plugins.Status) string {
	switch {
	case status.Active:
		return "Active"
	case status.Installed && !status.Healthy:
		return "Attention"
	case status.Installed && !status.Trusted:
		return "Untrusted"
	case status.Installed:
		return "Disabled"
	default:
		return "Missing"
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
			totalWidthStyle(workspaceBoxStyle.Copy(), left).Render(scenarios),
			strings.Repeat(" ", gutter),
			totalWidthStyle(workspaceBoxStyle.Copy(), right).Render(context),
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
