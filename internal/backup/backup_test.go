package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBackup_Success(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create authorized_keys file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	require.NoError(t, os.WriteFile(authKeysPath, []byte("ssh-ed25519 AAAA key"), 0600))

	// Create manager with fixed time and ID
	fixedTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	manager := NewWithDeps(
		func() (string, error) { return "abcdef", nil },
		func() time.Time { return fixedTime },
	)

	// Create backup
	uid := os.Getuid()
	gid := os.Getgid()
	backupPath, err := manager.CreateBackup(sshDir, uid, gid)

	require.NoError(t, err)
	assert.NotEmpty(t, backupPath)
	assert.Contains(t, backupPath, "authorized_keys_20240615_103045_abcdef")

	// Verify backup file exists and has correct content
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "ssh-ed25519 AAAA key", string(content))

	// Verify backup directory permissions
	backupDir := filepath.Join(sshDir, BackupDirName)
	stat, err := os.Stat(backupDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
	assert.Equal(t, os.FileMode(BackupDirMode), stat.Mode().Perm())
}

func TestCreateBackup_NoSourceFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	manager := New()
	uid := os.Getuid()
	gid := os.Getgid()

	backupPath, err := manager.CreateBackup(sshDir, uid, gid)

	require.NoError(t, err)
	assert.Empty(t, backupPath) // No backup created
}

func TestCreateBackup_EmptySourceFile(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create empty authorized_keys file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	require.NoError(t, os.WriteFile(authKeysPath, []byte(""), 0600))

	manager := New()
	uid := os.Getuid()
	gid := os.Getgid()

	backupPath, err := manager.CreateBackup(sshDir, uid, gid)

	require.NoError(t, err)
	assert.Empty(t, backupPath) // No backup created for empty file
}

func TestCreateBackup_BackupDirExists(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	// Create backup directory
	backupDir := filepath.Join(sshDir, BackupDirName)
	require.NoError(t, os.Mkdir(backupDir, BackupDirMode))

	// Create authorized_keys file
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	require.NoError(t, os.WriteFile(authKeysPath, []byte("ssh-ed25519 AAAA key"), 0600))

	manager := NewWithDeps(
		func() (string, error) { return "testid", nil },
		time.Now,
	)

	uid := os.Getuid()
	gid := os.Getgid()
	backupPath, err := manager.CreateBackup(sshDir, uid, gid)

	require.NoError(t, err)
	assert.NotEmpty(t, backupPath)
}

func TestRotateBackups_KeepsCorrectCount(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))
	backupDir := filepath.Join(sshDir, BackupDirName)
	require.NoError(t, os.Mkdir(backupDir, BackupDirMode))

	// Create 5 backup files with different timestamps
	backupFiles := []string{
		"authorized_keys_20240101_100000_aaaaaa",
		"authorized_keys_20240102_100000_bbbbbb",
		"authorized_keys_20240103_100000_cccccc",
		"authorized_keys_20240104_100000_dddddd",
		"authorized_keys_20240105_100000_eeeeee",
	}

	for _, name := range backupFiles {
		path := filepath.Join(backupDir, name)
		require.NoError(t, os.WriteFile(path, []byte("content"), 0600))
	}

	manager := New()
	deleted, err := manager.RotateBackups(sshDir, 3)

	require.NoError(t, err)
	assert.Len(t, deleted, 2) // Should delete 2 oldest

	// Verify oldest were deleted
	assert.Contains(t, deleted, "authorized_keys_20240101_100000_aaaaaa")
	assert.Contains(t, deleted, "authorized_keys_20240102_100000_bbbbbb")

	// Verify remaining files
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestRotateBackups_NoBackupDir(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	manager := New()
	deleted, err := manager.RotateBackups(sshDir, 10)

	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestRotateBackups_LessFilesThanRetention(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))
	backupDir := filepath.Join(sshDir, BackupDirName)
	require.NoError(t, os.Mkdir(backupDir, BackupDirMode))

	// Create 2 backup files
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240101_100000_aaaaaa"),
		[]byte("content"), 0600))
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240102_100000_bbbbbb"),
		[]byte("content"), 0600))

	manager := New()
	deleted, err := manager.RotateBackups(sshDir, 10)

	require.NoError(t, err)
	assert.Nil(t, deleted) // Nothing to delete
}

func TestRotateBackups_ZeroRetention(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))
	backupDir := filepath.Join(sshDir, BackupDirName)
	require.NoError(t, os.Mkdir(backupDir, BackupDirMode))

	// Create 2 backup files
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240101_100000_aaaaaa"),
		[]byte("content"), 0600))
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240102_100000_bbbbbb"),
		[]byte("content"), 0600))

	manager := New()
	deleted, err := manager.RotateBackups(sshDir, 0)

	require.NoError(t, err)
	assert.Len(t, deleted, 2) // All deleted
}

func TestRotateBackups_IgnoresNonBackupFiles(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))
	backupDir := filepath.Join(sshDir, BackupDirName)
	require.NoError(t, os.Mkdir(backupDir, BackupDirMode))

	// Create backup files and other files
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240101_100000_aaaaaa"),
		[]byte("content"), 0600))
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "some_other_file.txt"),
		[]byte("content"), 0600))
	require.NoError(t, os.WriteFile(
		filepath.Join(backupDir, "authorized_keys_20240102_100000_bbbbbb"),
		[]byte("content"), 0600))

	manager := New()
	deleted, err := manager.RotateBackups(sshDir, 1)

	require.NoError(t, err)
	assert.Len(t, deleted, 1) // Only oldest backup deleted
	assert.Contains(t, deleted, "authorized_keys_20240101_100000_aaaaaa")

	// Verify non-backup file still exists
	_, err = os.Stat(filepath.Join(backupDir, "some_other_file.txt"))
	require.NoError(t, err)
}

func TestRotateBackups_NegativeRetention(t *testing.T) {
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	require.NoError(t, os.Mkdir(sshDir, 0700))

	manager := New()
	_, err := manager.RotateBackups(sshDir, -1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention count cannot be negative")
}

func TestNew(t *testing.T) {
	manager := New()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.idGenerator)
	assert.NotNil(t, manager.timeNow)
}
