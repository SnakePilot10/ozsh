package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/fonts"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

func (m Model) themes() string {
	presets := themecatalog.List()
	var b strings.Builder
	b.WriteString("Theme gallery\n")
	fmt.Fprintf(&b, "%s\n", mutedStyle.Render(fmt.Sprintf("%d families · %d selectable presets", len(themecatalog.Families()), len(presets))))
	b.WriteString("Circuit variants  ")
	variantNames := make([]string, 0, len(themecatalog.Variants("circuit")))
	for _, variant := range themecatalog.Variants("circuit") {
		if preset, ok := themecatalog.Get("circuit", variant); ok {
			variantNames = append(variantNames, preset.Name)
		}
	}
	b.WriteString(mutedStyle.Render(strings.Join(variantNames, " · ")))
	b.WriteString("\n\n")
	if len(presets) == 0 {
		return "Theme gallery\n\nNo presets available."
	}
	start, end := m.visibleThemeRange(len(presets))
	if start > 0 {
		b.WriteString(mutedStyle.Render("  ↑ more"))
		b.WriteByte('\n')
	}
	for i := start; i < end; i++ {
		preset := presets[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		applied := " "
		if m.cfg.Theme.ID == preset.ID && m.cfg.Theme.Variant == preset.Variant {
			applied = "✓"
		}
		fmt.Fprintf(&b, "%s%s %-15s %s\n", prefix, applied, preset.Name, preset.Description)
	}
	if end < len(presets) {
		b.WriteString(mutedStyle.Render("  ↓ more"))
		b.WriteByte('\n')
	}

	selected, ok := m.selectedTheme()
	if !ok {
		return b.String()
	}
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(selected.Name))
	if selected.Variant != "" {
		fmt.Fprintf(&b, "  %s", mutedStyle.Render("variant: "+selected.Variant))
	}
	b.WriteString("\n")
	b.WriteString(selected.Description)
	b.WriteString("\n")
	b.WriteString(themeSwatches(selected))
	b.WriteString("\n")
	b.WriteString(prompt.Simulated(themecatalog.Apply(m.cfg, selected)))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("up/down choose · enter apply · [/] jump Circuit variants · c save custom"))
	return b.String()
}

func (m Model) visibleThemeRange(total int) (int, int) {
	if total <= 0 || m.height == 0 {
		return 0, total
	}
	rows := m.height - 18
	if rows < 4 {
		rows = 4
	}
	if rows > 8 {
		rows = 8
	}
	start := m.cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > total {
		start = total - rows
		if start < 0 {
			start = 0
		}
	}
	end := start + rows
	if end > total {
		end = total
	}
	return start, end
}

func themeSwatches(preset themecatalog.Preset) string {
	colors := []string{preset.Theme.Accent, preset.Theme.Success, preset.Theme.Warning, preset.Theme.Error}
	items := make([]string, 0, len(colors))
	for _, color := range colors {
		items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●"))
	}
	return strings.Join(items, " ") + "  " + mutedStyle.Render(strings.Join(colors, " · "))
}

func (m Model) plugins() string {
	var b strings.Builder
	b.WriteString("Plugins\n")
	b.WriteString(mutedStyle.Render("Recommended setup · selected by default"))
	b.WriteString("\n\n")
	for i, definition := range plugins.Catalog() {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		status := plugins.StatusFor(m.cfg, definition)
		check := "[ ]"
		if status.Selected {
			check = "[x]"
		}
		state := "Not installed"
		switch {
		case status.Active:
			state = "Active"
		case status.Installed && !status.Healthy:
			state = "Needs attention"
		case status.Installed && !status.Trusted:
			state = "Installed · not trusted"
		case status.Installed:
			state = "Installed · disabled"
		}
		fmt.Fprintf(&b, "%s%s %-20s %s\n", prefix, check, definition.Name, state)
		fmt.Fprintf(&b, "      %s\n", mutedStyle.Render(definition.Description))
	}
	missing := m.missingSelectedPlugins()
	b.WriteString("\n")
	if missing > 0 {
		fmt.Fprintf(&b, "[i] Install %d selected plugins\n", missing)
	} else {
		b.WriteString(accentStyle.Render("✓ Selected plugins are installed"))
		b.WriteByte('\n')
	}
	b.WriteString("[space] Toggle selection\n")
	b.WriteString("[x] Advanced custom plugins")
	if m.pluginAdvanced {
		b.WriteString("\n\nAdvanced\n")
		for displayIndex, itemIndex := range m.customPluginIndices() {
			item := m.cfg.Plugins.Items[itemIndex]
			prefix := "  "
			if m.cursor == len(plugins.Catalog())+displayIndex {
				prefix = "> "
			}
			fmt.Fprintf(&b, "%s%s · enabled=%t · trusted=%t\n", prefix, item.Name, item.Enabled, item.Trusted)
		}
		b.WriteString("\nAdd from repository\n")
		b.WriteString(m.pluginURL.View())
		b.WriteByte('\n')
		b.WriteString(m.pluginLoad.View())
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter/space toggle · i install · x advanced"))
	return b.String()
}

func (m Model) pluginInstallConfirmation() string {
	downloads := 0
	for _, definition := range plugins.Catalog() {
		status := plugins.StatusFor(m.cfg, definition)
		if status.Selected && !status.Installed {
			downloads++
		}
	}
	detail := "Existing managed checkouts will be validated and activated."
	if downloads > 0 {
		detail = fmt.Sprintf("%d curated repositories will be downloaded and validated.", downloads)
	}
	return fmt.Sprintf("Install selected plugins?\n\n%s\nThey will be enabled in the safe load order.\n\nPress y to install or n/esc to cancel.", detail)
}

func (m *Model) toggleSegment() {
	if m.tab != tabPrompt || m.cursor < 0 || m.cursor >= len(m.cfg.Prompt.Order) {
		return
	}
	name := m.cfg.Prompt.Order[m.cursor]
	seg := m.cfg.Prompt.Segments[name]
	seg.Enabled = !seg.Enabled
	m.cfg.Prompt.Segments[name] = seg
}

func (m *Model) moveSegment(delta int) {
	if m.tab != tabPrompt {
		return
	}
	next := m.cursor + delta
	if m.cursor < 0 || next < 0 || next >= len(m.cfg.Prompt.Order) {
		return
	}
	m.cfg.Prompt.Order[m.cursor], m.cfg.Prompt.Order[next] = m.cfg.Prompt.Order[next], m.cfg.Prompt.Order[m.cursor]
	m.cursor = next
}

func (m Model) updatePreviewInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "[":
			m.setPreviewScenario(m.previewScenario - 1)
			return m, nil
		case "]":
			m.setPreviewScenario(m.previewScenario + 1)
			return m, nil
		case "down":
			m.inputFocus = wrapIndex(m.inputFocus+1, len(m.inputs))
			m.cursor = m.inputFocus
			m.focusPreviewInput()
			return m, nil
		case "up":
			m.inputFocus = wrapIndex(m.inputFocus-1, len(m.inputs))
			m.cursor = m.inputFocus
			m.focusPreviewInput()
			return m, nil
		}
	}
	if len(m.inputs) == 0 {
		return m, nil
	}
	m.focusPreviewInput()
	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	m.syncPreviewInputs()
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyRunes {
		m.previewCustom = true
	}
	return m, cmd
}

