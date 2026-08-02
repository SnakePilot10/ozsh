package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
)

type themeItem struct {
	name   string
	preset config.ThemeConfig
	active bool
}

func (i themeItem) FilterValue() string { return i.name }
func (i themeItem) Title() string {
	if i.active {
		return "[active] " + i.name
	}
	return i.name
}
func (i themeItem) Description() string {
	return fmt.Sprintf("accent=%s success=%s error=%s", i.preset.Accent, i.preset.Success, i.preset.Error)
}

type pluginItem struct {
	item config.PluginItem
}

func (i pluginItem) FilterValue() string {
	return i.item.Name + " " + i.item.Source + " " + i.item.Load
}
func (i pluginItem) Title() string {
	state := "[ ]"
	if i.item.Enabled {
		state = "[x]"
	}
	return state + " " + i.item.Name
}
func (i pluginItem) Description() string {
	trust := "untrusted"
	if i.item.Trusted {
		trust = "trusted"
	}
	return fmt.Sprintf("%s | load=%s", trust, i.item.Load)
}

func newCatalogList(items []list.Item, singular, plural string, uiStyles styles) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	model := list.New(items, delegate, 80, 8)
	model.SetShowTitle(false)
	model.SetShowHelp(false)
	model.SetShowStatusBar(true)
	model.SetShowPagination(true)
	model.SetStatusBarItemName(singular, plural)
	model.DisableQuitKeybindings()
	styleList(&model, uiStyles)
	return model
}

func themeItems(cfg *config.Config) []list.Item {
	names := sortedThemeNames()
	items := make([]list.Item, 0, len(names))
	for _, name := range names {
		preset := config.Presets[name]
		items = append(items, themeItem{name: name, preset: preset, active: cfg.Theme.Name == name})
	}
	if cfg.Theme.Name == "custom" {
		items = append(items, themeItem{name: "custom", preset: cfg.Theme, active: true})
	}
	return items
}

func pluginItems(cfg *config.Config) []list.Item {
	items := make([]list.Item, 0, len(cfg.Plugins.Items))
	for _, item := range cfg.Plugins.Items {
		items = append(items, pluginItem{item: item})
	}
	return items
}

func (m Model) catalogListShouldHandle(model list.Model, msg tea.KeyMsg) bool {
	if model.SettingFilter() {
		return true
	}
	switch msg.String() {
	case "up", "down", "j", "k", "pgup", "pgdown", "home", "end", "/":
		return true
	case "esc":
		return model.IsFiltered()
	default:
		return false
	}
}

func (m Model) activeListSettingFilter() bool {
	switch m.tab {
	case tabBuilder:
		return m.builderList.SettingFilter()
	case tabThemes:
		return m.themesList.SettingFilter()
	case tabPlugins:
		return m.pluginsList.SettingFilter()
	default:
		return false
	}
}

func (m *Model) updateThemesList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.themesList.SettingFilter() {
		cmd = updateListFilter(&m.themesList, msg)
	} else {
		m.themesList, cmd = m.themesList.Update(msg)
	}
	m.syncThemeCursor()
	return *m, cmd
}

func (m *Model) updatePluginsList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.pluginsList.SettingFilter() {
		cmd = updateListFilter(&m.pluginsList, msg)
	} else {
		m.pluginsList, cmd = m.pluginsList.Update(msg)
	}
	m.syncPluginCursor()
	return *m, cmd
}

func (m Model) selectedThemeName() string {
	item, ok := m.themesList.SelectedItem().(themeItem)
	if !ok {
		return ""
	}
	return item.name
}

func (m Model) selectedThemePreset() (config.ThemeConfig, bool) {
	item, ok := m.themesList.SelectedItem().(themeItem)
	return item.preset, ok
}

func (m Model) selectedPluginName() string {
	item, ok := m.pluginsList.SelectedItem().(pluginItem)
	if !ok {
		return ""
	}
	return item.item.Name
}

func (m Model) pluginIndex() int {
	name := m.selectedPluginName()
	for i, item := range m.cfg.Plugins.Items {
		if item.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) syncThemeCursor() {
	name := m.selectedThemeName()
	for i, candidate := range sortedThemeNames() {
		if candidate == name {
			m.cursor = i
			return
		}
	}
}

func (m *Model) syncPluginCursor() {
	if index := m.pluginIndex(); index >= 0 {
		m.cursor = index
	}
}

func (m *Model) syncThemesList(selectedName string) {
	filter := m.themesList.FilterValue()
	m.themesList.SetItems(themeItems(m.cfg))
	if filter != "" {
		m.themesList.SetFilterText(filter)
	}
	selectVisibleItem(&m.themesList, selectedName)
	m.syncThemeCursor()
}

func (m *Model) syncPluginsList(selectedName string) {
	filter := m.pluginsList.FilterValue()
	m.pluginsList.SetItems(pluginItems(m.cfg))
	if filter != "" {
		m.pluginsList.SetFilterText(filter)
	}
	selectVisibleItem(&m.pluginsList, selectedName)
	m.syncPluginCursor()
}

func selectVisibleItem(model *list.Model, name string) {
	for i, raw := range model.VisibleItems() {
		switch item := raw.(type) {
		case themeItem:
			if item.name == name {
				model.Select(i)
				return
			}
		case pluginItem:
			if item.item.Name == name {
				model.Select(i)
				return
			}
		}
	}
	model.GoToStart()
}
