package pages

import (
	"fmt"
	"strings"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	settingSectionStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#374151")).
				Padding(0, 2).
				MarginBottom(1)

	settingTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	settingLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Width(20)

	settingValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB")).
				Bold(true)

	settingOnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	settingOffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	settingCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	settingKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)
)

type settingItem struct {
	label  string
	action string // key to identify the setting
}

// SettingsPage manages core, proxy, and TUN settings.
type SettingsPage struct {
	width, height int
	cursor        int
	apiClient     *api.Client

	// State
	mode       string
	tunEnabled bool
	allowLan   bool
	mixedPort  int
	logLevel   string

	coreRunning    bool
	coreVersion    string
	sysProxyOn     bool
	backgroundMode bool // keep mihomo running on exit

	items []settingItem

	// Status message
	statusMsg string
}

// SettingsUpdatedMsg signals that a setting was changed.
type SettingsUpdatedMsg struct {
	Setting string
	Err     error
}

func NewSettingsPage() *SettingsPage {
	return &SettingsPage{
		mode:     "rule",
		logLevel: "info",
		items: []settingItem{
			{label: "Mode", action: "mode"},
			{label: "System Proxy", action: "sysproxy"},
			{label: "TUN Mode", action: "tun"},
			{label: "Allow LAN", action: "allowlan"},
			{label: "Keep mihomo on exit", action: "background"},
			{label: "Restart Core", action: "restart"},
			{label: "Flush DNS Cache", action: "flushdns"},
			{label: "Flush FakeIP Cache", action: "flushfakeip"},
		},
	}
}

func (s *SettingsPage) SetAPIClient(client *api.Client) {
	s.apiClient = client
}

func (s *SettingsPage) Title() string     { return "Settings" }
func (s *SettingsPage) ShortHelp() string { return "j/k: navigate │ enter/space: toggle │ r: restart core" }

func (s *SettingsPage) Init() tea.Cmd { return nil }

func (s *SettingsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case coreStatusMsg:
		s.coreRunning = msg.running
		s.coreVersion = msg.version

	case modeMsg:
		s.mode = msg.mode

	case sysProxyMsg:
		s.sysProxyOn = msg.enabled

	case tunStatusMsg:
		s.tunEnabled = msg.enabled

	case allowLanMsg:
		s.allowLan = msg.enabled

	case backgroundModeMsg:
		s.backgroundMode = msg.enabled

	case SettingsUpdatedMsg:
		if msg.Err != nil {
			s.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			s.statusMsg = fmt.Sprintf("%s updated successfully", msg.Setting)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
		case "enter", " ":
			return s, s.executeAction()
		}
	}

	return s, nil
}

func (s *SettingsPage) executeAction() tea.Cmd {
	if s.cursor >= len(s.items) || s.apiClient == nil {
		return nil
	}

	action := s.items[s.cursor].action
	client := s.apiClient

	switch action {
	case "mode":
		// Cycle through modes: rule -> global -> direct -> rule
		nextMode := "rule"
		switch s.mode {
		case "rule":
			nextMode = "global"
		case "global":
			nextMode = "direct"
		case "direct":
			nextMode = "rule"
		}
		s.mode = nextMode
		return func() tea.Msg {
			err := client.SetMode(nextMode)
			return SettingsUpdatedMsg{Setting: "Mode → " + nextMode, Err: err}
		}

	case "tun":
		// Don't optimistically flip — mihomo may accept the PATCH but fail to
		// actually create the TUN device (e.g. missing CAP_NET_ADMIN). The
		// configRefreshedMsg broadcast after SettingsUpdatedMsg will set the
		// real state from /configs.
		newVal := !s.tunEnabled
		return func() tea.Msg {
			err := client.SetTunEnabled(newVal)
			label := "TUN Mode"
			if newVal {
				label = "TUN Mode → ON (verifying...)"
			} else {
				label = "TUN Mode → OFF"
			}
			return SettingsUpdatedMsg{Setting: label, Err: err}
		}

	case "allowlan":
		newVal := !s.allowLan
		return func() tea.Msg {
			err := client.SetAllowLan(newVal)
			return SettingsUpdatedMsg{Setting: "Allow LAN", Err: err}
		}

	case "background":
		// This is purely client-side state. The root model owns the actual
		// flag; we just emit a toggle event for it to handle.
		return func() tea.Msg {
			return BackgroundModeToggleMsg{}
		}

	case "restart":
		return func() tea.Msg {
			err := client.Restart()
			return SettingsUpdatedMsg{Setting: "Core Restart", Err: err}
		}

	case "flushdns":
		return func() tea.Msg {
			err := client.FlushDNS()
			return SettingsUpdatedMsg{Setting: "DNS Cache Flushed", Err: err}
		}

	case "flushfakeip":
		return func() tea.Msg {
			err := client.FlushFakeIP()
			return SettingsUpdatedMsg{Setting: "FakeIP Cache Flushed", Err: err}
		}
	}

	return nil
}

func (s *SettingsPage) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	contentWidth := s.width - 6
	if contentWidth < 30 {
		contentWidth = 30
	}

	var sections []string

	// Core Info section
	coreStatus := settingOffStyle.Render("stopped")
	if s.coreRunning {
		coreStatus = settingOnStyle.Render("running")
		if s.coreVersion != "" {
			coreStatus += settingValueStyle.Render(" (" + s.coreVersion + ")")
		}
	}

	coreInfo := []string{
		settingLabelStyle.Render("Core Status:") + " " + coreStatus,
	}

	sections = append(sections,
		settingSectionStyle.Width(contentWidth).Render(
			settingTitleStyle.Render("Core")+"\n"+strings.Join(coreInfo, "\n"),
		),
	)

	// Settings items
	var settingLines []string
	for i, item := range s.items {
		cursor := "  "
		if i == s.cursor {
			cursor = settingCursorStyle.Render("> ")
		}

		value := s.getSettingValue(item.action)
		line := cursor + settingLabelStyle.Render(item.label+":") + " " + value
		settingLines = append(settingLines, line)
	}

	sections = append(sections,
		settingSectionStyle.Width(contentWidth).Render(
			settingTitleStyle.Render("Network & Control")+"\n"+strings.Join(settingLines, "\n"),
		),
	)

	// Status message
	if s.statusMsg != "" {
		sections = append(sections,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Padding(0, 2).
				Render("  "+s.statusMsg),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (s *SettingsPage) getSettingValue(action string) string {
	switch action {
	case "mode":
		return settingValueStyle.Render(s.mode) + "  " +
			settingKeyStyle.Render("[enter]") + " cycle"
	case "sysproxy":
		if s.sysProxyOn {
			return settingOnStyle.Render("ON")
		}
		return settingOffStyle.Render("OFF")
	case "tun":
		if s.tunEnabled {
			return settingOnStyle.Render("ON")
		}
		return settingOffStyle.Render("OFF")
	case "allowlan":
		if s.allowLan {
			return settingOnStyle.Render("ON")
		}
		return settingOffStyle.Render("OFF")
	case "background":
		if s.backgroundMode {
			return settingOnStyle.Render("ON") + "  " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
					Render("(mihomo will keep running after q)")
		}
		return settingOffStyle.Render("OFF")
	case "restart":
		return settingKeyStyle.Render("[enter]") + " to restart"
	case "flushdns":
		return settingKeyStyle.Render("[enter]") + " to flush"
	case "flushfakeip":
		return settingKeyStyle.Render("[enter]") + " to flush"
	}
	return ""
}
