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

func TestLookup_NonExistentUserVariants(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{name: "random string", username: "xyzabc123nonexistent"},
		{name: "with spaces", username: "user with spaces"},
		{name: "with special chars", username: "user@special!"},
		{name: "unicode", username: "用户名"},
		{name: "empty string", username: ""},
		{name: "very long", username: string(make([]byte, 1000))},
		{name: "with null byte", username: "user\x00name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Lookup(tt.username)
			require.Error(t, err)
			// Most should be ErrUserNotFound, but some invalid formats might error differently
		})
	}
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

func TestSystemLookupProvider_CurrentUser(t *testing.T) {
	currentUser, err := user.Current()
	require.NoError(t, err)

	// Ensure .ssh directory exists for the test
	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: .ssh directory does not exist for current user")
	}

	provider := &SystemLookupProvider{}
	info, err := provider.Lookup(currentUser.Username)
	require.NoError(t, err)

	assert.Equal(t, currentUser.Username, info.Username)
	assert.NotEmpty(t, info.HomeDir)
	assert.NotEmpty(t, info.SSHDir)
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
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	_, err := Lookup("nonexistent_user_xyz123")
	require.Error(t, err)

	// Verify error wrapping
	assert.True(t, errors.Is(err, ErrUserNotFound))

	// Error message should contain the username
	assert.Contains(t, err.Error(), "nonexistent_user_xyz123")
}

func TestUserInfo_Fields(t *testing.T) {
	// Get current user to test with
	currentUser, err := user.Current()
	require.NoError(t, err)

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: .ssh directory does not exist for current user")
	}

	info, err := Lookup(currentUser.Username)
	require.NoError(t, err)

	// Verify all fields are populated correctly
	assert.NotEmpty(t, info.Username)
	assert.GreaterOrEqual(t, info.UID, 0)
	assert.GreaterOrEqual(t, info.GID, 0)
	assert.NotEmpty(t, info.HomeDir)
	assert.NotEmpty(t, info.SSHDir)
	assert.NotEmpty(t, info.AuthKeysPath)
	assert.NotEmpty(t, info.BackupDir)

	// Verify paths are correctly constructed
	assert.Equal(t, filepath.Join(info.HomeDir, ".ssh"), info.SSHDir)
	assert.Equal(t, filepath.Join(info.SSHDir, "authorized_keys"), info.AuthKeysPath)
	assert.Equal(t, filepath.Join(info.SSHDir, "authorized_keys_backups"), info.BackupDir)
}

func TestLookup_RootUser(t *testing.T) {
	// Test looking up root user (should exist on most systems)
	// But skip if .ssh doesn't exist for root
	info, err := Lookup("root")
	if err != nil {
		// Either user doesn't exist or .ssh doesn't exist
		if errors.Is(err, ErrUserNotFound) {
			t.Skip("root user does not exist on this system")
		}
		if errors.Is(err, ErrSSHDirNotFound) {
			t.Skip("root user exists but .ssh directory not found")
		}
		// Other errors are test failures
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, "root", info.Username)
	assert.Equal(t, 0, info.UID)
	assert.NotEmpty(t, info.HomeDir)
}
