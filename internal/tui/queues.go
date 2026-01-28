package tui

import (
	"fmt"
	"time"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// QueuesModel handles the queue list view
type QueuesModel struct {
	client        *jira.Client
	cfg           *config.Config
	projectKey    string
	keys          KeyMap
	queues        []jira.Queue
	selectedIndex int
	loading       bool
	err           error
	lastEscPress  time.Time
	lastGPress    time.Time
	showEscHint   bool
	width         int
	height        int
}

// NewQueuesModel creates a new queue list model
func NewQueuesModel(client *jira.Client, cfg *config.Config, projectKey string, keys KeyMap) *QueuesModel {
	return &QueuesModel{
		client:     client,
		cfg:        cfg,
		projectKey: projectKey,
		keys:       keys,
		loading:    true,
	}
}

type queuesLoadedMsg struct{ queues []jira.Queue }
type clearEscHintMsg struct{}

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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case queuesLoadedMsg:
		m.queues = msg.queues
		m.loading = false
		return m, nil

	case clearEscHintMsg:
		m.showEscHint = false
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

		case key.Matches(msg, m.keys.ToggleFavorite):
			// Toggle favorite for selected queue
			if len(m.queues) > 0 {
				selectedQueue := m.queues[m.selectedIndex]
				m.cfg.ToggleFavoriteQueue(selectedQueue.Name)
				if err := m.cfg.Save(); err != nil {
					m.err = err
					return m, nil
				}
				// Update client's favorite list
				m.client.UpdateFavorites(m.cfg.Queues.Favorites, m.cfg.Queues.HideNonFavorites)
				// Refresh queues to show updated favorite status
				m.loading = true
				return m, m.fetchQueues()
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleHideNonFavorites):
			// Toggle hiding non-favorite queues
			m.cfg.Queues.HideNonFavorites = !m.cfg.Queues.HideNonFavorites
			if err := m.cfg.Save(); err != nil {
				m.err = err
				return m, nil
			}
			// Update client settings
			m.client.UpdateFavorites(m.cfg.Queues.Favorites, m.cfg.Queues.HideNonFavorites)
			// Reset selection to avoid out of bounds
			m.selectedIndex = 0
			// Refresh queues to apply filter
			m.loading = true
			return m, m.fetchQueues()

		case key.Matches(msg, m.keys.Back):
			// Double-tap ESC to quit
			now := time.Now()
			// Check if last ESC was within 2 seconds
			if !m.lastEscPress.IsZero() && now.Sub(m.lastEscPress) < 2*time.Second {
				// Second ESC press within 2 seconds - quit
				return m, tea.Quit
			}
			// First ESC press - show hint
			m.lastEscPress = now
			m.showEscHint = true
			// Clear hint after 2 seconds
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return clearEscHintMsg{}
			})

		case key.Matches(msg, m.keys.GoToBottom):
			// G - go to bottom
			if len(m.queues) > 0 {
				m.selectedIndex = len(m.queues) - 1
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToTop):
			// gg - double tap g to go to top
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

	helpText := "↑↓/jk navigate • enter select • * favorite • h hide others • r refresh • esc quit"
	if m.showEscHint {
		helpText = "Press ESC again to quit"
	}
	s += "\n" + HelpStyle.Render(helpText)

	return s
}
