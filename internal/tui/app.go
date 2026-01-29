package tui

import (
	"fmt"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewType represents the current view
type ViewType int

const (
	QueueListView ViewType = iota
	IssueListView
	IssueDetailView
	TransitionView
	CommentView
	AssignView
	ConfirmView
	HelpView
	WorkflowView
)

// Model is the root application model
type Model struct {
	cfg         *config.Config
	client      *jira.Client
	keys        KeyMap
	width       int
	height      int
	currentView ViewType
	prevView    ViewType // For returning from confirm modal
	err         error

	// View models
	queuesView     *QueuesModel
	issuesView     *IssuesModel
	detailView     *DetailModel
	transitionView *TransitionModel
	commentView    *CommentModel
	assignView     *AssignModel
	confirmView    *ConfirmModel
	helpView       *HelpModel
	workflowView   *WorkflowModel

	// Command bar
	cmdBar *CmdBarModel

	// ZZ quit tracking
	waitingForZ bool
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
		cmdBar:      NewCmdBarModel(80),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return m.queuesView.Init()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size first
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.cmdBar.width = wsm.Width
	}

	// Handle command bar results FIRST (before routing to cmdbar)
	if result, ok := msg.(CmdBarResult); ok {
		if result.Mode == CmdBarCommand && !result.Aborted {
			// Execute command
			if execMsg := ExecuteCommand(result.Input); execMsg != nil {
				return m, func() tea.Msg { return execMsg }
			}
		} else if result.Mode == CmdBarSearch {
			// Apply or clear search filter
			return m, m.applySearch(result.Input)
		}
		return m, nil
	}

	// If command bar is actively being edited, route to it
	if m.cmdBar.IsActive() {
		var cmd tea.Cmd
		m.cmdBar, cmd = m.cmdBar.Update(msg)
		return m, cmd
	}

	// Handle close help (must be before HelpView routing)
	if _, ok := msg.(closeHelpMsg); ok {
		m.currentView = m.prevView
		m.helpView = nil
		return m, nil
	}

	// Handle help modal
	if m.currentView == HelpView {
		var cmd tea.Cmd
		m.helpView, cmd = m.helpView.Update(msg)
		return m, cmd
	}

	// Handle confirm modal
	if m.currentView == ConfirmView {
		var cmd tea.Cmd
		m.confirmView, cmd = m.confirmView.Update(msg)
		return m, cmd
	}

	// Handle confirm results
	if result, ok := msg.(ConfirmResult); ok {
		return m.handleConfirmResult(result)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key bindings (when not in command bar)

		// ZZ to quit
		if msg.String() == "Z" {
			if m.waitingForZ {
				return m, tea.Quit
			}
			m.waitingForZ = true
			return m, nil
		}
		// Reset Z wait on any other key
		if m.waitingForZ && msg.String() != "Z" {
			m.waitingForZ = false
		}

		// Ctrl+C force quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Command mode
		if key.Matches(msg, m.keys.Command) {
			return m, m.cmdBar.Open(CmdBarCommand)
		}

		// Search mode
		if key.Matches(msg, m.keys.Search) {
			return m, m.cmdBar.Open(CmdBarSearch)
		}

		// Clear search with esc when showing search results
		if key.Matches(msg, m.keys.Back) && m.cmdBar.Mode() == CmdBarShowSearch {
			m.cmdBar.ClearSearch()
			return m, m.applySearch("")
		}

		// Help modal
		if key.Matches(msg, m.keys.Help) {
			m.prevView = m.currentView
			m.helpView = NewHelpModel(m.width, m.height)
			m.currentView = HelpView
			return m, nil
		}

		// Rename (R) - only in detail view
		if key.Matches(msg, m.keys.Rename) && m.currentView == IssueDetailView {
			m.prevView = m.currentView
			m.confirmView = NewInputConfirmModel(
				"rename",
				"Rename Issue",
				fmt.Sprintf("Enter new summary for %s:", m.detailView.issue.Key),
				"New summary...",
				m.detailView.issue.Fields.Summary,
				m.width,
				m.height,
			)
			m.currentView = ConfirmView
			return m, m.confirmView.Init()
		}

		// Workflow (w) - only in detail view
		if key.Matches(msg, m.keys.Workflow) && m.currentView == IssueDetailView && m.detailView != nil {
			m.workflowView = NewWorkflowModel(
				m.cfg,
				m.detailView.issue,
				m.detailView.proformaForms,
				m.detailView.comments,
				m.keys,
				m.width,
				m.height,
			)
			m.currentView = WorkflowView
			return m, nil
		}

	case errorMsg:
		m.err = msg.err
		return m, nil

	case queueSelectedMsg:
		m.issuesView = NewIssuesModel(m.client, m.cfg.Project, msg.queue, m.keys, m.cfg.Username, m.width, m.height)
		m.currentView = IssueListView
		return m, m.issuesView.Init()

	case issueSelectedMsg:
		m.detailView = NewDetailModel(m.client, msg.issue, m.keys, m.width, m.height)
		m.currentView = IssueDetailView
		return m, m.detailView.Init()

	case backToQueuesMsg:
		m.currentView = QueueListView
		m.issuesView = nil
		return m, m.queuesView.Refresh()

	case backToIssuesMsg:
		m.currentView = IssueListView
		m.detailView = nil
		return m, m.issuesView.Refresh()

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
		return m, m.detailView.Refresh()

	case transitionCompletedMsg:
		m.currentView = IssueDetailView
		m.transitionView = nil
		return m, m.detailView.Refresh()

	case commentAddedMsg:
		m.currentView = IssueDetailView
		m.commentView = nil
		return m, m.detailView.Refresh()

	case openAssignMsg:
		m.assignView = NewAssignModel(m.client, msg.issue, m.keys, m.cfg.Username, m.width, m.height)
		m.currentView = AssignView
		return m, m.assignView.Init()

	case assignCompletedMsg:
		m.currentView = IssueDetailView
		m.assignView = nil
		return m, m.detailView.Refresh()

	case renameCompletedMsg:
		m.currentView = IssueDetailView
		return m, m.detailView.Refresh()

	case workflowCompletedMsg:
		m.currentView = IssueDetailView
		m.workflowView = nil
		return m, nil
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
	case AssignView:
		m.assignView, cmd = m.assignView.Update(msg)
	case WorkflowView:
		m.workflowView, cmd = m.workflowView.Update(msg)
	}

	return m, cmd
}

