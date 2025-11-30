// Package keyparser provides SSH public key parsing and validation.
package keyparser

import (
	"bufio"
	"io"
	"strings"
)

// ParsedKey represents a parsed SSH public key line
type ParsedKey struct {
	// Line is the trimmed key content
	Line string
	// LineNumber is the original line number (1-indexed)
	LineNumber int
}

// ParseResult contains the results of parsing keys from a source
type ParseResult struct {
	// Keys are the valid SSH keys found
	Keys []ParsedKey
	// DiscardedLines is the count of lines that were discarded
	DiscardedLines int
}

// Parse parses SSH public keys from a reader.
// It applies the parsing rules defined in the specification:
// - Empty lines are discarded
// - Comment lines (starting with #) are discarded
// - HTML/JSON error lines (starting with <, {, or [) are discarded
// - Valid lines must have at least 2 whitespace-separated fields
func Parse(r io.Reader) (*ParseResult, error) {
	result := &ParseResult{
		Keys: make([]ParsedKey, 0),
	}

	scanner := bufio.NewScanner(r)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if isValidKey(line) {
			result.Keys = append(result.Keys, ParsedKey{
				Line:       line,
				LineNumber: lineNumber,
			})
		} else if line != "" {
			// Only count non-empty lines as discarded
			result.DiscardedLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ParseString is a convenience function to parse keys from a string
func ParseString(content string) (*ParseResult, error) {
	return Parse(strings.NewReader(content))
}

// isValidKey checks if a trimmed line is a valid SSH public key
func isValidKey(line string) bool {
	// Empty lines are not valid
	if line == "" {
		return false
	}

	// Comment lines are not valid
	if strings.HasPrefix(line, "#") {
		return false
	}

	// HTML/JSON error lines are not valid
	if strings.HasPrefix(line, "<") || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
		return false
	}

	// Must have at least 2 whitespace-separated fields
	fields := strings.Fields(line)
	return len(fields) >= 2
}

// IsValidKey is exported for testing purposes
func IsValidKey(line string) bool {
	return isValidKey(strings.TrimSpace(line))
}
