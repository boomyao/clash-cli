package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keyboard shortcuts available everywhere.
type GlobalKeyMap struct {
	Quit       key.Binding
	Help       key.Binding
	NextTab    key.Binding
	PrevTab    key.Binding
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
	Tab5       key.Binding
	Tab6       key.Binding
	Tab7       key.Binding
}

// DefaultGlobalKeyMap returns the default global key bindings.
func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Tab1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "Home")),
		Tab2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Profiles")),
		Tab3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "Proxies")),
		Tab4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "Connections")),
		Tab5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "Rules")),
		Tab6: key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "Logs")),
		Tab7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "Settings")),
	}
}
