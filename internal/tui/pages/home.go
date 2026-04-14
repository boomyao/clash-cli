package pages

import (
	"fmt"
	"strings"

	"github.com/boomyao/clash-cli/internal/util"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	homeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2)

	homeTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	homeUpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))

	homeDownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	homeValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Bold(true)

	homeLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	homeKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	homeDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

// HomePage shows the dashboard with traffic, memory, and quick actions.
type HomePage struct {
	width, height int
	trafficUp     int64
	trafficDown   int64
	memoryInUse   int64
	connections   int
	mode          string
	coreRunning   bool
	coreVersion   string
	sysProxyOn    bool
}

// NewHomePage creates a new HomePage.
func NewHomePage() *HomePage {
	return &HomePage{
		mode: "rule",
	}
}

func (h *HomePage) Title() string { return "Home" }

func (h *HomePage) ShortHelp() string {
	return "m: mode │ s: sysproxy │ t: tun │ r: restart core"
}

func (h *HomePage) Init() tea.Cmd {
	return nil
}

func (h *HomePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle shared messages to update local state
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	case trafficMsg:
		h.trafficUp = msg.up
		h.trafficDown = msg.down
	case memoryMsg:
		h.memoryInUse = msg.inUse
	case coreStatusMsg:
		h.coreRunning = msg.running
		h.coreVersion = msg.version
	case modeMsg:
		h.mode = msg.mode
	case sysProxyMsg:
		h.sysProxyOn = msg.enabled
	case connectionsCountMsg:
		h.connections = msg.count
	}
	return h, nil
}

func (h *HomePage) View() string {
	if h.width == 0 {
		return "Loading..."
	}

	contentWidth := h.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Traffic section
	trafficContent := fmt.Sprintf(
		"%s  %s    %s  %s",
		homeUpStyle.Render("↑ Upload:"),
		homeValueStyle.Render(util.FormatBytesPerSec(h.trafficUp)),
		homeDownStyle.Render("↓ Download:"),
		homeValueStyle.Render(util.FormatBytesPerSec(h.trafficDown)),
	)

	trafficBox := homeBoxStyle.Width(contentWidth).Render(
		homeTitleStyle.Render("Traffic") + "\n" + trafficContent,
	)

	// System info section
	coreStatus := "stopped"
	if h.coreRunning {
		coreStatus = "running"
		if h.coreVersion != "" {
			coreStatus += " (" + h.coreVersion + ")"
		}
	}

	sysProxyStatus := "OFF"
	if h.sysProxyOn {
		sysProxyStatus = "ON"
	}

	systemLines := []string{
		fmt.Sprintf("%s %s  │  %s %s",
			homeLabelStyle.Render("Memory:"),
			homeValueStyle.Render(util.FormatBytes(h.memoryInUse)),
			homeLabelStyle.Render("Connections:"),
			homeValueStyle.Render(fmt.Sprintf("%d", h.connections)),
		),
		fmt.Sprintf("%s %s  │  %s %s",
			homeLabelStyle.Render("Mode:"),
			homeValueStyle.Render(h.mode),
			homeLabelStyle.Render("Core:"),
			homeValueStyle.Render(coreStatus),
		),
		fmt.Sprintf("%s %s",
			homeLabelStyle.Render("System Proxy:"),
			homeValueStyle.Render(sysProxyStatus),
		),
	}

	systemBox := homeBoxStyle.Width(contentWidth).Render(
		homeTitleStyle.Render("System") + "\n" + strings.Join(systemLines, "\n"),
	)

	// Quick actions
	actions := []string{
		homeKeyStyle.Render("[m]") + homeDescStyle.Render(" Toggle Mode"),
		homeKeyStyle.Render("[s]") + homeDescStyle.Render(" Toggle SysProxy"),
		homeKeyStyle.Render("[t]") + homeDescStyle.Render(" Toggle TUN"),
		homeKeyStyle.Render("[r]") + homeDescStyle.Render(" Restart Core"),
	}

	actionsBox := homeBoxStyle.Width(contentWidth).Render(
		homeTitleStyle.Render("Quick Actions") + "\n" + strings.Join(actions, "   "),
	)

	return lipgloss.JoinVertical(lipgloss.Left, trafficBox, systemBox, actionsBox)
}

// Internal message types broadcast by the root model to all pages.
type trafficMsg struct{ up, down int64 }
type memoryMsg struct{ inUse int64 }
type coreStatusMsg struct {
	running bool
	version string
}
type modeMsg struct{ mode string }
type sysProxyMsg struct{ enabled bool }
type tunStatusMsg struct{ enabled bool }
type allowLanMsg struct{ enabled bool }
type connectionsCountMsg struct{ count int }
type backgroundModeMsg struct{ enabled bool }

// SysProxyChangedMsg is emitted by the root app when the user toggles
// the system proxy with the global `s` shortcut. It is exported so the
// root model's Update can match on it (the root lives in a different package).
type SysProxyChangedMsg struct{ Enabled bool }

// BackgroundModeToggleMsg is emitted by the Settings page when the user toggles
// "Keep mihomo running on exit". The root app captures it and updates state,
// then re-broadcasts as backgroundModeMsg so all pages can show indicators.
type BackgroundModeToggleMsg struct{}

// Constructors used by app.go to push state updates into pages.
func NewTrafficMsg(up, down int64) tea.Msg              { return trafficMsg{up, down} }
func NewMemoryMsg(inUse int64) tea.Msg                  { return memoryMsg{inUse} }
func NewCoreStatusMsg(running bool, ver string) tea.Msg { return coreStatusMsg{running, ver} }
func NewModeMsg(mode string) tea.Msg                    { return modeMsg{mode} }
func NewSysProxyMsg(enabled bool) tea.Msg               { return sysProxyMsg{enabled} }
func NewTunStatusMsg(enabled bool) tea.Msg              { return tunStatusMsg{enabled} }
func NewAllowLanMsg(enabled bool) tea.Msg               { return allowLanMsg{enabled} }
func NewConnectionsCountMsg(count int) tea.Msg          { return connectionsCountMsg{count} }
func NewBackgroundModeMsg(enabled bool) tea.Msg         { return backgroundModeMsg{enabled} }
