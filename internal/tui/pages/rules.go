package pages

import (
	"fmt"
	"strings"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	ruleTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	ruleHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(lipgloss.Color("#1F2937"))

	ruleRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	ruleSelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	ruleFilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

// RulesLoadedMsg carries loaded rules.
type RulesLoadedMsg struct {
	Rules []api.Rule
	Err   error
}

// RulesPage displays the active routing rules.
type RulesPage struct {
	width, height int
	rules         []api.Rule
	cursor        int
	scroll        int
	filter        string
	filtering     bool
}

func NewRulesPage() *RulesPage {
	return &RulesPage{}
}

func (r *RulesPage) Title() string { return "Rules" }
func (r *RulesPage) ShortHelp() string {
	if r.filtering {
		return "type to filter │ esc: cancel │ enter: apply"
	}
	return "j/k: navigate │ /: filter │ g/G: top/bottom"
}

func (r *RulesPage) Init() tea.Cmd { return nil }

func (r *RulesPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height

	case RulesLoadedMsg:
		if msg.Err == nil {
			r.rules = msg.Rules
		}

	case tea.KeyMsg:
		if r.filtering {
			switch msg.String() {
			case "esc":
				r.filtering = false
				r.filter = ""
			case "enter":
				r.filtering = false
			case "backspace":
				if len(r.filter) > 0 {
					r.filter = r.filter[:len(r.filter)-1]
				}
			default:
				if len(msg.String()) == 1 {
					r.filter += msg.String()
				}
			}
			return r, nil
		}

		filtered := r.filteredRules()

		switch msg.String() {
		case "up", "k":
			if r.cursor > 0 {
				r.cursor--
			}
		case "down", "j":
			if r.cursor < len(filtered)-1 {
				r.cursor++
			}
		case "/":
			r.filtering = true
		case "g":
			r.cursor = 0
			r.scroll = 0
		case "G":
			r.cursor = len(filtered) - 1
			if r.cursor < 0 {
				r.cursor = 0
			}
		}
	}

	return r, nil
}

func (r *RulesPage) filteredRules() []api.Rule {
	if r.filter == "" {
		return r.rules
	}
	filter := strings.ToLower(r.filter)
	var result []api.Rule
	for _, rule := range r.rules {
		if strings.Contains(strings.ToLower(rule.Type), filter) ||
			strings.Contains(strings.ToLower(rule.Payload), filter) ||
			strings.Contains(strings.ToLower(rule.Proxy), filter) {
			result = append(result, rule)
		}
	}
	return result
}

func (r *RulesPage) View() string {
	if r.width == 0 {
		return "Loading..."
	}

	filtered := r.filteredRules()
	contentWidth := r.width - 4

	// Title
	title := ruleTitleStyle.Render(fmt.Sprintf(" Rules (%d)", len(filtered)))
	if r.filtering {
		title += "\n" + ruleFilterStyle.Render("  Filter: ") + r.filter + "█"
	} else if r.filter != "" {
		title += "  " + ruleFilterStyle.Render(fmt.Sprintf("[filter: %s]", r.filter))
	}

	// Header
	header := ruleHeaderStyle.Width(contentWidth).Render(
		fmt.Sprintf("  %-6s %-20s %-40s %-15s", "#", "Type", "Payload", "Proxy"),
	)

	// Rows
	visibleHeight := r.height - 6
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if r.cursor < r.scroll {
		r.scroll = r.cursor
	}
	if r.cursor >= r.scroll+visibleHeight {
		r.scroll = r.cursor - visibleHeight + 1
	}

	end := r.scroll + visibleHeight
	if end > len(filtered) {
		end = len(filtered)
	}

	var rows []string
	for i := r.scroll; i < end; i++ {
		rule := filtered[i]
		payload := rule.Payload
		if len(payload) > 40 {
			payload = payload[:37] + "..."
		}
		proxy := rule.Proxy
		if len(proxy) > 15 {
			proxy = proxy[:12] + "..."
		}

		line := fmt.Sprintf("  %-6d %-20s %-40s %-15s",
			i+1, rule.Type, payload, proxy,
		)

		if i == r.cursor {
			rows = append(rows, ruleSelectedRowStyle.Render(line))
		} else {
			rows = append(rows, ruleRowStyle.Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		header,
		strings.Join(rows, "\n"),
	)
}

// LoadRules creates a tea.Cmd to fetch rules.
func LoadRules(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetRules()
		if err != nil {
			return RulesLoadedMsg{Err: err}
		}
		return RulesLoadedMsg{Rules: resp.Rules}
	}
}
