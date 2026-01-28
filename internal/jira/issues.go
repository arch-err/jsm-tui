package jira

import (
	"fmt"
	"net/url"
)

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
	AccountID    string `json:"accountId,omitempty"`
	Name         string `json:"name,omitempty"` // For Jira Server/Data Center
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
	// Use cached service desk ID if available, otherwise get it
	serviceDeskID := c.serviceDeskID
	if serviceDeskID == "" {
		var err error
		serviceDeskID, err = c.GetServiceDeskID(projectKey)
		if err != nil {
			return nil, err
		}
		c.serviceDeskID = serviceDeskID
	}

	path := fmt.Sprintf("/rest/servicedeskapi/servicedesk/%s/queue/%s/issue?start=%d&limit=%d",
		serviceDeskID, queueID, start, limit)

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

// SearchAssignableUsers searches for users that can be assigned to an issue
func (c *Client) SearchAssignableUsers(issueKey, query string) ([]User, error) {
	// Jira Server uses 'username', Jira Cloud uses 'query' - include both for compatibility
	path := fmt.Sprintf("/rest/api/2/user/assignable/search?issueKey=%s&username=%s&query=%s&maxResults=10",
		url.QueryEscape(issueKey), url.QueryEscape(query), url.QueryEscape(query))

	var users []User
	if err := c.Get(path, &users); err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	return users, nil
}

// AssignIssue assigns an issue to a user. Pass nil to unassign.
func (c *Client) AssignIssue(issueKey string, user *User) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/assignee", issueKey)

	var body map[string]interface{}
	if user == nil {
		// Unassign - send null or -1 depending on Jira version
		body = map[string]interface{}{"name": nil}
	} else if user.Name != "" {
		// Jira Server/Data Center uses "name"
		body = map[string]interface{}{"name": user.Name}
	} else if user.AccountID != "" {
		// Jira Cloud uses "accountId"
		body = map[string]interface{}{"accountId": user.AccountID}
	}

	resp, err := c.doRequest("PUT", path, body)
	if err != nil {
		return fmt.Errorf("failed to assign issue: %w", err)
	}
	resp.Body.Close()

	return nil
}
