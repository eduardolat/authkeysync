// Package userinfo provides system user information lookup.
package userinfo

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

var (
	// ErrUserNotFound indicates the user does not exist in the system
	ErrUserNotFound = errors.New("user not found")
	// ErrNoHomeDir indicates the user has no home directory
	ErrNoHomeDir = errors.New("user has no home directory")
	// ErrSSHDirNotFound indicates the .ssh directory does not exist
	ErrSSHDirNotFound = errors.New(".ssh directory not found")
	// ErrSSHDirNotDir indicates .ssh exists but is not a directory
	ErrSSHDirNotDir = errors.New(".ssh exists but is not a directory")
)

// UserInfo contains information about a system user
type UserInfo struct {
	Username     string
	UID          int
	GID          int
	HomeDir      string
	SSHDir       string
	AuthKeysPath string
	BackupDir    string
}

// Lookup looks up a user by username and returns their information.
// Returns ErrUserNotFound if the user doesn't exist.
// Returns ErrNoHomeDir if the user has no home directory.
// Returns ErrSSHDirNotFound if the .ssh directory doesn't exist.
// Returns ErrSSHDirNotDir if .ssh exists but is not a directory.
func Lookup(username string) (*UserInfo, error) {
	u, err := user.Lookup(username)
	if err != nil {
		var unknownUserError user.UnknownUserError
		if errors.As(err, &unknownUserError) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, username)
		}
		return nil, fmt.Errorf("failed to lookup user %s: %w", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UID for user %s: %w", username, err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GID for user %s: %w", username, err)
	}

	if u.HomeDir == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoHomeDir, username)
	}

	sshDir := filepath.Join(u.HomeDir, ".ssh")
	stat, err := os.Stat(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSSHDirNotFound, sshDir)
		}
		return nil, fmt.Errorf("failed to stat .ssh directory for user %s: %w", username, err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrSSHDirNotDir, sshDir)
	}

	return &UserInfo{
		Username:     username,
		UID:          uid,
		GID:          gid,
		HomeDir:      u.HomeDir,
		SSHDir:       sshDir,
		AuthKeysPath: filepath.Join(sshDir, "authorized_keys"),
		BackupDir:    filepath.Join(sshDir, "authorized_keys_backups"),
	}, nil
}

// LookupProvider is an interface for looking up user information.
// This allows for dependency injection and easier testing.
type LookupProvider interface {
	Lookup(username string) (*UserInfo, error)
}

// SystemLookupProvider uses the real system user lookup
type SystemLookupProvider struct{}

// Lookup implements LookupProvider using the system
func (p *SystemLookupProvider) Lookup(username string) (*UserInfo, error) {
	return Lookup(username)
}
