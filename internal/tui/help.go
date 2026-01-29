package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModel displays keybindings help
type HelpModel struct {
	width    int
	height   int
	viewport viewport.Model
	ready    bool
}

// NewHelpModel creates a new help model
func NewHelpModel(width, height int) *HelpModel {
	m := &HelpModel{
		width:  width,
		height: height,
	}
	m.setupViewport()
	return m
}

// setupViewport initializes the viewport with content
func (m *HelpModel) setupViewport() {
	// Calculate modal dimensions
	modalWidth := 44
	maxModalHeight := m.height - 4 // Leave some margin

	// Build content
	content := m.buildContent()
	contentLines := strings.Count(content, "\n") + 1

	// Calculate actual modal height (content + padding + border)
	contentHeight := contentLines
	if contentHeight > maxModalHeight-4 {
		contentHeight = maxModalHeight - 4
	}

	// Create viewport
	m.viewport = viewport.New(modalWidth-4, contentHeight) // -4 for padding/border
	m.viewport.SetContent(content)
	m.ready = true
}

// buildContent builds the help text content
func (m *HelpModel) buildContent() string {
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
	content.WriteString(keyStyle.Render("e") + descStyle.Render("Edit comment (own)") + "\n")
	content.WriteString(keyStyle.Render("a") + descStyle.Render("Assign issue") + "\n")
	content.WriteString(keyStyle.Render("A") + descStyle.Render("Quick action") + "\n")
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
	content.WriteString(keyStyle.Render("h") + descStyle.Render("Show favorites only") + "\n")

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

	return content.String()
}

// closeHelpMsg is sent when help should close
type closeHelpMsg struct{}

// Update handles messages
func (m *HelpModel) Update(msg tea.Msg) (*HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.setupViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "?":
			// Close help
			return m, func() tea.Msg {
				return closeHelpMsg{}
			}
		case "j", "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k", "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
	}

	// Pass to viewport for other scrolling (pgup, pgdown, etc)
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the help modal
func (m *HelpModel) View() string {
	if !m.ready {
		return ""
	}

	// Title
	title := TitleStyle.Render("Keybindings")

	// Scroll indicator
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		percent := int(m.viewport.ScrollPercent() * 100)
		scrollInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf(" (%d%%)", percent))
	}

	// Footer
	footer := HelpStyle.Render("j/k scroll • q/esc close")

	// Calculate modal width
	modalWidth := 44

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title+scrollInfo,
		"",
		m.viewport.View(),
		"",
		footer,
	)

	// Popup style
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(modalWidth)

	popup := popupStyle.Render(content)

	// Center the popup
	popupHeight := lipgloss.Height(popup)
	popupWidth := lipgloss.Width(popup)

	verticalPadding := (m.height - popupHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	horizontalPadding := (m.width - popupWidth) / 2
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
