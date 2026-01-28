package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
)

// ViewType represents the current view
type ViewType int

const (
	QueueListView ViewType = iota
	IssueListView
	IssueDetailView
	TransitionView
	CommentView
)

// Model is the root application model
type Model struct {
	cfg        *config.Config
	client     *jira.Client
	keys       KeyMap
	width      int
	height     int
	currentView ViewType
	err        error

	// View models
	queuesView     *QueuesModel
	issuesView     *IssuesModel
	detailView     *DetailModel
	transitionView *TransitionModel
	commentView    *CommentModel
}

// NewModel creates a new application model
func NewModel(cfg *config.Config) Model {
	client := jira.NewClient(cfg)
	keys := DefaultKeyMap()

	return Model{
		cfg:         cfg,
		client:      client,
		keys:        keys,
		currentView: QueueListView,
		queuesView:  NewQueuesModel(client, cfg, cfg.Project, keys),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return m.queuesView.Init()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Don't return early - let child views handle this too

	case tea.KeyMsg:
		// Global key bindings
		switch msg.String() {
		case "ctrl+c":
			// Force quit from anywhere
			return m, tea.Quit
		}

	case errorMsg:
		m.err = msg.err
		return m, nil

	case queueSelectedMsg:
		// Navigate to issue list
		m.issuesView = NewIssuesModel(m.client, m.cfg.Project, msg.queue, m.keys, m.cfg.Username, m.width, m.height)
		m.currentView = IssueListView
		return m, m.issuesView.Init()

	case issueSelectedMsg:
		// Navigate to issue detail
		m.detailView = NewDetailModel(m.client, msg.issue, m.keys)
		m.currentView = IssueDetailView
		return m, m.detailView.Init()

	case backToQueuesMsg:
		m.currentView = QueueListView
		m.issuesView = nil
		return m, nil

	case backToIssuesMsg:
		m.currentView = IssueListView
		m.detailView = nil
		return m, nil

	case openTransitionMsg:
		m.transitionView = NewTransitionModel(m.client, msg.issue, m.keys)
		m.currentView = TransitionView
		return m, m.transitionView.Init()

	case openCommentMsg:
		m.commentView = NewCommentModel(m.client, msg.issue, m.keys)
		m.currentView = CommentView
		return m, nil

	case backToDetailMsg:
		m.currentView = IssueDetailView
		m.transitionView = nil
		m.commentView = nil
		// Refresh the detail view
		return m, m.detailView.Refresh()

	case transitionCompletedMsg:
		m.currentView = IssueDetailView
		m.transitionView = nil
		return m, m.detailView.Refresh()

	case commentAddedMsg:
		m.currentView = IssueDetailView
		m.commentView = nil
		return m, m.detailView.Refresh()
	}

	// Route to current view
	var cmd tea.Cmd
	switch m.currentView {
	case QueueListView:
		m.queuesView, cmd = m.queuesView.Update(msg)
	case IssueListView:
		m.issuesView, cmd = m.issuesView.Update(msg)
	case IssueDetailView:
		m.detailView, cmd = m.detailView.Update(msg)
	case TransitionView:
		m.transitionView, cmd = m.transitionView.Update(msg)
	case CommentView:
		m.commentView, cmd = m.commentView.Update(msg)
	}

	return m, cmd
}

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.currentView {
	case QueueListView:
		return m.queuesView.View()
	case IssueListView:
		return m.issuesView.View()
	case IssueDetailView:
		return m.detailView.View()
	case TransitionView:
		return m.transitionView.View()
	case CommentView:
		return m.commentView.View()
	}

	return "Unknown view"
}

// Custom messages for navigation
type errorMsg struct{ err error }
type queueSelectedMsg struct{ queue jira.Queue }
type issueSelectedMsg struct{ issue jira.Issue }
type backToQueuesMsg struct{}
type backToIssuesMsg struct{}
type openTransitionMsg struct{ issue jira.Issue }
type openCommentMsg struct{ issue jira.Issue }
type backToDetailMsg struct{}
type transitionCompletedMsg struct{}
type commentAddedMsg struct{}
