package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultipleSources verifies that authkeysync can fetch keys from multiple
// sources for a single user.
func TestMultipleSources(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with multiple sources
	configPath := createConfig(t, "multiple_sources", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
      - url: ${MOCK_SERVER_URL}/users/bob.keys
      - url: ${MOCK_SERVER_URL}/users/charlie.keys
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Verify exit code
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0")

	// Verify file exists
	authKeysPath := getAuthKeysPath(t, testUser1)
	assertFileExists(t, authKeysPath, "authorized_keys should exist")

	// Check content contains keys from all sources
	content := readAuthKeys(t, testUser1)
	assert.Contains(t, content, "alice@laptop", "Should contain alice's key")
	assert.Contains(t, content, "alice@workstation", "Should contain alice's second key")
	assert.Contains(t, content, "bob@desktop", "Should contain bob's key")
	assert.Contains(t, content, "charlie@server", "Should contain charlie's first key")
	assert.Contains(t, content, "charlie@laptop", "Should contain charlie's second key")

	// Check source comments are present
	assert.Contains(t, content, "# Source:")

	// Verify total key count
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 5, keyCount, "Should have 5 keys total (2+1+2)")
}
