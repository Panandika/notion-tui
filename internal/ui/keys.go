package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	ShowHelp key.Binding
	Refresh  key.Binding
	NewPage  key.Binding
	Edit     key.Binding
	Command  key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("left/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("right/l", "move right"),
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
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		ShowHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		NewPage: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new page"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Command: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "command palette"),
		),
	}
}

// Help returns all key bindings for help display.
func (k KeyMap) Help() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
		k.Left,
		k.Right,
		k.Enter,
		k.Back,
		k.Quit,
		k.ShowHelp,
		k.Refresh,
		k.NewPage,
		k.Edit,
		k.Command,
	}
}

// ShortHelp returns a subset of key bindings for compact help display.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
		k.Enter,
		k.Quit,
		k.ShowHelp,
	}
}

// FullHelp returns all key bindings grouped for full help display.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back, k.Quit},
		{k.Refresh, k.NewPage, k.Edit, k.Command},
		{k.ShowHelp},
	}
}
