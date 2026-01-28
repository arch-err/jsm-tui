package tui

import (
	"fmt"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// IssuesModel handles the issue list view
type IssuesModel struct {
	client        *jira.Client
	projectKey    string
	queue         jira.Queue
	keys          KeyMap
	issues        []jira.Issue
	selectedIndex int
	loading       bool
	err           error
}

// NewIssuesModel creates a new issue list model
func NewIssuesModel(client *jira.Client, projectKey string, queue jira.Queue, keys KeyMap) *IssuesModel {
	return &IssuesModel{
		client:     client,
		projectKey: projectKey,
		queue:      queue,
		keys:       keys,
		loading:    true,
	}
}

type issuesLoadedMsg struct{ issues []jira.Issue }

// Init initializes the view
func (m *IssuesModel) Init() tea.Cmd {
	return m.fetchIssues()
}

// fetchIssues loads issues from the API
func (m *IssuesModel) fetchIssues() tea.Cmd {
	return func() tea.Msg {
		issues, err := m.client.GetQueueIssues(m.projectKey, m.queue.ID, 0, 50)
		if err != nil {
			return errorMsg{err: err}
		}
		return issuesLoadedMsg{issues: issues}
	}
}

// Update handles messages
func (m *IssuesModel) Update(msg tea.Msg) (*IssuesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.issues)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if len(m.issues) > 0 {
				return m, func() tea.Msg {
					return issueSelectedMsg{issue: m.issues[m.selectedIndex]}
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg {
				return backToQueuesMsg{}
			}

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, m.fetchIssues()
		}
	}

	return m, nil
}

// View renders the issue list
func (m *IssuesModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading issues...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.issues) == 0 {
		return fmt.Sprintf("No issues in queue: %s", m.queue.Name)
	}

	s := HeaderStyle.Render(fmt.Sprintf("Issues - %s > %s", m.projectKey, m.queue.Name)) + "\n\n"

	// Table header
	s += TableHeaderStyle.Render(fmt.Sprintf("%-12s %-40s %-15s %-10s %-15s", "Key", "Summary", "Status", "Priority", "Assignee")) + "\n"

	for i, issue := range m.issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		statusStyle := GetStatusStyle(issue.Fields.Status.StatusCategory.Name)
		priorityStyle := GetPriorityStyle(issue.Fields.Priority.Name)

		// Truncate summary if too long
		summary := issue.Fields.Summary
		if len(summary) > 38 {
			summary = summary[:35] + "..."
		}

		// Truncate assignee if too long
		if len(assignee) > 13 {
			assignee = assignee[:10] + "..."
		}

		line := fmt.Sprintf("%-12s %-40s %-15s %-10s %-15s",
			issue.Key,
			summary,
			statusStyle.Render(issue.Fields.Status.Name),
			priorityStyle.Render(issue.Fields.Priority.Name),
			assignee,
		)

		if i == m.selectedIndex {
			line = SelectedStyle.Render(line)
		}

		s += line + "\n"
	}

	s += "\n" + HelpStyle.Render("↑/k up • ↓/j down • enter open • esc back • r refresh")

	return s
}
