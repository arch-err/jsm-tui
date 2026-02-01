package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/arch-err/jsm-tui/internal/storage"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentFocus represents which element is focused
type CommentFocus int

const (
	FocusInput CommentFocus = iota
	FocusSubmit
	FocusPassive
	FocusInternal
	FocusCancel
)

// CommentMode represents the current mode
type CommentMode int

const (
	ModeInsert  CommentMode = iota // Typing in textarea
	ModeNormal                     // Navigating between elements
	ModeMention                    // Selecting a user to mention
)

// CommentModel handles adding comments to issues
type CommentModel struct {
	client     *jira.Client
	keys       KeyMap
	issue      jira.Issue
	textarea   textarea.Model
	submitting bool
	err        error
	width      int
	height     int

	mode  CommentMode
	focus CommentFocus

	// Edit mode
	isEditMode bool
	commentID  string // Only set in edit mode

	// Mention picker state
	mentionQuery    string
	mentionUsers    []jira.User
	mentionSelected int
	mentionLoading  bool
}

// NewCommentModel creates a new comment model
func NewCommentModel(client *jira.Client, issue jira.Issue, keys KeyMap, width, height int) *CommentModel {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.ShowLineNumbers = true
	ta.Focus()
	ta.CharLimit = 0 // No limit
	ta.SetWidth(58)
	ta.SetHeight(12)

	// Clear default base styles to avoid double borders
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()

	// Style line numbers
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Width(4)
	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Width(4)

	// Remove the prompt/separator line
	ta.FocusedStyle.Prompt = lipgloss.NewStyle()
	ta.BlurredStyle.Prompt = lipgloss.NewStyle()
	ta.Prompt = ""

	// Cursor line style (no highlight)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	// Load any existing draft
	if draft, err := storage.LoadDraft(issue.Key, storage.DraftTypeComment); err == nil && draft != "" {
		ta.SetValue(draft)
	}

	return &CommentModel{
		client:   client,
		keys:     keys,
		issue:    issue,
		textarea: ta,
		width:    width,
		height:   height,
		mode:     ModeInsert,
		focus:    FocusInput,
	}
}

// NewEditCommentModel creates a comment model for editing an existing comment
func NewEditCommentModel(client *jira.Client, issue jira.Issue, keys KeyMap, width, height int, commentID, body string) *CommentModel {
	m := NewCommentModel(client, issue, keys, width, height)
	m.isEditMode = true
	m.commentID = commentID
	m.textarea.SetValue(body)
	return m
}

// editorFinishedMsg is sent when external editor closes
type editorFinishedMsg struct {
	content string
	err     error
}

// mentionSearchResultMsg is sent when user search completes
type mentionSearchResultMsg struct {
	users []jira.User
	err   error
}

// openInEditor opens the current text in $EDITOR
func (m *CommentModel) openInEditor() tea.Cmd {
	content := m.textarea.Value()

	// Create temp file before launching editor
	tmpFile, err := os.CreateTemp("", "jsm-comment-*.md")
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}
	tmpPath := tmpFile.Name()

	// Write current content
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("failed to write temp file: %w", err)}
		}
	}
	tmpFile.Close()

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim" // fallback
	}

	// Use tea.ExecProcess to properly suspend TUI and run editor
	c := exec.Command(editor, tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)

		if err != nil {
			return editorFinishedMsg{err: fmt.Errorf("editor failed: %w", err)}
		}

		// Read back content
		newContent, err := os.ReadFile(tmpPath)
		if err != nil {
			return editorFinishedMsg{err: fmt.Errorf("failed to read temp file: %w", err)}
		}

		return editorFinishedMsg{content: string(newContent)}
	})
}

// searchMentionUsers searches for users to mention
func (m *CommentModel) searchMentionUsers(query string) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.SearchAssignableUsers(m.issue.Key, query)
		if err != nil {
			return mentionSearchResultMsg{err: err}
		}
		return mentionSearchResultMsg{users: users}
	}
}

