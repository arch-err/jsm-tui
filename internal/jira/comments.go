package jira

import (
	"encoding/json"
	"fmt"
)

// DateField can be either a string (Jira API) or an object with iso8601 (Service Desk API)
type DateField struct {
	ISO8601 string `json:"iso8601,omitempty"`
	Raw     string `json:"-"` // For direct string value
}

// UnmarshalJSON handles both string and object date formats
func (d *DateField) UnmarshalJSON(data []byte) error {
	// Try as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		d.Raw = s
		return nil
	}

	// Try as object with iso8601
	var obj struct {
		ISO8601 string `json:"iso8601"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		d.ISO8601 = obj.ISO8601
		return nil
	}

	return nil
}

// String returns the date as a string
func (d DateField) String() string {
	if d.Raw != "" {
		return d.Raw
	}
	return d.ISO8601
}

// Comment represents a comment on an issue
type Comment struct {
	ID           string        `json:"id"`
	Author       User          `json:"author"`
	Body         string        `json:"body"`
	Created      DateField     `json:"created"`
	Updated      DateField     `json:"updated"`
	Public       *bool         `json:"public,omitempty"`    // Service Desk API: true = public, false = internal
	JsdPublic    *bool         `json:"jsdPublic,omitempty"` // Jira API fallback
	RenderedBody string        `json:"renderedBody,omitempty"`
}

// GetCreated returns the created date as a string
func (c *Comment) GetCreated() string {
	return c.Created.String()
}

// IsInternal returns true if this is an internal comment (not visible to customers)
func (c *Comment) IsInternal() bool {
	// Service Desk API uses "public" field
	if c.Public != nil {
		return !*c.Public
	}
	// Jira API fallback uses "jsdPublic" field
	if c.JsdPublic != nil {
		return !*c.JsdPublic
	}
	return false
}

// CommentsResponse represents the response from regular Jira comments API
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// ServiceDeskCommentsResponse represents the response from Service Desk API
type ServiceDeskCommentsResponse struct {
	Size       int       `json:"size"`
	Start      int       `json:"start"`
	Limit      int       `json:"limit"`
	IsLastPage bool      `json:"isLastPage"`
	Values     []Comment `json:"values"`
}

// CommentProperty represents a comment property for JSM
type CommentProperty struct {
	Key   string                 `json:"key"`
	Value map[string]interface{} `json:"value"`
}

// AddCommentRequest represents a request to add a comment
type AddCommentRequest struct {
	Body       string            `json:"body"`
	Properties []CommentProperty `json:"properties,omitempty"`
}

// GetComments fetches all comments for an issue using Service Desk API
func (c *Client) GetComments(issueKey string) ([]Comment, error) {
	// Use Service Desk API to get public/internal flag
	path := fmt.Sprintf("/rest/servicedeskapi/request/%s/comment?public=true&internal=true", issueKey)

	var resp ServiceDeskCommentsResponse
	if err := c.Get(path, &resp); err != nil {
		// Fallback to regular Jira API if Service Desk API fails
		return c.getCommentsJiraAPI(issueKey)
	}

	return resp.Values, nil
}

// getCommentsJiraAPI fetches comments using regular Jira API (fallback)
func (c *Client) getCommentsJiraAPI(issueKey string) ([]Comment, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment?expand=properties,renderedBody", issueKey)

	var resp CommentsResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	return resp.Comments, nil
}

// AddComment adds a public comment to an issue (visible to customers)
// This may trigger workflow transitions (e.g., status change to "Waiting for Customer")
func (c *Client) AddComment(issueKey, body string) error {
	return c.addCommentWithOptions(issueKey, body, false, true)
}

// AddPassiveComment adds a public comment without triggering workflow transitions
// The comment is visible to customers but won't change the issue status
func (c *Client) AddPassiveComment(issueKey, body string) error {
	return c.addCommentWithOptions(issueKey, body, false, false)
}

// AddInternalComment adds an internal comment to an issue (hidden from customers)
func (c *Client) AddInternalComment(issueKey, body string) error {
	return c.addCommentWithOptions(issueKey, body, true, false)
}

// addCommentWithOptions adds a comment with specified visibility and transition settings
func (c *Client) addCommentWithOptions(issueKey, body string, internal, allowTransition bool) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)

	properties := []CommentProperty{
		{
			Key:   "sd.public.comment",
			Value: map[string]interface{}{"internal": internal},
		},
	}

	// Add property to prevent workflow transitions if needed
	if !allowTransition {
		properties = append(properties, CommentProperty{
			Key:   "sd.allow.public.comment.transition",
			Value: map[string]interface{}{"allow": false},
		})
	}

	req := AddCommentRequest{
		Body:       body,
		Properties: properties,
	}

	var result Comment
	if err := c.Post(path, req, &result); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// UpdateComment updates an existing comment
func (c *Client) UpdateComment(issueKey, commentID, body string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment/%s", issueKey, commentID)

	req := map[string]string{"body": body}

	if err := c.Put(path, req); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}
