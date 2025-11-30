package config

import (
	"testing"
)

// FuzzParse tests the Parse function with random YAML inputs
func FuzzParse(f *testing.F) {
	// Add seed corpus with valid and invalid YAML
	seeds := []string{
		// Valid minimal config
		`users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"`,
		// Valid full config
		`policy:
  backup_enabled: true
  backup_retention_count: 10
  preserve_local_keys: true
users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
        method: "GET"
        timeout_seconds: 10`,
		// Empty string
		"",
		// Invalid YAML
		"not: valid: yaml: [",
		// Valid YAML but invalid config (no users)
		`policy:
  backup_enabled: true`,
		// Unicode content
		`users:
  - username: "用户"
    sources:
      - url: "https://example.com/keys"`,
		// Very long strings
		`users:
  - username: "a"
    sources:
      - url: "https://example.com/` + string(make([]byte, 1000)) + `"`,
		// Multiple users
		`users:
  - username: "user1"
    sources:
      - url: "https://example.com/keys1"
  - username: "user2"
    sources:
      - url: "https://example.com/keys2"`,
		// POST method with body
		`users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
        method: "POST"
        body: '{"role": "admin"}'
        headers:
          Authorization: "Bearer token"`,
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse should never panic
		cfg, err := Parse(data)

		// If parsing fails, that's fine for fuzz testing
		if err != nil {
			return
		}

		// Config should not be nil when err is nil
		if cfg == nil {
			t.Fatal("config should not be nil when err is nil")
		}

		// Users should be valid
		if cfg.Users == nil {
			t.Fatal("users should not be nil")
		}
		if len(cfg.Users) == 0 {
			t.Fatal("should have at least one user")
		}

		// Validate each user
		for i, user := range cfg.Users {
			if user.Username == "" {
				t.Errorf("user %d has empty username", i)
			}
			if len(user.Sources) == 0 {
				t.Errorf("user %d has no sources", i)
			}
			for j, source := range user.Sources {
				if source.URL == "" {
					t.Errorf("user %d source %d has empty URL", i, j)
				}
				method := source.GetMethod()
				if method != "GET" && method != "POST" {
					t.Errorf("user %d source %d has invalid method: %s", i, j, method)
				}
				if source.GetTimeoutSeconds() <= 0 {
					t.Errorf("user %d source %d has invalid timeout", i, j)
				}
			}
		}

		// Policy should have valid values
		if cfg.Policy.GetBackupRetentionCount() < 0 {
			t.Error("backup retention count should not be negative")
		}
	})
}
