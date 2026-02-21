package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Launch   key.Binding
	Mailpit  key.Binding
	XHGui    key.Binding
	Refresh  key.Binding
	Filter   key.Binding
	Help     key.Binding
	Quit     key.Binding
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Detail   key.Binding
	Back     key.Binding
	Logs     key.Binding
	SSH      key.Binding
	StartAll key.Binding
	StopAll  key.Binding
	Confirm  key.Binding
	Xdebug   key.Binding
	Poweroff key.Binding
	CopyURL  key.Binding
	Config   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "stop"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		Launch: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "launch"),
		),
		Mailpit: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mailpit"),
		),
		XHGui: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "xhgui"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Detail: key.NewBinding(
			key.WithKeys("enter", "d"),
			key.WithHelp("enter/d", "detail"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Logs: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "logs"),
		),
		SSH: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "ssh"),
		),
		StartAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "start all"),
		),
		StopAll: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "stop all"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Xdebug: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "xdebug toggle"),
		),
		Poweroff: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "poweroff"),
		),
		CopyURL: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy url"),
		),
		Config: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "config"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
	}
}
