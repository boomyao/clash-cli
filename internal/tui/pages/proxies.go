package pages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	groupListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1)

	groupFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(0, 1)

	nodeListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1)

	nodeFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	activeNodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	normalNodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	delayGoodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	delayMediumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	delayBadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	delayTimeoutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	proxyTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	proxyHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

// ProxiesPage shows proxy groups and allows node selection.
type ProxiesPage struct {
	width, height int

	// Data
	groups    []proxyGroup
	allNodes  map[string]api.Proxy
	apiClient *api.Client

	// UI state
	focusGroup    bool // true = group pane focused, false = node pane
	groupIdx      int
	nodeIdx       int
	groupScroll   int
	nodeScroll    int
	testing       bool
	testingGroup  string
	nodeDelays    map[string]int // proxy name -> delay ms
}

type proxyGroup struct {
	name    string
	typ     string
	now     string
	members []string
}

// ProxiesLoadedMsg carries loaded proxy data.
type ProxiesLoadedMsg struct {
	Groups   []proxyGroup
	AllNodes map[string]api.Proxy
	Err      error
}

// ProxySelectedMsg reports that a proxy was selected.
type ProxySelectedMsg struct {
	Group string
	Proxy string
	Err   error
}

// GroupDelayTestedMsg reports delay test results.
type GroupDelayTestedMsg struct {
	Group  string
	Delays map[string]int
	Err    error
}

func NewProxiesPage() *ProxiesPage {
	return &ProxiesPage{
		focusGroup: true,
		nodeDelays: make(map[string]int),
	}
}

func (p *ProxiesPage) Title() string { return "Proxies" }
func (p *ProxiesPage) ShortHelp() string {
	return "hjkl/arrows: navigate │ enter: select │ t: test │ tab: switch pane"
}

func (p *ProxiesPage) Init() tea.Cmd { return nil }

// SetAPIClient injects the API client.
func (p *ProxiesPage) SetAPIClient(client *api.Client) {
	p.apiClient = client
}

func (p *ProxiesPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case ProxiesLoadedMsg:
		if msg.Err == nil {
			p.groups = msg.Groups
			p.allNodes = msg.AllNodes
		}

	case ProxySelectedMsg:
		if msg.Err == nil {
			// Update the selected node in the group
			for i, g := range p.groups {
				if g.name == msg.Group {
					p.groups[i].now = msg.Proxy
					break
				}
			}
		}

	case GroupDelayTestedMsg:
		p.testing = false
		if msg.Err == nil {
			for name, delay := range msg.Delays {
				p.nodeDelays[name] = delay
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			p.focusGroup = !p.focusGroup
		case "up", "k":
			if p.focusGroup {
				if p.groupIdx > 0 {
					p.groupIdx--
				}
				p.nodeIdx = 0 // Reset node selection when switching groups
			} else {
				if p.nodeIdx > 0 {
					p.nodeIdx--
				}
			}
		case "down", "j":
			if p.focusGroup {
				if p.groupIdx < len(p.groups)-1 {
					p.groupIdx++
				}
				p.nodeIdx = 0
			} else {
				members := p.currentMembers()
				if p.nodeIdx < len(members)-1 {
					p.nodeIdx++
				}
			}
		case "left", "h":
			p.focusGroup = true
		case "right", "l":
			p.focusGroup = false
		case "enter":
			if !p.focusGroup && p.apiClient != nil {
				return p, p.selectNode()
			}
		case "t":
			if p.apiClient != nil && !p.testing {
				return p, p.testGroupDelay()
			}
		}
	}

	return p, nil
}

func (p *ProxiesPage) currentMembers() []string {
	if p.groupIdx >= 0 && p.groupIdx < len(p.groups) {
		return p.groups[p.groupIdx].members
	}
	return nil
}

func (p *ProxiesPage) selectNode() tea.Cmd {
	if p.groupIdx >= len(p.groups) {
		return nil
	}
	group := p.groups[p.groupIdx]
	members := group.members
	if p.nodeIdx >= len(members) {
		return nil
	}
	nodeName := members[p.nodeIdx]
	groupName := group.name
	client := p.apiClient

	return func() tea.Msg {
		err := client.SelectProxy(groupName, nodeName)
		return ProxySelectedMsg{Group: groupName, Proxy: nodeName, Err: err}
	}
}

func (p *ProxiesPage) testGroupDelay() tea.Cmd {
	if p.groupIdx >= len(p.groups) {
		return nil
	}
	p.testing = true
	groupName := p.groups[p.groupIdx].name
	p.testingGroup = groupName
	client := p.apiClient

	return func() tea.Msg {
		delays, err := client.TestGroupDelay(groupName, "", 5000)
		result := make(map[string]int)
		if err == nil {
			for k, v := range delays {
				result[k] = v
			}
		}
		return GroupDelayTestedMsg{Group: groupName, Delays: result, Err: err}
	}
}

func (p *ProxiesPage) View() string {
	if p.width == 0 {
		return "Loading..."
	}

	if len(p.groups) == 0 {
		return lipgloss.NewStyle().Padding(2, 4).Foreground(lipgloss.Color("#6B7280")).
			Render("No proxy groups loaded. Make sure mihomo is running and connected.")
	}

	// Calculate pane widths
	totalWidth := p.width - 4 // margin
	groupWidth := totalWidth / 3
	nodeWidth := totalWidth - groupWidth - 1 // -1 for separator
	if groupWidth < 15 {
		groupWidth = 15
	}

	contentHeight := p.height - 6 // account for borders and headers
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Render group pane
	groupPane := p.renderGroupPane(groupWidth, contentHeight)

	// Render node pane
	nodePane := p.renderNodePane(nodeWidth, contentHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, groupPane, " ", nodePane)
}

