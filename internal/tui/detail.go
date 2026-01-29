package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 30

// YankableField represents a field that can be copied
type YankableField struct {
	Label    string
	Value    string
	LineNum  int // Line number in viewport for scrolling
	IsMatch  bool // True if this field matches the search
}

// DetailModel handles the issue detail view
type DetailModel struct {
	client        *jira.Client
	keys          KeyMap
	issue         jira.Issue
	comments      []jira.Comment
	proformaForms []jira.ProformaForm
	viewport      viewport.Model
	loading       bool
	err           error
	width         int
	height        int

	// Yank/copy support
	yankableFields  []YankableField
	selectedField   int
	lastYank        string    // What was last copied
	yankFeedbackEnd time.Time // When to hide feedback
	waitingForYank  bool      // True after first 'y' press, waiting for second

	// Search support
	searchFilter string

	// Navigation
	lastGPress time.Time // For gg detection
	waitingForG bool     // For gx detection
}

// NewDetailModel creates a new issue detail model
func NewDetailModel(client *jira.Client, issue jira.Issue, keys KeyMap, width, height int) *DetailModel {
	return &DetailModel{
		client:  client,
		keys:    keys,
		issue:   issue,
		loading: true,
		width:   width,
		height:  height,
	}
}

type issueDetailLoadedMsg struct {
	issue         jira.Issue
	comments      []jira.Comment
	proformaForms []jira.ProformaForm
}

type clearYankFeedbackMsg struct{}
type yankTimeoutMsg struct{}

// Init initializes the view
func (m *DetailModel) Init() tea.Cmd {
	return m.fetchDetail()
}

// fetchDetail loads full issue details, comments, and proforma forms
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

		forms, _ := m.client.GetProformaForms(m.issue.Key)

		return issueDetailLoadedMsg{
			issue:         *issue,
			comments:      comments,
			proformaForms: forms,
		}
	}
}

// Refresh reloads the issue details
func (m *DetailModel) Refresh() tea.Cmd {
	m.loading = true
	return m.fetchDetail()
}

// SetSearchFilter sets the search filter for highlighting in content
func (m *DetailModel) SetSearchFilter(query string) {
	m.searchFilter = query
	m.buildYankableFields() // Rebuild to mark matches
	m.updateViewport()

	// Find and select the first matching field
	if query != "" {
		for i, field := range m.yankableFields {
			if field.IsMatch {
				m.selectedField = i
				// Scroll viewport to show the match
				if field.LineNum > 0 {
					m.viewport.SetYOffset(field.LineNum - 2)
				}
				break
			}
		}
	}
}