// insertMention inserts a user mention at the current position
func (m *CommentModel) insertMention(user jira.User) {
	// Get current text
	currentValue := m.textarea.Value()

	// Find the last @ that started this mention
	// Note: the query characters went to mentionQuery, not the textarea
	// So the textarea only has "@" without the query
	lastAt := strings.LastIndex(currentValue, "@")

	if lastAt >= 0 {
		// Build the mention text - prefer accountId for Cloud, name for Server
		var mention string
		if user.AccountID != "" {
			mention = fmt.Sprintf("[~accountId:%s]", user.AccountID)
		} else if user.Name != "" {
			mention = fmt.Sprintf("[~%s]", user.Name)
		} else {
			mention = user.DisplayName
		}

		// Replace @ with the mention
		newValue := currentValue[:lastAt] + mention + currentValue[lastAt+1:]
		m.textarea.SetValue(newValue)
	}
}

// submitComment submits the comment with the specified type (public, passive, internal)
func (m *CommentModel) submitComment(commentType string) tea.Cmd {
	// Trim leading/trailing whitespace (including newlines)
	body := strings.TrimSpace(m.textarea.Value())
	issueKey := m.issue.Key
	isEdit := m.isEditMode

	return func() tea.Msg {
		var err error
		if isEdit {
			// Update existing comment
			err = m.client.UpdateComment(issueKey, m.commentID, body)
			if err != nil {
				return errorMsg{err: err}
			}
			return commentEditedMsg{}
		}
		// Add new comment based on type
		switch commentType {
		case "internal":
			err = m.client.AddInternalComment(issueKey, body)
		case "passive":
			err = m.client.AddPassiveComment(issueKey, body)
		default: // "public"
			err = m.client.AddComment(issueKey, body)
		}
		if err != nil {
			return errorMsg{err: err}
		}
		// Delete draft after successful submission
		storage.DeleteDraft(issueKey, storage.DraftTypeComment)
		return commentAddedMsg{}
	}
}

