package e2e

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// Mock SSH keys for testing
const (
	aliceKey1     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAlice111111111111111111111111111111111111111 alice@laptop"
	aliceKey2     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQAlice222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222 alice@workstation"
	bobKey        = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBob1111111111111111111111111111111111111111111 bob@desktop"
	charlieKey1   = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICharlie11111111111111111111111111111111111111 charlie@server"
	charlieKey2   = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICharlie22222222222222222222222222222222222222 charlie@laptop"
	duplicateKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDuplicate111111111111111111111111111111111111 unique@duplicate"
	multilineKey1 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMulti1111111111111111111111111111111111111111 multi@host1"
	multilineKey2 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMulti2222222222222222222222222222222222222222 multi@host2"
	apiKey        = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAPIKEY1111111111111111111111111111111111111 api@authenticated"
)

// startMockServer starts a mock HTTP server for testing
func startMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// User keys endpoints
	mux.HandleFunc("/users/alice.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(aliceKey1 + "\n" + aliceKey2))
	})

	mux.HandleFunc("/users/bob.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bobKey))
	})

	mux.HandleFunc("/users/charlie.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(charlieKey1 + "\n" + charlieKey2))
	})

	// Duplicate keys endpoint (contains alice's first key for deduplication testing)
	mux.HandleFunc("/users/duplicate.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(aliceKey1 + "\n" + duplicateKey))
	})

	// Empty keys endpoint
	mux.HandleFunc("/users/empty.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	})

	// Multiline keys with comments
	mux.HandleFunc("/users/multiline.keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		content := `# This is a comment
` + multilineKey1 + `

# Another comment

` + multilineKey2 + `
`
		w.Write([]byte(content))
	})

	// Error endpoints
	mux.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	mux.HandleFunc("/error/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	mux.HandleFunc("/error/timeout", func(w http.ResponseWriter, r *http.Request) {
		// Wait for client to cancel or 30 seconds, whichever comes first
		// This prevents the handler from keeping connections open after client timeout
		select {
		case <-r.Context().Done():
			// Client cancelled the request (timeout)
			return
		case <-time.After(30 * time.Second):
			// Slow server response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ssh-ed25519 AAAA timeout@host"))
		}
	})

	// HTML error page (should be filtered out)
	mux.HandleFunc("/error/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Error</title></head>
<body>
<h1>404 Not Found</h1>
<p>The requested resource was not found.</p>
</body>
</html>`))
	})

	// JSON error response (should be filtered out)
	mux.HandleFunc("/error/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error": "unauthorized", "message": "Invalid API key"}`))
	})

	// POST endpoint with authentication
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		// Check Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Content-Type must be application/json"))
			return
		}

		// Read and verify body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Cannot read body"))
			return
		}

		// Check if body contains expected content
		if !strings.Contains(string(body), `"role"`) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Missing role in body"))
			return
		}

		// Return keys for authenticated request
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(apiKey))
	})

	return httptest.NewServer(mux)
}
