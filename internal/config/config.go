package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	URL      string       `yaml:"url"`
	Auth     Auth         `yaml:"auth"`
	Project  string       `yaml:"project"`
	Username string       `yaml:"username,omitempty"` // Display name to identify "me"
	Queues   QueuesConfig `yaml:"queues,omitempty"`
}

// QueuesConfig contains queue display settings
type QueuesConfig struct {
	Favorites        []string `yaml:"favorites,omitempty"`
	HideNonFavorites bool     `yaml:"hide_non_favorites,omitempty"`
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

// Save writes the config back to ~/.config/jsm-tui/config.yaml
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "jsm-tui", "config.yaml")

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file at %s: %w", configPath, err)
	}

	return nil
}

// ToggleFavoriteQueue adds or removes a queue from favorites
func (c *Config) ToggleFavoriteQueue(queueName string) {
	// Check if already a favorite
	for i, name := range c.Queues.Favorites {
		if name == queueName {
			// Remove from favorites
			c.Queues.Favorites = append(c.Queues.Favorites[:i], c.Queues.Favorites[i+1:]...)
			return
		}
	}
	// Add to favorites
	c.Queues.Favorites = append(c.Queues.Favorites, queueName)
}
