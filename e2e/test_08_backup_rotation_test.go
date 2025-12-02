package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBackupRotation verifies that old backups are rotated based on retention count.
func TestBackupRotation(t *testing.T) {
	_ = newTestCase(t, testUser1)

	retentionCount := 3

	// Create backup directory with existing backups
	createBackups(t, testUser1, 5)

	t.Log("Created 5 old backups")

	// Create current authorized_keys
	createExistingKeys(t, testUser1, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICURRENT1111111111111111111111111111111111111 current@key")

	// Create config with backup enabled and retention count of 3
	configPath := createConfig(t, "backup_rotation", `
policy:
  backup_enabled: true
  backup_retention_count: 3
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

	// Check backup count (should be retention count, not 6)
	// We had 5 + 1 new = 6, but rotation should keep only 3
	backupDir := getBackupDir(t, testUser1)
	backupCount := countBackups(t, backupDir)

	t.Logf("Backup count after rotation: %d", backupCount)

	// The backup count should be equal to retention count
	assert.Equal(t, retentionCount, backupCount, "Should have exactly %d backups after rotation", retentionCount)

	// Verify the oldest backups were deleted (01, 02, 03 should be gone)
	assertFileNotExists(t, backupDir+"/authorized_keys_20240101_120001_abcde1", "Oldest backup 1 should be deleted")
	assertFileNotExists(t, backupDir+"/authorized_keys_20240102_120002_abcde2", "Oldest backup 2 should be deleted")
	assertFileNotExists(t, backupDir+"/authorized_keys_20240103_120003_abcde3", "Oldest backup 3 should be deleted")
}
