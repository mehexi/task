package internal

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Quit          key.Binding
	Tab           key.Binding
	Focus1        key.Binding
	Focus2        key.Binding
	Focus3        key.Binding
	Enter         key.Binding
	Space         key.Binding
	New           key.Binding
	Subtask       key.Binding
	Delete        key.Binding
	EditNotes     key.Binding
	SetDueDate    key.Binding
	CyclePriority key.Binding
	Filter        key.Binding
	Rescan        key.Binding
	Escape        key.Binding
	Confirm       key.Binding
	Deny          key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.New, k.Delete}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.Enter},
		{k.New, k.Delete, k.Space, k.CyclePriority},
		{k.Filter, k.EditNotes, k.SetDueDate},
		{k.Quit},
	}
}

var Keys = KeyMap{
	Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Tab:           key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
	Focus1:        key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "projects")),
	Focus2:        key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "tasks")),
	Focus3:        key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "detail")),
	Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Space:         key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle done")),
	New:           key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Subtask:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "subtask")),
	Delete:        key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	EditNotes:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit notes")),
	SetDueDate:    key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "due date")),
	CyclePriority: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "priority")),
	Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Rescan:        key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rescan code")),
	Escape:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Confirm:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
	Deny:          key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "no")),
}
