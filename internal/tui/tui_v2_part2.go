package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/fonts"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("ozsh"))
	b.WriteString(" ")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.busy && m.operation == "plugins" {
		b.WriteString("Installing plugins…\n\nCloning and validating the selected repositories.")
	} else if m.busy && m.operation == "font" {
		b.WriteString("Installing Nerd Font…\n\nDownloading, verifying SHA-256, and activating the font.")
	} else if m.busy && m.operation == "font-restore" {
		b.WriteString("Restoring previous Termux font…")
	} else if m.busy && m.operation == "backup" {
		b.WriteString("Restoring backup…")
	} else if m.fontOpen {
		b.WriteString(m.fontDialog())
	} else if m.backupOpen {
		b.WriteString(m.backupDialog())
	} else if m.confirmPlugins {
		b.WriteString(m.pluginInstallConfirmation())
	} else if m.confirmApply || m.busy {
		b.WriteString(m.apply())
	} else if m.doctorOpen {
		b.WriteString(m.doctor())
	} else {
		switch m.tab {
		case tabHome:
			b.WriteString(m.home())
		case tabPrompt:
			b.WriteString(m.builder())
		case tabThemes:
			b.WriteString(m.themes())
		case tabPlugins:
			b.WriteString(m.plugins())
		case tabPreview:
			b.WriteString(m.preview())
		}
	}

	if m.msg != "" {
		b.WriteString("\n\n")
		if strings.Contains(strings.ToLower(m.msg), "error") || strings.Contains(strings.ToLower(m.msg), "failed") {
			b.WriteString(errorStyle.Render(m.msg))
		} else {
			b.WriteString(mutedStyle.Render(m.msg))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Ctrl+A apply · ? help · Ctrl+C quit"))
	contentWidth := m.contentWidth()
	content := fitBlock(b.String(), contentWidth)
	panel := panelStyle.Copy()
	if m.width > 0 {
		panel = panel.Width(contentWidth)
	}
	return panel.Render(content)
}

func (m Model) renderTabs() string {
	items := make([]string, len(tabs))
	for i, tab := range tabs {
		if i == m.tab {
			items[i] = accentStyle.Render(tab)
		} else {
			items[i] = mutedStyle.Render(tab)
		}
	}
	full := strings.Join(items, "  ")
	maxWidth := m.contentWidth() - lipgloss.Width("ozsh ")
	if maxWidth <= 0 || lipgloss.Width(full) <= maxWidth {
		return full
	}
	return strings.Join(items[:3], "  ") + "\n     " + strings.Join(items[3:], "  ")
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 76
	}
	width := m.width - panelStyle.GetHorizontalFrameSize()
	if width < 1 {
		return 1
	}
	return width
}

