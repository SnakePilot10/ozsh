package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/prompt"
)

var (
	workspaceBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(visualPalette.Border).
				Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(visualPalette.Text).
				Background(visualPalette.Surface).
				Bold(true).
				Padding(0, 1)

	stateOnStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Success).
			Bold(true)

	stateOffStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Muted)
)

func (m Model) promptWorkspace(spec layoutSpec) string {
	if m.promptEditingName {
		return renderSectionHeader("Prompt", "Edit the display identity") + "\n\n" +
			renderGroupLabel("Display name") + "\n" + m.promptName.View() + "\n\n" +
			renderHint("enter keep  ·  esc cancel")
	}

	var b strings.Builder
	b.WriteString(renderSectionHeader("Prompt", "Identity, layout, and visible segments"))
	b.WriteString("\n\n")

	if spec.wide {
		gutter := 2
		leftWidth := (spec.contentWidth - gutter) * 44 / 100
		if leftWidth < 28 {
			leftWidth = 28
		}
		rightWidth := spec.contentWidth - gutter - leftWidth
		if rightWidth < 28 {
			rightWidth = 28
			leftWidth = spec.contentWidth - gutter - rightWidth
		}
		configuration := m.promptConfigurationPanel(leftWidth, false)
		segments := m.promptSegmentsPanel(rightWidth, 8)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, configuration, strings.Repeat(" ", gutter), segments))
		b.WriteString("\n\n")
		b.WriteString(m.promptLivePreview(spec.contentWidth, false))
		if !spec.short {
			b.WriteString("\n\n")
			b.WriteString(m.selectedSegmentPanel(spec.contentWidth, false))
		}
		return fitHeight(b.String(), spec.contentWidth, spec.workspaceHeight)
	}

	b.WriteString(m.promptConfigurationPanel(spec.contentWidth, true))
	b.WriteString("\n")
	b.WriteString(m.promptSegmentsPanel(spec.contentWidth, compactSegmentRows(spec.workspaceHeight)))
	b.WriteString("\n")
	b.WriteString(m.promptLivePreview(spec.contentWidth, true))
	b.WriteString("\n")
	b.WriteString(m.selectedSegmentPanel(spec.contentWidth, true))
	return fitHeight(b.String(), spec.contentWidth, spec.workspaceHeight)
}

func compactSegmentRows(workspaceHeight int) int {
	if workspaceHeight < 16 {
		return 2
	}
	if workspaceHeight < 20 {
		return 3
	}
	return 4
}

func (m Model) promptConfigurationPanel(width int, compact bool) string {
	displayName := m.cfg.Prompt.DisplayName
	if displayName == "" {
		displayName = "System user"
	}
	layout := "One line"
	if m.cfg.Prompt.Layout == config.PromptLayoutTwoLine {
		layout = "Two lines"
	}
	iconMode := "Compatible"
	if m.cfg.Prompt.IconMode == config.IconModeNerd {
		iconMode = "Nerd Font"
	}
	rightPrompt := "Disabled"
	if m.cfg.Prompt.RightPrompt {
		rightPrompt = "Enabled"
	}
	heavy := "Enabled"
	if m.cfg.Prompt.DisableHeavySegments {
		heavy = "Disabled"
	}

	if compact {
		first := fmt.Sprintf("Name %s  ·  Layout %s", displayName, layout)
		second := fmt.Sprintf("Symbol %s  ·  Icons %s  ·  Heavy %s", m.cfg.Prompt.Symbol, iconMode, heavy)
		return renderGroupLabel("Configuration") + "\n" + fitBlock(first+"\n"+second, width)
	}

	var b strings.Builder
	b.WriteString(renderGroupLabel("Configuration"))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Display name", displayName))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Layout", layout))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Symbol", m.cfg.Prompt.Symbol))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Icons", iconMode))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Right prompt", rightPrompt))
	b.WriteString("\n")
	separator := m.cfg.Prompt.Separator
	if separator == "" {
		separator = "none"
	}
	b.WriteString(renderKeyValue("Separator", separator))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Heavy", heavy))
	return totalWidthStyle(workspaceBoxStyle.Copy(), width).Render(b.String())
}

