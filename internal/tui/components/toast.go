package components

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// ToastKind classifies a toast for styling.
type ToastKind int

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// ToastDismissMsg is sent when a toast should be removed.
type ToastDismissMsg struct{}

// Toast is a transient notification overlay.
type Toast struct {
	Message string
	Kind    ToastKind
	Visible bool
}

// NewToast creates an empty toast.
func NewToast() Toast {
	return Toast{}
}

// Show displays a toast and returns a tea.Cmd that dismisses it after 3 seconds.
func (t *Toast) Show(message string, kind ToastKind) tea.Cmd {
	t.Message = message
	t.Kind = kind
	t.Visible = true
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ToastDismissMsg{}
	})
}

// Dismiss hides the toast.
func (t *Toast) Dismiss() {
	t.Visible = false
	t.Message = ""
}

// View renders the toast.
func (t Toast) View(width int) string {
	if !t.Visible || t.Message == "" {
		return ""
	}

	var color lipgloss.Color
	var prefix string
	switch t.Kind {
	case ToastSuccess:
		color = lipgloss.Color("#10B981")
		prefix = "✓ "
	case ToastWarning:
		color = lipgloss.Color("#F59E0B")
		prefix = "⚠ "
	case ToastError:
		color = lipgloss.Color("#EF4444")
		prefix = "✗ "
	default:
		color = lipgloss.Color("#3B82F6")
		prefix = "ℹ "
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Foreground(color).
		Padding(0, 2).
		Bold(true)

	return style.Render(prefix + t.Message)
}
