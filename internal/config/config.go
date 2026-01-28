package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	URL             string   `yaml:"url"`
	Auth            Auth     `yaml:"auth"`
	Project         string   `yaml:"project"`
	FavoriteQueues  []string `yaml:"favorite_queues,omitempty"`
}

// Auth contains authentication details
type Auth struct {
	Type     string `yaml:"type"`     // "pat" or "basic"
	Token    string `yaml:"token"`    // for PAT
	Username string `yaml:"username"` // for basic auth
	Password string `yaml:"password"` // for basic auth
}

// Load reads the config file from ~/.config/jsm-tui/config.yaml
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "jsm-tui", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.URL == "" {
		return nil, fmt.Errorf("config: url is required")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("config: project is required")
	}
	if cfg.Auth.Type == "" {
		return nil, fmt.Errorf("config: auth.type is required")
	}
	if cfg.Auth.Type != "pat" && cfg.Auth.Type != "basic" {
		return nil, fmt.Errorf("config: auth.type must be 'pat' or 'basic'")
	}
	if cfg.Auth.Type == "pat" && cfg.Auth.Token == "" {
		return nil, fmt.Errorf("config: auth.token is required for PAT authentication")
	}
	if cfg.Auth.Type == "basic" && (cfg.Auth.Username == "" || cfg.Auth.Password == "") {
		return nil, fmt.Errorf("config: auth.username and auth.password are required for basic authentication")
	}

	return &cfg, nil
}
