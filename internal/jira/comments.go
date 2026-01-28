package jira

import "fmt"

// Comment represents a comment on an issue
type Comment struct {
	ID      string `json:"id"`
	Author  User   `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// CommentsResponse represents the response from comments API
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// AddCommentRequest represents a request to add a comment
type AddCommentRequest struct {
	Body string `json:"body"`
}

// GetComments fetches all comments for an issue
func (c *Client) GetComments(issueKey string) ([]Comment, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)

	var resp CommentsResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	return resp.Comments, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(issueKey, body string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)

	req := AddCommentRequest{Body: body}

	var result Comment
	if err := c.Post(path, req, &result); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}
