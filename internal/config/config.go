// Package config handles YAML configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigPath is the default configuration file path
	DefaultConfigPath = "/etc/authkeysync/config.yaml"

	// DefaultBackupRetentionCount is the default number of backups to keep
	DefaultBackupRetentionCount = 10

	// DefaultTimeoutSeconds is the default HTTP request timeout
	DefaultTimeoutSeconds = 10

	// DefaultMethod is the default HTTP method
	DefaultMethod = "GET"
)

// Config represents the complete application configuration
type Config struct {
	Policy Policy `yaml:"policy"`
	Users  []User `yaml:"users"`
}

// Policy defines global synchronization behavior
type Policy struct {
	BackupEnabled        *bool `yaml:"backup_enabled"`
	BackupRetentionCount *int  `yaml:"backup_retention_count"`
	PreserveLocalKeys    *bool `yaml:"preserve_local_keys"`
}

// IsBackupEnabled returns true if backups are enabled (default: true)
func (p Policy) IsBackupEnabled() bool {
	if p.BackupEnabled == nil {
		return true
	}
	return *p.BackupEnabled
}

// GetBackupRetentionCount returns the backup retention count (default: 10)
func (p Policy) GetBackupRetentionCount() int {
	if p.BackupRetentionCount == nil {
		return DefaultBackupRetentionCount
	}
	return *p.BackupRetentionCount
}

// IsPreserveLocalKeys returns true if local keys should be preserved (default: true)
func (p Policy) IsPreserveLocalKeys() bool {
	if p.PreserveLocalKeys == nil {
		return true
	}
	return *p.PreserveLocalKeys
}

// User represents a system user to manage
type User struct {
	Username string   `yaml:"username"`
	Sources  []Source `yaml:"sources"`
}

// Source defines an HTTP endpoint for fetching keys
type Source struct {
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Headers        map[string]string `yaml:"headers"`
	Body           string            `yaml:"body"`
	TimeoutSeconds *int              `yaml:"timeout_seconds"`
}

// GetMethod returns the HTTP method (default: GET)
func (s Source) GetMethod() string {
	if s.Method == "" {
		return DefaultMethod
	}
	return strings.ToUpper(s.Method)
}

// GetTimeoutSeconds returns the timeout in seconds (default: 10)
func (s Source) GetTimeoutSeconds() int {
	if s.TimeoutSeconds == nil {
		return DefaultTimeoutSeconds
	}
	return *s.TimeoutSeconds
}

// Load reads and parses a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return Parse(data)
}

// Parse parses YAML configuration data
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if len(c.Users) == 0 {
		return errors.New("config: at least one user must be defined")
	}

	if c.Policy.GetBackupRetentionCount() < 0 {
		return errors.New("config: backup_retention_count cannot be negative")
	}

	usernames := make(map[string]bool)
	for i, user := range c.Users {
		if user.Username == "" {
			return fmt.Errorf("config: user at index %d has empty username", i)
		}

		if usernames[user.Username] {
			return fmt.Errorf("config: duplicate username %q", user.Username)
		}
		usernames[user.Username] = true

		if len(user.Sources) == 0 {
			return fmt.Errorf("config: user %q has no sources defined", user.Username)
		}

		for j, source := range user.Sources {
			if source.URL == "" {
				return fmt.Errorf("config: user %q source at index %d has empty URL", user.Username, j)
			}

			method := source.GetMethod()
			if method != "GET" && method != "POST" {
				return fmt.Errorf("config: user %q source at index %d has invalid method %q (supported: GET, POST)", user.Username, j, method)
			}

			if source.GetTimeoutSeconds() <= 0 {
				return fmt.Errorf("config: user %q source at index %d has invalid timeout", user.Username, j)
			}
		}
	}

	return nil
}
