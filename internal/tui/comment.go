package tui

import (
	"fmt"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// CommentModel handles adding comments to issues
type CommentModel struct {
	client    *jira.Client
	keys      KeyMap
	issue     jira.Issue
	textarea  textarea.Model
	submitting bool
	err       error
}

// NewCommentModel creates a new comment model
func NewCommentModel(client *jira.Client, issue jira.Issue, keys KeyMap) *CommentModel {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.Focus()

	return &CommentModel{
		client:   client,
		keys:     keys,
		issue:    issue,
		textarea: ta,
	}
}

// submitComment submits the comment
func (m *CommentModel) submitComment() tea.Cmd {
	return func() tea.Msg {
		err := m.client.AddComment(m.issue.Key, m.textarea.Value())
		if err != nil {
			return errorMsg{err: err}
		}
		return commentAddedMsg{}
	}
}

// Update handles messages
func (m *CommentModel) Update(msg tea.Msg) (*CommentModel, tea.Cmd) {
	switch msg := msg.(type) {
	case commentAddedMsg:
		// Comment added, return to detail view
		return m, func() tea.Msg {
			return commentAddedMsg{}
		}

	case tea.KeyMsg:
		if m.submitting {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Submit):
			if m.textarea.Value() != "" {
				m.submitting = true
				return m, m.submitComment()
			}
			return m, nil

		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg {
				return backToDetailMsg{}
			}
		}
	}

	// Update textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the comment input view
func (m *CommentModel) View() string {
	if m.submitting {
		return SpinnerStyle.Render("Submitting comment...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	s := HeaderStyle.Render(fmt.Sprintf("Add Comment - %s", m.issue.Key)) + "\n\n"
	s += m.textarea.View() + "\n\n"
	s += HelpStyle.Render("ctrl+s submit • esc cancel")

	return s
}
