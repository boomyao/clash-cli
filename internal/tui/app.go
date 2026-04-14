package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/boomyao/clash-cli/internal/config"
	"github.com/boomyao/clash-cli/internal/core"
	"github.com/boomyao/clash-cli/internal/profile"
	"github.com/boomyao/clash-cli/internal/sysproxy"
	"github.com/boomyao/clash-cli/internal/tui/components"
	"github.com/boomyao/clash-cli/internal/tui/pages"
	"github.com/boomyao/clash-cli/internal/updater"
	"github.com/boomyao/clash-cli/internal/ws"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// CurrentVersion is set from main.go via SetVersion so the updater can
// compare against the version baked in at build time.
var currentVersion = "dev"

// SetVersion records the running build's version for update checks.
func SetVersion(v string) { currentVersion = v }

// Tab indices
const (
	TabHome = iota
	TabProfiles
	TabProxies
	TabConnections
	TabRules
	TabLogs
	TabSettings
)

// AppModel is the root bubbletea model that manages tab navigation and delegates to pages.
type AppModel struct {
	activeTab int
	tabBar    components.TabBar
	statusBar components.StatusBar
	toast     components.Toast
	pages     []pages.Page
	keys      GlobalKeyMap
	appConfig *config.AppConfig
	apiClient *api.Client
	profMgr   *profile.Manager
	sysProxy  sysproxy.SystemProxy
	coreMgr   *core.Manager // non-nil if we launched mihomo ourselves

	// WebSocket streams
	trafficStream *ws.TrafficStream
	memoryStream  *ws.MemoryStream
	logStream     *ws.LogStream
	connStream    *ws.ConnectionsStream

	width, height int

	// Shared state
	trafficUp      int64
	trafficDown    int64
	memoryInUse    int64
	mode           string
	mixedPort      int // from GET /configs
	httpPort       int
	socksPort      int
	coreRunning    bool
	coreVersion    string
	sysProxyOn     bool
	backgroundMode    bool   // if true, cleanup() leaves auto-launched mihomo running
	updateAvailable   string // tag name of newer release, or "" if none
	tunRequestPending bool   // true if user just asked to enable TUN; cleared on next refresh

	initialized bool
}

// NewAppModel creates the root model with all pages and wires up dependencies.
// coreMgr may be nil if mihomo was not started by us.
func NewAppModel(cfg *config.AppConfig, coreMgr *core.Manager) AppModel {
	apiClient := api.NewClient(cfg.API.ExternalController, cfg.API.Secret)
	profMgr := profile.NewManager(cfg)

	// Platform-specific system proxy
	sp := sysproxy.New()

	// Construct pages and inject dependencies
	homePage := pages.NewHomePage()
	profilesPage := pages.NewProfilesPage()
	profilesPage.SetProfileManager(profMgr)
	proxiesPage := pages.NewProxiesPage()
	proxiesPage.SetAPIClient(apiClient)
	connsPage := pages.NewConnectionsPage()
	connsPage.SetAPIClient(apiClient)
	rulesPage := pages.NewRulesPage()
	logsPage := pages.NewLogsPage()
	settingsPage := pages.NewSettingsPage()
	settingsPage.SetAPIClient(apiClient)

	allPages := []pages.Page{
		homePage,
		profilesPage,
		proxiesPage,
		connsPage,
		rulesPage,
		logsPage,
		settingsPage,
	}

	tabNames := make([]string, len(allPages))
	for i, p := range allPages {
		tabNames[i] = p.Title()
	}

	activeTab := cfg.UI.DefaultTab
	if activeTab < 0 || activeTab >= len(allPages) {
		activeTab = 0
	}

	// Detect initial sysproxy state
	sysProxyOn := false
	if sp != nil {
		if on, err := sp.IsEnabled(); err == nil {
			sysProxyOn = on
		}
	}

	return AppModel{
		activeTab:  activeTab,
		tabBar:     components.NewTabBar(tabNames),
		statusBar:  components.NewStatusBar(),
		toast:      components.NewToast(),
		pages:      allPages,
		keys:       DefaultGlobalKeyMap(),
		appConfig:  cfg,
		apiClient:  apiClient,
		profMgr:    profMgr,
		sysProxy:   sp,
		coreMgr:    coreMgr,
		mode:       "rule",
		sysProxyOn: sysProxyOn,
	}
}

// initMsg signals that initial data has been fetched.
type initMsg struct {
	version *api.VersionResponse
	config  *api.ConfigResponse
	err     error
}

// configRefreshedMsg signals a fresh /configs snapshot is available.
type configRefreshedMsg struct {
	cfg *api.ConfigResponse
}

