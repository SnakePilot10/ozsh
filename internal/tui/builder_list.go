package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/snakepilot10/ozsh/internal/config"
)

func newBuilderList(cfg *config.Config, uiStyles styles) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.NormalDesc = uiStyles.muted
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(uiStyles.accent.GetForeground()).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(uiStyles.accent.GetForeground())
	delegate.Styles.FilterMatch = uiStyles.warning

	model := list.New(segmentItems(cfg), delegate, 80, 8)
	model.SetShowTitle(false)
	model.SetShowHelp(false)
	model.SetShowStatusBar(true)
	model.SetShowPagination(true)
	model.SetStatusBarItemName("segment", "segments")
	model.DisableQuitKeybindings()
	styleList(&model, uiStyles)
	return model
}

func styleList(model *list.Model, uiStyles styles) {
	model.Styles.FilterPrompt = uiStyles.accent
	model.Styles.FilterCursor = uiStyles.accent
	model.Styles.DefaultFilterCharacterMatch = uiStyles.warning
	model.Styles.StatusBar = uiStyles.muted
	model.Styles.StatusEmpty = uiStyles.warning
	model.Styles.StatusBarActiveFilter = uiStyles.accent
	model.Styles.StatusBarFilterCount = uiStyles.muted
	model.Styles.NoItems = uiStyles.warning
	model.Styles.PaginationStyle = uiStyles.muted
	model.Styles.ActivePaginationDot = uiStyles.accent
	model.Styles.InactivePaginationDot = uiStyles.muted
	model.FilterInput.PromptStyle = uiStyles.accent
	model.FilterInput.PlaceholderStyle = uiStyles.muted
	model.FilterInput.Cursor.Style = uiStyles.accent

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.NormalDesc = uiStyles.muted
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(uiStyles.accent.GetForeground()).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(uiStyles.accent.GetForeground())
	delegate.Styles.FilterMatch = uiStyles.warning
	model.SetDelegate(delegate)
}

func segmentItems(cfg *config.Config) []list.Item {
	items := make([]list.Item, 0, len(cfg.Prompt.Order))
	for _, name := range cfg.Prompt.Order {
		segment := cfg.Prompt.Segments[name]
		items = append(items, segmentItem{
			name:    name,
			enabled: segment.Enabled,
			fg:      segment.FG,
			bg:      segment.BG,
			bold:    segment.Bold,
			icon:    segment.Icon,
		})
	}
	return items
}

func (m *Model) updateBuilderList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.builderList.SettingFilter() {
		cmd = updateListFilter(&m.builderList, msg)
	} else {
		m.builderList, cmd = m.builderList.Update(msg)
	}
	m.syncBuilderCursor()
	return *m, cmd
}

func updateListFilter(model *list.Model, msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, model.KeyMap.CancelWhileFiltering):
			model.ResetFilter()
			return nil
		case key.Matches(keyMsg, model.KeyMap.AcceptWhileFiltering):
			if model.FilterInput.Value() == "" {
				model.ResetFilter()
			} else {
				model.SetFilterState(list.FilterApplied)
			}
			return nil
		}
	}
	var cmd tea.Cmd
	model.FilterInput, cmd = model.FilterInput.Update(msg)
	model.SetFilterText(model.FilterInput.Value())
	model.SetFilterState(list.Filtering)
	return cmd
}

func (m Model) builderListShouldHandle(msg tea.KeyMsg) bool {
	if m.builderList.SettingFilter() {
		return true
	}
	switch msg.String() {
	case "up", "down", "j", "k", "pgup", "pgdown", "home", "end", "/":
		return true
	case "esc":
		return m.builderList.IsFiltered()
	default:
		return false
	}
}

func (m *Model) syncBuilderCursor() {
	name := m.selectedBuilderName()
	for i, candidate := range m.cfg.Prompt.Order {
		if candidate == name {
			m.cursor = i
			return
		}
	}
}

func (m Model) selectedBuilderName() string {
	item, ok := m.builderList.SelectedItem().(segmentItem)
	if !ok {
		return ""
	}
	return item.name
}

func (m Model) builderIndex() int {
	name := m.selectedBuilderName()
	for i, candidate := range m.cfg.Prompt.Order {
		if candidate == name {
			return i
		}
	}
	return -1
}

func (m *Model) syncBuilderList(selectedName string) {
	filter := m.builderList.FilterValue()
	m.builderList.SetItems(segmentItems(m.cfg))
	if filter != "" {
		m.builderList.SetFilterText(filter)
	}
	visible := m.builderList.VisibleItems()
	for i, item := range visible {
		segment, ok := item.(segmentItem)
		if ok && segment.name == selectedName {
			m.builderList.Select(i)
			m.syncBuilderCursor()
			return
		}
	}
	m.builderList.GoToStart()
	m.syncBuilderCursor()
}
