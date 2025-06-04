package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// keymap for help
type keyMap struct {
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Refresh  key.Binding
	Checkout key.Binding
	Quit     key.Binding
}

var Keys = keyMap{
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "prev category"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next category"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "open PR in browser"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh PR list"),
	),
	Checkout: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "checkout PR"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
