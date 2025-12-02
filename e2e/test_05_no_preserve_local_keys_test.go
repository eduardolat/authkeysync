package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoPreserveLocalKeys verifies that local keys are removed
// when preserve_local_keys is false.
func TestNoPreserveLocalKeys(t *testing.T) {
	_ = newTestCase(t, testUser1)

	localKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILOCALONLY111111111111111111111111111111111111 localonly@machine"

	// Create existing authorized_keys with local key
	createExistingKeys(t, testUser1, localKey)

	// Create config with preserve_local_keys: false
	configPath := createConfig(t, "no_preserve_local", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
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

	// Check remote keys are present
	assert.Contains(t, content, "alice@laptop", "Should contain remote key")
	assert.Contains(t, content, "alice@workstation", "Should contain remote key")

	// Check local keys are NOT present
	assert.NotContains(t, content, "localonly@machine", "Should NOT contain local key")
	assert.NotContains(t, content, "# Local (preserved)", "Should NOT have Local (preserved) section")

	// Verify key count
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount, "Should have only 2 remote keys")
}
