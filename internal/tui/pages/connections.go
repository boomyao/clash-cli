package pages

import (
	"fmt"
	"strings"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/boomyao/clash-cli/internal/util"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	connHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(lipgloss.Color("#1F2937"))

	connRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	connSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	connTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	connFilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

// ConnectionsPage shows active network connections.
type ConnectionsPage struct {
	width, height int
	connections   []api.Connection
	downloadTotal int64
	uploadTotal   int64
	cursor        int
	scroll        int
	filter        string
	filtering     bool
	apiClient     *api.Client
}

// ConnectionsUpdateMsg carries real-time connection data.
type ConnectionsUpdateMsg struct {
	Connections   []api.Connection
	DownloadTotal int64
	UploadTotal   int64
	Err           error
}

// ConnectionClosedMsg reports connection closure result.
type ConnectionClosedMsg struct {
	ID  string
	Err error
}

func NewConnectionsPage() *ConnectionsPage {
	return &ConnectionsPage{}
}

func (c *ConnectionsPage) SetAPIClient(client *api.Client) {
	c.apiClient = client
}

func (c *ConnectionsPage) Title() string { return "Connections" }
func (c *ConnectionsPage) ShortHelp() string {
	if c.filtering {
		return "type to filter │ esc: cancel │ enter: apply"
	}
	return "j/k: navigate │ x: close │ X: close all │ /: filter"
}

func (c *ConnectionsPage) Init() tea.Cmd { return nil }

func (c *ConnectionsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case ConnectionsUpdateMsg:
		if msg.Err == nil {
			c.connections = msg.Connections
			c.downloadTotal = msg.DownloadTotal
			c.uploadTotal = msg.UploadTotal
		}

	case ConnectionClosedMsg:
		// Connection closed, data will refresh on next WS message

	case tea.KeyMsg:
		if c.filtering {
			switch msg.Type {
			case tea.KeyEsc:
				c.filtering = false
				c.filter = ""
			case tea.KeyEnter:
				c.filtering = false
			case tea.KeyBackspace, tea.KeyCtrlH:
				if len(c.filter) > 0 {
					r := []rune(c.filter)
					c.filter = string(r[:len(r)-1])
				}
			case tea.KeyRunes, tea.KeySpace:
				c.filter += string(msg.Runes)
			}
			return c, nil
		}

		filtered := c.filteredConnections()

		switch msg.String() {
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "down", "j":
			if c.cursor < len(filtered)-1 {
				c.cursor++
			}
		case "/":
			c.filtering = true
		case "x":
			if c.cursor < len(filtered) && c.apiClient != nil {
				conn := filtered[c.cursor]
				client := c.apiClient
				return c, func() tea.Msg {
					err := client.CloseConnection(conn.ID)
					return ConnectionClosedMsg{ID: conn.ID, Err: err}
				}
			}
		case "X":
			if c.apiClient != nil {
				client := c.apiClient
				return c, func() tea.Msg {
					err := client.CloseAllConnections()
					return ConnectionClosedMsg{ID: "all", Err: err}
				}
			}
		}
	}

	return c, nil
}

func (c *ConnectionsPage) filteredConnections() []api.Connection {
	if c.filter == "" {
		return c.connections
	}
	filter := strings.ToLower(c.filter)
	var result []api.Connection
	for _, conn := range c.connections {
		host := strings.ToLower(conn.Metadata.Host)
		dest := strings.ToLower(conn.Metadata.Destination)
		chain := strings.ToLower(strings.Join(conn.Chains, ","))
		process := strings.ToLower(conn.Metadata.Process)

		if strings.Contains(host, filter) ||
			strings.Contains(dest, filter) ||
			strings.Contains(chain, filter) ||
			strings.Contains(process, filter) {
			result = append(result, conn)
		}
	}
	return result
}

func (c *ConnectionsPage) View() string {
	if c.width == 0 {
		return "Loading..."
	}

	contentWidth := c.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	filtered := c.filteredConnections()

	// Title line
	title := connTitleStyle.Render(fmt.Sprintf(" Active Connections (%d)", len(filtered)))
	title += "  " + connFilterStyle.Render(
		fmt.Sprintf("Total: ↑%s ↓%s",
			util.FormatBytes(c.uploadTotal),
			util.FormatBytes(c.downloadTotal),
		),
	)

	// Filter input
	if c.filtering {
		title += "\n" + connFilterStyle.Render("  Filter: ") + c.filter + "█"
	} else if c.filter != "" {
		title += "  " + connFilterStyle.Render(fmt.Sprintf("[filter: %s]", c.filter))
	}

	// Table header
	header := connHeaderStyle.Width(contentWidth).Render(
		fmt.Sprintf("  %-30s %-8s %-8s %-15s %-10s",
			"Host", "Network", "Type", "Chain", "DL/UL",
		),
	)

	// Table rows
	visibleHeight := c.height - 7
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Adjust scroll
	if c.cursor < c.scroll {
		c.scroll = c.cursor
	}
	if c.cursor >= c.scroll+visibleHeight {
		c.scroll = c.cursor - visibleHeight + 1
	}

	var rows []string
	end := c.scroll + visibleHeight
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := c.scroll; i < end; i++ {
		conn := filtered[i]
		host := conn.Metadata.Host
		if host == "" {
			host = conn.Metadata.Destination
		}
		if len(host) > 30 {
			host = host[:27] + "..."
		}

		chain := ""
		if len(conn.Chains) > 0 {
			chain = conn.Chains[0]
			if len(chain) > 15 {
				chain = chain[:12] + "..."
			}
		}

		dlul := fmt.Sprintf("%s/%s",
			util.FormatBytes(conn.Download),
			util.FormatBytes(conn.Upload),
		)

		line := fmt.Sprintf("  %-30s %-8s %-8s %-15s %-10s",
			host,
			conn.Metadata.Network,
			conn.Metadata.Type,
			chain,
			dlul,
		)

		if i == c.cursor {
			rows = append(rows, connSelectedStyle.Render(line))
		} else {
			rows = append(rows, connRowStyle.Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		header,
		strings.Join(rows, "\n"),
	)
}
