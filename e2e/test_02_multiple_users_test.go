package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultipleUsers verifies that authkeysync can sync keys for multiple users
// in a single run.
func TestMultipleUsers(t *testing.T) {
	_ = newTestCase(t, testUser1, testUser2)

	// Create config
	configPath := createConfig(t, "multiple_users", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys

  - username: `+testUser2+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/bob.keys
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Verify exit code
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0")

	// Verify user 1
	authKeysPath1 := getAuthKeysPath(t, testUser1)
	assertFileExists(t, authKeysPath1, "authorized_keys for %s should exist", testUser1)

	perms1 := getFilePermissions(t, authKeysPath1)
	assert.Equal(t, "600", perms1)

	owner1 := getFileOwner(t, authKeysPath1)
	assert.Equal(t, testUser1, owner1)

	content1 := readAuthKeys(t, testUser1)
	assert.Contains(t, content1, "alice@laptop", "User1 should have alice's key")
	assert.NotContains(t, content1, "bob@desktop", "User1 should NOT have bob's key")

	// Verify user 2
	authKeysPath2 := getAuthKeysPath(t, testUser2)
	assertFileExists(t, authKeysPath2, "authorized_keys for %s should exist", testUser2)

	perms2 := getFilePermissions(t, authKeysPath2)
	assert.Equal(t, "600", perms2)

	owner2 := getFileOwner(t, authKeysPath2)
	assert.Equal(t, testUser2, owner2)

	content2 := readAuthKeys(t, testUser2)
	assert.Contains(t, content2, "bob@desktop", "User2 should have bob's key")
	assert.NotContains(t, content2, "alice@laptop", "User2 should NOT have alice's key")

	// Verify key counts
	keyCount1 := countKeys(t, authKeysPath1)
	keyCount2 := countKeys(t, authKeysPath2)
	require.Equal(t, 2, keyCount1, "User1 should have 2 keys (alice has 2)")
	require.Equal(t, 1, keyCount2, "User2 should have 1 key (bob has 1)")
}
