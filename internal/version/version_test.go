package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserAgent(t *testing.T) {
	ua := UserAgent()
	assert.Contains(t, ua, "AuthKeySync/")
	assert.Contains(t, ua, Version)
}

func TestDefaultValues(t *testing.T) {
	// Default values should be set
	assert.NotEmpty(t, Version)
	assert.NotEmpty(t, Commit)
	assert.NotEmpty(t, Date)
}
