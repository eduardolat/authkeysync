// Package nanoid provides NanoID generation for random identifiers.
package nanoid

import (
	"crypto/rand"
)

const (
	// alphabet contains only lowercase letters as per spec
	alphabet = "abcdefghijklmnopqrstuvwxyz"
	// idLength is the length of generated IDs (6 characters = 26^6 = 308,915,776 combinations)
	idLength = 6
)

// Generate creates a new NanoID with 6 lowercase letters.
// Uses crypto/rand for secure random generation.
func Generate() (string, error) {
	bytes := make([]byte, idLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	result := make([]byte, idLength)
	for i := 0; i < idLength; i++ {
		result[i] = alphabet[int(bytes[i])%len(alphabet)]
	}

	return string(result), nil
}

// MustGenerate creates a new NanoID and panics on error.
// Use only when you're certain random generation won't fail.
func MustGenerate() string {
	id, err := Generate()
	if err != nil {
		panic(err)
	}
	return id
}
