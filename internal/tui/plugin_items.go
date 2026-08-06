package tui

import "github.com/snakepilot10/ozsh/internal/plugins"

type pluginItemKind uint8

const (
	pluginItemRecommended pluginItemKind = iota
	pluginItemCustom
)

type pluginListItem struct {
	Kind        pluginItemKind
	Definition  plugins.Definition
	ConfigIndex int
	Pending     bool
}

// pluginListItems keeps curated definitions first and appends only config
// entries that are not represented by the curated catalog.
func (m Model) pluginListItems() []pluginListItem {
	catalog := plugins.Catalog()
	items := make([]pluginListItem, 0, len(catalog)+len(m.cfg.Plugins.Items))
	for _, definition := range catalog {
		items = append(items, pluginListItem{
			Kind:        pluginItemRecommended,
			Definition:  definition,
			ConfigIndex: m.pluginConfigIndex(definition.ID),
		})
	}
	for index, item := range m.cfg.Plugins.Items {
		if _, curated := plugins.FindDefinition(item.Name); curated {
			continue
		}
		_, pending := m.pluginChanges.RepositoryURLFor(item.Name)
		items = append(items, pluginListItem{
			Kind: pluginItemCustom,
			Definition: plugins.Definition{
				ID:          item.Name,
				Name:        item.Name,
				Description: "Custom plugin managed by ozsh.",
				Load:        item.Load,
			},
			ConfigIndex: index,
			Pending:     pending,
		})
	}
	return items
}

func (m Model) selectedPluginListItem() (pluginListItem, bool) {
	items := m.pluginListItems()
	if len(items) == 0 {
		return pluginListItem{}, false
	}
	index := m.cursor
	if index < 0 {
		index = 0
	}
	if index >= len(items) {
		index = len(items) - 1
	}
	return items[index], true
}

func (m Model) pluginListItemByName(name string) (pluginListItem, bool) {
	for _, item := range m.pluginListItems() {
		if item.Kind == pluginItemCustom && item.Definition.ID == name {
			return item, true
		}
	}
	return pluginListItem{}, false
}

func (m Model) pluginConfigIndex(name string) int {
	for index, item := range m.cfg.Plugins.Items {
		if item.Name == name {
			return index
		}
	}
	return -1
}
