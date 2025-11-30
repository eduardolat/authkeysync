// Package sshfile handles atomic writing of authorized_keys files.
package sshfile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eduardolat/authkeysync/internal/nanoid"
)

const (
	// AuthKeysMode is the permission mode for authorized_keys files (0600)
	AuthKeysMode = 0600
	// TempFilePrefix is the prefix for temporary files
	TempFilePrefix = ".authkeysync_"
)

// Writer handles atomic file writes
type Writer struct {
	// idGenerator allows for dependency injection in tests
	idGenerator func() (string, error)
	// timeNow allows for dependency injection in tests
	timeNow func() time.Time
}

// New creates a new Writer
func New() *Writer {
	return &Writer{
		idGenerator: nanoid.Generate,
		timeNow:     time.Now,
	}
}

// NewWithDeps creates a new Writer with custom dependencies (for testing)
func NewWithDeps(idGen func() (string, error), timeNow func() time.Time) *Writer {
	return &Writer{
		idGenerator: idGen,
		timeNow:     timeNow,
	}
}

// WriteResult contains information about a write operation
type WriteResult struct {
	// Changed indicates whether the file content was different
	Changed bool
	// Path is the final path of the written file
	Path string
}

// WriteAtomic atomically writes content to the authorized_keys file.
// It uses the atomic write procedure specified in the spec:
// 1. Create temp file in the same directory
// 2. Set permissions (0600)
// 3. Set ownership (uid:gid)
// 4. Write content and fsync
// 5. Atomic rename
//
// Returns whether the file was changed (different content).
func (w *Writer) WriteAtomic(sshDir string, content []byte, uid, gid int) (*WriteResult, error) {
	authKeysPath := filepath.Join(sshDir, "authorized_keys")

	// Check if content is different from existing file
	existingContent, err := os.ReadFile(authKeysPath)
	if err == nil && bytes.Equal(existingContent, content) {
		return &WriteResult{Changed: false, Path: authKeysPath}, nil
	}

	// Generate temp filename
	timestamp := w.timeNow().UTC().Format("20060102_150405")
	id, err := w.idGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate temp file ID: %w", err)
	}
	tempFilename := fmt.Sprintf("%s%s_%s", TempFilePrefix, timestamp, id)
	tempPath := filepath.Join(sshDir, tempFilename)

	// Create temp file
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_EXCL, AuthKeysMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Ensure cleanup on error
	success := false
	defer func() {
		if !success {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	// Set permissions explicitly (in case umask affected file creation)
	if err := tempFile.Chmod(AuthKeysMode); err != nil {
		return nil, fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	// Set ownership
	if err := os.Chown(tempPath, uid, gid); err != nil {
		return nil, fmt.Errorf("failed to set temp file ownership: %w", err)
	}

	// Write content
	if _, err := tempFile.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close before rename
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, authKeysPath); err != nil {
		return nil, fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return &WriteResult{Changed: true, Path: authKeysPath}, nil
}

// ReadContent reads the current content of the authorized_keys file.
// Returns empty byte slice if file doesn't exist.
func ReadContent(sshDir string) ([]byte, error) {
	authKeysPath := filepath.Join(sshDir, "authorized_keys")

	content, err := os.ReadFile(authKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, nil
		}
		return nil, fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	return content, nil
}

// WriterProvider is an interface for atomic file writing
type WriterProvider interface {
	WriteAtomic(sshDir string, content []byte, uid, gid int) (*WriteResult, error)
}
