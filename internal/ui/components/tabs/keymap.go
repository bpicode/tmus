package tabs

import "charm.land/bubbles/v2/key"

// KeyMap defines the bindings for keyboard navigation.
type KeyMap struct {
	Next     key.Binding
	Previous key.Binding
}

// DefaultKeyMap returns Tab and Shift+Tab bindings for tab navigation.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		Previous: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous tab"),
		),
	}
}
