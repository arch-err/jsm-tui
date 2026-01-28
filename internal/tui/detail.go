package tui

import (
	"fmt"
	"strings"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailModel handles the issue detail view
type DetailModel struct {
	client   *jira.Client
	keys     KeyMap
	issue    jira.Issue
	comments []jira.Comment
	viewport viewport.Model
	loading  bool
	err      error
}

// NewDetailModel creates a new issue detail model
func NewDetailModel(client *jira.Client, issue jira.Issue, keys KeyMap) *DetailModel {
	vp := viewport.New(80, 30)
	return &DetailModel{
		client:   client,
		keys:     keys,
		issue:    issue,
		viewport: vp,
		loading:  true,
	}
}

type issueDetailLoadedMsg struct {
	issue    jira.Issue
	comments []jira.Comment
}

// Init initializes the view
func (m *DetailModel) Init() tea.Cmd {
	return m.fetchDetail()
}

// fetchDetail loads full issue details and comments
func (m *DetailModel) fetchDetail() tea.Cmd {
	return func() tea.Msg {
		issue, err := m.client.GetIssue(m.issue.Key)
		if err != nil {
			return errorMsg{err: err}
		}

		comments, err := m.client.GetComments(m.issue.Key)
		if err != nil {
			return errorMsg{err: err}
		}

		return issueDetailLoadedMsg{
			issue:    *issue,
			comments: comments,
		}
	}
}

// Refresh reloads the issue details
func (m *DetailModel) Refresh() tea.Cmd {
	m.loading = true
	return m.fetchDetail()
}

// Update handles messages
func (m *DetailModel) Update(msg tea.Msg) (*DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case issueDetailLoadedMsg:
		m.issue = msg.issue
		m.comments = msg.comments
		m.loading = false
		m.viewport.SetContent(m.renderContent())
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg {
				return backToIssuesMsg{}
			}

		case key.Matches(msg, m.keys.Transition):
			return m, func() tea.Msg {
				return openTransitionMsg{issue: m.issue}
			}

		case key.Matches(msg, m.keys.AddComment):
			return m, func() tea.Msg {
				return openCommentMsg{issue: m.issue}
			}

		case key.Matches(msg, m.keys.Refresh):
			return m, m.Refresh()
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// renderContent renders the issue details
func (m *DetailModel) renderContent() string {
	var s strings.Builder

	// Issue header
	s.WriteString(TitleStyle.Render(fmt.Sprintf("%s: %s", m.issue.Key, m.issue.Fields.Summary)))
	s.WriteString("\n\n")

	// Status, Priority, Assignee
	statusStyle := GetStatusStyle(m.issue.Fields.Status.StatusCategory.Name)
	priorityStyle := GetPriorityStyle(m.issue.Fields.Priority.Name)

	s.WriteString(fmt.Sprintf("Status: %s  ", statusStyle.Render(m.issue.Fields.Status.Name)))
	s.WriteString(fmt.Sprintf("Priority: %s\n", priorityStyle.Render(m.issue.Fields.Priority.Name)))

	assignee := "Unassigned"
	if m.issue.Fields.Assignee != nil {
		assignee = m.issue.Fields.Assignee.DisplayName
	}
	s.WriteString(fmt.Sprintf("Assignee: %s\n", assignee))

	if m.issue.Fields.Reporter != nil {
		s.WriteString(fmt.Sprintf("Reporter: %s\n", m.issue.Fields.Reporter.DisplayName))
	}

	s.WriteString(fmt.Sprintf("Type: %s\n", m.issue.Fields.IssueType.Name))
	s.WriteString(fmt.Sprintf("Created: %s\n", m.issue.Fields.Created))
	s.WriteString(fmt.Sprintf("Updated: %s\n", m.issue.Fields.Updated))

	// Description
	s.WriteString("\n")
	s.WriteString(TitleStyle.Render("Description"))
	s.WriteString("\n")
	if m.issue.Fields.Description != "" {
		s.WriteString(m.issue.Fields.Description)
	} else {
		s.WriteString("No description provided.")
	}
	s.WriteString("\n\n")

	// Comments
	s.WriteString(TitleStyle.Render(fmt.Sprintf("Comments (%d)", len(m.comments))))
	s.WriteString("\n\n")

	if len(m.comments) == 0 {
		s.WriteString("No comments yet.\n")
	} else {
		for _, comment := range m.comments {
			s.WriteString(fmt.Sprintf("─────────────────────────────────────────────────\n"))
			s.WriteString(fmt.Sprintf("%s • %s\n", comment.Author.DisplayName, comment.Created))
			s.WriteString(fmt.Sprintf("%s\n\n", comment.Body))
		}
	}

	return s.String()
}

// View renders the detail view
func (m *DetailModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading issue details...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	s := HeaderStyle.Render(fmt.Sprintf("Issue Detail - %s", m.issue.Key)) + "\n\n"
	s += m.viewport.View()
	s += "\n\n" + HelpStyle.Render("t transition • c comment • esc back • r refresh • ↑/↓ scroll")

	return s
}
