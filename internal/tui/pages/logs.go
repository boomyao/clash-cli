package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	logTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	logInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))

	logWarnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	logErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	logDebugStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	logTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	logPayloadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	logFilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

const maxLogEntries = 500

// LogEntry represents a single log line.
type LogEntry struct {
	Time    time.Time
	Type    string
	Payload string
}

// LogEntryMsg carries a new log entry.
type LogEntryMsg struct {
	Type    string
	Payload string
}

// LogsPage shows real-time log streaming.
type LogsPage struct {
	width, height int
	entries       []LogEntry
	level         string // current filter level
	levels        []string
	levelIdx      int
	paused        bool
	scroll        int
	autoScroll    bool
	filter        string
	filtering     bool
}

func NewLogsPage() *LogsPage {
	return &LogsPage{
		level:      "info",
		levels:     []string{"debug", "info", "warning", "error", "silent"},
		levelIdx:   1,
		autoScroll: true,
	}
}

func (l *LogsPage) Title() string { return "Logs" }
func (l *LogsPage) ShortHelp() string {
	if l.filtering {
		return "type to filter │ esc: cancel │ enter: apply"
	}
	status := ""
	if l.paused {
		status = " [PAUSED]"
	}
	return fmt.Sprintf("l: level (%s) │ /: filter │ p: pause%s │ c: clear", l.level, status)
}

func (l *LogsPage) Init() tea.Cmd { return nil }

func (l *LogsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height

	case LogEntryMsg:
		if !l.paused {
			entry := LogEntry{
				Time:    time.Now(),
				Type:    msg.Type,
				Payload: msg.Payload,
			}
			l.entries = append(l.entries, entry)
			if len(l.entries) > maxLogEntries {
				l.entries = l.entries[len(l.entries)-maxLogEntries:]
			}
			if l.autoScroll {
				l.scroll = len(l.filteredEntries()) - l.visibleHeight()
				if l.scroll < 0 {
					l.scroll = 0
				}
			}
		}

	case tea.KeyMsg:
		if l.filtering {
			switch msg.Type {
			case tea.KeyEsc:
				l.filtering = false
				l.filter = ""
			case tea.KeyEnter:
				l.filtering = false
			case tea.KeyBackspace, tea.KeyCtrlH:
				if len(l.filter) > 0 {
					r := []rune(l.filter)
					l.filter = string(r[:len(r)-1])
				}
			case tea.KeyRunes, tea.KeySpace:
				l.filter += string(msg.Runes)
			}
			return l, nil
		}

		switch msg.String() {
		case "l":
			l.levelIdx = (l.levelIdx + 1) % len(l.levels)
			l.level = l.levels[l.levelIdx]
		case "p":
			l.paused = !l.paused
		case "c":
			l.entries = nil
			l.scroll = 0
		case "/":
			l.filtering = true
		case "up", "k":
			l.autoScroll = false
			if l.scroll > 0 {
				l.scroll--
			}
		case "down", "j":
			filtered := l.filteredEntries()
			maxScroll := len(filtered) - l.visibleHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if l.scroll < maxScroll {
				l.scroll++
			}
			if l.scroll >= maxScroll {
				l.autoScroll = true
			}
		case "G":
			l.autoScroll = true
			filtered := l.filteredEntries()
			l.scroll = len(filtered) - l.visibleHeight()
			if l.scroll < 0 {
				l.scroll = 0
			}
		case "g":
			l.autoScroll = false
			l.scroll = 0
		}
	}

	return l, nil
}

func (l *LogsPage) visibleHeight() int {
	h := l.height - 5
	if h < 1 {
		return 1
	}
	return h
}

func (l *LogsPage) filteredEntries() []LogEntry {
	filter := strings.ToLower(l.filter)
	levelWeight := logLevelWeight(l.level)

	var result []LogEntry
	for _, e := range l.entries {
		// Level filter
		if logLevelWeight(e.Type) < levelWeight {
			continue
		}
		// Text filter
		if filter != "" && !strings.Contains(strings.ToLower(e.Payload), filter) {
			continue
		}
		result = append(result, e)
	}
	return result
}

func logLevelWeight(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warning", "warn":
		return 2
	case "error":
		return 3
	case "silent":
		return 4
	}
	return 0
}

func (l *LogsPage) View() string {
	if l.width == 0 {
		return "Loading..."
	}

	filtered := l.filteredEntries()

	// Title
	title := logTitleStyle.Render(fmt.Sprintf(" Logs (%d entries)", len(filtered)))
	title += "  " + logFilterStyle.Render(fmt.Sprintf("Level: %s", l.level))
	if l.paused {
		title += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render("[PAUSED]")
	}

	if l.filtering {
		title += "\n" + logFilterStyle.Render("  Filter: ") + l.filter + "█"
	} else if l.filter != "" {
		title += "  " + logFilterStyle.Render(fmt.Sprintf("[filter: %s]", l.filter))
	}

	// Log entries
	visible := l.visibleHeight()
	start := l.scroll
	end := start + visible
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	var lines []string
	for i := start; i < end; i++ {
		e := filtered[i]
		timeStr := logTimeStyle.Render(e.Time.Format("15:04:05"))

		var levelStr string
		switch strings.ToLower(e.Type) {
		case "info":
			levelStr = logInfoStyle.Render("[INF]")
		case "warning", "warn":
			levelStr = logWarnStyle.Render("[WRN]")
		case "error":
			levelStr = logErrorStyle.Render("[ERR]")
		case "debug":
			levelStr = logDebugStyle.Render("[DBG]")
		default:
			levelStr = logDebugStyle.Render("[" + strings.ToUpper(e.Type[:3]) + "]")
		}

		payload := e.Payload
		maxLen := l.width - 20
		if maxLen > 0 && len(payload) > maxLen {
			payload = payload[:maxLen-3] + "..."
		}

		line := fmt.Sprintf("  %s %s %s", timeStr, levelStr, logPayloadStyle.Render(payload))
		lines = append(lines, line)
	}

	// Scroll indicator
	scrollInfo := ""
	if len(filtered) > visible {
		scrollInfo = logFilterStyle.Render(
			fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(filtered)),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		strings.Join(lines, "\n"),
		scrollInfo,
	)
}