// updateCheckMsg carries the result of a background GitHub release check.
type updateCheckMsg struct {
	latest string // empty if no update or check failed
}

// checkForUpdate returns a tea.Cmd that asks GitHub for the latest release
// and produces an updateCheckMsg. Failures are silent (latest = "").
func checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		rel, err := updater.LatestRelease(5 * time.Second)
		if err != nil || rel == nil {
			return updateCheckMsg{}
		}
		if updater.IsNewer(currentVersion, rel.TagName) {
			return updateCheckMsg{latest: rel.TagName}
		}
		return updateCheckMsg{}
	}
}

func (m AppModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.pages {
		if cmd := p.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Fetch initial data from mihomo API
	client := m.apiClient
	cmds = append(cmds, func() tea.Msg {
		ver, _ := client.GetVersion()
		cfg, _ := client.GetConfig()
		return initMsg{version: ver, config: cfg}
	})

	// Background: check GitHub for a newer clashc release
	cmds = append(cmds, checkForUpdate())

	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case initMsg:
		if msg.version != nil {
			m.coreRunning = true
			m.coreVersion = msg.version.Version
		}
		if msg.config != nil {
			m.applyConfig(msg.config)
		}

		// Start WebSocket streams if core is running
		if m.coreRunning && !m.initialized {
			m.initialized = true
			cmds = append(cmds, m.startStreams()...)
		}

		// Broadcast initial state to all pages
		m.broadcast(pages.NewCoreStatusMsg(m.coreRunning, m.coreVersion), &cmds)
		if msg.config != nil {
			m.broadcast(pages.NewModeMsg(msg.config.Mode), &cmds)
			m.broadcast(pages.NewTunStatusMsg(msg.config.Tun.Enable), &cmds)
			m.broadcast(pages.NewAllowLanMsg(msg.config.AllowLan), &cmds)
		}
		m.broadcast(pages.NewSysProxyMsg(m.sysProxyOn), &cmds)

		// Auto-load proxies and rules in the background
		client := m.apiClient
		cmds = append(cmds, pages.LoadProxies(client))
		cmds = append(cmds, pages.LoadRules(client))

		return m, tea.Batch(cmds...)

	case updateCheckMsg:
		if msg.latest != "" {
			m.updateAvailable = msg.latest
			cmds = append(cmds, m.toast.Show(
				fmt.Sprintf("Update available: %s — run 'clashc update'", msg.latest),
				components.ToastInfo,
			))
		}
		return m, tea.Batch(cmds...)

	case configRefreshedMsg:
		if msg.cfg != nil {
			m.applyConfig(msg.cfg)
			m.broadcast(pages.NewModeMsg(msg.cfg.Mode), &cmds)
			m.broadcast(pages.NewTunStatusMsg(msg.cfg.Tun.Enable), &cmds)
			m.broadcast(pages.NewAllowLanMsg(msg.cfg.AllowLan), &cmds)

			// If the user just asked for TUN ON but mihomo failed to bring it
			// up (most commonly: missing CAP_NET_ADMIN), surface the reason.
			if m.tunRequestPending && !msg.cfg.Tun.Enable {
				cmds = append(cmds, m.toast.Show(
					"TUN failed — mihomo needs CAP_NET_ADMIN. Run: sudo setcap 'cap_net_admin,cap_net_bind_service=+ep' "+m.appConfig.Core.BinaryPath,
					components.ToastError,
				))
			}
			m.tunRequestPending = false
		}
		return m, tea.Batch(cmds...)

	case ws.TrafficMsg:
		if msg.Err == nil {
			m.trafficUp = msg.Up
			m.trafficDown = msg.Down
			up := msg.Up
			down := msg.Down
			cmds = append(cmds, func() tea.Msg {
				return pages.NewTrafficMsg(up, down)
			})
			if m.trafficStream != nil {
				cmds = append(cmds, m.trafficStream.WaitForTraffic())
			}
		}
		return m, tea.Batch(cmds...)

	case ws.MemoryMsg:
		if msg.Err == nil {
			m.memoryInUse = msg.InUse
			inUse := msg.InUse
			cmds = append(cmds, func() tea.Msg {
				return pages.NewMemoryMsg(inUse)
			})
			if m.memoryStream != nil {
				cmds = append(cmds, m.memoryStream.WaitForMemory())
			}
		}
		return m, tea.Batch(cmds...)

	case ws.LogMsg:
		if msg.Err == nil {
			logType := msg.Type
			payload := msg.Payload
			cmds = append(cmds, func() tea.Msg {
				return pages.LogEntryMsg{Type: logType, Payload: payload}
			})
			if m.logStream != nil {
				cmds = append(cmds, m.logStream.WaitForLog())
			}
		}
		return m, tea.Batch(cmds...)

	case ws.ConnectionsMsg:
		if msg.Err == nil {
			conns := msg.Connections
			dl := msg.DownloadTotal
			ul := msg.UploadTotal
			cmds = append(cmds, func() tea.Msg {
				return pages.ConnectionsUpdateMsg{
					Connections:   conns,
					DownloadTotal: dl,
					UploadTotal:   ul,
				}
			})
			count := len(conns)
			cmds = append(cmds, func() tea.Msg {
				return pages.NewConnectionsCountMsg(count)
			})
			if m.connStream != nil {
				cmds = append(cmds, m.connStream.WaitForConnections())
			}
		}
		return m, tea.Batch(cmds...)

	case pages.SettingsUpdatedMsg:
		// When user toggles a setting, refresh state from server.
		// For TUN ON: arm the failure detector so we can warn if the
		// PATCH was accepted but the tun device couldn't be brought up.
		if strings.Contains(msg.Setting, "TUN Mode → ON") {
			m.tunRequestPending = true
		}
		if msg.Err == nil {
			cmds = append(cmds, m.toast.Show(msg.Setting, components.ToastSuccess))
			client := m.apiClient
			cmds = append(cmds, func() tea.Msg {
				cfg, _ := client.GetConfig()
				if cfg != nil {
					return configRefreshedMsg{cfg: cfg}
				}
				return nil
			})
		} else {
			cmds = append(cmds, m.toast.Show(msg.Err.Error(), components.ToastError))
		}
		// Forward to settings page
		updated, cmd := m.pages[TabSettings].Update(msg)
		m.pages[TabSettings] = updated.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case pages.SysProxyChangedMsg:
		// Broadcast sysproxy change to all pages and update root state
		m.sysProxyOn = msg.Enabled
		m.broadcast(pages.NewSysProxyMsg(msg.Enabled), &cmds)
		return m, tea.Batch(cmds...)

	case pages.BackgroundModeToggleMsg:
		// User asked to toggle "keep mihomo on exit"
		if m.coreMgr == nil {
			cmds = append(cmds, m.toast.Show(
				"Background mode requires mihomo to be auto-launched by clashc",
				components.ToastWarning,
			))
			return m, tea.Batch(cmds...)
		}
		m.backgroundMode = !m.backgroundMode
		label := "Background mode OFF — mihomo will stop with clashc"
		kind := components.ToastInfo
		if m.backgroundMode {
			label = "Background mode ON — mihomo will keep running after q"
			kind = components.ToastSuccess
		}
		cmds = append(cmds, m.toast.Show(label, kind))
		m.broadcast(pages.NewBackgroundModeMsg(m.backgroundMode), &cmds)
		return m, tea.Batch(cmds...)

	case pages.ProfileActionMsg:
		if msg.Err == nil {
			cmds = append(cmds, m.toast.Show(msg.Action, components.ToastSuccess))
			// If the action was "Activated:" — tell mihomo to reload the profile file
			if len(msg.Action) > 10 && msg.Action[:10] == "Activated:" {
				if active := m.profMgr.ActiveProfile(); active != nil {
					if path, err := m.profMgr.GetProfilePath(active.ID); err == nil {
						client := m.apiClient
						cmds = append(cmds, func() tea.Msg {
							_ = client.ReloadConfig(path, true)
							return nil
						})
					}
				}
			}
		} else {
			cmds = append(cmds, m.toast.Show(msg.Err.Error(), components.ToastError))
		}
		updated, cmd := m.pages[TabProfiles].Update(msg)
		m.pages[TabProfiles] = updated.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case pages.ProxySelectedMsg:
		if msg.Err == nil {
			cmds = append(cmds, m.toast.Show("Selected: "+msg.Proxy, components.ToastSuccess))
		} else {
			cmds = append(cmds, m.toast.Show(msg.Err.Error(), components.ToastError))
		}
		updated, cmd := m.pages[TabProxies].Update(msg)
		m.pages[TabProxies] = updated.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case components.ToastDismissMsg:
		m.toast.Dismiss()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tabBar.Width = msg.Width
		m.statusBar.Width = msg.Width

		for i, p := range m.pages {
			updated, cmd := p.Update(msg)
			m.pages[i] = updated.(pages.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Global keys (only when no input mode is active)
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.cleanup()
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextTab):
			return m.switchTab((m.activeTab + 1) % len(m.pages))
		case key.Matches(msg, m.keys.PrevTab):
			return m.switchTab((m.activeTab - 1 + len(m.pages)) % len(m.pages))
		case key.Matches(msg, m.keys.Tab1):
			return m.switchTab(0)
		case key.Matches(msg, m.keys.Tab2):
			return m.switchTab(1)
		case key.Matches(msg, m.keys.Tab3):
			return m.switchTab(2)
		case key.Matches(msg, m.keys.Tab4):
			return m.switchTab(3)
		case key.Matches(msg, m.keys.Tab5):
			return m.switchTab(4)
		case key.Matches(msg, m.keys.Tab6):
			return m.switchTab(5)
		case key.Matches(msg, m.keys.Tab7):
			return m.switchTab(6)
		case msg.String() == "s":
			// Quick toggle system proxy from anywhere
			cmd := m.toggleSysProxy()
			if cmd != nil {
				return m, cmd
			}
		}
	}

	// Delegate to active page
	if m.activeTab >= 0 && m.activeTab < len(m.pages) {
		updated, cmd := m.pages[m.activeTab].Update(msg)
		m.pages[m.activeTab] = updated.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// applyConfig stores the relevant fields from a /configs response into the model.
func (m *AppModel) applyConfig(cfg *api.ConfigResponse) {
	m.mode = cfg.Mode
	m.mixedPort = cfg.MixedPort
	m.httpPort = cfg.Port
	m.socksPort = cfg.SocksPort
}

// broadcast forwards a tea.Msg to every page (not just the active one)
// by calling each page's Update directly. This is what state messages
// like ModeMsg/SysProxyMsg/CoreStatusMsg need so all tabs stay in sync.
func (m *AppModel) broadcast(msg tea.Msg, cmds *[]tea.Cmd) {
	for i, p := range m.pages {
		updated, cmd := p.Update(msg)
		m.pages[i] = updated.(pages.Page)
		if cmd != nil {
			*cmds = append(*cmds, cmd)
		}
	}
}

// startStreams connects all WebSocket streams and returns commands to listen on them.
func (m *AppModel) startStreams() []tea.Cmd {
	wsBase := m.apiClient.WSBaseURL()
	secret := m.apiClient.Secret()
	var cmds []tea.Cmd

	m.trafficStream = ws.NewTrafficStream(wsBase, secret)
	if err := m.trafficStream.Connect(); err == nil {
		cmds = append(cmds, m.trafficStream.WaitForTraffic())
	}

	m.memoryStream = ws.NewMemoryStream(wsBase, secret)
	if err := m.memoryStream.Connect(); err == nil {
		cmds = append(cmds, m.memoryStream.WaitForMemory())
	}

	m.logStream = ws.NewLogStream(wsBase, secret, "info")
	if err := m.logStream.Connect(); err == nil {
		cmds = append(cmds, m.logStream.WaitForLog())
	}

	m.connStream = ws.NewConnectionsStream(wsBase, secret)
	if err := m.connStream.Connect(); err == nil {
		cmds = append(cmds, m.connStream.WaitForConnections())
	}

	return cmds
}

// toggleSysProxy toggles the system proxy on/off.
// Reads the actual ports from mihomo's /configs (mixed-port preferred,
// falling back to separate http-port / socks-port).
func (m *AppModel) toggleSysProxy() tea.Cmd {
	if m.sysProxy == nil {
		return nil
	}

	if m.sysProxyOn {
		if err := m.sysProxy.Disable(); err != nil {
			return func() tea.Msg {
				return pages.SettingsUpdatedMsg{Setting: "Disable system proxy", Err: err}
			}
		}
		m.sysProxyOn = false
	} else {
		httpAddr, socksAddr := m.proxyAddrs()
		if httpAddr == "" && socksAddr == "" {
			return func() tea.Msg {
				return pages.SettingsUpdatedMsg{
					Setting: "System proxy",
					Err:     fmt.Errorf("no proxy port available — is mihomo connected?"),
				}
			}
		}
		if err := m.sysProxy.Enable(httpAddr, socksAddr); err != nil {
			return func() tea.Msg {
				return pages.SettingsUpdatedMsg{Setting: "Enable system proxy", Err: err}
			}
		}
		m.sysProxyOn = true
	}

	on := m.sysProxyOn
	port := m.activeProxyPort()
	label := "System proxy enabled"
	if !on {
		label = "System proxy disabled"
	} else if port > 0 {
		label = fmt.Sprintf("System proxy enabled (port %d)", port)
	}
	return tea.Batch(
		func() tea.Msg { return pages.SysProxyChangedMsg{Enabled: on} },
		m.toast.Show(label, components.ToastSuccess),
	)
}

// proxyAddrs returns the (http, socks) addresses to point system proxy at,
// preferring mixed-port (which serves both protocols on one port).
func (m *AppModel) proxyAddrs() (httpAddr, socksAddr string) {
	if m.mixedPort > 0 {
		addr := fmt.Sprintf("127.0.0.1:%d", m.mixedPort)
		return addr, addr
	}
	if m.httpPort > 0 {
		httpAddr = fmt.Sprintf("127.0.0.1:%d", m.httpPort)
	}
	if m.socksPort > 0 {
		socksAddr = fmt.Sprintf("127.0.0.1:%d", m.socksPort)
	}
	return httpAddr, socksAddr
}

// activeProxyPort returns whichever port we're currently exposing.
func (m *AppModel) activeProxyPort() int {
	if m.mixedPort > 0 {
		return m.mixedPort
	}
	if m.httpPort > 0 {
		return m.httpPort
	}
	return m.socksPort
}

// cleanup releases all resources before exit.
func (m *AppModel) cleanup() {
	if m.trafficStream != nil {
		m.trafficStream.Close()
	}
	if m.memoryStream != nil {
		m.memoryStream.Close()
	}
	if m.logStream != nil {
		m.logStream.Close()
	}
	if m.connStream != nil {
		m.connStream.Close()
	}

	// Disable system proxy if we enabled it and config says to
	if m.sysProxy != nil && m.sysProxyOn && m.appConfig.SystemProxy.DisableOnExit {
		_ = m.sysProxy.Disable()
	}

	// Stop mihomo if we launched it — UNLESS the user enabled background mode,
	// in which case we leave it running and orphaned (init/systemd takes over).
	if m.coreMgr != nil && m.coreMgr.IsRunning() && !m.backgroundMode {
		_ = m.coreMgr.Stop()
	}
}

// BackgroundMode reports whether mihomo should be kept running on exit.
// main.go reads this after the TUI has quit so its defer cleanup matches.
func (m *AppModel) BackgroundMode() bool {
	return m.backgroundMode
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Tab bar at top
	m.tabBar.ActiveTab = m.activeTab
	tabView := m.tabBar.View()

	// Status bar at bottom
	m.statusBar.Profile = m.getActiveProfileName()
	m.statusBar.Mode = m.mode
	m.statusBar.TrafficUp = m.trafficUp
	m.statusBar.TrafficDown = m.trafficDown
	m.statusBar.MemoryMB = float64(m.memoryInUse) / 1024 / 1024
	m.statusBar.SysProxyOn = m.sysProxyOn
	m.statusBar.CoreRunning = m.coreRunning
	m.statusBar.BackgroundMode = m.backgroundMode
	m.statusBar.UpdateAvailable = m.updateAvailable
	statusView := m.statusBar.View()

	// Page content
	pageView := ""
	if m.activeTab >= 0 && m.activeTab < len(m.pages) {
		pageView = m.pages[m.activeTab].View()
	}

	tabHeight := lipgloss.Height(tabView)
	statusHeight := lipgloss.Height(statusView)
	helpHeight := 1
	contentHeight := m.height - tabHeight - statusHeight - helpHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight)
	content := contentStyle.Render(pageView)

	helpText := StyleHelp.Render("  " + m.pages[m.activeTab].ShortHelp() + " │ tab: switch │ s: sysproxy │ q: quit")

	// Recompute content height if toast is visible (it eats one extra line)
	if m.toast.Visible {
		toastLine := lipgloss.PlaceHorizontal(m.width, lipgloss.Right, m.toast.View(m.width))
		content = contentStyle.Height(contentHeight - lipgloss.Height(toastLine)).Render(pageView)
		return lipgloss.JoinVertical(lipgloss.Left,
			tabView,
			content,
			toastLine,
			helpText,
			statusView,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		tabView,
		content,
		helpText,
		statusView,
	)
}

func (m AppModel) switchTab(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.pages) {
		return m, nil
	}
	m.activeTab = idx

	// Auto-refresh data when entering certain tabs
	var cmd tea.Cmd
	switch idx {
	case TabProxies:
		cmd = pages.LoadProxies(m.apiClient)
	case TabRules:
		cmd = pages.LoadRules(m.apiClient)
	}
	return m, cmd
}

func (m AppModel) getActiveProfileName() string {
	if m.appConfig.Profiles.Active == "" {
		return "None"
	}
	for _, p := range m.appConfig.Profiles.Items {
		if p.ID == m.appConfig.Profiles.Active {
			return p.Name
		}
	}
	return m.appConfig.Profiles.Active
}
