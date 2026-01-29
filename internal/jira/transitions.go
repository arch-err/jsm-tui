package jira

import (
	"fmt"
	"strings"
)

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
	Fields map[string]interface{} `json:"fields,omitempty"`
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
	return c.ExecuteTransitionWithFields(issueKey, transitionID, nil)
}

// ExecuteTransitionWithFields executes a transition with optional field updates
func (c *Client) ExecuteTransitionWithFields(issueKey, transitionID string, fields map[string]interface{}) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)

	req := TransitionRequest{}
	req.Transition.ID = transitionID
	if len(fields) > 0 {
		req.Fields = fields
	}

	if err := c.Post(path, req, nil); err != nil {
		return fmt.Errorf("failed to execute transition: %w", err)
	}

	return nil
}

// FindTransitionByStatusName finds a transition that leads to the given status name
func (c *Client) FindTransitionByStatusName(issueKey, statusName string) (*Transition, error) {
	transitions, err := c.GetTransitions(issueKey)
	if err != nil {
		return nil, err
	}

	statusLower := strings.ToLower(statusName)
	for _, t := range transitions {
		if strings.ToLower(t.To.Name) == statusLower {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("no transition found to status: %s", statusName)
}
