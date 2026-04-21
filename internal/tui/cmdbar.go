package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CmdBarMode represents the mode of the command bar
type CmdBarMode int

const (
	CmdBarHidden CmdBarMode = iota
	CmdBarCommand    // :command (editing)
	CmdBarSearch     // /search (editing)
	CmdBarShowSearch // showing search result (not editing)
)

// CmdBarResult is sent when command bar input is submitted
type CmdBarResult struct {
	Mode    CmdBarMode
	Input   string
	Aborted bool
}

// CmdBarModel is the command/search bar at the bottom
type CmdBarModel struct {
	mode         CmdBarMode
	textInput    textinput.Model
	width        int
	searchFilter string // persisted search filter
}

// NewCmdBarModel creates a new command bar
func NewCmdBarModel(width int) *CmdBarModel {
	ti := textinput.New()
	ti.Width = width - 6

	return &CmdBarModel{
		mode:      CmdBarHidden,
		textInput: ti,
		width:     width,
	}
}

// IsActive returns true if the command bar is in editing mode
func (m *CmdBarModel) IsActive() bool {
	return m.mode == CmdBarCommand || m.mode == CmdBarSearch
}

// IsVisible returns true if the command bar should be shown
func (m *CmdBarModel) IsVisible() bool {
	return m.mode != CmdBarHidden
}

// Mode returns the current mode
func (m *CmdBarModel) Mode() CmdBarMode {
	return m.mode
}

// HasActiveSearch returns true if there's an active search filter
func (m *CmdBarModel) HasActiveSearch() bool {
	return m.searchFilter != ""
}

// GetSearchFilter returns the current search filter
func (m *CmdBarModel) GetSearchFilter() string {
	return m.searchFilter
}

// ClearSearch clears the search filter
func (m *CmdBarModel) ClearSearch() {
	m.searchFilter = ""
	m.mode = CmdBarHidden
}

// Open opens the command bar in the specified mode
func (m *CmdBarModel) Open(mode CmdBarMode) tea.Cmd {
	m.mode = mode
	// Pre-fill with existing search filter if re-opening search
	if mode == CmdBarSearch && m.searchFilter != "" {
		m.textInput.SetValue(m.searchFilter)
	} else {
		m.textInput.SetValue("")
	}
	m.textInput.Focus()

	switch mode {
	case CmdBarCommand:
		m.textInput.Prompt = ":"
	case CmdBarSearch:
		m.textInput.Prompt = "/"
	}

	return textinput.Blink
}

// Close closes the command bar (but may keep showing search)
func (m *CmdBarModel) Close() {
	m.textInput.Blur()
	m.textInput.SetValue("")
	if m.searchFilter != "" {
		m.mode = CmdBarShowSearch
	} else {
		m.mode = CmdBarHidden
	}
}

// Update handles messages
func (m *CmdBarModel) Update(msg tea.Msg) (*CmdBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = msg.Width - 6
		return m, nil

	case tea.KeyMsg:
		// Handle ShowSearch mode (displaying search result)
		if m.mode == CmdBarShowSearch {
			switch msg.String() {
			case "esc", "enter":
				// Clear search and hide
				m.searchFilter = ""
				m.mode = CmdBarHidden
				return m, func() tea.Msg {
					return CmdBarResult{Mode: CmdBarSearch, Input: "", Aborted: true}
				}
			}
			return m, nil
		}

		// Handle editing modes
		if m.mode == CmdBarHidden {
			return m, nil
		}

		switch msg.String() {
		case "esc":
			mode := m.mode
			m.Close()
			return m, func() tea.Msg {
				return CmdBarResult{Mode: mode, Aborted: true}
			}

		case "enter":
			mode := m.mode
			input := m.textInput.Value()

			if mode == CmdBarSearch {
				// Store the search filter and switch to show mode
				m.searchFilter = input
				m.textInput.Blur()
				m.textInput.SetValue("")
				if input != "" {
					m.mode = CmdBarShowSearch
				} else {
					m.mode = CmdBarHidden
				}
			} else {
				m.Close()
			}

			return m, func() tea.Msg {
				return CmdBarResult{Mode: mode, Input: input, Aborted: false}
			}
		}

		// Pass to text input
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// SearchValue returns the current search value (for live filtering)
func (m *CmdBarModel) SearchValue() string {
	if m.mode == CmdBarSearch {
		return m.textInput.Value()
	}
	return m.searchFilter
}

// View renders the command bar with border
func (m *CmdBarModel) View() string {
	if m.mode == CmdBarHidden {
		return ""
	}

	var content string

	if m.mode == CmdBarShowSearch {
		// Show the active search filter
		content = fmt.Sprintf("/%s", m.searchFilter)
	} else {
		// Show the text input
		content = m.textInput.View()
	}

	barStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Width(m.width - 2).
		Padding(0, 1)

	return barStyle.Render(content)
}

// ViewWithHints renders the command bar with hints below
func (m *CmdBarModel) ViewWithHints(hints string) string {
	bar := m.View()
	if bar == "" {
		return HelpStyle.Render(hints)
	}

	// Hints go below the bar
	return lipgloss.JoinVertical(lipgloss.Left, bar, HelpStyle.Render(hints))
}

// browseIssueMsg signals to open the current issue in browser
type browseIssueMsg struct{}

// copyIssueURLMsg signals to copy the current issue URL to the clipboard
type copyIssueURLMsg struct{}

// ExecuteCommand executes a command and returns appropriate message
func ExecuteCommand(cmd string) tea.Msg {
	cmd = strings.TrimSpace(cmd)

	switch cmd {
	case "q", "quit", "exit":
		return tea.Quit()
	case "w", "write":
		// Could be used for saving in the future
		return nil
	case "wq":
		return tea.Quit()
	case "browse", "open", "b":
		return browseIssueMsg{}
	case "url":
		return copyIssueURLMsg{}
	}

	return nil
}
