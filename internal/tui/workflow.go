package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// WorkflowModel handles the workflow picker modal
type WorkflowModel struct {
	workflows     []config.WorkflowConfig
	issue         jira.Issue
	proformaForms []jira.ProformaForm
	comments      []jira.Comment
	keys          KeyMap
	selectedIndex int
	width         int
	height        int
	executing     bool
	err           error
	lastGPress    time.Time
}

// NewWorkflowModel creates a new workflow picker
func NewWorkflowModel(cfg *config.Config, issue jira.Issue, proformaForms []jira.ProformaForm, comments []jira.Comment, keys KeyMap, width, height int) *WorkflowModel {
	// Filter workflows by request type
	requestType := ""
	if issue.Fields.CustomerRequestType != nil && issue.Fields.CustomerRequestType.RequestType != nil {
		requestType = issue.Fields.CustomerRequestType.RequestType.Name
	}

	var filtered []config.WorkflowConfig
	for _, wf := range cfg.Workflows {
		if wf.RequestTypes.Matches(requestType) {
			filtered = append(filtered, wf)
		}
	}

	return &WorkflowModel{
		workflows:     filtered,
		issue:         issue,
		proformaForms: proformaForms,
		comments:      comments,
		keys:          keys,
		width:         width,
		height:        height,
	}
}

type workflowCompletedMsg struct{}
type workflowErrorMsg struct{ err error }

// Update handles messages
func (m *WorkflowModel) Update(msg tea.Msg) (*WorkflowModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case workflowErrorMsg:
		m.err = msg.err
		m.executing = false
		return m, nil

	case tea.KeyMsg:
		if m.executing {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.workflows)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if len(m.workflows) > 0 {
				m.executing = true
				return m, m.executeWorkflow(m.workflows[m.selectedIndex])
			}
			return m, nil

		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg {
				return backToDetailMsg{}
			}

		case key.Matches(msg, m.keys.GoToBottom):
			if len(m.workflows) > 0 {
				m.selectedIndex = len(m.workflows) - 1
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

// executeWorkflow runs the selected workflow script
func (m *WorkflowModel) executeWorkflow(wf config.WorkflowConfig) tea.Cmd {
	return func() tea.Msg {
		// Create temp directory
		tmpDir := "/tmp/jsm-tui"
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return workflowErrorMsg{err: fmt.Errorf("failed to create temp dir: %w", err)}
		}

		// Generate issue YAML
		issueYAML, err := m.generateIssueYAML()
		if err != nil {
			return workflowErrorMsg{err: fmt.Errorf("failed to generate issue YAML: %w", err)}
		}

		// Write to temp file
		tmpFile := filepath.Join(tmpDir, fmt.Sprintf("%s_%d.yaml", m.issue.Key, time.Now().Unix()))
		if err := os.WriteFile(tmpFile, []byte(issueYAML), 0644); err != nil {
			return workflowErrorMsg{err: fmt.Errorf("failed to write temp file: %w", err)}
		}

		// Get script path
		scriptPath, err := config.GetWorkflowScriptPath(wf.Script)
		if err != nil {
			return workflowErrorMsg{err: err}
		}

		// Check if script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return workflowErrorMsg{err: fmt.Errorf("workflow script not found: %s", scriptPath)}
		}

		// Execute script
		cmd := exec.Command(scriptPath, tmpFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return workflowErrorMsg{err: fmt.Errorf("failed to start workflow: %w", err)}
		}

		// Don't wait for completion - let it run in background
		go cmd.Wait()

		return workflowCompletedMsg{}
	}
}

// IssueExport is the YAML structure for exported issue data
type IssueExport struct {
	Key         string            `yaml:"key"`
	Summary     string            `yaml:"summary"`
	Description string            `yaml:"description,omitempty"`
	Status      string            `yaml:"status"`
	RequestType string            `yaml:"request_type,omitempty"`
	Assignee    string            `yaml:"assignee,omitempty"`
	Reporter    string            `yaml:"reporter,omitempty"`
	Created     string            `yaml:"created"`
	Updated     string            `yaml:"updated"`
	Priority    string            `yaml:"priority,omitempty"`
	Forms       []FormExport      `yaml:"forms,omitempty"`
	Comments    []CommentExport   `yaml:"comments,omitempty"`
	CustomData  map[string]string `yaml:"custom_data,omitempty"`
}

type FormExport struct {
	Name   string            `yaml:"name"`
	Fields map[string]string `yaml:"fields"`
}

type CommentExport struct {
	Author  string `yaml:"author"`
	Created string `yaml:"created"`
	Body    string `yaml:"body"`
}

// generateIssueYAML creates a YAML representation of the issue
func (m *WorkflowModel) generateIssueYAML() (string, error) {
	export := IssueExport{
		Key:         m.issue.Key,
		Summary:     m.issue.Fields.Summary,
		Description: m.issue.Fields.Description,
		Status:      m.issue.Fields.Status.Name,
		Created:     m.issue.Fields.Created,
		Updated:     m.issue.Fields.Updated,
	}

	if m.issue.Fields.CustomerRequestType != nil && m.issue.Fields.CustomerRequestType.RequestType != nil {
		export.RequestType = m.issue.Fields.CustomerRequestType.RequestType.Name
	}

	if m.issue.Fields.Assignee != nil {
		export.Assignee = m.issue.Fields.Assignee.DisplayName
	}

	if m.issue.Fields.Reporter != nil {
		export.Reporter = m.issue.Fields.Reporter.DisplayName
	}

	if m.issue.Fields.Priority.Name != "" {
		export.Priority = m.issue.Fields.Priority.Name
	}

	// Export forms
	for _, form := range m.proformaForms {
		formExport := FormExport{
			Name:   form.Name,
			Fields: make(map[string]string),
		}
		for _, field := range form.Fields {
			if field.Answer != "" && field.Answer != "-" {
				formExport.Fields[field.Label] = field.Answer
			}
		}
		if len(formExport.Fields) > 0 {
			export.Forms = append(export.Forms, formExport)
		}
	}

	// Export comments
	for _, comment := range m.comments {
		export.Comments = append(export.Comments, CommentExport{
			Author:  comment.Author.DisplayName,
			Created: comment.Created,
			Body:    comment.Body,
		})
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// View renders the workflow picker
func (m *WorkflowModel) View() string {
	if m.executing {
		return SpinnerStyle.Render("Launching workflow...")
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n" +
			HelpStyle.Render("esc to go back")
	}

	var content strings.Builder

	content.WriteString(TitleStyle.Render("Select Workflow") + "\n\n")

	if len(m.workflows) == 0 {
		content.WriteString(HelpStyle.Render("No workflows available for this request type.") + "\n")
	} else {
		for i, wf := range m.workflows {
			line := fmt.Sprintf("  %s", wf.Name)
			if i == m.selectedIndex {
				line = SelectedStyle.Render(line)
			}
			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n" + HelpStyle.Render("enter run • esc cancel"))

	// Render as centered popup
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
