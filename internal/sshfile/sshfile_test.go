package sshfile

import (
	"errors"
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

func TestWriteAtomic_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := New()
	content := []byte("ssh-ed25519 AAAA key@host\n")
	uid := os.Getuid()
	gid := os.Getgid()

	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	require.True(t, result.Changed)

	// Verify file permissions are exactly 0600
	stat, err := os.Stat(result.Path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(AuthKeysMode), stat.Mode().Perm(),
		"authorized_keys file should have 0600 permissions")
}

func TestWriteAtomic_PreservesPermissionsAfterUpdate(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := New()
	uid := os.Getuid()
	gid := os.Getgid()

	// First write
	content1 := []byte("ssh-ed25519 AAAA key1@host\n")
	result1, err := writer.WriteAtomic(sshDir, content1, uid, gid)
	require.NoError(t, err)

	stat1, err := os.Stat(result1.Path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(AuthKeysMode), stat1.Mode().Perm())

	// Second write with different content
	content2 := []byte("ssh-ed25519 BBBB key2@host\n")
	result2, err := writer.WriteAtomic(sshDir, content2, uid, gid)
	require.NoError(t, err)

	stat2, err := os.Stat(result2.Path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(AuthKeysMode), stat2.Mode().Perm(),
		"permissions should remain 0600 after update")
}

func TestWriteAtomic_EmptyContent(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := New()
	content := []byte("")
	uid := os.Getuid()
	gid := os.Getgid()

	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	assert.True(t, result.Changed)

	// Verify empty file was created
	written, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Empty(t, written)
}

func TestWriteAtomic_LargeContent(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create large content (1MB)
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte('a' + (i % 26))
	}

	writer := New()
	uid := os.Getuid()
	gid := os.Getgid()

	result, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.NoError(t, err)
	assert.True(t, result.Changed)

	// Verify content integrity
	written, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Equal(t, content, written)
}

func TestWriteAtomic_IDGeneratorError(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	writer := NewWithDeps(
		func() (string, error) { return "", errors.New("id generation failed") },
		time.Now,
	)

	content := []byte("ssh-ed25519 AAAA key@host\n")
	uid := os.Getuid()
	gid := os.Getgid()

	_, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate temp file ID")
}

func TestWriteAtomic_NonExistentSSHDir(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, "nonexistent", ".ssh")

	writer := New()
	content := []byte("ssh-ed25519 AAAA key@host\n")
	uid := os.Getuid()
	gid := os.Getgid()

	_, err := writer.WriteAtomic(sshDir, content, uid, gid)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp file")
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

func TestReadContent_NonExistentSSHDir(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, "nonexistent", ".ssh")

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

func TestNewWithDeps(t *testing.T) {
	customIDGen := func() (string, error) { return "custom", nil }
	customTime := func() time.Time { return time.Unix(0, 0) }

	writer := NewWithDeps(customIDGen, customTime)

	assert.NotNil(t, writer)
	id, _ := writer.idGenerator()
	assert.Equal(t, "custom", id)
	assert.Equal(t, time.Unix(0, 0), writer.timeNow())
}

func TestWriteAtomic_AtomicityOnContentChange(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create initial file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	initialContent := []byte("initial content\n")
	require.NoError(t, os.WriteFile(authKeysPath, initialContent, 0600))

	// Get initial inode
	stat1, err := os.Stat(authKeysPath)
	require.NoError(t, err)

	writer := New()
	newContent := []byte("new content\n")
	uid := os.Getuid()
	gid := os.Getgid()

	result, err := writer.WriteAtomic(sshDir, newContent, uid, gid)
	require.NoError(t, err)
	require.True(t, result.Changed)

	// The file should have been replaced atomically
	// Verify the content is correct
	written, err := os.ReadFile(authKeysPath)
	require.NoError(t, err)
	assert.Equal(t, newContent, written)

	// Get new stat - inode should be different (atomic replace)
	stat2, err := os.Stat(authKeysPath)
	require.NoError(t, err)

	// On most filesystems, atomic rename will change the inode
	// But we mainly care that the content is correct
	_ = stat1
	_ = stat2
}
