package jira

import (
	"fmt"
	"sort"
)

// ServiceDesk represents a Service Desk project
type ServiceDesk struct {
	ID         string `json:"id"`
	ProjectKey string `json:"projectKey"`
	ProjectName string `json:"projectName"`
}

// ServiceDesksResponse represents the response from service desk list API
type ServiceDesksResponse struct {
	Size       int           `json:"size"`
	Start      int           `json:"start"`
	Limit      int           `json:"limit"`
	IsLastPage bool          `json:"isLastPage"`
	Values     []ServiceDesk `json:"values"`
}

// Queue represents a Service Desk queue
type Queue struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Jql        string        `json:"jql"`
	IsFavorite bool          `json:"favourite,omitempty"` // British spelling from API
	Fields     []interface{} `json:"fields,omitempty"`    // Can be strings or objects depending on Jira version
}

// QueuesResponse represents the response from the queues API
type QueuesResponse struct {
	Size       int     `json:"size"`
	Start      int     `json:"start"`
	Limit      int     `json:"limit"`
	IsLastPage bool    `json:"isLastPage"`
	Values     []Queue `json:"values"`
}

// GetServiceDeskID fetches the service desk ID for a project key
func (c *Client) GetServiceDeskID(projectKey string) (string, error) {
	// Try to get all service desks and find the one matching our project key
	path := "/rest/servicedeskapi/servicedesk"

	var resp ServiceDesksResponse
	if err := c.Get(path, &resp); err != nil {
		return "", fmt.Errorf("failed to get service desks: %w", err)
	}

	// Find the service desk with matching project key
	for _, sd := range resp.Values {
		if sd.ProjectKey == projectKey {
			return sd.ID, nil
		}
	}

	return "", fmt.Errorf("no service desk found for project key: %s", projectKey)
}

// GetQueues fetches all queues for a Service Desk project
func (c *Client) GetQueues(projectKey string) ([]Queue, error) {
	// First, get the service desk ID (cache it for future use)
	if c.serviceDeskID == "" {
		serviceDeskID, err := c.GetServiceDeskID(projectKey)
		if err != nil {
			return nil, err
		}
		c.serviceDeskID = serviceDeskID
	}

	path := fmt.Sprintf("/rest/servicedeskapi/servicedesk/%s/queue", c.serviceDeskID)

	var resp QueuesResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	// Build favorite index map for ordering (lower index = higher priority)
	favoriteIndex := make(map[string]int)
	for i, name := range c.favoriteQueues {
		favoriteIndex[name] = i
	}

	// Mark queues as favorites based on config
	for i := range resp.Values {
		if _, ok := favoriteIndex[resp.Values[i].Name]; ok {
			resp.Values[i].IsFavorite = true
		}
	}

	// Filter out non-favorites if configured
	var queues []Queue
	if c.hideNonFavorites {
		for _, q := range resp.Values {
			if q.IsFavorite {
				queues = append(queues, q)
			}
		}
	} else {
		queues = resp.Values
	}

	// Sort queues: favorites first (in config order), then non-favorites by name
	sort.SliceStable(queues, func(i, j int) bool {
		iFav := queues[i].IsFavorite
		jFav := queues[j].IsFavorite

		if iFav && jFav {
			// Both are favorites - sort by config order
			return favoriteIndex[queues[i].Name] < favoriteIndex[queues[j].Name]
		}
		if iFav != jFav {
			// Favorites come first
			return iFav
		}
		// Both are non-favorites - sort by name
		return queues[i].Name < queues[j].Name
	})

	return queues, nil
}
