package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
)

// KeyMap defines the TUI's key bindings. The detail and stream views scroll
// with the viewport's defaults; see viewportKeyMap.
type KeyMap struct {
	// Navigation. Up and Down move the selection in the project list, Home
	// and End move it or scroll the viewports, which have no keys for that.
	Detail   key.Binding
	Up       key.Binding
	Down     key.Binding
	Home     key.Binding
	End      key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Back     key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding

	// The project list's filter, and the keys that end typing it.
	Filter       key.Binding
	ClearFilter  key.Binding
	AcceptFilter key.Binding
	CancelFilter key.Binding

	// Projects. SSH to Update act only in the detail view.
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Delete   key.Binding
	Launch   key.Binding
	Mailpit  key.Binding
	XHGui    key.Binding
	OpenDir  key.Binding
	OpenVSC  key.Binding
	OpenPhpS key.Binding
	SSH      key.Binding
	Logs     key.Binding
	Xdebug   key.Binding
	CopyURL  key.Binding
	Snapshot key.Binding
	Restore  key.Binding
	Update   key.Binding
	StartAll key.Binding
	StopAll  key.Binding
	Poweroff key.Binding
	Config   key.Binding

	// Confirmations of the project keys.
	Confirm  key.Binding
	OmitSnap key.Binding
	Cancel   key.Binding
}

// DefaultKeyMap returns the default key bindings. The key tables in
// docs/content/users/usage/cli.md list them in the same groups and order, so
// change them together.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Detail:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Home:     key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "top")),
		End:      key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "bottom")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "b"), key.WithHelp("b/pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "f"), key.WithHelp("f/pgdn", "page down")),
		Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		// Up and down also end typing, to move on to the matches.
		AcceptFilter: key.NewBinding(key.WithKeys("enter", "tab", "up", "down"), key.WithHelp("enter", "apply filter")),
		CancelFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),

		Start:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
		Stop:     key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "stop")),
		Restart:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restart")),
		Delete:   key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete")),
		Launch:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "launch")),
		Mailpit:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mailpit")),
		XHGui:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "xhgui")),
		OpenDir:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open directory")),
		OpenVSC:  key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "open in VS Code")),
		OpenPhpS: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "open in PhpStorm")),
		SSH:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "ssh")),
		Logs:     key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "logs")),
		Xdebug:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "toggle xdebug")),
		CopyURL:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy url")),
		Snapshot: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "snapshot")),
		Restore:  key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "restore latest snapshot")),
		Update:   key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "update add-ons")),
		StartAll: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "start all")),
		StopAll:  key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "stop all")),
		Poweroff: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "poweroff")),
		Config:   key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "config current dir")),

		Confirm:  key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
		OmitSnap: key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "delete without snapshot")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("any other key", "cancel")),
	}
}

// viewportKeyMap returns the viewport's default keys without horizontal
// scrolling, whose l would shadow the launch key, and without space for a
// page down.
func viewportKeyMap() viewport.KeyMap {
	km := viewport.DefaultKeyMap()
	km.Left.Unbind()
	km.Right.Unbind()
	km.PageDown.SetKeys("pgdown", "f")
	return km
}

// helpKeys implements help.KeyMap for a given set of bindings.
type helpKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpKeys) ShortHelp() []key.Binding  { return h.short }
func (h helpKeys) FullHelp() [][]key.Binding { return h.full }
