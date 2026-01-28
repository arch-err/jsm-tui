package jira

import "fmt"

// Issue represents a Jira issue
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains issue field data
type IssueFields struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	Priority    Priority    `json:"priority"`
	Assignee    *User       `json:"assignee"`
	Reporter    *User       `json:"reporter"`
	Created     string      `json:"created"`
	Updated     string      `json:"updated"`
	IssueType   IssueType   `json:"issuetype"`
	Comment     CommentList `json:"comment,omitempty"`
}

// Status represents issue status
type Status struct {
	Name           string `json:"name"`
	StatusCategory struct {
		Name string `json:"name"`
	} `json:"statusCategory"`
}

// Priority represents issue priority
type Priority struct {
	Name    string `json:"name"`
	IconURL string `json:"iconUrl"`
}

// User represents a Jira user
type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Active       bool   `json:"active"`
}

// IssueType represents the type of issue
type IssueType struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CommentList contains a list of comments
type CommentList struct {
	Comments []Comment `json:"comments"`
}

// IssuesResponse represents paginated issues response
type IssuesResponse struct {
	Size       int     `json:"size"`
	Start      int     `json:"start"`
	Limit      int     `json:"limit"`
	IsLastPage bool    `json:"isLastPage"`
	Values     []Issue `json:"values"`
}

// GetQueueIssues fetches issues from a specific queue
func (c *Client) GetQueueIssues(projectKey, queueID string, start, limit int) ([]Issue, error) {
	path := fmt.Sprintf("/rest/servicedeskapi/servicedesk/%s/queue/%s/issue?start=%d&limit=%d",
		projectKey, queueID, start, limit)

	var resp IssuesResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get queue issues: %w", err)
	}

	return resp.Values, nil
}

// GetIssue fetches full details for a specific issue
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)

	var issue Issue
	if err := c.Get(path, &issue); err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	return &issue, nil
}
