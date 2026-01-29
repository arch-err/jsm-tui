package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ActionPhase represents the current phase of the action view
type ActionPhase int

const (
	ActionPhaseSelect  ActionPhase = iota // Selecting an action
	ActionPhaseConfirm                    // Confirming with editable comment
)

// ActionMode represents the current mode in confirm phase
type ActionMode int

const (
	ActionModeInsert ActionMode = iota // Typing in textarea
	ActionModeNormal                   // Navigating between elements
)

// ActionFocus represents which element is focused in confirm phase
type ActionFocus int

const (
	ActionFocusInput   ActionFocus = iota // Textarea
	ActionFocusExecute                    // Execute button
)

// ActionModel handles quick actions on issues
type ActionModel struct {
	client  *jira.Client
	keys    KeyMap
	issue   jira.Issue
	actions []config.ActionConfig
	width   int
	height  int

	phase         ActionPhase
	selectedIndex int
	executing     bool
	err           error
	lastGPress    time.Time

	// Confirm phase
	selectedAction *config.ActionConfig
	textarea       textarea.Model
	mode           ActionMode
	focus          ActionFocus
}

// NewActionModel creates a new action model
func NewActionModel(client *jira.Client, issue jira.Issue, actions []config.ActionConfig, keys KeyMap, width, height int) *ActionModel {
	// Filter actions by request type
	requestType := ""
	if issue.Fields.CustomerRequestType != nil && issue.Fields.CustomerRequestType.RequestType != nil {
		requestType = issue.Fields.CustomerRequestType.RequestType.Name
	}

	var filteredActions []config.ActionConfig
	for _, action := range actions {
		if action.RequestTypes.Matches(requestType) {
			filteredActions = append(filteredActions, action)
		}
	}

	// Setup textarea for comment editing
	ta := textarea.New()
	ta.Placeholder = "Enter comment (optional)..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.SetWidth(50)
	ta.SetHeight(8)

	// Clear default styles
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Width(4)
	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Width(4)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle()
	ta.BlurredStyle.Prompt = lipgloss.NewStyle()
	ta.Prompt = ""
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	return &ActionModel{
		client:  client,
		keys:    keys,
		issue:   issue,
		actions: filteredActions,
		width:   width,
		height:  height,
		phase:   ActionPhaseSelect,
		textarea: ta,
	}
}

// actionCompletedMsg signals that action execution completed
type actionCompletedMsg struct{}

// Update handles messages
func (m *ActionModel) Update(msg tea.Msg) (*ActionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case actionCompletedMsg:
		return m, func() tea.Msg {
			return actionCompletedMsg{}
		}

	case tea.KeyMsg:
		if m.executing {
			return m, nil
		}

		if m.phase == ActionPhaseSelect {
			return m.updateSelectPhase(msg)
		}
		return m.updateConfirmPhase(msg)
	}

	return m, nil
}

// updateSelectPhase handles input during action selection
func (m *ActionModel) updateSelectPhase(msg tea.KeyMsg) (*ActionModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < len(m.actions)-1 {
			m.selectedIndex++
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if len(m.actions) > 0 {
			m.selectedAction = &m.actions[m.selectedIndex]
			m.phase = ActionPhaseConfirm
			m.mode = ActionModeInsert
			m.focus = ActionFocusInput
			// Pre-fill comment from action config
			if m.selectedAction.Comment != "" {
				m.textarea.SetValue(m.selectedAction.Comment)
			}
			m.textarea.Focus()
		}
		return m, nil

	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Cancel):
		return m, func() tea.Msg {
			return backToDetailMsg{}
		}

	case key.Matches(msg, m.keys.GoToBottom):
		if len(m.actions) > 0 {
			m.selectedIndex = len(m.actions) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.GoToTop):
		now := time.Now()
		if !m.lastGPress.IsZero() && now.Sub(m.lastGPress) < 500*time.Millisecond {
			m.selectedIndex = 0
			m.lastGPress = time.Time{}
		} else {
			m.lastGPress = now
		}
		return m, nil
	}

	return m, nil
}

