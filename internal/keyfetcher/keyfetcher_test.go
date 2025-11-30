package keyfetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardolat/authkeysync/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_Success(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ another@host`))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{
		URL: server.URL,
	}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	require.Len(t, result.Keys, 2)
	assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit user@host", result.Keys[0].Line)
}

func TestFetch_NonOKStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "404 Not Found", statusCode: http.StatusNotFound},
		{name: "500 Server Error", statusCode: http.StatusInternalServerError},
		{name: "401 Unauthorized", statusCode: http.StatusUnauthorized},
		{name: "403 Forbidden", statusCode: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			fetcher := New()
			source := config.Source{URL: server.URL}

			result := fetcher.Fetch(context.Background(), source)

			require.Error(t, result.Error)
			assert.Equal(t, tt.statusCode, result.StatusCode)
			assert.Contains(t, result.Error.Error(), "unexpected status code")
		})
	}
}

func TestFetch_WithHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key"))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{
		URL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "custom-value",
		},
	}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Equal(t, "Bearer token123", receivedHeaders.Get("Authorization"))
	assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom"))
}

func TestFetch_DefaultUserAgent(t *testing.T) {
	var receivedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key"))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{URL: server.URL}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Equal(t, DefaultUserAgent, receivedUserAgent)
}

func TestFetch_CustomUserAgent(t *testing.T) {
	var receivedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key"))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{
		URL: server.URL,
		Headers: map[string]string{
			"User-Agent": "MyCustomAgent/1.0",
		},
	}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Equal(t, "MyCustomAgent/1.0", receivedUserAgent)
}

func TestFetch_POSTWithBody(t *testing.T) {
	var receivedMethod string
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key"))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{
		URL:    server.URL,
		Method: "POST",
		Body:   `{"role": "admin"}`,
	}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, `{"role": "admin"}`, receivedBody)
}

func TestFetch_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := New()
	timeout := 1
	source := config.Source{
		URL:            server.URL,
		TimeoutSeconds: &timeout,
	}

	result := fetcher.Fetch(context.Background(), source)

	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "context deadline exceeded")
}

func TestFetch_InvalidURL(t *testing.T) {
	fetcher := New()
	source := config.Source{
		URL: "http://invalid.local.nonexistent:12345",
	}

	result := fetcher.Fetch(context.Background(), source)

	require.Error(t, result.Error)
}

func TestFetch_HTMLErrorPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>Error</body></html>`))
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{URL: server.URL}

	result := fetcher.Fetch(context.Background(), source)

	require.NoError(t, result.Error)
	assert.Len(t, result.Keys, 0)
	assert.Greater(t, result.DiscardedLines, 0)
}

func TestFetchAll_AllSuccess(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-rsa BBBB key2"))
	}))
	defer server2.Close()

	fetcher := New()
	sources := []config.Source{
		{URL: server1.URL},
		{URL: server2.URL},
	}

	results, err := fetcher.FetchAll(context.Background(), sources)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Len(t, results[0].Keys, 1)
	assert.Len(t, results[1].Keys, 1)
}

func TestFetchAll_FirstSourceFails(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key"))
	}))
	defer server2.Close()

	fetcher := New()
	sources := []config.Source{
		{URL: server1.URL},
		{URL: server2.URL},
	}

	results, err := fetcher.FetchAll(context.Background(), sources)

	require.Error(t, err)
	// Should have only tried the first source before failing
	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

func TestFetchAll_SecondSourceFails(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAA key1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	fetcher := New()
	sources := []config.Source{
		{URL: server1.URL},
		{URL: server2.URL},
	}

	results, err := fetcher.FetchAll(context.Background(), sources)

	require.Error(t, err)
	require.Len(t, results, 2)
	assert.NoError(t, results[0].Error)
	assert.Error(t, results[1].Error)
}

func TestFetchAll_EmptySources(t *testing.T) {
	fetcher := New()
	results, err := fetcher.FetchAll(context.Background(), []config.Source{})

	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestNewWithClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	fetcher := NewWithClient(customClient)
	assert.NotNil(t, fetcher)
	assert.Equal(t, customClient, fetcher.client)
}

func TestFetch_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := New()
	source := config.Source{URL: server.URL}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := fetcher.Fetch(ctx, source)

	require.Error(t, result.Error)
}
