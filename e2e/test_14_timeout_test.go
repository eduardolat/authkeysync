package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTimeout verifies that authkeysync properly handles request timeouts.
func TestTimeout(t *testing.T) {
	_ = newTestCase(t, testUser1)

	originalKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITIMEOUT11111111111111111111111111111111111111 timeout@original"

	// Create existing authorized_keys
	createExistingKeys(t, testUser1, originalKey)

	originalContent := readAuthKeys(t, testUser1)

	// Create config with very short timeout and slow endpoint
	configPath := createConfig(t, "timeout", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/error/timeout
        timeout_seconds: 2
`)

	// Execute and measure time
	startTime := time.Now()
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	duration := time.Since(startTime)

	t.Logf("Output: %s", output)
	t.Logf("Execution took %v", duration)

	// Should exit with code 1 (failure due to timeout)
	assert.Equal(t, 1, exitCode, "authkeysync should exit with code 1 on timeout")

	// Should have timed out reasonably quickly (less than 10 seconds)
	assert.Less(t, duration, 10*time.Second, "Timeout took too long")

	// Original file should be unchanged
	currentContent := readAuthKeys(t, testUser1)
	assert.Equal(t, originalContent, currentContent, "Content should be unchanged after timeout")
}
