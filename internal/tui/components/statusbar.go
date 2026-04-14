package components

import (
	"fmt"

	"github.com/boomyao/clash-cli/internal/util"
	"github.com/charmbracelet/lipgloss"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1)

	statusItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	statusLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	statusInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444"))

	statusUploadStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6"))

	statusDownloadStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))
)

// StatusBar shows system status at the bottom of the TUI.
type StatusBar struct {
	Profile         string
	Mode            string
	TrafficUp       int64
	TrafficDown     int64
	MemoryMB        float64
	SysProxyOn      bool
	CoreRunning     bool
	BackgroundMode  bool
	UpdateAvailable string // tag name of newer release, empty if up to date
	Width           int
}

// NewStatusBar creates a new StatusBar with defaults.
func NewStatusBar() StatusBar {
	return StatusBar{
		Profile: "None",
		Mode:    "rule",
	}
}

// View renders the status bar.
func (s StatusBar) View() string {
	sep := separatorStyle.Render(" │ ")

	parts := []string{}

	// Profile
	parts = append(parts, statusLabelStyle.Render("Profile: ")+statusItemStyle.Render(s.Profile))

	// Mode
	parts = append(parts, statusLabelStyle.Render("Mode: ")+statusItemStyle.Render(s.Mode))

	// Traffic
	up := statusUploadStyle.Render("↑ " + util.FormatBytesPerSec(s.TrafficUp))
	down := statusDownloadStyle.Render("↓ " + util.FormatBytesPerSec(s.TrafficDown))
	parts = append(parts, up+" "+down)

	// Memory
	if s.MemoryMB > 0 {
		parts = append(parts, statusLabelStyle.Render("Mem: ")+statusItemStyle.Render(fmt.Sprintf("%.0fM", s.MemoryMB)))
	}

	// System Proxy
	if s.SysProxyOn {
		parts = append(parts, statusLabelStyle.Render("SysProxy: ")+statusActiveStyle.Render("ON"))
	} else {
		parts = append(parts, statusLabelStyle.Render("SysProxy: ")+statusInactiveStyle.Render("OFF"))
	}

	// Core status
	if s.CoreRunning {
		parts = append(parts, statusLabelStyle.Render("Core: ")+statusActiveStyle.Render("●"))
	} else {
		parts = append(parts, statusLabelStyle.Render("Core: ")+statusInactiveStyle.Render("○"))
	}

	// Background mode indicator
	if s.BackgroundMode {
		bg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Render("⏚ BG")
		parts = append(parts, bg)
	}

	// Update-available indicator
	if s.UpdateAvailable != "" {
		up := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Bold(true).
			Render("↑ " + s.UpdateAvailable)
		parts = append(parts, up)
	}

	content := ""
	for i, part := range parts {
		if i > 0 {
			content += sep
		}
		content += part
	}

	return statusBarStyle.Width(s.Width).Render(content)
}