// updateConfirmPhase handles input during confirmation
func (m *ActionModel) updateConfirmPhase(msg tea.KeyMsg) (*ActionModel, tea.Cmd) {
	k := msg.String()

	if m.mode == ActionModeInsert {
		// In insert mode, Esc switches to normal mode
		if msg.Type == tea.KeyEsc {
			m.mode = ActionModeNormal
			m.textarea.Blur()
			return m, nil
		}

		// Pass all other keys to textarea
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	// Normal mode - navigate with hjkl
	switch k {
	case "esc":
		// Go back to selection phase
		m.phase = ActionPhaseSelect
		m.selectedAction = nil
		m.textarea.SetValue("")
		m.textarea.Blur()
		return m, nil

	case "j":
		// Move down: Input -> Execute
		if m.focus == ActionFocusInput {
			m.focus = ActionFocusExecute
		}
		return m, nil

	case "k":
		// Move up: Execute -> Input
		if m.focus == ActionFocusExecute {
			m.focus = ActionFocusInput
		}
		return m, nil

	case "enter":
		switch m.focus {
		case ActionFocusInput:
			// Enter insert mode
			m.mode = ActionModeInsert
			m.textarea.Focus()
			return m, nil

		case ActionFocusExecute:
			m.executing = true
			return m, m.executeAction()
		}

	case "i":
		// Also allow 'i' to enter insert mode when on input
		if m.focus == ActionFocusInput {
			m.mode = ActionModeInsert
			m.textarea.Focus()
			return m, nil
		}
	}

	return m, nil
}

// executeAction executes the selected action
func (m *ActionModel) executeAction() tea.Cmd {
	return func() tea.Msg {
		action := m.selectedAction
		comment := strings.TrimSpace(m.textarea.Value())

		// Execute transition if status is specified
		if action.Status != "" {
			transition, err := m.client.FindTransitionByStatusName(m.issue.Key, action.Status)
			if err != nil {
				return errorMsg{err: fmt.Errorf("failed to find transition: %w", err)}
			}

			// Build fields for transition (e.g., pending reason)
			var fields map[string]interface{}
			if action.PendingReason != "" {
				// Note: The actual field ID for pending reason varies by Jira instance
				// This is a common pattern, may need adjustment
				fields = map[string]interface{}{
					"customfield_10002": action.PendingReason, // Adjust field ID as needed
				}
			}

			if err := m.client.ExecuteTransitionWithFields(m.issue.Key, transition.ID, fields); err != nil {
				return errorMsg{err: fmt.Errorf("failed to execute transition: %w", err)}
			}
		}

		// Add comment if provided
		if comment != "" {
			if err := m.client.AddComment(m.issue.Key, comment); err != nil {
				return errorMsg{err: fmt.Errorf("failed to add comment: %w", err)}
			}
		}

		return actionCompletedMsg{}
	}
}

// View renders the action view
func (m *ActionModel) View() string {
	if m.executing {
		return SpinnerStyle.Render("Executing action...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.phase == ActionPhaseSelect {
		return m.viewSelectPhase()
	}
	return m.viewConfirmPhase()
}

// viewSelectPhase renders the action selection view
func (m *ActionModel) viewSelectPhase() string {
	if len(m.actions) == 0 {
		return "No actions available for this issue type.\n\n" +
			HelpStyle.Render("esc back")
	}

	s := HeaderStyle.Render(fmt.Sprintf("Quick Actions - %s", m.issue.Key)) + "\n\n"

	for i, action := range m.actions {
		line := fmt.Sprintf("  %s", action.Name)
		if action.Status != "" {
			line += fmt.Sprintf(" → %s", action.Status)
		}
		if i == m.selectedIndex {
			line = SelectedStyle.Render(line)
		}
		s += line + "\n"
	}

	s += "\n" + HelpStyle.Render("enter select • esc cancel")

	return s
}

// viewConfirmPhase renders the confirmation view with editable comment
func (m *ActionModel) viewConfirmPhase() string {
	action := m.selectedAction

	// Colors
	focusedBorder := lipgloss.Color("#7B68EE")
	unfocusedBorder := lipgloss.Color("#555555")
	insertModeBorder := lipgloss.Color("#98C379")

	// Modal width
	modalWidth := 60

	// Title
	title := TitleStyle.Render(fmt.Sprintf("Confirm: %s", action.Name))

	// Action summary
	var summary strings.Builder
	summary.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Issue: "))
	summary.WriteString(m.issue.Key + "\n")

	if action.Status != "" {
		summary.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Status: "))
		summary.WriteString(m.issue.Fields.Status.Name + " → " + action.Status + "\n")
	}

	if action.PendingReason != "" {
		summary.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Pending Reason: "))
		summary.WriteString(action.PendingReason + "\n")
	}

	// Textarea border color based on focus and mode
	textareaBorderColor := unfocusedBorder
	if m.focus == ActionFocusInput {
		if m.mode == ActionModeInsert {
			textareaBorderColor = insertModeBorder
		} else {
			textareaBorderColor = focusedBorder
		}
	}

	textareaStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(textareaBorderColor).
		Padding(0, 1)

	textareaBox := textareaStyle.Render(m.textarea.View())

	// Submit button
	submitBorderColor := unfocusedBorder
	if m.focus == ActionFocusExecute {
		submitBorderColor = focusedBorder
	}
	submitStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(submitBorderColor).
		Padding(0, 2)

	submitBtn := submitStyle.Render("Execute")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		summary.String(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Comment:"),
		textareaBox,
		"",
		submitBtn,
	)

	// Modal container
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content)

	// Help text based on mode
	var help string
	if m.mode == ActionModeInsert {
		help = HelpStyle.Render("esc normal mode")
	} else {
		help = HelpStyle.Render("i/enter edit • j/k navigate • enter execute • esc back")
	}

	return m.renderCentered(lipgloss.JoinVertical(lipgloss.Left, modal, help))
}

// renderCentered centers content on the screen
func (m *ActionModel) renderCentered(content string) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		if lipgloss.Width(line) > contentWidth {
			contentWidth = lipgloss.Width(line)
		}
	}

	verticalPadding := (m.height - contentHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	horizontalPadding := (m.width - contentWidth) / 2
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	var view strings.Builder
	for i := 0; i < verticalPadding; i++ {
		view.WriteString("\n")
	}

	for _, line := range lines {
		view.WriteString(strings.Repeat(" ", horizontalPadding))
		view.WriteString(line)
		view.WriteString("\n")
	}

	return view.String()
}