// Update handles messages
func (m *DetailModel) Update(msg tea.Msg) (*DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewport()
		return m, nil

	case issueDetailLoadedMsg:
		m.issue = msg.issue
		m.comments = msg.comments
		m.proformaForms = msg.proformaForms
		m.loading = false
		m.buildYankableFields()
		m.updateViewport()
		return m, nil

	case clearYankFeedbackMsg:
		m.lastYank = ""
		return m, nil

	case yankTimeoutMsg:
		// Timeout after first 'y' - copy selected field if any
		if m.waitingForYank {
			m.waitingForYank = false
			if len(m.yankableFields) > 0 && m.selectedField >= 0 {
				return m, m.yankSelectedField()
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		// Handle yank (y/yy)
		if key.Matches(msg, m.keys.Yank) {
			if m.waitingForYank {
				// Second 'y' - copy issue key (yy)
				m.waitingForYank = false
				return m, m.yankIssueKey()
			}
			// First 'y' - wait briefly for possible second 'y'
			m.waitingForYank = true
			return m, tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
				return yankTimeoutMsg{}
			})
		}

		// Any other key cancels waiting for yank
		if m.waitingForYank {
			m.waitingForYank = false
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

		case key.Matches(msg, m.keys.Assign):
			return m, func() tea.Msg {
				return openAssignMsg{issue: m.issue}
			}

		case key.Matches(msg, m.keys.Refresh):
			return m, m.Refresh()

		// Field selection with j/k (vim style)
		case key.Matches(msg, m.keys.Down):
			m.waitingForG = false
			if len(m.yankableFields) > 0 {
				m.selectedField++
				if m.selectedField >= len(m.yankableFields) {
					m.selectedField = 0
				}
				m.scrollToSelected()
			}
			return m, nil

		case key.Matches(msg, m.keys.Up):
			m.waitingForG = false
			if len(m.yankableFields) > 0 {
				m.selectedField--
				if m.selectedField < 0 {
					m.selectedField = len(m.yankableFields) - 1
				}
				m.scrollToSelected()
			}
			return m, nil

		// G - go to bottom (last field)
		case key.Matches(msg, m.keys.GoToBottom):
			m.waitingForG = false
			if len(m.yankableFields) > 0 {
				m.selectedField = len(m.yankableFields) - 1
				m.scrollToSelected()
			}
			return m, nil

		// n - next search match
		case key.Matches(msg, m.keys.NextMatch):
			m.waitingForG = false
			if m.searchFilter != "" {
				m.goToNextMatch()
			}
			return m, nil

		// N - previous search match
		case key.Matches(msg, m.keys.PrevMatch):
			m.waitingForG = false
			if m.searchFilter != "" {
				m.goToPrevMatch()
			}
			return m, nil

		// g - potential gg (top) or gx (open URL)
		case key.Matches(msg, m.keys.GoToTop):
			now := time.Now()
			if m.waitingForG && now.Sub(m.lastGPress) < 500*time.Millisecond {
				// gg - go to top
				m.waitingForG = false
				m.selectedField = 0
				m.viewport.GotoTop()
				m.updateViewport()
				return m, nil
			}
			m.waitingForG = true
			m.lastGPress = now
			return m, nil
		}

		// Handle 'x' after 'g' for gx (open URL)
		if m.waitingForG && msg.String() == "x" {
			m.waitingForG = false
			return m, m.openURLInSelected()
		}

		// Reset g wait on other keys
		if m.waitingForG && msg.String() != "g" {
			m.waitingForG = false
		}
	}

	// Update viewport for scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// buildYankableFields builds the list of fields that can be copied
func (m *DetailModel) buildYankableFields() {
	m.yankableFields = []YankableField{}
	searchLower := strings.ToLower(m.searchFilter)

	// Helper to check if value matches search
	matchesSearch := func(value string) bool {
		if m.searchFilter == "" {
			return false
		}
		return strings.Contains(strings.ToLower(value), searchLower)
	}

	// Add description if present
	if m.issue.Fields.Description != "" {
		m.yankableFields = append(m.yankableFields, YankableField{
			Label:   "Description",
			Value:   m.issue.Fields.Description,
			IsMatch: matchesSearch(m.issue.Fields.Description),
		})
	}

	// Add proforma form fields
	for _, form := range m.proformaForms {
		for _, field := range form.Fields {
			if field.Answer != "" && field.Answer != "-" {
				m.yankableFields = append(m.yankableFields, YankableField{
					Label:   field.Label,
					Value:   field.Answer,
					IsMatch: matchesSearch(field.Answer),
				})
			}
		}
	}

	// Add comments
	for _, comment := range m.comments {
		m.yankableFields = append(m.yankableFields, YankableField{
			Label:   fmt.Sprintf("Comment by %s", comment.Author.DisplayName),
			Value:   comment.Body,
			IsMatch: matchesSearch(comment.Body),
		})
	}

	// Reset selection if not searching
	if m.searchFilter == "" {
		m.selectedField = -1
	}
}

// yankIssueKey copies the issue key to clipboard
func (m *DetailModel) yankIssueKey() tea.Cmd {
	CopyToClipboard(m.issue.Key)
	m.lastYank = m.issue.Key
	m.yankFeedbackEnd = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearYankFeedbackMsg{}
	})
}

// yankSelectedField copies the selected field value to clipboard
func (m *DetailModel) yankSelectedField() tea.Cmd {
	if m.selectedField >= 0 && m.selectedField < len(m.yankableFields) {
		field := m.yankableFields[m.selectedField]
		CopyToClipboard(field.Value)
		m.lastYank = field.Value
		if len(m.lastYank) > 30 {
			m.lastYank = m.lastYank[:30] + "..."
		}
		m.yankFeedbackEnd = time.Now().Add(2 * time.Second)
		return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearYankFeedbackMsg{}
		})
	}
	return nil
}

