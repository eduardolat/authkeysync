package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnchangedContent verifies that authkeysync works correctly when run
// multiple times (multiple sync runs).
func TestUnchangedContent(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with backups disabled to simplify test
	configPath := createConfig(t, "unchanged_content", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
`)

	authKeysPath := getAuthKeysPath(t, testUser1)

	// First run - create initial file
	t.Log("First run")
	output1, exitCode1 := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output1)

	assert.Equal(t, 0, exitCode1, "First run should succeed")
	assertFileExists(t, authKeysPath)

	// Verify first run content
	content1 := readAuthKeys(t, testUser1)
	assert.Contains(t, content1, "alice@laptop", "Should contain alice's first key")
	assert.Contains(t, content1, "alice@workstation", "Should contain alice's second key")

	keyCount1 := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount1, "Should have 2 keys after first run")

	// Second run - should work correctly
	t.Log("Second run")
	output2, exitCode2 := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output2)

	assert.Equal(t, 0, exitCode2, "Second run should succeed")
	assertFileExists(t, authKeysPath)

	// Content should still be valid
	content2 := readAuthKeys(t, testUser1)
	assert.Contains(t, content2, "alice@laptop", "Should still contain alice's first key")
	assert.Contains(t, content2, "alice@workstation", "Should still contain alice's second key")

	keyCount2 := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount2, "Should still have 2 keys after second run")

	// Third run for good measure
	t.Log("Third run")
	output3, exitCode3 := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output3)

	assert.Equal(t, 0, exitCode3, "Third run should succeed")

	keyCount3 := countKeys(t, authKeysPath)
	require.Equal(t, 2, keyCount3, "Should still have 2 keys after third run")

	// No backups should exist (backups disabled)
	backupDir := getBackupDir(t, testUser1)
	if _, err := os.Stat(backupDir); err == nil {
		backupCount := countBackups(t, backupDir)
		assert.Equal(t, 0, backupCount, "No backups should exist (backups disabled)")
	}
}
