# AuthKeySync End-to-End Tests

This directory contains end-to-end (e2e) tests for AuthKeySync. These tests verify the complete functionality of the application in a realistic environment with real system users and file operations.

## Quick Start

This commands should be executed in the root of the project.

```bash
# Run all tests (including e2e) via CI pipeline
task ci

# Run only e2e tests
task e2e
```

## Architecture

The e2e tests are written in Go using the standard `testing` package with `testify` for assertions. They run as a single test binary that:

1. Builds the authkeysync binary
2. Creates temporary test users (e2euser1-4)
3. Starts an in-process mock HTTP server
4. Runs all test cases
5. Cleans up (removes test users)

```
e2e/
├── e2e_test.go                  # Test setup and TestMain
├── helpers_test.go              # Common test utilities
├── mock_server_test.go          # HTTP mock server implementation
├── test_00_description_test.go  # Test cases files
```

## Test Users

The tests create the following system users:

- `e2euser1` - Normal user with `.ssh` directory
- `e2euser2` - Normal user with `.ssh` directory
- `e2euser3` - User WITHOUT `.ssh` directory (for skip tests)
- `e2euser4` - User with `.ssh` directory

## Mock Server Endpoints

The mock server provides the following endpoints:

| Endpoint                | Description                            |
| ----------------------- | -------------------------------------- |
| `/users/alice.keys`     | Returns 2 SSH keys                     |
| `/users/bob.keys`       | Returns 1 SSH key                      |
| `/users/charlie.keys`   | Returns 2 SSH keys                     |
| `/users/duplicate.keys` | Returns keys with duplicates           |
| `/users/empty.keys`     | Returns empty response                 |
| `/users/multiline.keys` | Returns response with comments         |
| `/error/500`            | Returns HTTP 500 error                 |
| `/error/404`            | Returns HTTP 404 error                 |
| `/error/timeout`        | Delays response for 30 seconds         |
| `/error/html`           | Returns HTML error page                |
| `/error/json`           | Returns JSON error response            |
| `/api/keys`             | POST endpoint requiring authentication |
| `/health`               | Health check endpoint                  |

## Writing New Tests

Create a new test file following the naming convention `test_XX_description_test.go`:

```go
package e2e

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewFeature(t *testing.T) {
    // Setup test users
    _ = newTestCase(t, testUser1)
    
    // Create config
    configPath := createConfig(t, "new_feature", `
policy:
  backup_enabled: false

users:
  - username: `+testUser1+`
    sources:
      - url: ${MOCK_SERVER_URL}/users/alice.keys
`)
    
    // Execute
    output, exitCode := runAuthKeySync(t, "--config", configPath)
    t.Logf("Output: %s", output)
    
    // Assert
    assert.Equal(t, 0, exitCode)
    
    content := readAuthKeys(t, testUser1)
    assert.Contains(t, content, "alice@laptop")
}
```

## Helper Functions

```go
// Test execution
runAuthKeySync(t, args...) (output string, exitCode int)

// Config management
createConfig(t, name, content string) string

// User SSH management
createExistingKeys(t, username, content string)
getAuthKeysPath(t, username string) string
getBackupDir(t, username string) string
getSSHDir(t, username string) string
readAuthKeys(t, username string) string

// File assertions
assertFileExists(t, path string, msgAndArgs...)
assertFileNotExists(t, path string, msgAndArgs...)
assertDirExists(t, path string, msgAndArgs...)

// File metadata
getFilePermissions(t, path string) string
getFileOwner(t, path string) string
getFileGroup(t, path string) string

// Counting
countKeys(t, filepath string) int
countBackups(t, backupDir string) int

// Backup management
createBackups(t, username string, count int)
```

## Requirements

This requirements are met if you are running the tests in the project's DevContainer.

- Root access
- Golang installed
- Linux environment
