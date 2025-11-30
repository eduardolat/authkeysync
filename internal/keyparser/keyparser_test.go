package keyparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidKeys(t *testing.T) {
	content := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host
ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY comment with spaces`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 3)

	assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host", result.Keys[0].Line)
	assert.Equal(t, 1, result.Keys[0].LineNumber)

	assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host", result.Keys[1].Line)
	assert.Equal(t, 2, result.Keys[1].LineNumber)

	assert.Equal(t, "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY comment with spaces", result.Keys[2].Line)
	assert.Equal(t, 3, result.Keys[2].LineNumber)

	assert.Equal(t, 0, result.DiscardedLines)
}

func TestParse_WithComments(t *testing.T) {
	content := `# This is a comment
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host
# Another comment
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)
	assert.Equal(t, 2, result.DiscardedLines)
}

func TestParse_WithEmptyLines(t *testing.T) {
	content := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host

   
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)
	assert.Equal(t, 0, result.DiscardedLines) // Empty lines don't count as discarded
}

func TestParse_HTMLErrorPage(t *testing.T) {
	content := `<html>
<body>
<h1>404 Not Found</h1>
</body>
</html>`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 0)
	assert.Equal(t, 5, result.DiscardedLines)
}

func TestParse_JSONError(t *testing.T) {
	content := `{"error": "unauthorized"}
[{"message": "error"}]`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 0)
	assert.Equal(t, 2, result.DiscardedLines)
}

func TestParse_MixedContent(t *testing.T) {
	content := `# Header comment
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host

<html>error page</html>
{"error": "something"}
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ valid@key
invalid-single-field
# Trailing comment`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)
	assert.Equal(t, 5, result.DiscardedLines) // 2 comments + html + json + invalid
}

func TestParse_EmptyContent(t *testing.T) {
	result, err := ParseString("")
	require.NoError(t, err)
	require.Len(t, result.Keys, 0)
	assert.Equal(t, 0, result.DiscardedLines)
}

func TestParse_OnlyWhitespace(t *testing.T) {
	content := `   
	
    `

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 0)
	assert.Equal(t, 0, result.DiscardedLines)
}

func TestParse_KeyWithOptions(t *testing.T) {
	content := `restrict,port-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host
no-pty,no-agent-forwarding ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ user@host`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)
}

func TestParse_WhitespaceTrimming(t *testing.T) {
	content := `   ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host   
	ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host	`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)

	// Keys should be trimmed
	assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host", result.Keys[0].Line)
	assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host", result.Keys[1].Line)
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{name: "valid ssh-ed25519", line: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host", expected: true},
		{name: "valid ssh-rsa", line: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ", expected: true},
		{name: "valid with options", line: "restrict ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit", expected: true},
		{name: "valid sk-ssh-ed25519", line: "sk-ssh-ed25519@openssh.com AAAA blob comment", expected: true},
		{name: "empty", line: "", expected: false},
		{name: "comment", line: "# comment", expected: false},
		{name: "html", line: "<html>", expected: false},
		{name: "json object", line: "{\"error\": true}", expected: false},
		{name: "json array", line: "[\"error\"]", expected: false},
		{name: "single field", line: "ssh-ed25519", expected: false},
		{name: "whitespace only", line: "   ", expected: false},
		{name: "with leading whitespace", line: "  ssh-ed25519 AAAA", expected: true},
		{name: "with trailing whitespace", line: "ssh-ed25519 AAAA  ", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidKey(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParse_LineNumbers(t *testing.T) {
	content := `# Comment on line 1

ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host

ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host`

	result, err := ParseString(content)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)

	assert.Equal(t, 3, result.Keys[0].LineNumber)
	assert.Equal(t, 5, result.Keys[1].LineNumber)
}

func TestParse_ReaderError(t *testing.T) {
	// Create a reader that will error
	r := &errorReader{err: assert.AnError}
	_, err := Parse(r)
	require.Error(t, err)
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestParse_LargeContent(t *testing.T) {
	var builder strings.Builder
	for i := range 1000 {
		builder.WriteString("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user")
		builder.WriteString(string(rune('0' + i%10)))
		builder.WriteString("@host\n")
	}

	result, err := ParseString(builder.String())
	require.NoError(t, err)
	assert.Len(t, result.Keys, 1000)
}
