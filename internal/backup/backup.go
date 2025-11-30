// Package backup handles creation and rotation of authorized_keys backups.
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eduardolat/authkeysync/internal/nanoid"
)

const (
	// BackupDirName is the name of the backup directory
	BackupDirName = "authorized_keys_backups"
	// BackupDirMode is the permission mode for the backup directory
	BackupDirMode = 0700
	// BackupFileMode is the permission mode for backup files
	BackupFileMode = 0600
	// BackupPrefix is the prefix for backup filenames
	BackupPrefix = "authorized_keys_"
)

// Manager handles backup creation and rotation
type Manager struct {
	// idGenerator allows for dependency injection in tests
	idGenerator func() (string, error)
	// timeNow allows for dependency injection in tests
	timeNow func() time.Time
}

// New creates a new backup Manager
func New() *Manager {
	return &Manager{
		idGenerator: nanoid.Generate,
		timeNow:     time.Now,
	}
}

// NewWithDeps creates a new backup Manager with custom dependencies (for testing)
func NewWithDeps(idGen func() (string, error), timeNow func() time.Time) *Manager {
	return &Manager{
		idGenerator: idGen,
		timeNow:     timeNow,
	}
}

// CreateBackup creates a backup of the authorized_keys file.
// Returns the backup file path, or empty string if no backup was created.
// If the source file doesn't exist or is empty, no backup is created.
func (m *Manager) CreateBackup(sshDir string, uid, gid int) (string, error) {
	authKeysPath := filepath.Join(sshDir, "authorized_keys")

	// Check if source file exists
	stat, err := os.Stat(authKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No file to backup
			return "", nil
		}
		return "", fmt.Errorf("failed to stat authorized_keys: %w", err)
	}

	// Skip if file is empty
	if stat.Size() == 0 {
		return "", nil
	}

	// Ensure backup directory exists
	backupDir := filepath.Join(sshDir, BackupDirName)
	if err := m.ensureBackupDir(backupDir, uid, gid); err != nil {
		return "", err
	}

	// Generate backup filename
	timestamp := m.timeNow().UTC().Format("20060102_150405")
	id, err := m.idGenerator()
	if err != nil {
		return "", fmt.Errorf("failed to generate backup ID: %w", err)
	}
	backupFilename := fmt.Sprintf("%s%s_%s", BackupPrefix, timestamp, id)
	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy file
	if err := m.copyFile(authKeysPath, backupPath, uid, gid); err != nil {
		return "", err
	}

	return backupPath, nil
}

// ensureBackupDir creates the backup directory if it doesn't exist
func (m *Manager) ensureBackupDir(backupDir string, uid, gid int) error {
	stat, err := os.Stat(backupDir)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("backup path exists but is not a directory: %s", backupDir)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat backup directory: %w", err)
	}

	// Create directory
	if err := os.Mkdir(backupDir, BackupDirMode); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Set ownership
	if err := os.Chown(backupDir, uid, gid); err != nil {
		return fmt.Errorf("failed to set backup directory ownership: %w", err)
	}

	return nil
}

// copyFile copies a file and sets proper permissions and ownership
func (m *Manager) copyFile(src, dst string, uid, gid int) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, BackupFileMode)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync backup file: %w", err)
	}

	// Set ownership
	if err := os.Chown(dst, uid, gid); err != nil {
		return fmt.Errorf("failed to set backup file ownership: %w", err)
	}

	return nil
}

// RotateBackups removes old backups, keeping only the specified count.
// Oldest files are deleted first (based on filename which includes timestamp).
func (m *Manager) RotateBackups(sshDir string, retentionCount int) ([]string, error) {
	if retentionCount < 0 {
		return nil, fmt.Errorf("retention count cannot be negative")
	}

	backupDir := filepath.Join(sshDir, BackupDirName)

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, nil
	}

	// List backup files
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter and collect backup files
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), BackupPrefix) {
			backups = append(backups, entry.Name())
		}
	}

	// Sort by filename (which includes timestamp, so alphabetical = chronological)
	sort.Strings(backups)

	// Calculate how many to delete
	deleteCount := len(backups) - retentionCount
	if deleteCount <= 0 {
		return nil, nil
	}

	// Delete oldest files
	deleted := make([]string, 0, deleteCount)
	for i := 0; i < deleteCount; i++ {
		path := filepath.Join(backupDir, backups[i])
		if err := os.Remove(path); err != nil {
			return deleted, fmt.Errorf("failed to remove backup %s: %w", backups[i], err)
		}
		deleted = append(deleted, backups[i])
	}

	return deleted, nil
}

// ManagerProvider is an interface for backup management
type ManagerProvider interface {
	CreateBackup(sshDir string, uid, gid int) (string, error)
	RotateBackups(sshDir string, retentionCount int) ([]string, error)
}
