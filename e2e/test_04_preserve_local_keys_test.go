package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreserveLocalKeys verifies that existing local keys are preserved
// when preserve_local_keys is true.
func TestPreserveLocalKeys(t *testing.T) {
	_ = newTestCase(t, testUser1)

	localKey1 := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILOCAL11111111111111111111111111111111111111111 local@machine1"
	localKey2 := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQLOCAL2222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222 local@machine2"

	// Create existing authorized_keys with local keys
	createExistingKeys(t, testUser1, localKey1+"\n"+localKey2)

	// Create config with preserve_local_keys: true
	configPath := createConfig(t, "preserve_local", `
policy:
  backup_enabled: false
  preserve_local_keys: true

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
	assert.Contains(t, content, "alice@laptop", "Should contain remote key from alice")
	assert.Contains(t, content, "alice@workstation", "Should contain alice's second remote key")

	// Check local keys are preserved
	assert.Contains(t, content, "local@machine1", "Should preserve local key 1")
	assert.Contains(t, content, "local@machine2", "Should preserve local key 2")

	// Check local section header
	assert.Contains(t, content, "# Local (preserved)", "Should have Local (preserved) section")

	// Verify total key count
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 4, keyCount, "Should have 4 keys total (2 remote + 2 local)")
}
