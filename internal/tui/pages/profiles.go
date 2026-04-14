package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/boomyao/clash-cli/internal/profile"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	profileTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	profileActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	profileInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB"))

	profileSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	profileMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	profileTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4"))

	profileInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(1, 2)

	profileKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)
)

type profileMode int

const (
	profileModeList profileMode = iota
	profileModeAddURL
	profileModeAddName
)

// ProfilesPage manages subscription profiles.
type ProfilesPage struct {
	width, height int
	profiles      []profile.Profile
	activeID      string
	cursor        int
	scroll        int
	mode          profileMode
	inputText     string
	addURL        string
	statusMsg     string

	profileMgr *profile.Manager
}

// ProfileActionMsg reports the result of a profile action.
type ProfileActionMsg struct {
	Action string
	Err    error
}

// ProfilesRefreshMsg triggers a profiles reload.
type ProfilesRefreshMsg struct{}

func NewProfilesPage() *ProfilesPage {
	return &ProfilesPage{}
}

func (p *ProfilesPage) SetProfileManager(mgr *profile.Manager) {
	p.profileMgr = mgr
	p.refreshProfiles()
}

func (p *ProfilesPage) refreshProfiles() {
	if p.profileMgr != nil {
		p.profiles = p.profileMgr.List()
		p.activeID = p.profileMgr.ActiveID()
	}
}

func (p *ProfilesPage) Title() string { return "Profiles" }
func (p *ProfilesPage) ShortHelp() string {
	if p.mode != profileModeList {
		return "type input │ enter: confirm │ esc: cancel"
	}
	return "a: add │ u: update │ d: delete │ enter: activate"
}

func (p *ProfilesPage) Init() tea.Cmd { return nil }

func (p *ProfilesPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case ProfileActionMsg:
		if msg.Err != nil {
			p.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			p.statusMsg = msg.Action
		}
		p.refreshProfiles()

	case ProfilesRefreshMsg:
		p.refreshProfiles()

	case tea.KeyMsg:
		// Handle input modes
		if p.mode == profileModeAddURL {
			return p.handleURLInput(msg)
		}
		if p.mode == profileModeAddName {
			return p.handleNameInput(msg)
		}

		// List mode
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.profiles)-1 {
				p.cursor++
			}
		case "a":
			p.mode = profileModeAddURL
			p.inputText = ""
			p.addURL = ""
		case "enter":
			return p, p.activateProfile()
		case "u":
			return p, p.updateProfile()
		case "d":
			return p, p.deleteProfile()
		}
	}

	return p, nil
}

func (p *ProfilesPage) handleURLInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		p.mode = profileModeList
		p.inputText = ""
		return p, nil
	case tea.KeyEnter:
		p.addURL = p.inputText
		p.inputText = ""
		p.mode = profileModeAddName
		return p, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(p.inputText) > 0 {
			// Trim one rune (handles multibyte)
			r := []rune(p.inputText)
			p.inputText = string(r[:len(r)-1])
		}
		return p, nil
	case tea.KeyRunes, tea.KeySpace:
		// Single key OR pasted text both arrive here
		p.inputText += string(msg.Runes)
		return p, nil
	}
	return p, nil
}

func (p *ProfilesPage) handleNameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		p.mode = profileModeList
		p.inputText = ""
		return p, nil
	case tea.KeyEnter:
		name := p.inputText
		url := p.addURL
		p.mode = profileModeList
		p.inputText = ""

		if name == "" {
			name = "Subscription"
		}

		mgr := p.profileMgr
		if mgr == nil {
			return p, nil
		}

		profileType := "remote"
		if url == "" {
			profileType = "local"
		}

		return p, func() tea.Msg {
			_, err := mgr.Add(name, profileType, url)
			if err != nil {
				return ProfileActionMsg{Action: "Add failed", Err: err}
			}
			return ProfileActionMsg{Action: fmt.Sprintf("Added profile: %s", name)}
		}
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(p.inputText) > 0 {
			r := []rune(p.inputText)
			p.inputText = string(r[:len(r)-1])
		}
		return p, nil
	case tea.KeyRunes, tea.KeySpace:
		p.inputText += string(msg.Runes)
		return p, nil
	}
	return p, nil
}

