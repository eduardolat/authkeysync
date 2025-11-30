// Package nanoid provides NanoID generation for random identifiers.
package nanoid

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
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
	return gonanoid.Generate(alphabet, idLength)
}

// MustGenerate creates a new NanoID and panics on error.
// Use only when you're certain random generation won't fail.
func MustGenerate() string {
	return gonanoid.MustGenerate(alphabet, idLength)
}
