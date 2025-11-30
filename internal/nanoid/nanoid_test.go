package nanoid

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "generates valid nanoid"},
		{name: "generates another valid nanoid"},
		{name: "generates yet another valid nanoid"},
	}

	pattern := regexp.MustCompile(`^[a-z]{6}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Generate()
			require.NoError(t, err)
			assert.Len(t, id, 6)
			assert.Regexp(t, pattern, id)
		})
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		id, err := Generate()
		require.NoError(t, err)
		assert.False(t, ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func TestGenerate_OnlyLowercaseLetters(t *testing.T) {
	for i := 0; i < 100; i++ {
		id, err := Generate()
		require.NoError(t, err)

		for _, char := range id {
			assert.True(t, char >= 'a' && char <= 'z',
				"character %c is not a lowercase letter", char)
		}
	}
}

func TestMustGenerate(t *testing.T) {
	// Should not panic under normal conditions
	assert.NotPanics(t, func() {
		id := MustGenerate()
		assert.Len(t, id, 6)
	})
}