func (p *ProfilesPage) activateProfile() tea.Cmd {
	if p.cursor >= len(p.profiles) || p.profileMgr == nil {
		return nil
	}
	prof := p.profiles[p.cursor]
	mgr := p.profileMgr
	return func() tea.Msg {
		err := mgr.SetActive(prof.ID)
		if err != nil {
			return ProfileActionMsg{Action: "Activate failed", Err: err}
		}
		return ProfileActionMsg{Action: fmt.Sprintf("Activated: %s", prof.Name)}
	}
}

func (p *ProfilesPage) updateProfile() tea.Cmd {
	if p.cursor >= len(p.profiles) || p.profileMgr == nil {
		return nil
	}
	prof := p.profiles[p.cursor]
	if !prof.IsRemote() {
		p.statusMsg = "Only remote profiles can be updated"
		return nil
	}
	mgr := p.profileMgr
	return func() tea.Msg {
		err := mgr.Update(prof.ID)
		if err != nil {
			return ProfileActionMsg{Action: "Update failed", Err: err}
		}
		return ProfileActionMsg{Action: fmt.Sprintf("Updated: %s", prof.Name)}
	}
}

func (p *ProfilesPage) deleteProfile() tea.Cmd {
	if p.cursor >= len(p.profiles) || p.profileMgr == nil {
		return nil
	}
	prof := p.profiles[p.cursor]
	mgr := p.profileMgr
	return func() tea.Msg {
		err := mgr.Delete(prof.ID)
		if err != nil {
			return ProfileActionMsg{Action: "Delete failed", Err: err}
		}
		return ProfileActionMsg{Action: fmt.Sprintf("Deleted: %s", prof.Name)}
	}
}

func (p *ProfilesPage) View() string {
	if p.width == 0 {
		return "Loading..."
	}

	contentWidth := p.width - 6

	// Show input modal if adding
	if p.mode == profileModeAddURL {
		return p.renderAddModal(contentWidth, "Enter subscription URL:", p.inputText)
	}
	if p.mode == profileModeAddName {
		return p.renderAddModal(contentWidth, "Enter profile name:", p.inputText)
	}

	// Title
	title := profileTitleStyle.Render(fmt.Sprintf(" Profiles (%d)", len(p.profiles)))

	if len(p.profiles) == 0 {
		empty := profileMetaStyle.Render("\n  No profiles yet. Press ") +
			profileKeyStyle.Render("[a]") +
			profileMetaStyle.Render(" to add a subscription.\n")
		return lipgloss.JoinVertical(lipgloss.Left, title, empty)
	}

	// Profile list
	var lines []string
	for i, prof := range p.profiles {
		isActive := prof.ID == p.activeID
		isCursor := i == p.cursor

		prefix := "  "
		if isActive {
			prefix = "● "
		}
		if isCursor {
			prefix = "> "
		}

		name := prof.Name
		typeBadge := profileTypeStyle.Render(fmt.Sprintf("(%s)", prof.Type))

		updatedStr := ""
		if !prof.UpdatedAt.IsZero() {
			ago := time.Since(prof.UpdatedAt)
			updatedStr = profileMetaStyle.Render(fmt.Sprintf("  Updated: %s ago", formatDuration(ago)))
		}

		line := prefix + name + " " + typeBadge + updatedStr

		switch {
		case isCursor:
			lines = append(lines, profileSelectedStyle.Render(line))
		case isActive:
			lines = append(lines, profileActiveStyle.Render(line))
		default:
			lines = append(lines, profileInactiveStyle.Render(line))
		}
	}

	content := strings.Join(lines, "\n")

	// Status message
	status := ""
	if p.statusMsg != "" {
		status = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Padding(0, 2).
			Render("  "+p.statusMsg)
	}

	// Help
	help := "\n" + profileMetaStyle.Render(
		"  "+profileKeyStyle.Render("[a]")+" add  "+
			profileKeyStyle.Render("[enter]")+" activate  "+
			profileKeyStyle.Render("[u]")+" update  "+
			profileKeyStyle.Render("[d]")+" delete",
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, content, help, status)
}

func (p *ProfilesPage) renderAddModal(width int, prompt, value string) string {
	return profileInputStyle.Width(width).Render(
		profileTitleStyle.Render("Add Profile") + "\n\n" +
			profileMetaStyle.Render(prompt) + "\n" +
			value + "█" + "\n\n" +
			profileMetaStyle.Render("enter: confirm │ esc: cancel"),
	)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
