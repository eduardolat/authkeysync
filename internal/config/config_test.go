package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidConfig(t *testing.T) {
	yamlData := `
policy:
  backup_enabled: true
  backup_retention_count: 5
  preserve_local_keys: false

users:
  - username: "admin"
    sources:
      - url: "https://github.com/admin.keys"
      - url: "https://api.example.com/keys"
        method: "POST"
        headers:
          Authorization: "Bearer token123"
        body: '{"role": "admin"}'
        timeout_seconds: 5
`

	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	assert.True(t, cfg.Policy.IsBackupEnabled())
	assert.Equal(t, 5, cfg.Policy.GetBackupRetentionCount())
	assert.False(t, cfg.Policy.IsPreserveLocalKeys())

	require.Len(t, cfg.Users, 1)
	assert.Equal(t, "admin", cfg.Users[0].Username)
	require.Len(t, cfg.Users[0].Sources, 2)

	// First source - defaults
	assert.Equal(t, "https://github.com/admin.keys", cfg.Users[0].Sources[0].URL)
	assert.Equal(t, "GET", cfg.Users[0].Sources[0].GetMethod())
	assert.Equal(t, 10, cfg.Users[0].Sources[0].GetTimeoutSeconds())

	// Second source - custom values
	assert.Equal(t, "https://api.example.com/keys", cfg.Users[0].Sources[1].URL)
	assert.Equal(t, "POST", cfg.Users[0].Sources[1].GetMethod())
	assert.Equal(t, 5, cfg.Users[0].Sources[1].GetTimeoutSeconds())
	assert.Equal(t, "Bearer token123", cfg.Users[0].Sources[1].Headers["Authorization"])
	assert.Equal(t, `{"role": "admin"}`, cfg.Users[0].Sources[1].Body)
}

func TestParse_DefaultValues(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: "https://github.com/admin.keys"
`

	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	// Check defaults
	assert.True(t, cfg.Policy.IsBackupEnabled())
	assert.Equal(t, 10, cfg.Policy.GetBackupRetentionCount())
	assert.True(t, cfg.Policy.IsPreserveLocalKeys())
}

func TestParse_ExplicitFalseValues(t *testing.T) {
	yamlData := `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: "admin"
    sources:
      - url: "https://github.com/admin.keys"
`

	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	assert.False(t, cfg.Policy.IsBackupEnabled())
	assert.False(t, cfg.Policy.IsPreserveLocalKeys())
}

func TestValidate_NoUsers(t *testing.T) {
	yamlData := `
policy:
  backup_enabled: true
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one user must be defined")
}

func TestValidate_EmptyUsername(t *testing.T) {
	yamlData := `
users:
  - username: ""
    sources:
      - url: "https://example.com/keys"
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty username")
}

func TestValidate_DuplicateUsername(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
  - username: "admin"
    sources:
      - url: "https://example.com/keys2"
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate username")
}

func TestValidate_NoSources(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources: []
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no sources defined")
}

func TestValidate_EmptyURL(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: ""
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty URL")
}

func TestValidate_InvalidMethod(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
        method: "PUT"
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid method")
}

func TestValidate_NegativeBackupRetention(t *testing.T) {
	yamlData := `
policy:
  backup_retention_count: -1

users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup_retention_count cannot be negative")
}

func TestValidate_InvalidTimeout(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: "https://example.com/keys"
        timeout_seconds: 0
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout")
}

func TestSource_MethodCaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "get", expected: "GET"},
		{input: "GET", expected: "GET"},
		{input: "Get", expected: "GET"},
		{input: "post", expected: "POST"},
		{input: "POST", expected: "POST"},
		{input: "Post", expected: "POST"},
		{input: "", expected: "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			s := Source{Method: tt.input}
			assert.Equal(t, tt.expected, s.GetMethod())
		})
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	yamlData := `
this is not valid yaml: [
`

	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestParse_MultipleUsers(t *testing.T) {
	yamlData := `
users:
  - username: "admin"
    sources:
      - url: "https://github.com/admin.keys"
  - username: "deploy"
    sources:
      - url: "https://github.com/deploy.keys"
  - username: "backup"
    sources:
      - url: "https://github.com/backup.keys"
`

	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)
	require.Len(t, cfg.Users, 3)

	assert.Equal(t, "admin", cfg.Users[0].Username)
	assert.Equal(t, "deploy", cfg.Users[1].Username)
	assert.Equal(t, "backup", cfg.Users[2].Username)
}
