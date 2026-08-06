package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/plugins"
	themecatalog "github.com/snakepilot10/ozsh/internal/themes"
)

func (m *Model) cycleThemeVariant(delta int) {
	presets := themecatalog.List()
	if m.cursor < 0 || m.cursor >= len(presets) || presets[m.cursor].ID != "circuit" {
		return
	}
	indices := make([]int, 0, len(themecatalog.Variants("circuit")))
	position := 0
	for index, preset := range presets {
		if preset.ID != "circuit" {
			continue
		}
		if index == m.cursor {
			position = len(indices)
		}
		indices = append(indices, index)
	}
	if len(indices) == 0 {
		return
	}
	m.cursor = indices[wrapIndex(position+delta, len(indices))]
}

func (m *Model) togglePluginAtCursor() {
	item, ok := m.selectedPluginListItem()
	if !ok {
		return
	}
	if item.Kind == pluginItemCustom {
		m.toggleCustomPlugin(item)
		return
	}

	id := item.Definition.ID
	selected := !containsString(m.cfg.Plugins.Selected, id)
	if selected {
		m.cfg.Plugins.Selected = append(m.cfg.Plugins.Selected, id)
	} else {
		m.cfg.Plugins.Selected = removeString(m.cfg.Plugins.Selected, id)
	}
	if item.ConfigIndex >= 0 && item.ConfigIndex < len(m.cfg.Plugins.Items) {
		m.cfg.Plugins.Items[item.ConfigIndex].Enabled = selected
	}
	m.msg = "plugin selection updated; Review & Apply to activate"
}

func (m *Model) toggleCustomPlugin(item pluginListItem) {
	if item.Kind != pluginItemCustom || item.ConfigIndex < 0 || item.ConfigIndex >= len(m.cfg.Plugins.Items) {
		m.msg = "custom plugin configuration is unavailable"
		return
	}
	m.cfg.Plugins.Items[item.ConfigIndex].Enabled = !m.cfg.Plugins.Items[item.ConfigIndex].Enabled
	m.msg = "plugin state updated; Review & Apply to activate pending changes"
}

func (m *Model) trustPluginAtCursor() {
	item, ok := m.selectedPluginListItem()
	if !ok || item.Kind != pluginItemCustom {
		return
	}
	if err := m.trustCustomPlugin(item, true); err != nil {
		m.msg = "plugin trust error: " + err.Error()
		return
	}
	m.msg = "plugin trusted; Review & Apply to activate pending changes"
}

func (m *Model) untrustPluginAtCursor() {
	item, ok := m.selectedPluginListItem()
	if !ok || item.Kind != pluginItemCustom {
		return
	}
	if err := m.trustCustomPlugin(item, false); err != nil {
		m.msg = "plugin untrust error: " + err.Error()
		return
	}
	m.msg = "plugin untrusted; Review & Apply to activate pending changes"
}

func (m *Model) trustCustomPlugin(item pluginListItem, trusted bool) error {
	if item.Kind != pluginItemCustom || item.ConfigIndex < 0 || item.ConfigIndex >= len(m.cfg.Plugins.Items) {
		return fmt.Errorf("custom plugin configuration is unavailable")
	}
	configured := &m.cfg.Plugins.Items[item.ConfigIndex]
	if configured.Name != item.Definition.ID {
		return fmt.Errorf("custom plugin selection changed")
	}
	if trusted {
		root := m.pluginChanges.RootFor(configured.Name, configured.Source)
		if err := plugins.ValidateCandidate(root, configured.Load); err != nil {
			return fmt.Errorf("validate load file: %w", err)
		}
	}
	configured.Trusted = trusted
	return nil
}

func (m *Model) openLoadFilePicker(item pluginListItem) error {
	if item.Kind != pluginItemCustom || item.ConfigIndex < 0 || item.ConfigIndex >= len(m.cfg.Plugins.Items) {
		return fmt.Errorf("custom plugin configuration is unavailable")
	}
	configured := m.cfg.Plugins.Items[item.ConfigIndex]
	if configured.Name != item.Definition.ID {
		return fmt.Errorf("custom plugin selection changed")
	}
	root := m.pluginChanges.RootFor(configured.Name, configured.Source)
	candidates, err := plugins.DiscoverCandidates(root, configured.Name, plugins.DefaultScanLimits)
	if err != nil {
		return fmt.Errorf("discover load files: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no supported load files found")
	}

	requestID := m.pluginWizard.RequestID
	m.pluginWizard = newPluginWizardModel()
	m.pluginWizard.RequestID = requestID
	m.pluginWizard.Mode = pluginWizardChangeLoad
	m.pluginWizard.Step = pluginWizardCandidates
	m.pluginWizard.Candidates = append([]plugins.Candidate(nil), candidates...)
	m.pluginWizard.TargetName = configured.Name
	m.pluginWizard.TargetRoot = root
	m.pluginWizard.TargetConfigIndex = item.ConfigIndex
	for index, candidate := range candidates {
		if candidate.RelativePath == configured.Load {
			m.pluginWizard.Candidate = index
			break
		}
	}
	m.msg = ""
	return nil
}

