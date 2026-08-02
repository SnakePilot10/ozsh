package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	NextTab       key.Binding
	PrevTab       key.Binding
	NextTabForm   key.Binding
	PrevTabForm   key.Binding
	Up            key.Binding
	Down          key.Binding
	Toggle        key.Binding
	ReorderDown   key.Binding
	ReorderUp     key.Binding
	Heavy         key.Binding
	EditSegment   key.Binding
	ApplyEdit     key.Binding
	Discard       key.Binding
	Filter        key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	BodyUp        key.Binding
	BodyDown      key.Binding
	Save          key.Binding
	Apply         key.Binding
	Confirm       key.Binding
	Cancel        key.Binding
	AddPlugin     key.Binding
	TrustPlugin   key.Binding
	UntrustPlugin key.Binding
	NextField     key.Binding
	PrevField     key.Binding
	Submit        key.Binding
	CloseForm     key.Binding
	Select        key.Binding
	Fix           key.Binding
	ApplyTheme    key.Binding
	TogglePlugin  key.Binding
	CustomTheme   key.Binding
	Refresh       key.Binding
	Help          key.Binding
	HelpForm      key.Binding
	Quit          key.Binding
	ForceQuit     key.Binding
}

var keys = keyMap{
	NextTab:       key.NewBinding(key.WithKeys("tab", "right"), key.WithHelp("tab/right", "next tab")),
	PrevTab:       key.NewBinding(key.WithKeys("shift+tab", "left"), key.WithHelp("shift+tab/left", "previous tab")),
	NextTabForm:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	PrevTabForm:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous tab")),
	Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
	Toggle:        key.NewBinding(key.WithKeys(" ", "enter"), key.WithHelp("space/enter", "toggle")),
	ReorderDown:   key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "move down")),
	ReorderUp:     key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "move up")),
	Heavy:         key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "toggle heavy")),
	EditSegment:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit segment")),
	ApplyEdit:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "apply edit")),
	Discard:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "discard changes")),
	Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	PageUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
	PageDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "next page")),
	BodyUp:        key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "scroll up")),
	BodyDown:      key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "scroll down")),
	Save:          key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save")),
	Apply:         key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "review apply")),
	Confirm:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
	Cancel:        key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "cancel")),
	AddPlugin:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "add plugin")),
	TrustPlugin:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "trust")),
	UntrustPlugin: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "untrust")),
	NextField:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
	PrevField:     key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field")),
	Submit:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	CloseForm:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Select:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Fix:           key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "fix issues")),
	ApplyTheme:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply and save")),
	TogglePlugin:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle plugin")),
	CustomTheme:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "save colors as custom")),
	Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Help:          key.NewBinding(key.WithKeys("f1", "?"), key.WithHelp("f1/?", "more help")),
	HelpForm:      key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "more help")),
	Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	ForceQuit:     key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
}

type contextualKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (m contextualKeyMap) ShortHelp() []key.Binding  { return m.short }
func (m contextualKeyMap) FullHelp() [][]key.Binding { return m.full }
