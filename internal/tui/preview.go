package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PreviewModel displays a field's content in a read-only modal
type PreviewModel struct {
	title    string
	content  string
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewPreviewModel creates a new preview modal
func NewPreviewModel(title, content string, width, height int) *PreviewModel {
	m := &PreviewModel{
		title:   title,
		content: content,
		width:   width,
		height:  height,
	}
	m.setupViewport()
	return m
}

// setupViewport initializes the viewport
func (m *PreviewModel) setupViewport() {
	// Modal dimensions
	modalWidth := m.width - 10
	if modalWidth > 100 {
		modalWidth = 100
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Account for: outer border (2), padding (4), title (1), separators (2), footer (1)
	modalHeight := m.height - 16
	if modalHeight < 5 {
		modalHeight = 5
	}

	// Viewport width: modal width - border (2) - padding (4)
	viewportWidth := modalWidth - 6
	if viewportWidth < 30 {
		viewportWidth = 30
	}

	// Viewport for content
	m.viewport = viewport.New(viewportWidth, modalHeight)
	m.viewport.SetContent(wrapText(m.content, viewportWidth-2))
	m.ready = true
}

// closePreviewMsg signals to close the preview modal
type closePreviewMsg struct{}

// previewEditorFinishedMsg is sent when the editor closes
type previewEditorFinishedMsg struct {
	err error
}

// openInEditorReadOnly opens the content in $EDITOR as read-only
func (m *PreviewModel) openInEditorReadOnly() tea.Cmd {
	content := m.content

	// Create temp file
	tmpFile, err := os.CreateTemp("", "jsm-preview-*.txt")
	if err != nil {
		return func() tea.Msg {
			return previewEditorFinishedMsg{err: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}
	tmpPath := tmpFile.Name()

	// Write content
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return func() tea.Msg {
			return previewEditorFinishedMsg{err: fmt.Errorf("failed to write temp file: %w", err)}
		}
	}
	tmpFile.Close()

	// Make file read-only
	os.Chmod(tmpPath, 0444)

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}

	// Build command with read-only flags based on editor
	var c *exec.Cmd
	switch {
	case strings.Contains(editor, "vim") || strings.Contains(editor, "nvim"):
		c = exec.Command(editor, "-R", tmpPath) // -R for read-only
	case strings.Contains(editor, "nano"):
		c = exec.Command(editor, "-v", tmpPath) // -v for view mode
	case strings.Contains(editor, "emacs"):
		c = exec.Command(editor, "--eval", "(view-file \""+tmpPath+"\")")
	default:
		c = exec.Command(editor, tmpPath)
	}

	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(tmpPath)
		return previewEditorFinishedMsg{err: err}
	})
}

// Update handles messages
func (m *PreviewModel) Update(msg tea.Msg) (*PreviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.setupViewport()
		return m, nil

	case previewEditorFinishedMsg:
		// Editor closed, just stay in preview
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return closePreviewMsg{}
			}

		case "ctrl+g":
			return m, m.openInEditorReadOnly()

		case "j", "down":
			m.viewport.LineDown(1)
			return m, nil

		case "k", "up":
			m.viewport.LineUp(1)
			return m, nil

		case "g":
			m.viewport.GotoTop()
			return m, nil

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "d", "ctrl+d":
			m.viewport.HalfViewDown()
			return m, nil

		case "u", "ctrl+u":
			m.viewport.HalfViewUp()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the preview modal
func (m *PreviewModel) View() string {
	if !m.ready {
		return ""
	}

	// Modal dimensions
	modalWidth := m.width - 10
	if modalWidth > 100 {
		modalWidth = 100
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Title
	title := TitleStyle.Render(m.title)

	// Scroll indicator
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		percent := int(m.viewport.ScrollPercent() * 100)
		scrollInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf(" (%d%%)", percent))
	}

	// Footer
	footer := HelpStyle.Render("j/k scroll • ctrl+g open in editor • q/esc close")

	// Combine content (no extra border around viewport)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title+scrollInfo,
		strings.Repeat("─", modalWidth-6),
		m.viewport.View(),
		strings.Repeat("─", modalWidth-6),
		footer,
	)

	// Modal container
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7B68EE")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content)

	// Center the modal
	return m.renderCentered(modal)
}

// renderCentered centers content on screen
func (m *PreviewModel) renderCentered(content string) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > contentWidth {
			contentWidth = w
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
