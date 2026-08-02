package tui

import (
	"fmt"

	"github.com/snakepilot10/ozsh/internal/config"
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
	catalog := plugins.Catalog()
	if m.cursor >= 0 && m.cursor < len(catalog) {
		id := catalog[m.cursor].ID
		selected := !containsString(m.cfg.Plugins.Selected, id)
		if selected {
			m.cfg.Plugins.Selected = append(m.cfg.Plugins.Selected, id)
		} else {
			m.cfg.Plugins.Selected = removeString(m.cfg.Plugins.Selected, id)
		}
		for i := range m.cfg.Plugins.Items {
			if m.cfg.Plugins.Items[i].Name == id {
				m.cfg.Plugins.Items[i].Enabled = selected
			}
		}
		m.msg = "plugin selection updated"
		return
	}
	itemIndex, ok := m.customPluginIndexAtCursor()
	if !ok {
		return
	}
	original := cloneConfig(m.cfg)
	m.cfg.Plugins.Items[itemIndex].Enabled = !m.cfg.Plugins.Items[itemIndex].Enabled
	if err := config.Save(m.cfg); err != nil {
		m.cfg = original
		m.msg = "plugin save error: " + err.Error()
		return
	}
	m.msg = "plugin state saved"
}

func (m *Model) trustPluginAtCursor() {
	itemIndex, ok := m.customPluginIndexAtCursor()
	if !ok {
		return
	}
	name := m.cfg.Plugins.Items[itemIndex].Name
	original := cloneConfig(m.cfg)
	if err := plugins.SetTrusted(m.cfg, name, true); err != nil {
		m.msg = "plugin trust error: " + err.Error()
		return
	}
	if err := config.Save(m.cfg); err != nil {
		m.cfg = original
		m.msg = "plugin trust save error: " + err.Error()
		return
	}
	m.msg = "plugin trusted and saved: " + name
}

func (m *Model) untrustPluginAtCursor() {
	itemIndex, ok := m.customPluginIndexAtCursor()
	if !ok {
		return
	}
	name := m.cfg.Plugins.Items[itemIndex].Name
	original := cloneConfig(m.cfg)
	if err := plugins.SetTrusted(m.cfg, name, false); err != nil {
		m.msg = "plugin untrust error: " + err.Error()
		return
	}
	if err := config.Save(m.cfg); err != nil {
		m.cfg = original
		m.msg = "plugin untrust save error: " + err.Error()
		return
	}
	m.msg = "plugin untrusted and saved: " + name
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
	name, err := plugins.AddAndSave(m.cfg, url, load)
	if err != nil {
		m.msg = "plugin add error: " + err.Error()
		return
	}
	m.pluginURL.SetValue("")
	m.pluginLoad.SetValue("")
	m.pluginFocus = 0
	m.focusPluginInput()
	m.cursor = len(plugins.Catalog()) + len(m.customPluginIndices()) - 1
	m.msg = "plugin added: " + name
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

func (m Model) customPluginIndexAtCursor() (int, bool) {
	customCursor := m.cursor - len(plugins.Catalog())
	indices := m.customPluginIndices()
	if customCursor < 0 || customCursor >= len(indices) {
		return 0, false
	}
	return indices[customCursor], true
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
	presets := themecatalog.List()
	names := make([]string, len(presets))
	for i, preset := range presets {
		names[i] = preset.ID
	}
	return names
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
