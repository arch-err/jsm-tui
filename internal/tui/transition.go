package tui

import (
	"fmt"
	"time"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TransitionModel handles the transition selection view
type TransitionModel struct {
	client        *jira.Client
	keys          KeyMap
	issue         jira.Issue
	transitions   []jira.Transition
	selectedIndex int
	loading       bool
	executing     bool
	err           error
	lastGPress    time.Time
}

// NewTransitionModel creates a new transition model
func NewTransitionModel(client *jira.Client, issue jira.Issue, keys KeyMap) *TransitionModel {
	return &TransitionModel{
		client:  client,
		keys:    keys,
		issue:   issue,
		loading: true,
	}
}

type transitionsLoadedMsg struct{ transitions []jira.Transition }

// Init initializes the view
func (m *TransitionModel) Init() tea.Cmd {
	return m.fetchTransitions()
}

// fetchTransitions loads available transitions
func (m *TransitionModel) fetchTransitions() tea.Cmd {
	return func() tea.Msg {
		transitions, err := m.client.GetTransitions(m.issue.Key)
		if err != nil {
			return errorMsg{err: err}
		}
		return transitionsLoadedMsg{transitions: transitions}
	}
}

// executeTransition executes the selected transition
func (m *TransitionModel) executeTransition(transitionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.ExecuteTransition(m.issue.Key, transitionID)
		if err != nil {
			return errorMsg{err: err}
		}
		return transitionCompletedMsg{}
	}
}

// Update handles messages
func (m *TransitionModel) Update(msg tea.Msg) (*TransitionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case transitionsLoadedMsg:
		m.transitions = msg.transitions
		m.loading = false
		return m, nil

	case transitionCompletedMsg:
		// Transition completed, return to detail view
		return m, func() tea.Msg {
			return transitionCompletedMsg{}
		}

	case tea.KeyMsg:
		if m.loading || m.executing {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.transitions)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if len(m.transitions) > 0 {
				m.executing = true
				return m, m.executeTransition(m.transitions[m.selectedIndex].ID)
			}
			return m, nil

		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg {
				return backToDetailMsg{}
			}

		case key.Matches(msg, m.keys.GoToBottom):
			if len(m.transitions) > 0 {
				m.selectedIndex = len(m.transitions) - 1
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
	}

	return m, nil
}

// View renders the transition selection view
func (m *TransitionModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading transitions...")
	}

	if m.executing {
		return SpinnerStyle.Render("Executing transition...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.transitions) == 0 {
		return "No transitions available for this issue.\n\n" +
			HelpStyle.Render("esc back")
	}

	s := HeaderStyle.Render(fmt.Sprintf("Select Transition - %s", m.issue.Key)) + "\n\n"

	for i, transition := range m.transitions {
		line := fmt.Sprintf("  %s → %s", transition.Name, transition.To.Name)
		if i == m.selectedIndex {
			line = SelectedStyle.Render(line)
		}
		s += line + "\n"
	}

	s += "\n" + HelpStyle.Render("↑↓/jk navigate • enter execute • esc cancel")

	return s
}