func (p *ProxiesPage) renderGroupPane(width, height int) string {
	title := proxyTitleStyle.Render(" Groups")

	var lines []string
	visibleStart := p.groupScroll
	visibleEnd := visibleStart + height - 2
	if visibleEnd > len(p.groups) {
		visibleEnd = len(p.groups)
	}

	// Adjust scroll to keep selected item visible
	if p.groupIdx < visibleStart {
		p.groupScroll = p.groupIdx
		visibleStart = p.groupScroll
		visibleEnd = visibleStart + height - 2
		if visibleEnd > len(p.groups) {
			visibleEnd = len(p.groups)
		}
	}
	if p.groupIdx >= visibleEnd {
		p.groupScroll = p.groupIdx - height + 3
		if p.groupScroll < 0 {
			p.groupScroll = 0
		}
		visibleStart = p.groupScroll
		visibleEnd = visibleStart + height - 2
		if visibleEnd > len(p.groups) {
			visibleEnd = len(p.groups)
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		g := p.groups[i]
		prefix := "  "
		if i == p.groupIdx {
			prefix = "> "
		}

		name := g.name
		if len(name) > width-8 {
			name = name[:width-8] + "..."
		}

		typeBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf(" [%s]", strings.ToLower(g.typ)))

		line := prefix + name + typeBadge
		if i == p.groupIdx {
			line = selectedItemStyle.Render(line)
		}
		lines = append(lines, line)
	}

	content := title + "\n" + strings.Join(lines, "\n")

	style := groupListStyle
	if p.focusGroup {
		style = groupFocusedStyle
	}

	return style.Width(width).Height(height).Render(content)
}

func (p *ProxiesPage) renderNodePane(width, height int) string {
	if p.groupIdx >= len(p.groups) {
		return nodeListStyle.Width(width).Height(height).Render("No group selected")
	}

	group := p.groups[p.groupIdx]
	title := proxyTitleStyle.Render(fmt.Sprintf(" %s", group.name))
	if p.testing && p.testingGroup == group.name {
		title += proxyHelpStyle.Render("  testing...")
	}

	members := group.members
	var lines []string

	visibleStart := p.nodeScroll
	visibleEnd := visibleStart + height - 2
	if visibleEnd > len(members) {
		visibleEnd = len(members)
	}

	if p.nodeIdx < visibleStart {
		p.nodeScroll = p.nodeIdx
		visibleStart = p.nodeScroll
		visibleEnd = visibleStart + height - 2
		if visibleEnd > len(members) {
			visibleEnd = len(members)
		}
	}
	if p.nodeIdx >= visibleEnd {
		p.nodeScroll = p.nodeIdx - height + 3
		if p.nodeScroll < 0 {
			p.nodeScroll = 0
		}
		visibleStart = p.nodeScroll
		visibleEnd = visibleStart + height - 2
		if visibleEnd > len(members) {
			visibleEnd = len(members)
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		name := members[i]
		isActive := name == group.now
		isCursor := i == p.nodeIdx && !p.focusGroup

		prefix := "  "
		if isActive {
			prefix = "● "
		}
		if isCursor {
			prefix = "> "
			if isActive {
				prefix = "●>"
			}
		}

		// Format delay
		delayStr := ""
		if delay, ok := p.nodeDelays[name]; ok {
			if delay > 0 {
				delayStr = formatDelay(delay)
			} else {
				delayStr = delayTimeoutStyle.Render("timeout")
			}
		}

		// Truncate name to fit
		maxNameLen := width - 14
		displayName := name
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen] + "..."
		}

		// Build the line
		var line string
		if isActive {
			line = activeNodeStyle.Render(prefix+displayName) + "  " + delayStr
		} else if isCursor {
			line = selectedItemStyle.Render(prefix+displayName) + "  " + delayStr
		} else {
			line = normalNodeStyle.Render(prefix+displayName) + "  " + delayStr
		}

		lines = append(lines, line)
	}

	content := title + "\n" + strings.Join(lines, "\n")

	style := nodeListStyle
	if !p.focusGroup {
		style = nodeFocusedStyle
	}

	return style.Width(width).Height(height).Render(content)
}

func formatDelay(ms int) string {
	s := fmt.Sprintf("%dms", ms)
	switch {
	case ms < 200:
		return delayGoodStyle.Render(s)
	case ms < 500:
		return delayMediumStyle.Render(s)
	default:
		return delayBadStyle.Render(s)
	}
}

// LoadProxies creates a tea.Cmd to fetch proxy data.
func LoadProxies(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		proxiesResp, err := client.GetProxies()
		if err != nil {
			return ProxiesLoadedMsg{Err: err}
		}

		var groups []proxyGroup
		allNodes := proxiesResp.Proxies

		// Extract groups (Selector, URLTest, Fallback, LoadBalance)
		for name, proxy := range allNodes {
			switch proxy.Type {
			case "Selector", "URLTest", "Fallback", "LoadBalance":
				g := proxyGroup{
					name:    name,
					typ:     proxy.Type,
					now:     proxy.Now,
					members: proxy.All,
				}
				groups = append(groups, g)
			}
		}

		// Sort groups by name
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].name < groups[j].name
		})

		return ProxiesLoadedMsg{Groups: groups, AllNodes: allNodes}
	}
}
