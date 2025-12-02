package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMalformedResponses verifies that authkeysync correctly handles
// HTML and JSON error responses.
func TestMalformedResponses(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with sources that return HTML and JSON
	configPath := createConfig(t, "malformed_responses", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
      - url: ${MOCK_SERVER_URL}/error/html
      - url: ${MOCK_SERVER_URL}/error/json
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

	// Should contain valid keys from alice
	assert.Contains(t, content, "alice@laptop", "Should contain alice's key")
	assert.Contains(t, content, "alice@workstation", "Should contain alice's second key")

	// Should NOT contain HTML or JSON content
	assert.NotContains(t, content, "<!DOCTYPE", "Should not contain HTML")
	assert.NotContains(t, content, "<html>", "Should not contain HTML tag")
	assert.NotContains(t, content, `"error"`, "Should not contain JSON")

	// Verify key count (only valid keys)
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount, "Should have only 2 valid keys (from alice)")
}