// scrollToSelected scrolls viewport to show the selected field
func (m *DetailModel) scrollToSelected() {
	m.updateViewport()
	if m.selectedField >= 0 && m.selectedField < len(m.yankableFields) {
		field := m.yankableFields[m.selectedField]
		if field.LineNum > 0 {
			// Scroll so the field is visible (with some padding)
			targetLine := field.LineNum - 2
			if targetLine < 0 {
				targetLine = 0
			}
			m.viewport.SetYOffset(targetLine)
		}
	}
}

// goToNextMatch moves to the next matching field
func (m *DetailModel) goToNextMatch() {
	if len(m.yankableFields) == 0 {
		return
	}

	// Start searching from current position + 1
	start := m.selectedField + 1
	if start < 0 {
		start = 0
	}

	// Search forward, wrapping around
	for i := 0; i < len(m.yankableFields); i++ {
		idx := (start + i) % len(m.yankableFields)
		if m.yankableFields[idx].IsMatch {
			m.selectedField = idx
			m.scrollToSelected()
			return
		}
	}
}

// goToPrevMatch moves to the previous matching field
func (m *DetailModel) goToPrevMatch() {
	if len(m.yankableFields) == 0 {
		return
	}

	// Start searching from current position - 1
	start := m.selectedField - 1
	if start < 0 {
		start = len(m.yankableFields) - 1
	}

	// Search backward, wrapping around
	for i := 0; i < len(m.yankableFields); i++ {
		idx := start - i
		if idx < 0 {
			idx += len(m.yankableFields)
		}
		if m.yankableFields[idx].IsMatch {
			m.selectedField = idx
			m.scrollToSelected()
			return
		}
	}
}

// openURLInSelected finds and opens the first URL in the selected field
func (m *DetailModel) openURLInSelected() tea.Cmd {
	if m.selectedField < 0 || m.selectedField >= len(m.yankableFields) {
		return nil
	}

	field := m.yankableFields[m.selectedField]
	url := findFirstURL(field.Value)
	if url == "" {
		return nil
	}

	return func() tea.Msg {
		OpenInBrowser(url)
		return nil
	}
}

// findFirstURL finds the first http/https URL in text
func findFirstURL(text string) string {
	// Simple URL detection
	for _, prefix := range []string{"https://", "http://"} {
		idx := strings.Index(text, prefix)
		if idx >= 0 {
			// Find end of URL (space, newline, or end of string)
			end := idx
			for end < len(text) && !isURLTerminator(text[end]) {
				end++
			}
			return text[idx:end]
		}
	}
	return ""
}

// isURLTerminator checks if a character ends a URL
func isURLTerminator(c byte) bool {
	return c == ' ' || c == '\n' || c == '\t' || c == '"' || c == '\'' || c == '>' || c == ')' || c == ']'
}

// updateViewport sets up the viewport with correct dimensions
func (m *DetailModel) updateViewport() {
	headerHeight := 4 // header + status line
	helpHeight := 2   // help text at bottom
	mainContentWidth := m.width - sidebarWidth - 3 // 3 for borders/padding

	if mainContentWidth < 40 {
		mainContentWidth = 40
	}

	contentHeight := m.height - headerHeight - helpHeight
	if contentHeight < 10 {
		contentHeight = 10
	}

	m.viewport = viewport.New(mainContentWidth, contentHeight)
	m.viewport.SetContent(m.renderMainContent(mainContentWidth))
}

// renderHeader renders the top header with issue key, summary, status, and request type
func (m *DetailModel) renderHeader() string {
	// Status box with color
	statusStyle := GetStatusStyle(m.issue.Fields.Status.StatusCategory.Name)
	statusBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(statusStyle.GetForeground()).
		Padding(0, 1).
		Render(statusStyle.Render(m.issue.Fields.Status.Name))

	// Request type box
	requestType := "Unknown"
	if m.issue.Fields.CustomerRequestType != nil && m.issue.Fields.CustomerRequestType.RequestType != nil {
		requestType = m.issue.Fields.CustomerRequestType.RequestType.Name
	}
	requestTypeBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1).
		Render(requestType)

	// Calculate available width for summary
	boxesWidth := lipgloss.Width(statusBox) + lipgloss.Width(requestTypeBox) + 2
	summaryWidth := m.width - boxesWidth - 4
	if summaryWidth < 20 {
		summaryWidth = 20
	}

	// Truncate summary if needed
	summary := m.issue.Fields.Summary
	if len(summary) > summaryWidth {
		summary = summary[:summaryWidth-3] + "..."
	}

	// Issue key and summary
	issueTitle := TitleStyle.Render(fmt.Sprintf("%s: %s", m.issue.Key, summary))

	// Combine header elements
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		issueTitle,
		"  ",
		statusBox,
		" ",
		requestTypeBox,
	)

	return header
}