func (m Model) promptSegmentsPanel(width, maxRows int) string {
	var b strings.Builder
	total := len(m.cfg.Prompt.Order)
	enabled := 0
	for _, name := range m.cfg.Prompt.Order {
		if m.cfg.Prompt.Segments[name].Enabled {
			enabled++
		}
	}
	fmt.Fprintf(&b, "%s %s\n", renderGroupLabel("Segments"), renderHint(fmt.Sprintf("(%d/%d)", enabled, total)))
	if total == 0 {
		b.WriteString("No segments configured.")
		return b.String()
	}

	if maxRows < 1 {
		maxRows = 1
	}
	start := m.cursor - maxRows/2
	if start < 0 {
		start = 0
	}
	if start+maxRows > total {
		start = total - maxRows
		if start < 0 {
			start = 0
		}
	}
	end := start + maxRows
	if end > total {
		end = total
	}
	if start > 0 {
		b.WriteString(renderHint("↑ more"))
		b.WriteByte('\n')
	}
	for i := start; i < end; i++ {
		name := m.cfg.Prompt.Order[i]
		segment := m.cfg.Prompt.Segments[name]
		state := stateOffStyle.Render("[ ]")
		if segment.Enabled {
			state = stateOnStyle.Render("[x]")
		}
		icon := promptSegmentIcon(m.cfg.Prompt, segment)
		if icon != "" {
			icon += " "
		}
		row := fmt.Sprintf("%s %s%s", state, icon, segmentLabel(name))
		if i == m.cursor {
			row = selectedRowStyle.Copy().Width(innerBoxWidth(width)).Render("› " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		if i+1 < end {
			b.WriteByte('\n')
		}
	}
	if end < total {
		b.WriteByte('\n')
		b.WriteString(renderHint("↓ more"))
	}
	if width >= 34 {
		return totalWidthStyle(workspaceBoxStyle.Copy(), width).Render(b.String())
	}
	return b.String()
}

func (m Model) promptLivePreview(width int, compact bool) string {
	content := prompt.Simulated(m.previewConfig())
	if compact {
		return renderGroupLabel("Live preview") + "\n" + fitBlock(content, width)
	}
	return renderPreviewBox("Live preview", content, width)
}

func (m Model) selectedSegmentPanel(width int, compact bool) string {
	name, segment, ok := m.selectedSegment()
	if !ok {
		return renderGroupLabel("Selected segment") + "\n" + renderHint("No segment selected")
	}
	state := "Disabled"
	stateStyle := stateOffStyle
	if segment.Enabled {
		state = "Enabled"
		stateStyle = stateOnStyle
	}
	label := segmentLabel(name)
	if compact {
		summary := fmt.Sprintf("%s · %s · %s · %s", label, state, segment.FG, segmentCondition(name, segment))
		return renderGroupLabel("Selected segment") + "\n" + fitBlock(summary, width)
	}

	iconMode := "Compatible"
	icon := segment.CompatibleIcon
	if m.cfg.Prompt.IconMode == config.IconModeNerd {
		iconMode = "Nerd Font"
		icon = segment.NerdIcon
	}
	if icon == "" {
		icon = "none"
	}
	style := "Regular"
	if segment.Bold {
		style = "Bold"
	}

	var b strings.Builder
	b.WriteString(renderGroupLabel("Selected segment"))
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(label))
	b.WriteString("  ")
	b.WriteString(stateStyle.Render(state))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Description", segmentDescription(name)))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Icon", iconMode+" · "+icon))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Color", segment.FG))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Style", style))
	b.WriteString("\n")
	b.WriteString(renderKeyValue("Condition", segmentCondition(name, segment)))
	return totalWidthStyle(workspaceBoxStyle.Copy(), width).Render(b.String())
}

func (m Model) selectedSegment() (string, config.SegmentConfig, bool) {
	if len(m.cfg.Prompt.Order) == 0 {
		return "", config.SegmentConfig{}, false
	}
	index := m.cursor
	if index < 0 {
		index = 0
	}
	if index >= len(m.cfg.Prompt.Order) {
		index = len(m.cfg.Prompt.Order) - 1
	}
	name := m.cfg.Prompt.Order[index]
	segment, ok := m.cfg.Prompt.Segments[name]
	return name, segment, ok
}

func segmentDescription(name string) string {
	descriptions := map[string]string{
		"user":    "Shows the current shell identity.",
		"cwd":     "Shows the current working directory.",
		"git":     "Shows the active Git branch and repository state.",
		"status":  "Shows the previous command exit state.",
		"time":    "Shows the current local time.",
		"host":    "Shows the current host name.",
		"venv":    "Shows the active Python virtual environment.",
		"node":    "Shows the detected Node.js project state.",
		"go":      "Shows the detected Go module state.",
		"battery": "Shows the device battery level when available.",
		"jobs":    "Shows the number of background jobs.",
	}
	if description, ok := descriptions[name]; ok {
		return description
	}
	return "Shows contextual prompt information."
}

func segmentCondition(name string, segment config.SegmentConfig) string {
	switch name {
	case "git":
		return "Inside Git repositories"
	case "status":
		if segment.ShowSuccess {
			return "Always"
		}
		return "On command failure"
	case "venv":
		return "When a Python environment is active"
	case "node":
		return "Inside Node.js projects"
	case "go":
		return "Inside Go modules"
	case "battery":
		return "When battery data is available"
	case "jobs":
		return "When background jobs exist"
	default:
		return "Always"
	}
}

func innerBoxWidth(width int) int {
	inner := width - workspaceBoxStyle.GetHorizontalFrameSize()
	if inner < 1 {
		return 1
	}
	return inner
}
