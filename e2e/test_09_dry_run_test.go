package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDryRun verifies that --dry-run flag prevents any file modifications.
func TestDryRun(t *testing.T) {
	_ = newTestCase(t, testUser1)

	originalKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDRYRUN1111111111111111111111111111111111111111 dryrun@original"

	// Create existing authorized_keys
	createExistingKeys(t, testUser1, originalKey)

	// Record original content
	originalContent := readAuthKeys(t, testUser1)

	// Create config
	configPath := createConfig(t, "dry_run", `
policy:
  backup_enabled: true
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
`)

	// Execute with --dry-run
	output, exitCode := runAuthKeySync(t, "--config", configPath, "--dry-run")
	t.Logf("Output: %s", output)

	// Verify exit code
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0")

	// File should still exist
	authKeysPath := getAuthKeysPath(t, testUser1)
	assertFileExists(t, authKeysPath, "authorized_keys should still exist")

	// Content should be unchanged
	currentContent := readAuthKeys(t, testUser1)
	assert.Equal(t, originalContent, currentContent, "Content should be unchanged")

	// The original key should still be there
	assert.Contains(t, currentContent, "dryrun@original", "Original key should still be present")

	// New keys should NOT be present
	assert.NotContains(t, currentContent, "alice@laptop", "New key should NOT be present")

	// No backup should have been created
	backupDir := getBackupDir(t, testUser1)
	if _, err := os.Stat(backupDir); err == nil {
		backupCount := countBackups(t, backupDir)
		assert.Equal(t, 0, backupCount, "No backups should be created in dry-run mode")
	}
}