func fitBlock(value string, width int) string {
	if width <= 0 {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) home() string {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if shell.IsTermux() {
		platform = "Termux · " + platform
	}
	return fmt.Sprintf("Welcome\n\nStatus      %s\nTheme       %s\nIcon mode   %s\nPlugins     %d selected\nBackups     %d\n\nActions\n[d] Run Doctor\n[f] Manage Nerd Font\n[r] Restore Backup\n[a] Review & Apply",
		platform, m.cfg.Theme.Name, m.cfg.Prompt.IconMode, len(m.cfg.Plugins.Selected), backupCount())
}

func (m Model) fontDialog() string {
	manifest := fonts.Manifest()
	if len(manifest) == 0 {
		return "Nerd Fonts\n\nNo verified fonts are available."
	}
	selected := manifest[wrapIndex(m.fontCursor, len(manifest))]
	if m.confirmFontRestore {
		return "Restore the previous Termux font?\n\nThe ozsh font backup will replace ~/.termux/font.ttf and Termux settings will reload.\n\nPress y to restore or n/esc to go back."
	}
	if m.confirmFont {
		platformNote := "The font will be installed for your user. Select it in your terminal settings afterward."
		if shell.IsTermux() {
			platformNote = "Your current Termux font will be backed up, then Termux settings will reload."
		}
		return fmt.Sprintf("Download and install %s?\n\nPinned release: %s\nThe archive is accepted only after SHA-256 verification.\n%s\nNerd icons will be enabled after success.\n\nPress y to install or n/esc to go back.", selected.Name, selected.Version, platformNote)
	}

	var b strings.Builder
	b.WriteString("Nerd Fonts\n")
	b.WriteString(mutedStyle.Render("Optional · Compatible icons remain the default"))
	b.WriteString("\n\n")
	for i, font := range manifest {
		prefix := "  "
		if i == m.fontCursor {
			prefix = "> "
		}
		recommended := ""
		if font.Recommended {
			recommended = " · Recommended"
		}
		fmt.Fprintf(&b, "%s%s%s\n", prefix, font.Name, recommended)
	}
	b.WriteString("\n")
	if shell.IsTermux() {
		b.WriteString("Termux: backup, install, and reload automatically.\n")
		b.WriteString("[r] Restore previous Termux font\n")
	} else {
		b.WriteString("Linux: install for this user, then choose the font in terminal settings.\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("up/down choose · enter continue · esc close"))
	return b.String()
}

func (m Model) backupDialog() string {
	if len(m.backupPaths) == 0 {
		return "Backups\n\nNo backups available."
	}
	selected := m.backupPaths[wrapIndex(m.backupCursor, len(m.backupPaths))]
	if m.confirmBackup {
		return fmt.Sprintf("Restore this backup?\n\n%s\n\nThe current .zshrc will be replaced atomically.\nPress y to restore or n/esc to go back.", filepath.Base(selected))
	}
	var b strings.Builder
	b.WriteString("Backups\n")
	b.WriteString(mutedStyle.Render("Choose a .zshrc snapshot"))
	b.WriteString("\n\n")
	for i, path := range m.backupPaths {
		prefix := "  "
		if i == m.backupCursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, filepath.Base(path))
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("up/down choose · enter continue · esc close"))
	return b.String()
}

func (m Model) builder() string {
	var b strings.Builder
	b.WriteString("Prompt\n")
	b.WriteString(mutedStyle.Render("Identity and layout"))
	b.WriteString("\n\n")
	if m.promptEditingName {
		b.WriteString(m.promptName.View())
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("enter keep · esc cancel"))
		return b.String()
	}
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
	fmt.Fprintf(&b, "[u] Display name  %s\n", accentStyle.Render(displayName))
	fmt.Fprintf(&b, "[l] Layout       %s\n", layout)
	fmt.Fprintf(&b, "[o] Symbol       %s\n", m.cfg.Prompt.Symbol)
	fmt.Fprintf(&b, "[i] Icons        %s\n", iconMode)
	b.WriteString("\nSegments\n")
	if len(m.cfg.Prompt.Order) == 0 {
		b.WriteString("No segments configured.\n")
	} else {
		for i, name := range m.cfg.Prompt.Order {
			seg := m.cfg.Prompt.Segments[name]
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			state := "[ ]"
			if seg.Enabled {
				state = "[x]"
			}
			icon := promptSegmentIcon(m.cfg.Prompt, seg)
			if icon != "" {
				icon += " "
			}
			fmt.Fprintf(&b, "%s%s %s%s\n", prefix, state, icon, segmentLabel(name))
		}
	}
	if m.promptAdvanced && len(m.cfg.Prompt.Order) > 0 {
		name := m.cfg.Prompt.Order[m.cursor]
		segment := m.cfg.Prompt.Segments[name]
		b.WriteString("\nAdvanced segment details\n")
		fmt.Fprintf(&b, "Color %s · Bold %t · Compatible %q · Nerd %q\n", segment.FG, segment.Bold, segment.CompatibleIcon, segment.NerdIcon)
	}
	b.WriteString("\n")
	b.WriteString(prompt.Simulated(m.previewConfig()))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("space toggle · J/K reorder · v advanced"))
	return b.String()
}

func (m Model) preview() string {
	m.syncPreviewInputs()
	var b strings.Builder
	b.WriteString("Preview\n")
	b.WriteString(mutedStyle.Render("Scenarios · [/] switch"))
	b.WriteString("\n\n")
	for i, label := range previewScenarioLabels() {
		if i > 0 {
			if i == 3 {
				b.WriteString("\n")
			} else {
				b.WriteString(" · ")
			}
		}
		if i == m.previewScenario && !m.previewCustom {
			b.WriteString(accentStyle.Render("[" + label + "]"))
		} else {
			b.WriteString(label)
		}
	}
	if m.previewCustom {
		b.WriteString("\n")
		b.WriteString(accentStyle.Render("[Custom context]"))
	}
	b.WriteString("\n\nContext\n\n")
	for i := range m.inputs {
		prefix := "  "
		if i == m.inputFocus {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(m.inputs[i].View())
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(prompt.SimulatedWithContext(m.previewConfig(), m.previewCtx))
	if m.previewError != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.previewError))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("[/] scenario · up/down field · type to customize"))
	return b.String()
}

