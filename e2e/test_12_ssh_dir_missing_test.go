package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSSHDirMissing verifies that authkeysync skips users without .ssh
// directory but exits 0.
func TestSSHDirMissing(t *testing.T) {
	// e2euser3 was created without .ssh directory
	// Ensure .ssh directory does NOT exist for e2euser3
	removeSSHDir(t, testUser3)

	sshDir := getSSHDir(t, testUser3)
	assertFileNotExists(t, sshDir, ".ssh directory should not exist")

	// Create config
	configPath := createConfig(t, "ssh_dir_missing", `
policy:
  backup_enabled: false

users:
  - username: `+testUser3+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Should exit with code 0 (skipped users are not errors)
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0 (skip is not an error)")

	// Output should contain warning about .ssh directory
	assert.Contains(t, output, "skipping", "Output should mention skipping")

	// .ssh directory should NOT have been created
	authKeysPath := getAuthKeysPath(t, testUser3)
	assertFileNotExists(t, authKeysPath, "authorized_keys should not be created")
}
