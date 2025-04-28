package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// keymap for help
type keyMap struct {
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
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
	Checkout: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "checkout PR"),
	),
}
