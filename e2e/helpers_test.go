package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runAuthKeySync executes the authkeysync binary with the given arguments
// Returns stdout, stderr combined output, and exit code
func runAuthKeySync(t *testing.T, args ...string) (output string, exitCode int) {
	t.Helper()

	cmd := exec.Command(env.binaryPath, args...)
	outputBytes, err := cmd.CombinedOutput()
	output = string(outputBytes)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return output, exitError.ExitCode()
		}
		t.Fatalf("Failed to run authkeysync: %v\nOutput: %s", err, output)
	}

	return output, 0
}

// createConfig creates a temporary config file with the given content
func createConfig(t *testing.T, name string, content string) string {
	t.Helper()

	configPath := filepath.Join(configTempDir, fmt.Sprintf("authkeysync_test_%s.yaml", name))

	// Replace placeholder with actual mock server URL
	content = strings.ReplaceAll(content, "${MOCK_SERVER_URL}", env.mockServerURL)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(configPath)
	})

	return configPath
}

// createExistingKeys creates an authorized_keys file for a user with the given content
func createExistingKeys(t *testing.T, username, content string) {
	t.Helper()

	u, err := user.Lookup(username)
	require.NoError(t, err, "User lookup failed")

	sshDir := filepath.Join(u.HomeDir, ".ssh")
	authKeysPath := filepath.Join(sshDir, "authorized_keys")

	// Ensure .ssh directory exists
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	// Write authorized_keys
	require.NoError(t, os.WriteFile(authKeysPath, []byte(content), 0600))

	// Set ownership
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	require.NoError(t, os.Chown(authKeysPath, uid, gid))
	require.NoError(t, os.Chown(sshDir, uid, gid))
}

// getAuthKeysPath returns the path to the authorized_keys file for a user
func getAuthKeysPath(t *testing.T, username string) string {
	t.Helper()

	u, err := user.Lookup(username)
	require.NoError(t, err, "User lookup failed")

	return filepath.Join(u.HomeDir, ".ssh", "authorized_keys")
}

// getBackupDir returns the path to the backup directory for a user
func getBackupDir(t *testing.T, username string) string {
	t.Helper()

	u, err := user.Lookup(username)
	require.NoError(t, err, "User lookup failed")

	return filepath.Join(u.HomeDir, ".ssh", "authorized_keys_backups")
}

// getSSHDir returns the path to the .ssh directory for a user
func getSSHDir(t *testing.T, username string) string {
	t.Helper()

	u, err := user.Lookup(username)
	require.NoError(t, err, "User lookup failed")

	return filepath.Join(u.HomeDir, ".ssh")
}

// readAuthKeys reads the authorized_keys file content for a user
func readAuthKeys(t *testing.T, username string) string {
	t.Helper()

	content, err := os.ReadFile(getAuthKeysPath(t, username))
	require.NoError(t, err, "Failed to read authorized_keys")

	return string(content)
}

// countKeys counts the number of SSH keys in the authorized_keys file
func countKeys(t *testing.T, filePath string) int {
	t.Helper()

	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "ssh-") {
			count++
		}
	}

	return count
}

// countBackups counts the number of backup files in the backup directory
func countBackups(t *testing.T, backupDir string) int {
	t.Helper()

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "authorized_keys_") {
			count++
		}
	}

	return count
}

// getFilePermissions returns the file permissions as a string (e.g., "600")
func getFilePermissions(t *testing.T, filePath string) string {
	t.Helper()

	info, err := os.Stat(filePath)
	require.NoError(t, err, "Failed to stat file")

	return fmt.Sprintf("%o", info.Mode().Perm())
}

// getFileOwner returns the owner username of a file
func getFileOwner(t *testing.T, filePath string) string {
	t.Helper()

	info, err := os.Stat(filePath)
	require.NoError(t, err, "Failed to stat file")

	stat := info.Sys().(*syscall.Stat_t)
	u, err := user.LookupId(strconv.Itoa(int(stat.Uid)))
	require.NoError(t, err, "Failed to lookup user")

	return u.Username
}

// getFileGroup returns the group name of a file
func getFileGroup(t *testing.T, filePath string) string {
	t.Helper()

	info, err := os.Stat(filePath)
	require.NoError(t, err, "Failed to stat file")

	stat := info.Sys().(*syscall.Stat_t)

	// Try to get group name, fall back to gid if not found
	cmd := exec.Command("getent", "group", strconv.Itoa(int(stat.Gid)))
	output, err := cmd.Output()
	if err != nil {
		return strconv.Itoa(int(stat.Gid))
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) > 0 {
		return parts[0]
	}

	return strconv.Itoa(int(stat.Gid))
}

// assertFileExists asserts that a file exists
func assertFileExists(t *testing.T, path string, msgAndArgs ...interface{}) {
	t.Helper()

	_, err := os.Stat(path)
	assert.NoError(t, err, msgAndArgs...)
}

// assertFileNotExists asserts that a file does not exist
func assertFileNotExists(t *testing.T, path string, msgAndArgs ...interface{}) {
	t.Helper()

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), msgAndArgs...)
}

// assertDirExists asserts that a directory exists
func assertDirExists(t *testing.T, path string, msgAndArgs ...interface{}) {
	t.Helper()

	info, err := os.Stat(path)
	assert.NoError(t, err, msgAndArgs...)
	assert.True(t, info.IsDir(), msgAndArgs...)
}

// chownRecursive changes ownership of a directory and all its contents
func chownRecursive(path, uidStr, gidStr string) error {
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}

	gid, err := strconv.Atoi(gidStr)
	if err != nil {
		return fmt.Errorf("invalid gid: %w", err)
	}

	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(name, uid, gid)
	})
}

// createBackups creates fake backup files for testing backup rotation
func createBackups(t *testing.T, username string, count int) {
	t.Helper()

	backupDir := getBackupDir(t, username)

	// Ensure backup directory exists
	require.NoError(t, os.MkdirAll(backupDir, 0700))

	u, err := user.Lookup(username)
	require.NoError(t, err)

	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	// Set ownership on backup directory
	require.NoError(t, os.Chown(backupDir, uid, gid))

	// Create backup files with chronological names
	for i := 1; i <= count; i++ {
		backupName := fmt.Sprintf("authorized_keys_2024010%d_12000%d_abcde%d", i, i, i)
		backupPath := filepath.Join(backupDir, backupName)

		content := fmt.Sprintf("ssh-ed25519 BACKUP%d backup%d@old\n", i, i)
		require.NoError(t, os.WriteFile(backupPath, []byte(content), 0600))
		require.NoError(t, os.Chown(backupPath, uid, gid))
	}
}

// removeSSHDir removes the .ssh directory for a user
func removeSSHDir(t *testing.T, username string) {
	t.Helper()

	sshDir := getSSHDir(t, username)
	os.RemoveAll(sshDir)
}