func previewScenarioIDs() []string {
	return []string{"clean", "git-dirty", "command-failed", "dev-project", "low-battery"}
}

func previewScenarioLabels() []string {
	return []string{"Clean", "Git dirty", "Command failed", "Dev project", "Low battery"}
}

func (m *Model) setPreviewScenario(index int) {
	ids := previewScenarioIDs()
	if len(ids) == 0 {
		return
	}
	m.previewScenario = wrapIndex(index, len(ids))
	ctx, ok := prompt.PreviewScenario(ids[m.previewScenario])
	if !ok {
		return
	}
	m.previewCtx = ctx
	m.inputs = previewInputs(ctx)
	m.inputFocus = 0
	m.cursor = 0
	m.previewCustom = false
	m.previewError = ""
	m.focusPreviewInput()
}

func (m Model) updatePromptName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := sanitizeInput(m.promptName.Value())
		if strings.ContainsAny(value, "\x00\n\r") {
			m.msg = "display name contains unsupported characters"
			return m, nil
		}
		m.cfg.Prompt.DisplayName = value
		m.promptEditingName = false
		m.promptName.Blur()
		m.msg = "display name updated"
		return m, nil
	case "esc":
		m.promptName.SetValue(m.cfg.Prompt.DisplayName)
		m.promptEditingName = false
		m.promptName.Blur()
		m.msg = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.promptName, cmd = m.promptName.Update(msg)
	return m, cmd
}

func (m Model) updateFontDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	manifest := fonts.Manifest()
	if len(manifest) == 0 {
		m.fontOpen = false
		return m, nil
	}
	if m.confirmFontRestore {
		switch msg.String() {
		case "y", "enter":
			m.confirmFontRestore = false
			m.fontOpen = false
			m.busy = true
			m.operation = "font-restore"
			m.msg = ""
			return m, doRestoreTermuxFont()
		case "n", "esc":
			m.confirmFontRestore = false
		}
		return m, nil
	}
	if m.confirmFont {
		switch msg.String() {
		case "y", "enter":
			selected := manifest[wrapIndex(m.fontCursor, len(manifest))]
			m.confirmFont = false
			m.fontOpen = false
			m.busy = true
			m.operation = "font"
			m.msg = ""
			return m, doInstallFont(m.cfg, selected)
		case "n", "esc":
			m.confirmFont = false
			return m, nil
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.fontCursor = wrapIndex(m.fontCursor-1, len(manifest))
	case "down", "j":
		m.fontCursor = wrapIndex(m.fontCursor+1, len(manifest))
	case "enter":
		m.confirmFont = true
	case "r":
		if shell.IsTermux() {
			m.confirmFontRestore = true
		}
	case "esc", "q":
		m.fontOpen = false
	}
	return m, nil
}

func (m Model) updateBackupDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.backupPaths) == 0 {
		m.backupOpen = false
		return m, nil
	}
	if m.confirmBackup {
		switch msg.String() {
		case "y", "enter":
			path := m.backupPaths[wrapIndex(m.backupCursor, len(m.backupPaths))]
			m.confirmBackup = false
			m.backupOpen = false
			m.busy = true
			m.operation = "backup"
			m.msg = ""
			return m, doRestoreBackup(path)
		case "n", "esc":
			m.confirmBackup = false
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.backupCursor = wrapIndex(m.backupCursor-1, len(m.backupPaths))
	case "down", "j":
		m.backupCursor = wrapIndex(m.backupCursor+1, len(m.backupPaths))
	case "enter":
		m.confirmBackup = true
	case "esc", "q":
		m.backupOpen = false
	}
	return m, nil
}

func (m *Model) cyclePromptSymbol() {
	symbols := []string{"❯", ">", "$", "λ", "›"}
	index := 0
	for i, symbol := range symbols {
		if symbol == m.cfg.Prompt.Symbol {
			index = i
			break
		}
	}
	m.cfg.Prompt.Symbol = symbols[(index+1)%len(symbols)]
}
