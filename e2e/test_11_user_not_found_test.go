package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUserNotFound verifies that authkeysync skips non-existent users
// with a warning but exits 0.
func TestUserNotFound(t *testing.T) {
	// No user setup needed - we're testing with a non-existent user

	// Create config with a non-existent user
	configPath := createConfig(t, "user_not_found", `
policy:
  backup_enabled: false

users:
  - username: nonexistentuser12345
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Should exit with code 0 (skipped users are not errors)
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0 (skip is not an error)")

	// Output should contain warning about user not found
	assert.Contains(t, output, "skipping", "Output should mention skipping")
}
