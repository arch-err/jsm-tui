package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmResult is sent when the confirm modal completes
type ConfirmResult struct {
	Confirmed bool
	Input     string // For input modals
	ID        string // To identify which confirm this is
}

// ConfirmModel is a reusable confirmation/input modal
type ConfirmModel struct {
	id          string
	title       string
	message     string
	showInput   bool
	textInput   textinput.Model
	width       int
	height      int
	confirmed   bool
	cancelled   bool
}

// NewConfirmModel creates a simple yes/no confirmation modal
func NewConfirmModel(id, title, message string, width, height int) *ConfirmModel {
	return &ConfirmModel{
		id:      id,
		title:   title,
		message: message,
		width:   width,
		height:  height,
	}
}

// NewInputConfirmModel creates a confirmation modal with text input
func NewInputConfirmModel(id, title, message, placeholder, initialValue string, width, height int) *ConfirmModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(initialValue)
	ti.Focus()
	ti.Width = 40

	return &ConfirmModel{
		id:        id,
		title:     title,
		message:   message,
		showInput: true,
		textInput: ti,
		width:     width,
		height:    height,
	}
}

// Init initializes the modal
func (m *ConfirmModel) Init() tea.Cmd {
	if m.showInput {
		return textinput.Blink
	}
	return nil
}

// Update handles messages
func (m *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: false, ID: m.id}
			}

		case "enter":
			m.confirmed = true
			return m, func() tea.Msg {
				return ConfirmResult{
					Confirmed: true,
					Input:     m.textInput.Value(),
					ID:        m.id,
				}
			}

		case "y", "Y":
			if !m.showInput {
				m.confirmed = true
				return m, func() tea.Msg {
					return ConfirmResult{Confirmed: true, ID: m.id}
				}
			}

		case "n", "N":
			if !m.showInput {
				m.cancelled = true
				return m, func() tea.Msg {
					return ConfirmResult{Confirmed: false, ID: m.id}
				}
			}
		}

		// Pass to text input if showing input
		if m.showInput {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the modal
func (m *ConfirmModel) View() string {
	var content strings.Builder

	// Title
	content.WriteString(TitleStyle.Render(m.title))
	content.WriteString("\n\n")

	// Message
	content.WriteString(m.message)
	content.WriteString("\n\n")

	// Input if showing
	if m.showInput {
		content.WriteString(m.textInput.View())
		content.WriteString("\n\n")
		content.WriteString(HelpStyle.Render("enter confirm • esc cancel"))
	} else {
		content.WriteString(HelpStyle.Render("y/enter confirm • n/esc cancel"))
	}

	// Popup style
	popupWidth := 50
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(popupWidth)

	popup := popupStyle.Render(content.String())

	// Center the popup
	popupHeight := strings.Count(popup, "\n") + 1
	verticalPadding := (m.height - popupHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	horizontalPadding := (m.width - popupWidth - 4) / 2
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	// Build centered view
	var view strings.Builder
	for i := 0; i < verticalPadding; i++ {
		view.WriteString("\n")
	}

	lines := strings.Split(popup, "\n")
	for _, line := range lines {
		view.WriteString(strings.Repeat(" ", horizontalPadding))
		view.WriteString(line)
		view.WriteString("\n")
	}

	return view.String()
}
