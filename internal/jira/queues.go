package jira

import "fmt"

// Queue represents a Service Desk queue
type Queue struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Jql    string `json:"jql"`
	Fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"fields"`
}

// QueuesResponse represents the response from the queues API
type QueuesResponse struct {
	Size       int     `json:"size"`
	Start      int     `json:"start"`
	Limit      int     `json:"limit"`
	IsLastPage bool    `json:"isLastPage"`
	Values     []Queue `json:"values"`
}

// GetQueues fetches all queues for a Service Desk project
func (c *Client) GetQueues(projectKey string) ([]Queue, error) {
	path := fmt.Sprintf("/rest/servicedeskapi/servicedesk/%s/queue", projectKey)

	var resp QueuesResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	return resp.Values, nil
}
