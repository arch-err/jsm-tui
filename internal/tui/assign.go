package tui

import (
	"fmt"
	"strings"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AssignOption represents an assignable option (user or unassign)
type AssignOption struct {
	User        *jira.User
	DisplayName string
	IsUnassign  bool
}

// AssignModel handles the user assignment popup
type AssignModel struct {
	client        *jira.Client
	keys          KeyMap
	issue         jira.Issue
	currentUser   string       // username of the current user
	meUser        *jira.User   // cached user object for current user
	lastUsers     []jira.User  // last search results for rebuilding options
	searchQuery   string
	options       []AssignOption
	selectedIndex int
	searchMode    bool
	searching     bool
	assigning     bool
	err           error
	width         int
	height        int
}

// NewAssignModel creates a new assignment model
func NewAssignModel(client *jira.Client, issue jira.Issue, keys KeyMap, currentUser string, width, height int) *AssignModel {
	return &AssignModel{
		client:      client,
		keys:        keys,
		issue:       issue,
		currentUser: currentUser,
		searchQuery: "",
		searchMode:  false,
		width:       width,
		height:      height,
	}
}

type usersSearchedMsg struct{ users []jira.User }
type meUserFoundMsg struct{ user *jira.User }
type assignCompletedMsg struct{}

// Init initializes the view
func (m *AssignModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.searchUsers("")}
	if m.currentUser != "" && m.issue.Fields.Assignee == nil {
		cmds = append(cmds, m.findMeUser())
	}
	return tea.Batch(cmds...)
}

// findMeUser searches for the current user to cache their User object
func (m *AssignModel) findMeUser() tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.SearchAssignableUsers(m.issue.Key, m.currentUser)
		if err != nil {
			return meUserFoundMsg{user: nil}
		}
		for i := range users {
			if users[i].DisplayName == m.currentUser {
				return meUserFoundMsg{user: &users[i]}
			}
		}
		return meUserFoundMsg{user: nil}
	}
}

// searchUsers searches for assignable users
func (m *AssignModel) searchUsers(query string) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.SearchAssignableUsers(m.issue.Key, query)
		if err != nil {
			return errorMsg{err: err}
		}
		return usersSearchedMsg{users: users}
	}
}

// assignUser assigns the issue to the selected user
func (m *AssignModel) assignUser(option AssignOption) tea.Cmd {
	return func() tea.Msg {
		var err error
		if option.IsUnassign {
			err = m.client.AssignIssue(m.issue.Key, nil)
		} else {
			err = m.client.AssignIssue(m.issue.Key, option.User)
		}
		if err != nil {
			return errorMsg{err: err}
		}
		return assignCompletedMsg{}
	}
}

// Update handles messages
func (m *AssignModel) Update(msg tea.Msg) (*AssignModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case usersSearchedMsg:
		m.searching = false
		m.updateOptionsWithUsers(msg.users)
		return m, nil

	case meUserFoundMsg:
		m.meUser = msg.user
		if len(m.options) > 0 || len(m.lastUsers) > 0 {
			m.rebuildOptions()
		}
		return m, nil

	case assignCompletedMsg:
		return m, func() tea.Msg {
			return assignCompletedMsg{}
		}

	case errorMsg:
		m.err = msg.err
		m.searching = false
		m.assigning = false
		return m, nil

	case tea.KeyMsg:
		if m.assigning {
			return m, nil
		}

		// Search mode - handle text input manually
		if m.searchMode {
			keyStr := msg.String()
			switch keyStr {
			case "esc":
				m.searchMode = false
				return m, nil
			case "enter":
				m.searchMode = false
				if len(m.options) > 0 {
					m.assigning = true
					return m, m.assignUser(m.options[m.selectedIndex])
				}
				return m, nil
			case "up":
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
				return m, nil
			case "down":
				if m.selectedIndex < len(m.options)-1 {
					m.selectedIndex++
				}
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.searching = true
					m.selectedIndex = 0
					return m, m.searchUsers(m.searchQuery)
				}
				return m, nil
			default:
				// For printable characters, append to search query
				if len(keyStr) == 1 {
					char := keyStr[0]
					if char >= 32 && char < 127 {
						m.searchQuery += keyStr
						m.searching = true
						m.selectedIndex = 0
						return m, m.searchUsers(m.searchQuery)
					}
				}
			}
			return m, nil
		}

		// Navigation mode
		switch {
		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg {
				return backToDetailMsg{}
			}

		case msg.String() == "/":
			// Enter search mode
			m.searchMode = true
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.options)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if len(m.options) > 0 {
				m.assigning = true
				return m, m.assignUser(m.options[m.selectedIndex])
			}
			return m, nil
		}
	}

	return m, nil
}

