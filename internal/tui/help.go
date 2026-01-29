package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModel displays keybindings help
type HelpModel struct {
	width  int
	height int
}

// NewHelpModel creates a new help model
func NewHelpModel(width, height int) *HelpModel {
	return &HelpModel{
		width:  width,
		height: height,
	}
}

// closeHelpMsg is sent when help should close
type closeHelpMsg struct{}

// Update handles messages
func (m *HelpModel) Update(msg tea.Msg) (*HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Any key closes help
		return m, func() tea.Msg {
			return closeHelpMsg{}
		}
	}

	return m, nil
}

// View renders the help modal
func (m *HelpModel) View() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Width(12)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("215")).
		Bold(true).
		MarginTop(1)

	var content strings.Builder

	// Title
	content.WriteString(TitleStyle.Render("Keybindings") + "\n")

	// Navigation
	content.WriteString(sectionStyle.Render("Navigation") + "\n")
	content.WriteString(keyStyle.Render("j / ↓") + descStyle.Render("Move down") + "\n")
	content.WriteString(keyStyle.Render("k / ↑") + descStyle.Render("Move up") + "\n")
	content.WriteString(keyStyle.Render("g g") + descStyle.Render("Go to top") + "\n")
	content.WriteString(keyStyle.Render("G") + descStyle.Render("Go to bottom") + "\n")
	content.WriteString(keyStyle.Render("g x") + descStyle.Render("Open URL in browser") + "\n")
	content.WriteString(keyStyle.Render("Enter") + descStyle.Render("Select / Open") + "\n")
	content.WriteString(keyStyle.Render("Esc") + descStyle.Render("Back / Cancel") + "\n")

	// Actions
	content.WriteString(sectionStyle.Render("Actions") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Transition status") + "\n")
	content.WriteString(keyStyle.Render("c") + descStyle.Render("Add comment") + "\n")
	content.WriteString(keyStyle.Render("a") + descStyle.Render("Assign issue") + "\n")
	content.WriteString(keyStyle.Render("R") + descStyle.Render("Rename issue") + "\n")
	content.WriteString(keyStyle.Render("w") + descStyle.Render("Run workflow") + "\n")
	content.WriteString(keyStyle.Render("r") + descStyle.Render("Refresh") + "\n")

	// Yank/Copy
	content.WriteString(sectionStyle.Render("Copy") + "\n")
	content.WriteString(keyStyle.Render("y y") + descStyle.Render("Copy issue key") + "\n")
	content.WriteString(keyStyle.Render("y") + descStyle.Render("Copy selected field") + "\n")

	// Queues
	content.WriteString(sectionStyle.Render("Queues") + "\n")
	content.WriteString(keyStyle.Render("*") + descStyle.Render("Toggle favorite") + "\n")
	content.WriteString(keyStyle.Render("h") + descStyle.Render("Toggle show favorites only") + "\n")

	// Search & Command
	content.WriteString(sectionStyle.Render("Search & Command") + "\n")
	content.WriteString(keyStyle.Render("/") + descStyle.Render("Search / Filter") + "\n")
	content.WriteString(keyStyle.Render("n") + descStyle.Render("Next match") + "\n")
	content.WriteString(keyStyle.Render("N") + descStyle.Render("Previous match") + "\n")
	content.WriteString(keyStyle.Render(":") + descStyle.Render("Command mode") + "\n")
	content.WriteString(keyStyle.Render(":q") + descStyle.Render("Quit") + "\n")

	// Quit
	content.WriteString(sectionStyle.Render("Quit") + "\n")
	content.WriteString(keyStyle.Render("Z Z") + descStyle.Render("Quit") + "\n")
	content.WriteString(keyStyle.Render("Ctrl+C") + descStyle.Render("Force quit") + "\n")

	// Footer
	content.WriteString("\n" + HelpStyle.Render("Press any key to close"))

	// Popup style
	popupWidth := 40
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
