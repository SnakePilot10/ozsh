package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const wideLayoutMinWidth = 72

type layoutSpec struct {
	terminalWidth    int
	terminalHeight   int
	contentWidth     int
	contentHeight    int
	workspaceWidth   int
	workspaceHeight  int
	workspaceContentHeight int
	wide             bool
	short            bool
}

func (m Model) layout() layoutSpec {
	terminalWidth := m.width
	if terminalWidth <= 0 {
		terminalWidth = 82
	}
	terminalHeight := m.height
	if terminalHeight <= 0 {
		terminalHeight = 30
	}

	workspaceWidth := terminalWidth - panelStyle.GetHorizontalFrameSize()
	if workspaceWidth < 1 {
		workspaceWidth = 1
	}
	workspaceContentHeight := terminalHeight - panelStyle.GetVerticalFrameSize()
	if workspaceContentHeight < 1 {
		workspaceContentHeight = 1
	}

	headerHeight := lipgloss.Height(renderHeader(m.tab, workspaceWidth))
	footerHeight := 1
	statusHeight := 0
	if strings.TrimSpace(m.msg) != "" {
		statusHeight = 1
	}
	workspaceHeight := workspaceContentHeight - headerHeight - footerHeight - statusHeight - 2
	if workspaceHeight < 1 {
		workspaceHeight = 1
	}

	return layoutSpec{
		terminalWidth:          terminalWidth,
		terminalHeight:         terminalHeight,
		contentWidth:           terminalWidth,
		contentHeight:          terminalHeight,
		workspaceWidth:         workspaceWidth,
		workspaceHeight:        workspaceHeight,
		workspaceContentHeight: workspaceContentHeight,
		wide:                   workspaceWidth >= wideLayoutMinWidth,
		short:                  workspaceContentHeight < 22,
	}
}

func (m Model) workspaceContent(spec layoutSpec) string {
	viewSpec := spec
	viewSpec.contentWidth = spec.workspaceWidth
	viewSpec.contentHeight = spec.workspaceContentHeight

	switch {
	case m.busy && m.operation == "plugins":
		return "Installing plugins…\n\nCloning and validating the selected repositories."
	case m.busy && m.operation == "font":
		return "Installing Nerd Font…\n\nDownloading, verifying SHA-256, and activating the font."
	case m.busy && m.operation == "font-restore":
		return "Restoring previous Termux font…"
	case m.busy && m.operation == "backup":
		return "Restoring backup…"
	case m.fontOpen:
		return m.fontDialog()
	case m.backupOpen:
		return m.backupDialog()
	case m.confirmPlugins:
		return m.pluginInstallConfirmation()
	case m.confirmApply || m.busy:
		return m.apply()
	case m.doctorOpen:
		return m.doctor()
	}

	switch m.tab {
	case tabHome:
		return m.homeWorkspace(viewSpec)
	case tabPrompt:
		return m.promptWorkspace(viewSpec)
	case tabThemes:
		return m.themesWorkspace(viewSpec)
	case tabPlugins:
		return m.pluginsWorkspace(viewSpec)
	case tabPreview:
		return m.previewWorkspace(viewSpec)
	default:
		return ""
	}
}

func screenFooter(tab int) string {
	switch tab {
	case tabHome:
		return renderHint("d doctor  ·  f font  ·  r restore  ·  Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	case tabPrompt:
		return renderHint("space toggle  ·  J/K reorder  ·  v details  ·  Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	case tabThemes:
		return renderHint("up/down choose  ·  enter apply  ·  [/] Circuit  ·  Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	case tabPlugins:
		return renderHint("space toggle  ·  i install  ·  x advanced  ·  Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	case tabPreview:
		return renderHint("[/] scenario  ·  up/down field  ·  Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	default:
		return renderHint("Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	}
}

func composeFullscreen(header, body, status, footer string, spec layoutSpec) string {
	width := spec.workspaceWidth
	height := spec.workspaceContentHeight
	header = truncateBlock(header, width)
	body = padBlock(body, width, spec.workspaceHeight)
	if ansi.StringWidth(footer) > width {
		footer = renderHint("Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
	}
	footer = fitHeight(footer, width, 1)

	parts := []string{header, "", body}
	if strings.TrimSpace(status) != "" {
		parts = append(parts, fitHeight(status, width, 1))
	}
	parts = append(parts, "", footer)
	return fitHeight(strings.Join(parts, "\n"), width, height)
}

func padBlock(value string, width, height int) string {
	if height <= 0 {
		return ""
	}
	value = fitHeight(value, width, height)
	lines := strings.Split(value, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func fitHeight(value string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	value = truncateBlock(value, width)
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func truncateBlock(value string, width int) string {
	if width <= 0 {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if ansi.StringWidth(line) > width {
			lines[i] = ansi.Truncate(line, width, "")
		}
	}
	return strings.Join(lines, "\n")
}
