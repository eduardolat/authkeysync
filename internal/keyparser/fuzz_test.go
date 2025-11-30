package keyparser

import (
	"strings"
	"testing"
)

// FuzzParseString tests the ParseString function with random inputs
func FuzzParseString(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		"",
		"ssh-ed25519 AAAA user@host",
		"ssh-rsa BBBB comment",
		"# comment",
		"<html>error</html>",
		`{"error": "json"}`,
		"[1, 2, 3]",
		"single-field",
		"   ",
		"\n\n\n",
		"\t\t\t",
		"ssh-ed25519 AAAA user@host\nssh-rsa BBBB another@host",
		"# comment\nssh-ed25519 AAAA user@host\n# another comment",
		"restrict,port-forwarding ssh-ed25519 AAAA user@host",
		"sk-ssh-ed25519@openssh.com AAAA blob comment",
		strings.Repeat("a", 10000),
		"ssh-ed25519 " + strings.Repeat("A", 10000) + " user@host",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseString should never panic
		result, err := ParseString(input)

		// If there's an error, that's fine for fuzz testing
		if err != nil {
			return
		}

		// Result should not be nil when err is nil
		if result == nil {
			t.Fatal("result should not be nil when err is nil")
		}
		if result.Keys == nil {
			t.Fatal("result.Keys should not be nil")
		}
		if result.DiscardedLines < 0 {
			t.Fatal("discarded lines should not be negative")
		}

		// Each key should be trimmed and have at least 2 fields
		for i, key := range result.Keys {
			if key.Line != strings.TrimSpace(key.Line) {
				t.Errorf("key %d is not trimmed: %q", i, key.Line)
			}
			if len(strings.Fields(key.Line)) < 2 {
				t.Errorf("key %d has less than 2 fields: %q", i, key.Line)
			}
			if key.LineNumber < 1 {
				t.Errorf("key %d has invalid line number: %d", i, key.LineNumber)
			}
		}
	})
}

// FuzzIsValidKey tests the IsValidKey function with random inputs
func FuzzIsValidKey(f *testing.F) {
	seeds := []string{
		"",
		"ssh-ed25519 AAAA",
		"ssh-rsa BBBB user@host",
		"# comment",
		"<html>",
		"{json}",
		"[array]",
		"single",
		"   spaces   ",
		"\ttabs\t",
		"restrict ssh-ed25519 AAAA user@host",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// IsValidKey should never panic
		result := IsValidKey(input)

		// Verify consistency with ParseString
		parsed, err := ParseString(input)
		if err != nil {
			return
		}

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			if result {
				t.Error("empty/whitespace input should return false")
			}
			return
		}

		// If input has one line, result should match IsValidKey
		lines := strings.Split(input, "\n")
		if len(lines) == 1 {
			if result && len(parsed.Keys) == 0 {
				t.Errorf("IsValidKey returned true but ParseString found no keys for: %q", input)
			}
			if !result && len(parsed.Keys) > 0 {
				t.Errorf("IsValidKey returned false but ParseString found keys for: %q", input)
			}
		}
	})
}
