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
	ActionView
	PreviewView
)

// ModelOptions contains optional configuration for the Model
type ModelOptions struct {
	InitialIssueKey string // If set, open directly to this issue
}

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
	actionView     *ActionModel
	previewView    *PreviewModel

	// Command bar
	cmdBar *CmdBarModel

	// ZZ quit tracking
	waitingForZ bool

	// Pending action data
	pendingRename string

	// Initial issue to load (from CLI)
	initialIssueKey string

	// Current user info (fetched from Jira)
	currentUsername string
}

// NewModel creates a new application model
func NewModel(cfg *config.Config) Model {
	return NewModelWithOptions(cfg, ModelOptions{})
}

// NewModelWithOptions creates a new application model with options
func NewModelWithOptions(cfg *config.Config, opts ModelOptions) Model {
	client := jira.NewClient(cfg)
	keys := DefaultKeyMap()

	return Model{
		cfg:             cfg,
		client:          client,
		keys:            keys,
		currentView:     QueueListView,
		queuesView:      NewQueuesModel(client, cfg, cfg.Project, keys),
		cmdBar:          NewCmdBarModel(80),
		initialIssueKey: opts.InitialIssueKey,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	// Fetch current user info
	fetchUser := func() tea.Msg {
		user, err := m.client.GetCurrentUser()
		if err != nil {
			// Fall back to config username if API fails
			return currentUserLoadedMsg{user: jira.User{DisplayName: m.currentUsername}}
		}
		return currentUserLoadedMsg{user: *user}
	}

	// If we have an initial issue key, load that issue directly
	if m.initialIssueKey != "" {
		return tea.Batch(fetchUser, m.loadInitialIssue(m.initialIssueKey))
	}
	return tea.Batch(fetchUser, m.queuesView.Init())
}

// loadInitialIssue fetches and opens a specific issue
func (m *Model) loadInitialIssue(issueKey string) tea.Cmd {
	return func() tea.Msg {
		issue, err := m.client.GetIssue(issueKey)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to load issue %s: %w", issueKey, err)}
		}
		return initialIssueLoadedMsg{issue: *issue}
	}
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

	// Handle confirm results FIRST (before routing to confirmView)
	if result, ok := msg.(ConfirmResult); ok {
		return m.handleConfirmResult(result)
	}

	// Handle confirm modal
	if m.currentView == ConfirmView {
		var cmd tea.Cmd
		m.confirmView, cmd = m.confirmView.Update(msg)
		return m, cmd
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

		// Check if we're in a modal view (don't allow command/search/help in modals)
		inModal := m.currentView == AssignView || m.currentView == TransitionView ||
			m.currentView == CommentView || m.currentView == ConfirmView ||
			m.currentView == HelpView || m.currentView == WorkflowView ||
			m.currentView == ActionView || m.currentView == PreviewView

		// Command mode (not in modals)
		if key.Matches(msg, m.keys.Command) && !inModal {
			return m, m.cmdBar.Open(CmdBarCommand)
		}

		// Search mode (not in modals)
		if key.Matches(msg, m.keys.Search) && !inModal {
			return m, m.cmdBar.Open(CmdBarSearch)
		}

		// Clear search with esc when showing search results
		if key.Matches(msg, m.keys.Back) && m.cmdBar.Mode() == CmdBarShowSearch {
			m.cmdBar.ClearSearch()
			return m, m.applySearch("")
		}

		// Help modal (not in modals)
		if key.Matches(msg, m.keys.Help) && !inModal {
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

		// Quick Action (A) - only in detail view
		if key.Matches(msg, m.keys.QuickAction) && m.currentView == IssueDetailView && m.detailView != nil {
			m.actionView = NewActionModel(
				m.client,
				m.detailView.issue,
				m.cfg.Actions,
				m.keys,
				m.width,
				m.height,
			)
			m.currentView = ActionView
			return m, nil
		}

	case errorMsg:
		m.err = msg.err
		return m, nil

	case currentUserLoadedMsg:
		m.currentUsername = msg.user.DisplayName
		return m, nil

	case browseIssueMsg:
		// Open current issue in browser
		if m.currentView == IssueDetailView && m.detailView != nil {
			url := fmt.Sprintf("%s/browse/%s", m.cfg.URL, m.detailView.issue.Key)
			OpenInBrowser(url)
		}
		return m, nil

	case queueSelectedMsg:
		m.issuesView = NewIssuesModel(m.client, m.cfg.Project, msg.queue, m.keys, m.currentUsername, m.width, m.height)
		m.currentView = IssueListView
		return m, m.issuesView.Init()

	case issueSelectedMsg:
		m.detailView = NewDetailModel(m.client, msg.issue, m.keys, m.width, m.height, m.currentUsername)
		m.currentView = IssueDetailView
		return m, m.detailView.Init()

	case initialIssueLoadedMsg:
		// Opened directly to an issue via CLI flag
		m.detailView = NewDetailModel(m.client, msg.issue, m.keys, m.width, m.height, m.currentUsername)
		m.currentView = IssueDetailView
		return m, m.detailView.Init()

	case backToQueuesMsg:
		m.currentView = QueueListView
		m.issuesView = nil
		return m, m.queuesView.Refresh()

	case backToIssuesMsg:
		m.detailView = nil
		// If opened directly to an issue (no issues view), go to queue list instead
		if m.issuesView == nil {
			m.currentView = QueueListView
			return m, m.queuesView.Refresh()
		}
		m.currentView = IssueListView
		return m, m.issuesView.Refresh()

	case openTransitionMsg:
		m.transitionView = NewTransitionModel(m.client, msg.issue, m.keys)
		m.currentView = TransitionView
		return m, m.transitionView.Init()

	case openCommentMsg:
		m.commentView = NewCommentModel(m.client, msg.issue, m.keys, m.width, m.height)
		m.currentView = CommentView
		return m, nil

	case openEditCommentMsg:
		m.commentView = NewEditCommentModel(m.client, msg.issue, m.keys, m.width, m.height, msg.commentID, msg.body)
		m.currentView = CommentView
		return m, nil

	case openPreviewMsg:
		m.previewView = NewPreviewModel(msg.title, msg.content, m.width, m.height)
		m.currentView = PreviewView
		return m, nil

	case closePreviewMsg:
		m.currentView = IssueDetailView
		m.previewView = nil
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

	case commentEditedMsg:
		m.currentView = IssueDetailView
		m.commentView = nil
		return m, m.detailView.Refresh()

	case openAssignMsg:
		m.assignView = NewAssignModel(m.client, msg.issue, m.keys, m.currentUsername, m.width, m.height)
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

	case actionCompletedMsg:
		m.currentView = IssueDetailView
		m.actionView = nil
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
	case AssignView:
		m.assignView, cmd = m.assignView.Update(msg)
	case WorkflowView:
		m.workflowView, cmd = m.workflowView.Update(msg)
	case ActionView:
		m.actionView, cmd = m.actionView.Update(msg)
	case PreviewView:
		m.previewView, cmd = m.previewView.Update(msg)
	}

	return m, cmd
}

// handleConfirmResult handles results from confirmation modals
func (m *Model) handleConfirmResult(result ConfirmResult) (tea.Model, tea.Cmd) {
	if !result.Confirmed {
		m.currentView = m.prevView
		m.confirmView = nil
		return m, nil
	}

	switch result.ID {
	case "rename":
		// First step: got the new name, now ask for confirmation
		if result.Input != "" && m.detailView != nil {
			m.confirmView = NewConfirmModel(
				"rename-confirm",
				"Confirm Rename",
				fmt.Sprintf("Rename issue %s to:\n\n\"%s\"?", m.detailView.issue.Key, result.Input),
				m.width,
				m.height,
			)
			// Store the new name temporarily
			m.pendingRename = result.Input
			m.currentView = ConfirmView
			return m, nil
		}

	case "rename-confirm":
		// Second step: confirmed, do the rename
		m.currentView = m.prevView
		m.confirmView = nil
		if m.pendingRename != "" && m.detailView != nil {
			newName := m.pendingRename
			m.pendingRename = ""
			return m, m.renameIssue(m.detailView.issue.Key, newName)
		}
	}

	m.currentView = m.prevView
	m.confirmView = nil
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
	case ActionView:
		content = m.actionView.View()
	case PreviewView:
		content = m.previewView.View()
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
type currentUserLoadedMsg struct{ user jira.User }
type queueSelectedMsg struct{ queue jira.Queue }
type issueSelectedMsg struct{ issue jira.Issue }
type initialIssueLoadedMsg struct{ issue jira.Issue }
type backToQueuesMsg struct{}
type backToIssuesMsg struct{}
type openTransitionMsg struct{ issue jira.Issue }
type openCommentMsg struct{ issue jira.Issue }
type openEditCommentMsg struct {
	issue     jira.Issue
	commentID string
	body      string
}
type openPreviewMsg struct {
	title   string
	content string
}
type backToDetailMsg struct{}
type transitionCompletedMsg struct{}
type commentAddedMsg struct{}
type commentEditedMsg struct{}
type openAssignMsg struct{ issue jira.Issue }
type renameCompletedMsg struct{}
