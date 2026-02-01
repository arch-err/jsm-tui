package storage

import (
	"os"
	"path/filepath"
	"strings"
)

// DraftType represents the type of draft
type DraftType string

const (
	DraftTypeComment DraftType = "comment"
	DraftTypeAction  DraftType = "action"
)

// GetDataDir returns the data directory path (~/.local/share/jsm-tui)
func GetDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir := filepath.Join(home, ".local", "share", "jsm-tui")
	return dataDir, nil
}

// GetDraftsDir returns the drafts directory path
func GetDraftsDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}

	draftsDir := filepath.Join(dataDir, "drafts")
	return draftsDir, nil
}

// getDraftPath returns the path for a specific draft
func getDraftPath(issueKey string, draftType DraftType) (string, error) {
	draftsDir, err := GetDraftsDir()
	if err != nil {
		return "", err
	}

	// Sanitize issue key for filename
	safeKey := strings.ReplaceAll(issueKey, "/", "_")
	filename := safeKey + "." + string(draftType) + ".txt"

	return filepath.Join(draftsDir, filename), nil
}

// SaveDraft saves a comment draft for an issue
func SaveDraft(issueKey string, draftType DraftType, content string) error {
	// Don't save empty drafts
	if strings.TrimSpace(content) == "" {
		// Delete any existing draft
		return DeleteDraft(issueKey, draftType)
	}

	draftsDir, err := GetDraftsDir()
	if err != nil {
		return err
	}

	// Ensure drafts directory exists
	if err := os.MkdirAll(draftsDir, 0755); err != nil {
		return err
	}

	draftPath, err := getDraftPath(issueKey, draftType)
	if err != nil {
		return err
	}

	return os.WriteFile(draftPath, []byte(content), 0644)
}

// LoadDraft loads a comment draft for an issue
func LoadDraft(issueKey string, draftType DraftType) (string, error) {
	draftPath, err := getDraftPath(issueKey, draftType)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(draftPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No draft exists
		}
		return "", err
	}

	return string(content), nil
}

// DeleteDraft removes a draft for an issue
func DeleteDraft(issueKey string, draftType DraftType) error {
	draftPath, err := getDraftPath(issueKey, draftType)
	if err != nil {
		return err
	}

	err = os.Remove(draftPath)
	if os.IsNotExist(err) {
		return nil // Already deleted
	}
	return err
}

// HasDraft checks if a draft exists for an issue
func HasDraft(issueKey string, draftType DraftType) bool {
	draftPath, err := getDraftPath(issueKey, draftType)
	if err != nil {
		return false
	}

	_, err = os.Stat(draftPath)
	return err == nil
}