// renderSidebar renders the fixed sidebar with metadata
func (m *DetailModel) renderSidebar(height int) string {
	var sb strings.Builder

	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth - 2).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	// Assignee
	assignee := "Unassigned"
	if m.issue.Fields.Assignee != nil {
		assignee = m.issue.Fields.Assignee.DisplayName
	}
	sb.WriteString(labelStyle.Render("Assignee") + "\n")
	sb.WriteString(valueStyle.Render(truncate(assignee, sidebarWidth-4)) + "\n\n")

	// Reporter
	reporter := "Unknown"
	if m.issue.Fields.Reporter != nil {
		reporter = m.issue.Fields.Reporter.DisplayName
	}
	sb.WriteString(labelStyle.Render("Reporter") + "\n")
	sb.WriteString(valueStyle.Render(truncate(reporter, sidebarWidth-4)) + "\n\n")

	// Request Participants
	sb.WriteString(labelStyle.Render("Participants") + "\n")
	if len(m.issue.Fields.RequestParticipants) == 0 {
		sb.WriteString(HelpStyle.Render("None") + "\n")
	} else {
		for _, p := range m.issue.Fields.RequestParticipants {
			sb.WriteString(valueStyle.Render("• "+truncate(p.DisplayName, sidebarWidth-6)) + "\n")
		}
	}
	sb.WriteString("\n")

	// Dates
	sb.WriteString(labelStyle.Render("Created") + "\n")
	sb.WriteString(valueStyle.Render(formatDate(m.issue.Fields.Created)) + "\n\n")

	sb.WriteString(labelStyle.Render("Updated") + "\n")
	sb.WriteString(valueStyle.Render(formatDate(m.issue.Fields.Updated)) + "\n")

	// Add border to sidebar
	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Height(height).
		Width(sidebarWidth).
		Render(sidebarStyle.Render(sb.String()))

	return sidebarBox
}