func (m Model) apply() string {
	if m.busy {
		return "applying configuration…"
	}
	if m.confirmApply {
		cfg := m.reviewedConfig
		if cfg == nil {
			return "Review & Apply\n\nReview expired. Press n/esc, then open Review & Apply again."
		}
		active, missing := reviewedPluginCounts(cfg)
		var b strings.Builder
		b.WriteString("Review & Apply\n\n")
		fmt.Fprintf(&b, "Theme        %s\n", cfg.Theme.Name)
		fmt.Fprintf(&b, "Display name %s\n", displayNameLabel(cfg.Prompt.DisplayName))
		fmt.Fprintf(&b, "Layout       %s\n", cfg.Prompt.Layout)
		fmt.Fprintf(&b, "Icons        %s\n", cfg.Prompt.IconMode)
		fmt.Fprintf(&b, "Plugins      %d ready · %d need setup\n", active, missing)
		if missing > 0 {
			b.WriteString(errorStyle.Render("Warning: selected plugins that need setup will not load."))
			b.WriteString("\n")
		}
		b.WriteString("\nFiles\n")
		fmt.Fprintf(&b, "• %s\n• %s\n• %s\n", config.Path(), shell.OmegaZshPath(), shell.ZshrcPath())
		b.WriteString("\n[t] Technical details")
		if m.showApplyTechnical {
			b.WriteString("\n\nPlanned .zshrc diff\n\n")
			b.WriteString(m.applyDiff)
		}
		b.WriteString("\n\nPress y to apply this exact snapshot or n/esc to cancel.")
		return b.String()
	}
	before, after, err := shell.PreviewInjectBlock()
	if err != nil {
		return "preview error: " + err.Error()
	}
	return "Review & Apply\n\nPlanned .zshrc diff\n\n" + shell.DiffLines(before, after) + "\n\nThis also updates config.toml and omega.zsh.\nPress a to review and confirm."
}

func displayNameLabel(value string) string {
	if value == "" {
		return "system user"
	}
	return value
}

func reviewedPluginCounts(cfg *config.Config) (active, missing int) {
	for _, definition := range plugins.Catalog() {
		status := plugins.StatusFor(cfg, definition)
		if !status.Selected {
			continue
		}
		if status.Active {
			active++
		} else {
			missing++
		}
	}
	return active, missing
}

func (m Model) doctor() string {
	if m.confirmDoctor {
		return "Run automatic repairs?\n\nDoctor may create config.toml and .zshrc when they are missing.\nIt will not inject the managed block without Review & Apply.\n\nPress y to repair or n/esc to go back."
	}
	var b strings.Builder
	writeCheck(&b, shell.HasZsh(), "zsh installed")
	writeCheck(&b, shell.ConfigExists(), "config.toml exists")
	writeCheck(&b, shell.ZshrcExists(), ".zshrc exists")
	writeCheck(&b, shell.HasBlock(), "ozsh block present")
	writeCheck(&b, shell.ZshIsDefaultShell(), "zsh is default shell")
	b.WriteString("\n\npress enter to fix auto-resolvable issues")
	return b.String()
}

func (m *Model) fixDoctor() string {
	if !shell.ConfigExists() {
		if err := config.Save(m.cfg); err != nil {
			return "config fix error: " + err.Error()
		}
	}
	if !shell.ZshrcExists() {
		if err := os.MkdirAll(filepath.Dir(shell.ZshrcPath()), 0o700); err != nil {
			return "zshrc fix error: " + err.Error()
		}
		if err := os.WriteFile(shell.ZshrcPath(), []byte{}, 0o600); err != nil {
			return "zshrc fix error: " + err.Error()
		}
	}
	if !shell.HasBlock() {
		return "open Review & Apply and confirm the .zshrc change"
	}
	return "no auto-fixes needed"
}
