package pages

import tea "github.com/charmbracelet/bubbletea"

// Page is the interface all TUI pages must implement.
type Page interface {
	tea.Model
	// Title returns the display name for the tab bar.
	Title() string
	// ShortHelp returns short help text for the current page state.
	ShortHelp() string
}
