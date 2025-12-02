package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackupCreation verifies that backups are created when content changes
// and backup is enabled.
func TestBackupCreation(t *testing.T) {
	_ = newTestCase(t, testUser1)

	originalKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIORIG111111111111111111111111111111111111111111 original@key"

	// Create existing authorized_keys
	createExistingKeys(t, testUser1, originalKey)

	// Create config with backup enabled
	configPath := createConfig(t, "backup_creation", `
policy:
  backup_enabled: true
  backup_retention_count: 5
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

	// Check backup directory was created
	backupDir := getBackupDir(t, testUser1)
	assertDirExists(t, backupDir, "Backup directory should exist")

	// Check backup directory has correct permissions
	info, err := os.Stat(backupDir)
	require.NoError(t, err)
	perms := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0700), perms, "Backup directory should have 700 permissions")

	// Check backup directory ownership
	owner := getFileOwner(t, backupDir)
	assert.Equal(t, testUser1, owner, "Backup directory should be owned by %s", testUser1)

	// Check that a backup file was created
	backupCount := countBackups(t, backupDir)
	assert.Equal(t, 1, backupCount, "Should have exactly 1 backup file")

	// Get the backup file
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)

	var backupFile string
	for _, entry := range entries {
		if !entry.IsDir() {
			backupFile = filepath.Join(backupDir, entry.Name())
			break
		}
	}
	require.NotEmpty(t, backupFile, "No backup file found")

	t.Logf("Backup file: %s", backupFile)

	// Check backup file permissions
	backupPerms := getFilePermissions(t, backupFile)
	assert.Equal(t, "600", backupPerms, "Backup file should have 600 permissions")

	// Check backup file ownership
	backupOwner := getFileOwner(t, backupFile)
	assert.Equal(t, testUser1, backupOwner, "Backup file should be owned by %s", testUser1)

	// Check backup content matches original
	backupContent, err := os.ReadFile(backupFile)
	require.NoError(t, err)
	assert.Contains(t, string(backupContent), "original@key", "Backup should contain original key")

	// Check new authorized_keys has new content
	newContent := readAuthKeys(t, testUser1)
	assert.Contains(t, newContent, "alice@laptop", "New content should have alice's key")
	assert.NotContains(t, newContent, "original@key", "New content should NOT have original key")
}
