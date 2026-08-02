package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const wideLayoutMinWidth = 72

type layoutSpec struct {
	terminalWidth  int
	terminalHeight int
	contentWidth   int
	contentHeight  int
	workspaceHeight int
	wide           bool
	short          bool
}

func (m Model) layout() layoutSpec {
	terminalWidth := m.width
	if terminalWidth <= 0 {
		terminalWidth = 80
	}
	terminalHeight := m.height
	if terminalHeight <= 0 {
		terminalHeight = 30
	}

	contentWidth := terminalWidth - panelStyle.GetHorizontalFrameSize()
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := terminalHeight - panelStyle.GetVerticalFrameSize()
	if contentHeight < 1 {
		contentHeight = 1
	}

	headerHeight := lipgloss.Height(renderHeader(m.tab, contentWidth))
	footerHeight := 1
	statusHeight := 0
	if strings.TrimSpace(m.msg) != "" {
		statusHeight = 1
	}
	workspaceHeight := contentHeight - headerHeight - footerHeight - statusHeight - 2
	if workspaceHeight < 1 {
		workspaceHeight = 1
	}

	return layoutSpec{
		terminalWidth:  terminalWidth,
		terminalHeight: terminalHeight,
		contentWidth:   contentWidth,
		contentHeight:  contentHeight,
		workspaceHeight: workspaceHeight,
		wide:           contentWidth >= wideLayoutMinWidth,
		short:          contentHeight < 22,
	}
}

func (m Model) workspaceContent(spec layoutSpec) string {
	_ = spec
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
		return m.home()
	case tabPrompt:
		return m.builder()
	case tabThemes:
		return m.themes()
	case tabPlugins:
		return m.plugins()
	case tabPreview:
		return m.preview()
	default:
		return ""
	}
}

func screenFooter(tab int) string {
	_ = tab
	return renderHint("Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")
}

func composeFullscreen(header, body, status, footer string, spec layoutSpec) string {
	header = fitBlock(header, spec.contentWidth)
	body = padBlock(body, spec.contentWidth, spec.workspaceHeight)
	footer = fitHeight(footer, spec.contentWidth, 1)

	parts := []string{header, "", body}
	if strings.TrimSpace(status) != "" {
		parts = append(parts, fitHeight(status, spec.contentWidth, 1))
	}
	parts = append(parts, "", footer)
	return fitHeight(strings.Join(parts, "\n"), spec.contentWidth, spec.contentHeight)
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
	value = fitBlock(value, width)
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