// updateOptionsWithUsers updates options based on search results
func (m *AssignModel) updateOptionsWithUsers(users []jira.User) {
	m.lastUsers = users
	m.rebuildOptions()
}

// rebuildOptions rebuilds the options list using lastUsers and meUser
func (m *AssignModel) rebuildOptions() {
	m.options = []AssignOption{}

	// If currently assigned, show "Unassign" option first
	if m.issue.Fields.Assignee != nil {
		m.options = append(m.options, AssignOption{
			DisplayName: "Unassign",
			IsUnassign:  true,
		})
	}

	// Add current user at top if unassigned
	if m.issue.Fields.Assignee == nil && m.meUser != nil {
		m.options = append(m.options, AssignOption{
			User:        m.meUser,
			DisplayName: m.meUser.DisplayName,
		})
	}

	// Add other users from search results (skip "me" if already added)
	for i := range m.lastUsers {
		user := &m.lastUsers[i]
		if m.meUser != nil && user.Name == m.meUser.Name {
			continue // Skip, already added as "Me"
		}
		m.options = append(m.options, AssignOption{
			User:        user,
			DisplayName: user.DisplayName,
		})
	}

	// Reset selection if out of bounds
	if m.selectedIndex >= len(m.options) {
		m.selectedIndex = 0
	}
}

// View renders the assignment popup
func (m *AssignModel) View() string {
	if m.assigning {
		return m.renderPopup(SpinnerStyle.Render("Assigning..."))
	}

	var content strings.Builder

	// Title
	content.WriteString(TitleStyle.Render("Assign Issue"))
	content.WriteString("\n\n")

	// Current assignee info
	currentAssignee := "Unassigned"
	if m.issue.Fields.Assignee != nil {
		currentAssignee = m.issue.Fields.Assignee.DisplayName
	}
	content.WriteString(fmt.Sprintf("Current: %s\n\n", currentAssignee))

	// Search input
	if m.searchMode {
		content.WriteString("> ")
		content.WriteString(m.searchQuery)
		content.WriteString("█") // Cursor
	} else {
		if m.searchQuery == "" {
			content.WriteString(HelpStyle.Render("/ to search"))
		} else {
			content.WriteString(fmt.Sprintf("Search: \"%s\" (/ to edit)", m.searchQuery))
		}
	}
	content.WriteString("\n\n")

	// Options list
	if m.searching {
		content.WriteString(HelpStyle.Render("Searching..."))
	} else if len(m.options) == 0 {
		content.WriteString(HelpStyle.Render("No users found"))
	} else {
		for i, opt := range m.options {
			line := "  " + opt.DisplayName
			if i == m.selectedIndex {
				line = SelectedStyle.Render(line)
			}
			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n")
	if m.searchMode {
		content.WriteString(HelpStyle.Render("enter select • esc back"))
	} else {
		content.WriteString(HelpStyle.Render("/ search • enter assign • esc cancel"))
	}

	return m.renderPopup(content.String())
}

// renderPopup renders content in a centered popup box
func (m *AssignModel) renderPopup(content string) string {
	// Popup style
	popupWidth := 50
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(popupWidth)

	popup := popupStyle.Render(content)

	// Center the popup
	popupHeight := strings.Count(popup, "\n") + 1
	verticalPadding := (m.height - popupHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	horizontalPadding := (m.width - popupWidth - 4) / 2
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	// Build centered view
	var view strings.Builder
	for i := 0; i < verticalPadding; i++ {
		view.WriteString("\n")
	}

	lines := strings.Split(popup, "\n")
	for _, line := range lines {
		view.WriteString(strings.Repeat(" ", horizontalPadding))
		view.WriteString(line)
		view.WriteString("\n")
	}

	return view.String()
}
