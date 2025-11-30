package sshfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAtomic_NewFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := New()
	content := []byte("ssh-ed25519 AAAA key@host\n")

	uid := os.Getuid()
	gid := os.Getgid()
	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.Equal(t, filepath.Join(sshDir, "authorized_keys"), result.Path)

	// Verify file content
	written, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Equal(t, content, written)

	// Verify permissions
	stat, err := os.Stat(result.Path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(AuthKeysMode), stat.Mode().Perm())
}

func TestWriteAtomic_ExistingFileWithSameContent(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create existing file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	content := []byte("ssh-ed25519 AAAA key@host\n")
	require.NoError(t, os.WriteFile(authKeysPath, content, 0600))

	writer := New()
	uid := os.Getuid()
	gid := os.Getgid()
	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	assert.False(t, result.Changed) // No change
}

func TestWriteAtomic_ExistingFileWithDifferentContent(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create existing file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	oldContent := []byte("ssh-ed25519 AAAA old@host\n")
	require.NoError(t, os.WriteFile(authKeysPath, oldContent, 0600))

	writer := New()
	newContent := []byte("ssh-ed25519 BBBB new@host\n")
	uid := os.Getuid()
	gid := os.Getgid()
	result, err := writer.WriteAtomic(sshDir, newContent, uid, gid)

	require.NoError(t, err)
	assert.True(t, result.Changed)

	// Verify new content
	written, err := os.ReadFile(authKeysPath)
	require.NoError(t, err)
	assert.Equal(t, newContent, written)
}

func TestWriteAtomic_TempFileCleanup(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := New()
	content := []byte("ssh-ed25519 AAAA key@host\n")
	uid := os.Getuid()
	gid := os.Getgid()

	_, err := writer.WriteAtomic(sshDir, content, uid, gid)
	require.NoError(t, err)

	// Verify no temp files left
	entries, err := os.ReadDir(sshDir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.False(t, len(entry.Name()) > 0 && entry.Name()[0] == '.',
			"temp file left behind: %s", entry.Name())
	}
}

func TestWriteAtomic_CustomDeps(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	fixedTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	writer := NewWithDeps(
		func() (string, error) { return "testid", nil },
		func() time.Time { return fixedTime },
	)

	content := []byte("ssh-ed25519 AAAA key@host\n")
	uid := os.Getuid()
	gid := os.Getgid()
	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	assert.True(t, result.Changed)
}

func TestReadContent_ExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	content := []byte("ssh-ed25519 AAAA key@host\n")
	require.NoError(t, os.WriteFile(authKeysPath, content, 0600))

	read, err := ReadContent(sshDir)
	require.NoError(t, err)
	assert.Equal(t, content, read)
}

func TestReadContent_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	read, err := ReadContent(sshDir)
	require.NoError(t, err)
	assert.Empty(t, read)
}

func TestReadContent_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create empty file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	require.NoError(t, os.WriteFile(authKeysPath, []byte{}, 0600))

	read, err := ReadContent(sshDir)
	require.NoError(t, err)
	assert.Empty(t, read)
}

func TestNew(t *testing.T) {
	writer := New()
	assert.NotNil(t, writer)
	assert.NotNil(t, writer.idGenerator)
	assert.NotNil(t, writer.timeNow)
}
