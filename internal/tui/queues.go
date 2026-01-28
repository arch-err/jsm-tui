package tui

import (
	"fmt"

	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// QueuesModel handles the queue list view
type QueuesModel struct {
	client        *jira.Client
	projectKey    string
	keys          KeyMap
	queues        []jira.Queue
	selectedIndex int
	loading       bool
	err           error
}

// NewQueuesModel creates a new queue list model
func NewQueuesModel(client *jira.Client, projectKey string, keys KeyMap) *QueuesModel {
	return &QueuesModel{
		client:     client,
		projectKey: projectKey,
		keys:       keys,
		loading:    true,
	}
}

type queuesLoadedMsg struct{ queues []jira.Queue }

// Init initializes the view
func (m *QueuesModel) Init() tea.Cmd {
	return m.fetchQueues()
}

// fetchQueues loads queues from the API
func (m *QueuesModel) fetchQueues() tea.Cmd {
	return func() tea.Msg {
		queues, err := m.client.GetQueues(m.projectKey)
		if err != nil {
			return errorMsg{err: err}
		}
		return queuesLoadedMsg{queues: queues}
	}
}

// Update handles messages
func (m *QueuesModel) Update(msg tea.Msg) (*QueuesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case queuesLoadedMsg:
		m.queues = msg.queues
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
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.queues)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if len(m.queues) > 0 {
				return m, func() tea.Msg {
					return queueSelectedMsg{queue: m.queues[m.selectedIndex]}
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, m.fetchQueues()
		}
	}

	return m, nil
}

// View renders the queue list
func (m *QueuesModel) View() string {
	if m.loading {
		return SpinnerStyle.Render("Loading queues...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.queues) == 0 {
		return "No queues found."
	}

	s := HeaderStyle.Render(fmt.Sprintf("Queues - %s", m.projectKey)) + "\n\n"

	for i, queue := range m.queues {
		// Add star indicator for favorite queues
		prefix := "  "
		if queue.IsFavorite {
			prefix = "★ "
		}
		line := fmt.Sprintf("%s%s", prefix, queue.Name)
		if i == m.selectedIndex {
			line = SelectedStyle.Render(line)
		}
		s += line + "\n"
	}

	s += "\n" + HelpStyle.Render("↑/k up • ↓/j down • enter select • r refresh • q quit")

	return s
}
