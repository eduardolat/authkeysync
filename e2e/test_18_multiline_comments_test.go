package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultilineWithComments verifies that authkeysync correctly parses
// responses with comments and empty lines.
func TestMultilineWithComments(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with multiline source (has comments and empty lines)
	configPath := createConfig(t, "multiline_comments", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/multiline.keys
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

	// Should contain valid keys
	assert.Contains(t, content, "multi@host1", "Should contain first valid key")
	assert.Contains(t, content, "multi@host2", "Should contain second valid key")

	// Verify key count (should be 2 valid keys, comments and empty lines discarded)
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount, "Should have 2 valid keys")
}