// renderMainContent renders the scrollable main content area
func (m *DetailModel) renderMainContent(width int) string {
	var sb strings.Builder

	// Track which yankable field we're rendering
	fieldIndex := 0
	lineNum := 0

	// Style for selected field (gray background)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("255"))

	// Style for search match highlight (orange background, inline)
	matchHighlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("208")). // Orange
		Foreground(lipgloss.Color("232")).  // Dark text for contrast
		Bold(true)

	// Helper to highlight search matches inline
	highlightMatches := func(text string) string {
		if m.searchFilter == "" {
			return text
		}
		searchLower := strings.ToLower(m.searchFilter)
		textLower := strings.ToLower(text)

		var result strings.Builder
		lastEnd := 0

		for {
			idx := strings.Index(textLower[lastEnd:], searchLower)
			if idx == -1 {
				result.WriteString(text[lastEnd:])
				break
			}

			// Add text before match
			result.WriteString(text[lastEnd : lastEnd+idx])
			// Add highlighted match (preserve original case)
			matchEnd := lastEnd + idx + len(m.searchFilter)
			result.WriteString(matchHighlightStyle.Render(text[lastEnd+idx : matchEnd]))
			lastEnd = matchEnd
		}

		return result.String()
	}

	// Helper to render text with optional selection and inline match highlighting
	renderField := func(text string, isSelected bool, isMatch bool) string {
		// First apply inline match highlighting if there are matches
		if isMatch {
			text = highlightMatches(text)
		}
		// Then wrap in selection style if selected
		if isSelected {
			return selectedStyle.Render(text)
		}
		return text
	}

	// Description
	sb.WriteString(TitleStyle.Render("Description") + "\n")
	lineNum++
	sb.WriteString(strings.Repeat("─", width-2) + "\n")
	lineNum++
	if m.issue.Fields.Description != "" {
		// Update line number for this field
		if fieldIndex < len(m.yankableFields) {
			m.yankableFields[fieldIndex].LineNum = lineNum
		}
		descText := wrapText(m.issue.Fields.Description, width-2)
		isMatch := fieldIndex < len(m.yankableFields) && m.yankableFields[fieldIndex].IsMatch
		sb.WriteString(renderField(descText, m.selectedField == fieldIndex, isMatch) + "\n")
		lineNum += strings.Count(descText, "\n") + 1
		fieldIndex++
	} else {
		sb.WriteString(HelpStyle.Render("No description provided.") + "\n")
		lineNum++
	}
	sb.WriteString("\n")
	lineNum++

	// Proforma Forms
	for _, form := range m.proformaForms {
		sb.WriteString(TitleStyle.Render(fmt.Sprintf("Form: %s", form.Name)) + "\n")
		lineNum++
		sb.WriteString(strings.Repeat("─", width-2) + "\n")
		lineNum++
		for _, field := range form.Fields {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(field.Label) + "\n")
			lineNum++

			answerText := wrapText(field.Answer, width-2)
			if field.Answer != "" && field.Answer != "-" {
				// Update line number for this field
				if fieldIndex < len(m.yankableFields) {
					m.yankableFields[fieldIndex].LineNum = lineNum
				}
				isMatch := fieldIndex < len(m.yankableFields) && m.yankableFields[fieldIndex].IsMatch
				sb.WriteString(renderField(answerText, m.selectedField == fieldIndex, isMatch) + "\n\n")
				lineNum += strings.Count(answerText, "\n") + 2
				fieldIndex++
			} else {
				sb.WriteString(answerText + "\n\n")
				lineNum += strings.Count(answerText, "\n") + 2
			}
		}
	}

	// Comments
	sb.WriteString(TitleStyle.Render(fmt.Sprintf("Comments (%d)", len(m.comments))) + "\n")
	lineNum++
	sb.WriteString(strings.Repeat("─", width-2) + "\n")
	lineNum++

	if len(m.comments) == 0 {
		sb.WriteString(HelpStyle.Render("No comments yet.") + "\n")
	} else {
		for _, comment := range m.comments {
			authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
			dateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

			sb.WriteString(authorStyle.Render(comment.Author.DisplayName))
			sb.WriteString(dateStyle.Render(" • "+formatDate(comment.Created)) + "\n")
			lineNum++

			// Update line number for this comment field
			if fieldIndex < len(m.yankableFields) {
				m.yankableFields[fieldIndex].LineNum = lineNum
			}
			commentText := wrapText(comment.Body, width-2)
			isMatch := fieldIndex < len(m.yankableFields) && m.yankableFields[fieldIndex].IsMatch
			sb.WriteString(renderField(commentText, m.selectedField == fieldIndex, isMatch) + "\n\n")
			lineNum += strings.Count(commentText, "\n") + 2
			fieldIndex++
		}
	}

	return sb.String()
}

// View renders the detail view
func (m *DetailModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading issue details...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Header
	header := m.renderHeader()

	// Calculate content height
	headerHeight := lipgloss.Height(header) + 1
	helpHeight := 2
	contentHeight := m.height - headerHeight - helpHeight
	if contentHeight < 10 {
		contentHeight = 10
	}

	// Sidebar (fixed)
	sidebar := m.renderSidebar(contentHeight)

	// Main content (scrollable viewport)
	mainContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - sidebarWidth - 2).
		Height(contentHeight).
		Render(m.viewport.View())

	// Combine main content and sidebar
	body := lipgloss.JoinHorizontal(lipgloss.Top, mainContent, sidebar)

	// Help text or yank feedback
	var help string
	if m.lastYank != "" {
		yankStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
		help = yankStyle.Render(fmt.Sprintf("✓ Copied: %s", m.lastYank))
	} else if m.waitingForYank {
		help = HelpStyle.Render("y again to copy issue key")
	} else {
		help = HelpStyle.Render("? help • esc back")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help)
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDate(dateStr string) string {
	// Input format: 2026-01-28T15:24:12.490+0100
	if len(dateStr) >= 16 {
		return dateStr[:10] + " " + dateStr[11:16]
	}
	return dateStr
}

func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Wrap long lines
		for len(line) > width {
			// Find last space before width
			breakPoint := width
			for j := width - 1; j > 0; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}
			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
	}

	return result.String()
}
