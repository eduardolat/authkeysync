package userinfo

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_CurrentUser(t *testing.T) {
	// Get the current user for testing
	currentUser, err := user.Current()
	require.NoError(t, err)

	// Ensure .ssh directory exists for the test
	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: .ssh directory does not exist for current user")
	}

	info, err := Lookup(currentUser.Username)
	require.NoError(t, err)

	assert.Equal(t, currentUser.Username, info.Username)
	assert.Equal(t, currentUser.HomeDir, info.HomeDir)
	assert.Equal(t, filepath.Join(currentUser.HomeDir, ".ssh"), info.SSHDir)
	assert.Equal(t, filepath.Join(currentUser.HomeDir, ".ssh", "authorized_keys"), info.AuthKeysPath)
	assert.Equal(t, filepath.Join(currentUser.HomeDir, ".ssh", "authorized_keys_backups"), info.BackupDir)
	assert.Greater(t, info.UID, -1)
	assert.Greater(t, info.GID, -1)
}

func TestLookup_NonExistentUser(t *testing.T) {
	_, err := Lookup("nonexistent_user_that_does_not_exist_xyz123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestLookup_MissingSSHDir(t *testing.T) {
	// We can't easily test this without mocking user.Lookup
	// This test documents the expected behavior
	t.Skip("Requires mocking user.Lookup to test missing .ssh directory")
}

func TestSystemLookupProvider(t *testing.T) {
	provider := &SystemLookupProvider{}

	// Test that provider implements the interface
	var _ LookupProvider = provider

	// Test with a non-existent user
	_, err := provider.Lookup("nonexistent_user_that_does_not_exist_xyz123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "ErrUserNotFound", err: ErrUserNotFound},
		{name: "ErrNoHomeDir", err: ErrNoHomeDir},
		{name: "ErrSSHDirNotFound", err: ErrSSHDirNotFound},
		{name: "ErrSSHDirNotDir", err: ErrSSHDirNotDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
		})
	}
}
