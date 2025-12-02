package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeduplication verifies that duplicate keys across sources are deduplicated
// (first occurrence wins).
func TestDeduplication(t *testing.T) {
	_ = newTestCase(t, testUser1)

	// Create config with sources that have overlapping keys
	// duplicate.keys contains the same first key as alice.keys
	configPath := createConfig(t, "deduplication", `
policy:
  backup_enabled: false
  preserve_local_keys: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
      - url: ${MOCK_SERVER_URL}/users/duplicate.keys
`)

	// Execute
	output, exitCode := runAuthKeySync(t, "--config", configPath)
	t.Logf("Output: %s", output)

	// Verify exit code
	assert.Equal(t, 0, exitCode, "authkeysync should exit with code 0")

	// Verify file exists
	authKeysPath := getAuthKeysPath(t, testUser1)
	content := readAuthKeys(t, testUser1)

	// The duplicate key (alice@laptop from alice.keys) should appear only once
	// even though it's in both alice.keys and duplicate.keys
	aliceLaptopCount := strings.Count(content, "alice@laptop")
	assert.Equal(t, 1, aliceLaptopCount, "alice@laptop key should appear exactly once")

	// The unique key from duplicate.keys should still be present
	assert.Contains(t, content, "unique@duplicate", "Unique key from duplicate source should be present")

	// Total keys: alice has 2, duplicate has 2 but 1 is duplicate
	// So we should have 3 unique keys total
	keyCount := countKeys(t, authKeysPath)
	require.Equal(t, 3, keyCount, "Should have 3 unique keys (2 from alice + 1 unique from duplicate)")
}