func (m *Model) queueCustomPluginRemoval(item pluginListItem) error {
	if item.Kind != pluginItemCustom || item.ConfigIndex < 0 || item.ConfigIndex >= len(m.cfg.Plugins.Items) {
		return fmt.Errorf("custom plugin configuration is unavailable")
	}
	name := m.cfg.Plugins.Items[item.ConfigIndex].Name
	if name != item.Definition.ID {
		return fmt.Errorf("custom plugin selection changed")
	}
	if err := m.pluginChanges.QueueRemove(m.cfg, name); err != nil {
		return err
	}
	m.cursor = minInt(m.cursor, len(m.pluginListItems())-1)
	m.syncCursor()
	m.msg = "plugin removal queued; Review & Apply to activate pending changes"
	return nil
}

func (m *Model) openPluginRemovalAtCursor() {
	item, ok := m.selectedPluginListItem()
	if !ok {
		return
	}
	if item.Kind == pluginItemRecommended {
		m.msg = "recommended plugins can be deselected but not removed"
		return
	}
	if item.ConfigIndex < 0 || item.ConfigIndex >= len(m.cfg.Plugins.Items) {
		m.msg = "custom plugin configuration is unavailable"
		return
	}
	m.pluginRemoveConfirm = true
	m.pluginRemoveName = m.cfg.Plugins.Items[item.ConfigIndex].Name
	m.msg = "confirm pending plugin removal"
}

func (m Model) updatePluginRemoveConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		item, ok := m.pluginListItemByName(m.pluginRemoveName)
		if !ok {
			m.msg = "plugin removal target is unavailable"
		} else if err := m.queueCustomPluginRemoval(item); err != nil {
			m.msg = "plugin removal error: " + err.Error()
		}
		m.pluginRemoveConfirm = false
		m.pluginRemoveName = ""
	case "n", "esc":
		m.pluginRemoveConfirm = false
		m.pluginRemoveName = ""
		m.msg = "plugin removal cancelled"
	}
	return m, nil
}

func (m *Model) addPluginFromInputs() {
	url := sanitizeInput(m.pluginURL.Value())
	load := sanitizeInput(m.pluginLoad.Value())
	if url == "" {
		m.msg = "plugin URL is required"
		return
	}
	if load == "" {
		m.msg = "plugin load file is required"
		return
	}
	m.openPluginWizard()
	m.pluginWizard.URL.SetValue(url)
	m.msg = "legacy form migrated; press Enter to inspect repository candidates"
}

func (m Model) customPluginIndices() []int {
	indices := make([]int, 0, len(m.cfg.Plugins.Items))
	for i, item := range m.cfg.Plugins.Items {
		if _, curated := plugins.FindDefinition(item.Name); !curated {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m Model) missingSelectedPlugins() int {
	missing := 0
	for _, definition := range plugins.Catalog() {
		status := plugins.StatusFor(m.cfg, definition)
		if status.Selected && !status.Active {
			missing++
		}
	}
	return missing
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func removeString(values []string, value string) []string {
	result := values[:0]
	for _, candidate := range values {
		if candidate != value {
			result = append(result, candidate)
		}
	}
	return result
}

func sortedThemeNames() []string {
	presets := themecatalog.Families()
	names := make([]string, len(presets))
	for i, preset := range presets {
		names[i] = preset.ID
	}
	return names
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// Kept as a small public-to-package value object for existing callers and tests.
type segmentItem struct {
	name    string
	enabled bool
	fg      string
	bold    bool
	icon    string
}

func (i segmentItem) FilterValue() string { return i.name }

func (i segmentItem) Title() string {
	check := "[ ]"
	if i.enabled {
		check = "[x]"
	}
	return fmt.Sprintf("%s %s", check, i.name)
}

func (i segmentItem) Description() string {
	weight := "normal"
	if i.bold {
		weight = "bold"
	}
	icon := i.icon
	if icon == "" {
		icon = "none"
	}
	return fmt.Sprintf("fg=%s bold=%s icon=%s", i.fg, weight, icon)
}
