package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/arch-err/jsm-tui/internal/config"
)

// Client handles communication with Jira API
type Client struct {
	baseURL        string
	httpClient     *http.Client
	auth           config.Auth
	serviceDeskID  string   // Cached service desk ID
	favoriteQueues []string // List of favorite queue names from config
}

// NewClient creates a new Jira API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:        strings.TrimRight(cfg.URL, "/"),
		httpClient:     &http.Client{},
		auth:           cfg.Auth,
		favoriteQueues: cfg.FavoriteQueues,
	}
}

// doRequest executes an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authentication
	switch c.auth.Type {
	case "pat":
		req.Header.Set("Authorization", "Bearer "+c.auth.Token)
	case "basic":
		req.SetBasicAuth(c.auth.Username, c.auth.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d for %s %s: %s", resp.StatusCode, method, url, string(bodyBytes))
	}

	return resp, nil
}

// Get performs a GET request and unmarshals the response
func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// Post performs a POST request and unmarshals the response
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Put performs a PUT request
func (c *Client) Put(path string, body interface{}) error {
	resp, err := c.doRequest("PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
