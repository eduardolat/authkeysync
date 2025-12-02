// Package e2e contains end-to-end tests for AuthKeySync.
// These tests verify the complete functionality of the application
// in a realistic environment with real system users and file operations.
package e2e

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
)

// Test constants
const (
	testUser1     = "e2euser1"
	testUser2     = "e2euser2"
	testUser3     = "e2euser3" // User without .ssh directory
	testUser4     = "e2euser4"
	binaryName    = "authkeysync"
	configTempDir = "/tmp"
)

// testEnv holds the test environment configuration
type testEnv struct {
	binaryPath    string
	mockServerURL string
}

var env *testEnv

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Check if running as root
	if os.Geteuid() != 0 {
		log.Fatal("E2E tests must be run as root (required for user management)")
	}

	// Build the binary
	binaryPath, err := buildBinary()
	if err != nil {
		log.Fatalf("Failed to build binary: %v", err)
	}

	// Start mock server
	mockServer := startMockServer()

	// Store environment
	env = &testEnv{
		binaryPath:    binaryPath,
		mockServerURL: mockServer.URL,
	}

	// Create test users
	if err := createTestUsers(); err != nil {
		mockServer.Close()
		log.Fatalf("Failed to create test users: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	mockServer.Close()
	cleanupTestUsers()

	os.Exit(code)
}

// buildBinary compiles the authkeysync binary for testing
func buildBinary() (string, error) {
	projectRoot := getProjectRoot()
	outputPath := filepath.Join(projectRoot, "dist", binaryName)

	cmd := exec.Command("go", "build",
		"-ldflags", "-s -w -X github.com/eduardolat/authkeysync/internal/version.Version=e2e-test",
		"-o", outputPath,
		"./cmd",
	)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}

	return outputPath, nil
}

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	// Navigate from e2e directory to project root
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// If we're in the e2e directory, go up one level
	if filepath.Base(wd) == "e2e" {
		return filepath.Dir(wd)
	}

	// Check if we're in the project root
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}

	log.Fatalf("Could not determine project root from: %s", wd)
	return ""
}

// createTestUsers creates the test users needed for e2e tests
func createTestUsers() error {
	// e2euser1 - normal user with .ssh directory
	if err := createUser(testUser1, true); err != nil {
		return fmt.Errorf("failed to create %s: %w", testUser1, err)
	}

	// e2euser2 - normal user with .ssh directory
	if err := createUser(testUser2, true); err != nil {
		return fmt.Errorf("failed to create %s: %w", testUser2, err)
	}

	// e2euser3 - user WITHOUT .ssh directory (for skip tests)
	if err := createUser(testUser3, false); err != nil {
		return fmt.Errorf("failed to create %s: %w", testUser3, err)
	}

	// e2euser4 - normal user with .ssh directory
	if err := createUser(testUser4, true); err != nil {
		return fmt.Errorf("failed to create %s: %w", testUser4, err)
	}

	return nil
}

// createUser creates a system user for testing
func createUser(username string, createSSHDir bool) error {
	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		// User exists, ensure proper setup
		if createSSHDir {
			return setupUserSSH(username)
		}
		return nil
	}

	// Create user with home directory
	cmd := exec.Command("useradd", "-m", "-s", "/bin/sh", username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %v\nOutput: %s", err, output)
	}

	// Create .ssh directory if requested
	if createSSHDir {
		return setupUserSSH(username)
	}

	return nil
}

// cleanupTestUsers removes test users and their home directories
func cleanupTestUsers() {
	for _, username := range []string{testUser1, testUser2, testUser3, testUser4} {
		if _, err := user.Lookup(username); err == nil {
			cmd := exec.Command("userdel", "-rf", username)
			_ = cmd.Run() // Ignore errors during cleanup
		}
	}
}

// setupUserSSH creates the .ssh directory for a user with proper permissions
func setupUserSSH(username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("user lookup failed: %w", err)
	}

	sshDir := filepath.Join(u.HomeDir, ".ssh")

	// Create .ssh directory
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}

	// Change ownership
	if err := chownRecursive(sshDir, u.Uid, u.Gid); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}

	return nil
}

// resetUserSSH resets the user's .ssh directory to a clean state
func resetUserSSH(username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("user lookup failed: %w", err)
	}

	sshDir := filepath.Join(u.HomeDir, ".ssh")

	// Remove authorized_keys if it exists
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	_ = os.Remove(authKeysPath)

	// Remove backups directory if it exists
	backupDir := filepath.Join(sshDir, "authorized_keys_backups")
	_ = os.RemoveAll(backupDir)

	// Ensure .ssh directory exists with proper permissions
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}

	// Change ownership
	if err := chownRecursive(sshDir, u.Uid, u.Gid); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}

	return nil
}

// testCase represents a single e2e test case
type testCase struct {
	t *testing.T
}

// newTestCase creates a new test case with the required setup
func newTestCase(t *testing.T, users ...string) *testCase {
	t.Helper()

	// Reset environment for users
	for _, username := range users {
		if err := resetUserSSH(username); err != nil {
			t.Fatalf("Failed to reset user %s: %v", username, err)
		}
	}

	return &testCase{t: t}
}
