package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostWithAuth verifies that authkeysync can make POST requests
// with headers and body.
func TestPostWithAuth(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with POST source and authentication headers
	configPath := createConfig(t, "post_auth", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/api/keys
        method: POST
        headers:
          Authorization: "Bearer secret-token-123"
          Content-Type: "application/json"
        body: '{"role": "admin", "environment": "production"}'
        timeout_seconds: 10
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Verify exit code
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0")

	// Verify file exists
	authKeysPath := getAuthKeysPath(t, testUser1)
	assertFileExists(t, authKeysPath, "authorized_keys should exist")

	// Check content
	content := readAuthKeys(t, testUser1)

	// Check the API key is present (returned by authenticated POST endpoint)
	assert.Contains(t, content, "api@authenticated", "Should contain key from authenticated API")

	// Verify key count
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 1, keyCount, "Should have 1 key from API")
}