// handleConfirmResult handles results from confirmation modals
func (m *Model) handleConfirmResult(result ConfirmResult) (tea.Model, tea.Cmd) {
	m.currentView = m.prevView
	m.confirmView = nil

	if !result.Confirmed {
		return m, nil
	}

	switch result.ID {
	case "rename":
		if result.Input != "" && m.detailView != nil {
			return m, m.renameIssue(m.detailView.issue.Key, result.Input)
		}
	}

	return m, nil
}

// renameIssue renames the current issue
func (m *Model) renameIssue(issueKey, newSummary string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.UpdateIssueSummary(issueKey, newSummary)
		if err != nil {
			return errorMsg{err: err}
		}
		return renameCompletedMsg{}
	}
}

// applySearch applies search filter to the current view
func (m *Model) applySearch(query string) tea.Cmd {
	switch m.currentView {
	case IssueListView:
		if m.issuesView != nil {
			m.issuesView.SetSearchFilter(query)
		}
	case IssueDetailView:
		if m.detailView != nil {
			m.detailView.SetSearchFilter(query)
		}
	}
	return nil
}

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	var content string

	switch m.currentView {
	case QueueListView:
		content = m.queuesView.View()
	case IssueListView:
		content = m.issuesView.View()
	case IssueDetailView:
		content = m.detailView.View()
	case TransitionView:
		content = m.transitionView.View()
	case CommentView:
		content = m.commentView.View()
	case AssignView:
		content = m.assignView.View()
	case ConfirmView:
		content = m.confirmView.View()
	case HelpView:
		content = m.helpView.View()
	case WorkflowView:
		content = m.workflowView.View()
	default:
		content = "Unknown view"
	}

	// Add command bar and hints at bottom if visible
	if m.cmdBar.IsVisible() {
		// Get hints based on mode
		var hints string
		if m.cmdBar.Mode() == CmdBarShowSearch {
			hints = "enter/esc clear search"
		} else {
			hints = "enter submit • esc cancel"
		}
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.cmdBar.ViewWithHints(hints))
	}

	return content
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
type openAssignMsg struct{ issue jira.Issue }
type renameCompletedMsg struct{}
