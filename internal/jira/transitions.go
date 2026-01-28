package jira

import "fmt"

// Transition represents an available workflow transition
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

// TransitionsResponse represents the response from transitions API
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// TransitionRequest represents a transition execution request
type TransitionRequest struct {
	Transition struct {
		ID string `json:"id"`
	} `json:"transition"`
}

// GetTransitions fetches available transitions for an issue
func (c *Client) GetTransitions(issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)

	var resp TransitionsResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}

	return resp.Transitions, nil
}

// ExecuteTransition executes a transition on an issue
func (c *Client) ExecuteTransition(issueKey, transitionID string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)

	req := TransitionRequest{}
	req.Transition.ID = transitionID

	if err := c.Post(path, req, nil); err != nil {
		return fmt.Errorf("failed to execute transition: %w", err)
	}

	return nil
}
