package jira

import (
	"fmt"
	"net/url"
	"strings"
)

// Issue represents a Jira issue
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains issue field data
type IssueFields struct {
	Summary             string              `json:"summary"`
	Description         string              `json:"description"`
	Status              Status              `json:"status"`
	Priority            Priority            `json:"priority"`
	Assignee            *User               `json:"assignee"`
	Reporter            *User               `json:"reporter"`
	Created             string              `json:"created"`
	Updated             string              `json:"updated"`
	IssueType           IssueType           `json:"issuetype"`
	Comment             CommentList         `json:"comment,omitempty"`
	Attachment          []Attachment         `json:"attachment,omitempty"`
	RequestParticipants []User              `json:"customfield_10000,omitempty"` // Request participants
	CustomerRequestType *CustomerRequestType `json:"customfield_10001,omitempty"` // Service request type
}

// Attachment represents a file attached to an issue
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"` // download URL
	Author   *User  `json:"author,omitempty"`
	Created  string `json:"created"`
}

// IssueDetail is an enriched issue with proforma forms included
type IssueDetail struct {
	Issue
	ProformaForms []ProformaForm `json:"proformaForms,omitempty"`
}

// GetIssueDetail fetches an issue with proforma forms and attachments
func (c *Client) GetIssueDetail(issueKey string) (*IssueDetail, error) {
	issue, err := c.GetIssue(issueKey)
	if err != nil {
		return nil, err
	}

	detail := &IssueDetail{Issue: *issue}

	forms, err := c.GetProformaForms(issueKey)
	if err == nil && forms != nil {
		detail.ProformaForms = forms
	}

	return detail, nil
}

// CustomerRequestType represents the JSM request type
type CustomerRequestType struct {
	RequestType *RequestType `json:"requestType,omitempty"`
}

// RequestType contains request type details
type RequestType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

// ProformaFormsList contains the list of proforma forms on an issue
type ProformaFormsList struct {
	Key   string `json:"key"`
	Value struct {
		Forms []ProformaFormRef `json:"forms"`
	} `json:"value"`
}

// ProformaFormRef is a reference to a proforma form
type ProformaFormRef struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Submitted bool   `json:"submitted"`
}

// ProformaFormData contains the full form data
type ProformaFormData struct {
	Key   string `json:"key"`
	Value struct {
		Design struct {
			Questions map[string]ProformaQuestion `json:"questions"`
			Settings  struct {
				Name string `json:"name"`
			} `json:"settings"`
		} `json:"design"`
		State struct {
			Answers map[string]ProformaAnswer `json:"answers"`
		} `json:"state"`
	} `json:"value"`
}

// ProformaQuestion represents a question in a proforma form
type ProformaQuestion struct {
	Label       string           `json:"label"`
	Type        string           `json:"type"`
	Description string           `json:"description"`
	Choices     []ProformaChoice `json:"choices,omitempty"`
}

// ProformaChoice represents a choice option in a dropdown/select question
type ProformaChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ProformaAnswer represents an answer in a proforma form
type ProformaAnswer struct {
	Text    string   `json:"text,omitempty"`
	Date    string   `json:"date,omitempty"`
	Choices []string `json:"choices,omitempty"`
}

// ProformaForm is a processed form with questions and answers paired
type ProformaForm struct {
	Name   string
	Fields []ProformaField
}

// ProformaField is a question-answer pair
type ProformaField struct {
	Label  string
	Answer string
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

// GetUserByAccountID fetches a user by their account ID (Jira Cloud)
func (c *Client) GetUserByAccountID(accountID string) (*User, error) {
	path := fmt.Sprintf("/rest/api/2/user?accountId=%s", url.QueryEscape(accountID))

	var user User
	if err := c.Get(path, &user); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUsername fetches a user by their username (Jira Server/Data Center)
func (c *Client) GetUserByUsername(username string) (*User, error) {
	path := fmt.Sprintf("/rest/api/2/user?username=%s", url.QueryEscape(username))

	var user User
	if err := c.Get(path, &user); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetProformaForms fetches all proforma forms for an issue
func (c *Client) GetProformaForms(issueKey string) ([]ProformaForm, error) {
	// First get the list of forms
	listPath := fmt.Sprintf("/rest/api/2/issue/%s/properties/proforma.forms", issueKey)
	var formsList ProformaFormsList
	if err := c.Get(listPath, &formsList); err != nil {
		// No forms is not an error
		return nil, nil
	}

	var forms []ProformaForm
	for _, ref := range formsList.Value.Forms {
		// Fetch each form's data
		formPath := fmt.Sprintf("/rest/api/2/issue/%s/properties/proforma.forms.i%d", issueKey, ref.ID)
		var formData ProformaFormData
		if err := c.Get(formPath, &formData); err != nil {
			continue
		}

		form := ProformaForm{
			Name: formData.Value.Design.Settings.Name,
		}

		// Match questions with answers
		for qID, question := range formData.Value.Design.Questions {
			answer := formData.Value.State.Answers[qID]
			answerText := answer.Text
			if answerText == "" && answer.Date != "" {
				answerText = answer.Date
			}
			if answerText == "" && len(answer.Choices) > 0 {
				// Map choice IDs to their labels
				var choiceLabels []string
				for _, choiceID := range answer.Choices {
					for _, choice := range question.Choices {
						if choice.ID == choiceID {
							choiceLabels = append(choiceLabels, choice.Label)
							break
						}
					}
				}
				if len(choiceLabels) > 0 {
					answerText = strings.Join(choiceLabels, ", ")
				} else {
					answerText = "-"
				}
			}
			if answerText == "" {
				answerText = "-"
			}

			form.Fields = append(form.Fields, ProformaField{
				Label:  question.Label,
				Answer: answerText,
			})
		}

		forms = append(forms, form)
	}

	return forms, nil
}

// UpdateIssueSummary updates the summary of an issue
func (c *Client) UpdateIssueSummary(issueKey, newSummary string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)

	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"summary": newSummary,
		},
	}

	resp, err := c.doRequest("PUT", path, body)
	if err != nil {
		return fmt.Errorf("failed to update issue summary: %w", err)
	}
	resp.Body.Close()

	return nil
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
