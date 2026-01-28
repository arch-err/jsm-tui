package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7B68EE")
	successColor   = lipgloss.Color("#00FF00")
	warningColor   = lipgloss.Color("#FFA500")
	errorColor     = lipgloss.Color("#FF0000")
	infoColor      = lipgloss.Color("#00BFFF")
	subtleColor    = lipgloss.Color("#666666")
	yellowColor    = lipgloss.Color("#FFFF00")
	orangeColor    = lipgloss.Color("#FF8C00")
	blueColor      = lipgloss.Color("#4169E1")
	tealColor      = lipgloss.Color("#20B2AA")
	dimGrayColor   = lipgloss.Color("#555555")

	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Width(80)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	// Help bar style
	HelpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	// Table header style
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	// Selected row style
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Bold(true)

	// Status badge styles
	StatusOpenStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true)

	StatusInProgressStyle = lipgloss.NewStyle().
				Foreground(yellowColor).
				Bold(true)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	StatusEscalatedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	StatusWaitingSupportStyle = lipgloss.NewStyle().
					Foreground(orangeColor).
					Bold(true)

	StatusPendingStyle = lipgloss.NewStyle().
				Foreground(blueColor).
				Bold(true)

	StatusWaitingCustomerStyle = lipgloss.NewStyle().
					Foreground(successColor).
					Bold(true)

	// Assignee styles
	AssigneeMeStyle = lipgloss.NewStyle().
			Foreground(tealColor)

	AssigneeUnassignedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	AssigneeOtherStyle = lipgloss.NewStyle().
				Foreground(dimGrayColor)

	// Dimmed row style (for issues assigned to others)
	DimmedTextStyle = lipgloss.NewStyle().
			Foreground(dimGrayColor)

	// Priority styles
	PriorityHighStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	PriorityMediumStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	PriorityLowStyle = lipgloss.NewStyle().
				Foreground(infoColor)

	// Error message style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errorColor)

	// Loading spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Key binding style
	KeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Description style
	DescStyle = lipgloss.NewStyle().
			Foreground(subtleColor)
)

// GetStatusStyle returns the appropriate style for a status name
func GetStatusStyle(statusName string) lipgloss.Style {
	switch statusName {
	case "Escalated":
		return StatusEscalatedStyle
	case "In Progress", "In progress":
		return StatusInProgressStyle
	case "Waiting for support", "Waiting for Support":
		return StatusWaitingSupportStyle
	case "Pending":
		return StatusPendingStyle
	case "Waiting for customer", "Waiting for Customer":
		return StatusWaitingCustomerStyle
	case "Done", "Resolved", "Closed":
		return StatusDoneStyle
	default:
		return StatusOpenStyle
	}
}

// GetAssigneeStyle returns the appropriate style for an assignee
func GetAssigneeStyle(assigneeName string, currentUser string) lipgloss.Style {
	if assigneeName == "Unassigned" || assigneeName == "" {
		return AssigneeUnassignedStyle
	}
	if assigneeName == currentUser {
		return AssigneeMeStyle
	}
	return AssigneeOtherStyle
}

// IsAssignedToOther returns true if the issue is assigned to someone other than the current user
func IsAssignedToOther(assigneeName string, currentUser string) bool {
	if assigneeName == "Unassigned" || assigneeName == "" {
		return false
	}
	return assigneeName != currentUser
}

// GetPriorityStyle returns the appropriate style for a priority
func GetPriorityStyle(priority string) lipgloss.Style {
	switch priority {
	case "Highest", "High":
		return PriorityHighStyle
	case "Medium":
		return PriorityMediumStyle
	case "Low", "Lowest":
		return PriorityLowStyle
	default:
		return lipgloss.NewStyle()
	}
}
