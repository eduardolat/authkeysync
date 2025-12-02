package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmptySource verifies that authkeysync handles sources that return no keys.
func TestEmptySource(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with empty source and valid source
	configPath := createConfig(t, "empty_source", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/empty.keys
      - url: ${MOCK_SERVER_URL}/users/bob.keys
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

	// Should contain bob's key
	assert.Contains(t, content, "bob@desktop", "Should contain bob's key")

	// Verify key count
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 1, keyCount, "Should have 1 key (from bob)")
}
