package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottomForeground(lipgloss.Color("#374151"))
)

// TabBar renders a horizontal tab bar.
type TabBar struct {
	Tabs      []string
	ActiveTab int
	Width     int
}

// NewTabBar creates a new TabBar.
func NewTabBar(tabs []string) TabBar {
	return TabBar{
		Tabs:      tabs,
		ActiveTab: 0,
	}
}

// View renders the tab bar.
func (t TabBar) View() string {
	var tabs []string

	for i, tab := range t.Tabs {
		if i == t.ActiveTab {
			tabs = append(tabs, activeTabStyle.Render(tab))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(tab))
		}
	}

	row := strings.Join(tabs, "")
	bar := tabBarStyle.Width(t.Width).Render(row)
	return bar
}
