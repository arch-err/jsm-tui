package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Enter          key.Binding
	Back           key.Binding
	Quit           key.Binding
	Refresh        key.Binding
	Help           key.Binding
	Transition     key.Binding
	AddComment     key.Binding
	Submit         key.Binding
	Cancel         key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	ToggleFavorite         key.Binding
	ToggleHideNonFavorites key.Binding
	GoToTop                key.Binding
	GoToBottom             key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Transition: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "transition"),
		),
		AddComment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		ToggleFavorite: key.NewBinding(
			key.WithKeys("*"),
			key.WithHelp("*", "toggle favorite"),
		),
		ToggleHideNonFavorites: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "toggle hide non-favorites"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
	}
}
