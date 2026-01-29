package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// IssuesModel handles the issue list view
type IssuesModel struct {
	client         *jira.Client
	projectKey     string
	queue          jira.Queue
	keys           KeyMap
	issues         []jira.Issue
	filteredIssues []jira.Issue
	searchFilter   string
	selectedIndex  int
	scrollOffset   int
	loading        bool
	err            error
	width          int
	height         int
	currentUser    string
	lastGPress     time.Time
}

// NewIssuesModel creates a new issue list model
func NewIssuesModel(client *jira.Client, projectKey string, queue jira.Queue, keys KeyMap, currentUser string, width, height int) *IssuesModel {
	return &IssuesModel{
		client:      client,
		projectKey:  projectKey,
		queue:       queue,
		keys:        keys,
		loading:     true,
		currentUser: currentUser,
		width:       width,
		height:      height,
	}
}

type issuesLoadedMsg struct{ issues []jira.Issue }

// Init initializes the view
func (m *IssuesModel) Init() tea.Cmd {
	return m.fetchIssues()
}

// Refresh reloads the issues
func (m *IssuesModel) Refresh() tea.Cmd {
	m.loading = true
	return m.fetchIssues()
}

// SetSearchFilter sets the search filter and updates filtered issues
func (m *IssuesModel) SetSearchFilter(query string) {
	m.searchFilter = query
	m.updateFilteredIssues()
	m.selectedIndex = 0
	m.scrollOffset = 0
}

// updateFilteredIssues filters issues by summary
func (m *IssuesModel) updateFilteredIssues() {
	if m.searchFilter == "" {
		m.filteredIssues = m.issues
		return
	}

	query := strings.ToLower(m.searchFilter)
	m.filteredIssues = nil
	for _, issue := range m.issues {
		if strings.Contains(strings.ToLower(issue.Fields.Summary), query) ||
			strings.Contains(strings.ToLower(issue.Key), query) {
			m.filteredIssues = append(m.filteredIssues, issue)
		}
	}
}

// getDisplayIssues returns the issues to display (filtered or all)
func (m *IssuesModel) getDisplayIssues() []jira.Issue {
	if m.searchFilter != "" {
		return m.filteredIssues
	}
	return m.issues
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.updateFilteredIssues()
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
				m.adjustScroll()
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			issues := m.getDisplayIssues()
			if m.selectedIndex < len(issues)-1 {
				m.selectedIndex++
				m.adjustScroll()
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			issues := m.getDisplayIssues()
			if len(issues) > 0 {
				return m, func() tea.Msg {
					return issueSelectedMsg{issue: issues[m.selectedIndex]}
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

		case key.Matches(msg, m.keys.GoToBottom):
			// G - go to bottom
			issues := m.getDisplayIssues()
			if len(issues) > 0 {
				m.selectedIndex = len(issues) - 1
				m.adjustScroll()
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToTop):
			// gg - double tap g to go to top
			now := time.Now()
			if !m.lastGPress.IsZero() && now.Sub(m.lastGPress) < 500*time.Millisecond {
				m.selectedIndex = 0
				m.scrollOffset = 0
				m.lastGPress = time.Time{}
			} else {
				m.lastGPress = now
			}
			return m, nil
		}
	}

	return m, nil
}

// visibleRows returns the number of issue rows that can be displayed
func (m *IssuesModel) visibleRows() int {
	// Reserve lines for: header (3), table header (1), scroll indicators (2), blank + help (2)
	reserved := 8
	available := m.height - reserved
	if available < 1 {
		available = 1
	}
	return available
}

// adjustScroll ensures the selected item is visible
func (m *IssuesModel) adjustScroll() {
	visible := m.visibleRows()

	// Scroll up if selection is above viewport
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}

	// Scroll down if selection is below viewport
	if m.selectedIndex >= m.scrollOffset+visible {
		m.scrollOffset = m.selectedIndex - visible + 1
	}
}

