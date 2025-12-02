package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFilePermissions verifies that authorized_keys files have correct
// permissions (0600) and ownership.
func TestFilePermissions(t *testing.T) {
	_ = newTestCase(t, testUser1, testUser2)

	// Create config for both users
	configPath := createConfig(t, "file_permissions", `
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

	// Verify user 1 file permissions and ownership
	t.Log("Checking user1 authorized_keys:")
	authKeysPath1 := getAuthKeysPath(t, testUser1)
	assertFileExists(t, authKeysPath1)

	perms1 := getFilePermissions(t, authKeysPath1)
	assert.Equal(t, "600", perms1, "authorized_keys should have 0600 permissions")

	owner1 := getFileOwner(t, authKeysPath1)
	assert.Equal(t, testUser1, owner1, "authorized_keys should be owned by %s", testUser1)

	group1 := getFileGroup(t, authKeysPath1)
	assert.Equal(t, testUser1, group1, "Group should be %s", testUser1)

	// Verify user 2 file permissions and ownership
	t.Log("Checking user2 authorized_keys:")
	authKeysPath2 := getAuthKeysPath(t, testUser2)
	assertFileExists(t, authKeysPath2)

	perms2 := getFilePermissions(t, authKeysPath2)
	assert.Equal(t, "600", perms2, "authorized_keys should have 0600 permissions")

	owner2 := getFileOwner(t, authKeysPath2)
	assert.Equal(t, testUser2, owner2, "authorized_keys should be owned by %s", testUser2)

	group2 := getFileGroup(t, authKeysPath2)
	assert.Equal(t, testUser2, group2, "Group should be %s", testUser2)

	// Verify .ssh directory permissions
	t.Log("Checking .ssh directory permissions:")
	sshDir1 := getSSHDir(t, testUser1)
	sshDir2 := getSSHDir(t, testUser2)

	sshDirPerms1 := getFilePermissions(t, sshDir1)
	sshDirPerms2 := getFilePermissions(t, sshDir2)

	assert.Equal(t, "700", sshDirPerms1, "%s should have 700 permissions", sshDir1)
	assert.Equal(t, "700", sshDirPerms2, "%s should have 700 permissions", sshDir2)
}
