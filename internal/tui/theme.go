package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorText      = lipgloss.Color("#E5E7EB") // Light gray
	ColorBg        = lipgloss.Color("#1F2937") // Dark bg
	ColorBorder    = lipgloss.Color("#374151") // Border gray
	ColorHighlight = lipgloss.Color("#8B5CF6") // Light purple
	ColorUpload    = lipgloss.Color("#3B82F6") // Blue (upload)
	ColorDownload  = lipgloss.Color("#10B981") // Green (download)
)

// Common styles
var (
	// Base text styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleDanger = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// Status indicator styles
	StyleActive = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StyleInactive = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Box/panel styles
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StyleFocusedBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// Help text
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Upload/Download speed
	StyleUpload = lipgloss.NewStyle().
			Foreground(ColorUpload)

	StyleDownload = lipgloss.NewStyle().
			Foreground(ColorDownload)
)