// Update handles messages
func (m *CommentModel) Update(msg tea.Msg) (*CommentModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Set the content from editor, trim trailing newline
		m.textarea.SetValue(strings.TrimSuffix(msg.content, "\n"))
		m.mode = ModeInsert
		m.focus = FocusInput
		m.textarea.Focus()
		return m, nil

	case commentAddedMsg:
		return m, func() tea.Msg {
			return commentAddedMsg{}
		}

	case mentionSearchResultMsg:
		m.mentionLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.mentionUsers = msg.users
		m.mentionSelected = 0
		return m, nil

	case tea.KeyMsg:
		if m.submitting {
			return m, nil
		}

		key := msg.String()

		// Ctrl+G opens external editor (works in insert and normal modes, not mention)
		if key == "ctrl+g" && m.mode != ModeMention {
			m.textarea.Blur()
			return m, m.openInEditor()
		}

		// Handle mention mode
		if m.mode == ModeMention {
			switch msg.Type {
			case tea.KeyEsc:
				// Cancel mention, return to insert mode
				m.mode = ModeInsert
				m.mentionQuery = ""
				m.mentionUsers = nil
				m.textarea.Focus()
				return m, nil

			case tea.KeyEnter:
				// Select the highlighted user
				if len(m.mentionUsers) > 0 && m.mentionSelected < len(m.mentionUsers) {
					m.insertMention(m.mentionUsers[m.mentionSelected])
				}
				m.mode = ModeInsert
				m.mentionQuery = ""
				m.mentionUsers = nil
				m.textarea.Focus()
				return m, nil

			case tea.KeyUp:
				if m.mentionSelected > 0 {
					m.mentionSelected--
				}
				return m, nil

			case tea.KeyDown:
				if m.mentionSelected < len(m.mentionUsers)-1 {
					m.mentionSelected++
				}
				return m, nil

			case tea.KeyBackspace:
				if len(m.mentionQuery) > 0 {
					m.mentionQuery = m.mentionQuery[:len(m.mentionQuery)-1]
					m.mentionLoading = true
					return m, m.searchMentionUsers(m.mentionQuery)
				} else {
					// Backspace with empty query cancels mention mode
					m.mode = ModeInsert
					m.mentionUsers = nil
					m.textarea.Focus()
					return m, nil
				}

			default:
				// Add character to query if it's printable
				if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
					m.mentionQuery += key
					m.mentionLoading = true
					m.mentionSelected = 0
					return m, m.searchMentionUsers(m.mentionQuery)
				}
			}
			return m, nil
		}

		if m.mode == ModeInsert {
			// In insert mode, Esc switches to normal mode
			if msg.Type == tea.KeyEsc {
				m.mode = ModeNormal
				m.textarea.Blur()
				return m, nil
			}

			// Detect @ to start mention mode
			if key == "@" {
				// First, let the textarea handle the @
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				// Then switch to mention mode
				m.mode = ModeMention
				m.mentionQuery = ""
				m.mentionUsers = nil
				m.mentionSelected = 0
				m.mentionLoading = true
				// Start searching immediately (empty query gets all)
				return m, tea.Batch(cmd, m.searchMentionUsers(""))
			}

			// Pass all other keys to textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

		// Normal mode - navigate with hjkl
		switch key {
		case "esc":
			// Save draft and go back (only for new comments, not edits)
			if !m.isEditMode {
				storage.SaveDraft(m.issue.Key, storage.DraftTypeComment, m.textarea.Value())
			}
			return m, func() tea.Msg {
				return backToDetailMsg{}
			}

		case "h":
			// Move left on button row
			if m.isEditMode {
				// In edit mode: Cancel -> Submit (no Passive/Internal)
				if m.focus == FocusCancel {
					m.focus = FocusSubmit
				}
			} else {
				// Normal mode: Cancel -> Internal -> Passive -> Submit
				switch m.focus {
				case FocusCancel:
					m.focus = FocusInternal
				case FocusInternal:
					m.focus = FocusPassive
				case FocusPassive:
					m.focus = FocusSubmit
				}
			}

		case "l":
			// Move right on button row
			if m.isEditMode {
				// In edit mode: Submit -> Cancel (no Passive/Internal)
				if m.focus == FocusSubmit {
					m.focus = FocusCancel
				}
			} else {
				// Normal mode: Submit -> Passive -> Internal -> Cancel
				switch m.focus {
				case FocusSubmit:
					m.focus = FocusPassive
				case FocusPassive:
					m.focus = FocusInternal
				case FocusInternal:
					m.focus = FocusCancel
				}
			}

		case "j":
			// Move down: Input -> Submit
			if m.focus == FocusInput {
				m.focus = FocusSubmit
			}

		case "k":
			// Move up: any button -> Input
			if m.focus == FocusSubmit || m.focus == FocusInternal || m.focus == FocusCancel {
				m.focus = FocusInput
			}

		case "enter":
			switch m.focus {
			case FocusInput:
				// Enter insert mode
				m.mode = ModeInsert
				m.textarea.Focus()
				return m, nil

			case FocusSubmit:
				if m.textarea.Value() != "" {
					m.submitting = true
					return m, m.submitComment("public")
				}

			case FocusPassive:
				if m.textarea.Value() != "" {
					m.submitting = true
					return m, m.submitComment("passive")
				}

			case FocusInternal:
				if m.textarea.Value() != "" {
					m.submitting = true
					return m, m.submitComment("internal")
				}

			case FocusCancel:
				// Save draft when canceling (only for new comments)
				if !m.isEditMode {
					storage.SaveDraft(m.issue.Key, storage.DraftTypeComment, m.textarea.Value())
				}
				return m, func() tea.Msg {
					return backToDetailMsg{}
				}
			}

		case "i":
			// Also allow 'i' to enter insert mode when on input
			if m.focus == FocusInput {
				m.mode = ModeInsert
				m.textarea.Focus()
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the comment input view
func (m *CommentModel) View() string {
	if m.submitting {
		return m.renderCentered(SpinnerStyle.Render("Submitting comment..."))
	}

	// Colors
	focusedBorder := lipgloss.Color("#7B68EE")
	unfocusedBorder := lipgloss.Color("#555555")
	insertModeBorder := lipgloss.Color("#98C379") // Green for insert mode

	// Determine border color for input
	mentionModeBorder := lipgloss.Color("#61AFEF") // Blue for mention mode
	inputBorderColor := unfocusedBorder
	if m.focus == FocusInput {
		if m.mode == ModeInsert {
			inputBorderColor = insertModeBorder
		} else if m.mode == ModeMention {
			inputBorderColor = mentionModeBorder
		} else {
			inputBorderColor = focusedBorder
		}
	}

	// Modal width
	modalWidth := 70

	// Title
	var titleText string
	if m.isEditMode {
		titleText = fmt.Sprintf("Edit Comment - %s", m.issue.Key)
	} else {
		titleText = fmt.Sprintf("Add Comment - %s", m.issue.Key)
	}
	title := TitleStyle.Render(titleText)

	// Error if any
	var errView string
	if m.err != nil {
		errView = ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	// Input box - wrap textarea in our own border
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Padding(0, 1)

	inputBox := inputStyle.Render(m.textarea.View())

	// Mention picker (shown when in mention mode)
	var mentionPicker string
	if m.mode == ModeMention {
		var pickerContent strings.Builder
		pickerContent.WriteString(fmt.Sprintf("@%s", m.mentionQuery))
		if m.mentionLoading {
			pickerContent.WriteString(" (searching...)")
		}
		pickerContent.WriteString("\n")

		if len(m.mentionUsers) == 0 && !m.mentionLoading {
			pickerContent.WriteString(HelpStyle.Render("  No users found"))
		} else {
			for i, user := range m.mentionUsers {
				prefix := "  "
				if i == m.mentionSelected {
					prefix = "> "
				}
				userLine := fmt.Sprintf("%s%s", prefix, user.DisplayName)
				if i == m.mentionSelected {
					userLine = lipgloss.NewStyle().
						Foreground(lipgloss.Color("#61AFEF")).
						Bold(true).
						Render(userLine)
				}
				pickerContent.WriteString(userLine + "\n")
			}
		}
		pickerContent.WriteString(HelpStyle.Render("\n↑↓ select • enter confirm • esc cancel"))

		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mentionModeBorder).
			Padding(0, 1).
			Width(40)
		mentionPicker = pickerStyle.Render(pickerContent.String())
	}

	// Button styles
	buttonStyle := func(focused bool) lipgloss.Style {
		borderColor := unfocusedBorder
		if focused {
			borderColor = focusedBorder
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 2)
	}

	submitBtn := buttonStyle(m.focus == FocusSubmit).Render("Submit")
	cancelBtn := buttonStyle(m.focus == FocusCancel).Render("Cancel")

	// Buttons row - hide Passive/Internal buttons in edit mode (can't change visibility of existing comment)
	var buttons string
	if m.isEditMode {
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, submitBtn, "  ", cancelBtn)
	} else {
		passiveBtn := buttonStyle(m.focus == FocusPassive).Render("Passive")
		internalBtn := buttonStyle(m.focus == FocusInternal).Render("Internal")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, submitBtn, "  ", passiveBtn, "  ", internalBtn, "  ", cancelBtn)
	}

	// Combine all elements
	var content string
	if mentionPicker != "" {
		content = lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			errView+inputBox,
			"",
			mentionPicker,
			"",
			buttons,
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			errView+inputBox,
			"",
			buttons,
		)
	}

	// Modal container
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content)

	return m.renderCentered(modal)
}

// renderCentered centers content on the screen
func (m *CommentModel) renderCentered(content string) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		if lipgloss.Width(line) > contentWidth {
			contentWidth = lipgloss.Width(line)
		}
	}

	// Vertical centering
	verticalPadding := (m.height - contentHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Horizontal centering
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
