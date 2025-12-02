package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSourceFails verifies that authkeysync aborts user sync and returns
// exit code 1 when source fails.
func TestSourceFails(t *testing.T) {
	_ = newTestCase(t, testUser1)

	originalKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAISRCFAIL11111111111111111111111111111111111 srcfail@original"

	// Create existing authorized_keys
	createExistingKeys(t, testUser1, originalKey)

	originalContent := readAuthKeys(t, testUser1)

	// Create config with a source that returns 500 error
	configPath := createConfig(t, "source_fails", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/error/500
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Should exit with code 1 (failure)
	assert.Equal(t, 1, exitCode, "authkeysync should exit with code 1 on source failure")

	// Original file should be unchanged (abort on failure)
	currentContent := readAuthKeys(t, testUser1)
	assert.Equal(t, originalContent, currentContent, "Content should be unchanged after failure")
	assert.Contains(t, currentContent, "srcfail@original", "Original key should still be present")
}