// View renders the issue list
func (m *IssuesModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading issues...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	issues := m.getDisplayIssues()
	if len(issues) == 0 {
		if m.searchFilter != "" {
			return fmt.Sprintf("No issues matching '%s' in queue: %s", m.searchFilter, m.queue.Name)
		}
		return fmt.Sprintf("No issues in queue: %s", m.queue.Name)
	}

	// Calculate column widths based on terminal width
	keyWidth := 9 // 8 chars + 1 space
	statusWidth := 20
	assigneeWidth := 20
	padding := 3 // spaces between columns
	showStatus := m.queue.Name != "All"

	// Use full terminal width, or default to 120 if width not set
	availableWidth := m.width
	if availableWidth == 0 {
		availableWidth = 120
	}

	// Calculate summary width (remaining space)
	columnsWidth := keyWidth + assigneeWidth + (padding * 2) + 2
	if showStatus {
		columnsWidth += statusWidth + padding
	}
	summaryWidth := availableWidth - columnsWidth
	if summaryWidth < 20 {
		summaryWidth = 20 // minimum width
	}

	s := HeaderStyle.Width(availableWidth).Render(fmt.Sprintf("Issues - %s > %s", m.projectKey, m.queue.Name)) + "\n\n"

	// Table header
	var headerFormat string
	if showStatus {
		headerFormat = fmt.Sprintf("%%-%ds   %%-%ds   %%-%ds   %%-%ds", keyWidth, summaryWidth, statusWidth, assigneeWidth)
		s += TableHeaderStyle.Render(fmt.Sprintf(headerFormat, "Key", "Summary", "Status", "Assignee")) + "\n"
	} else {
		headerFormat = fmt.Sprintf("%%-%ds   %%-%ds   %%-%ds", keyWidth, summaryWidth, assigneeWidth)
		s += TableHeaderStyle.Render(fmt.Sprintf(headerFormat, "Key", "Summary", "Assignee")) + "\n"
	}

	// Calculate visible range
	visible := m.visibleRows()
	startIdx := m.scrollOffset
	endIdx := m.scrollOffset + visible
	if endIdx > len(issues) {
		endIdx = len(issues)
	}

	// Show scroll indicator if there are items above
	if startIdx > 0 {
		s += HelpStyle.Render(fmt.Sprintf("  ↑ %d more above", startIdx)) + "\n"
	}

	for i := startIdx; i < endIdx; i++ {
		issue := issues[i]
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		// Get styles based on status name and assignee
		assigneeStyle := GetAssigneeStyle(assignee, m.currentUser)
		isDimmed := IsAssignedToOther(assignee, m.currentUser)

		// Truncate summary if too long
		summary := issue.Fields.Summary
		if len(summary) > summaryWidth {
			summary = summary[:summaryWidth-3] + "..."
		}

		// Truncate assignee if too long
		displayAssignee := assignee
		if len(displayAssignee) > assigneeWidth {
			displayAssignee = displayAssignee[:assigneeWidth-3] + "..."
		}

		// Pad fields before styling to avoid ANSI codes breaking alignment
		paddedKey := fmt.Sprintf("%-*s", keyWidth, issue.Key)
		paddedSummary := fmt.Sprintf("%-*s", summaryWidth, summary)
		paddedAssignee := fmt.Sprintf("%-*s", assigneeWidth, displayAssignee)

		// Apply styles - dim text if assigned to others (except status)
		var styledKey, styledSummary string
		if isDimmed {
			styledKey = DimmedTextStyle.Render(paddedKey)
			styledSummary = DimmedTextStyle.Render(paddedSummary)
		} else {
			styledKey = paddedKey
			styledSummary = paddedSummary
		}
		styledAssignee := assigneeStyle.Render(paddedAssignee)

		var line string
		if showStatus {
			// Truncate status if too long
			status := issue.Fields.Status.Name
			if len(status) > statusWidth {
				status = status[:statusWidth-3] + "..."
			}
			paddedStatus := fmt.Sprintf("%-*s", statusWidth, status)
			statusStyle := GetStatusStyle(issue.Fields.Status.Name)
			styledStatus := statusStyle.Render(paddedStatus)

			line = fmt.Sprintf("%s   %s   %s   %s",
				styledKey,
				styledSummary,
				styledStatus,
				styledAssignee,
			)
		} else {
			line = fmt.Sprintf("%s   %s   %s",
				styledKey,
				styledSummary,
				styledAssignee,
			)
		}

		if i == m.selectedIndex {
			line = SelectedStyle.Render(line)
		}

		s += line + "\n"
	}

	// Show scroll indicator if there are items below
	remaining := len(issues) - endIdx
	if remaining > 0 {
		s += HelpStyle.Render(fmt.Sprintf("  ↓ %d more below", remaining)) + "\n"
	}

	s += "\n" + HelpStyle.Render("? help • enter open • esc back")

	return s
}
