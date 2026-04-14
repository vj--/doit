package tui

import "github.com/charmbracelet/bubbles/key"

// ShortHelp / FullHelp let keymap satisfy bubbles/help.KeyMap so the help
// bubble can render the same bindings we react to in Update.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Up, k.Down, k.New, k.Edit, k.Delete, k.Help, k.Quit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.MoveLeft, k.MoveRight, k.MoveUp, k.MoveDown},
		{k.New, k.Edit, k.Delete, k.Filter},
		{k.ToggleHidden, k.Help, k.Quit},
	}
}

type keymap struct {
	Left, Right      key.Binding
	Up, Down         key.Binding
	MoveLeft, MoveRight key.Binding
	MoveUp, MoveDown    key.Binding
	New, Edit, Delete   key.Binding
	Filter, Help        key.Binding
	ToggleHidden        key.Binding
	Confirm, Cancel     key.Binding
	Quit                key.Binding
}

func newKeymap() keymap {
	return keymap{
		Left:      key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "prev column")),
		Right:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "next column")),
		Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "prev card")),
		Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "next card")),
		MoveLeft:  key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "move card left")),
		MoveRight: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "move card right")),
		MoveUp:    key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "move card up")),
		MoveDown:  key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "move card down")),
		New:       key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new card")),
		Edit:      key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e/⏎", "edit card")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete card")),
		Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		ToggleHidden: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "show hidden done")),
		Confirm:   key.NewBinding(key.WithKeys("y", "Y")),
		Cancel:    key.NewBinding(key.WithKeys("esc", "n", "N")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
