// Package keyfetcher provides HTTP fetching of SSH public keys from remote sources.
package keyfetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eduardolat/authkeysync/internal/config"
	"github.com/eduardolat/authkeysync/internal/keyparser"
	"github.com/eduardolat/authkeysync/internal/version"
)

const (
	// MaxResponseSize is the maximum response body size (10MB)
	MaxResponseSize = 10 * 1024 * 1024
)

// FetchResult contains the result of fetching keys from a source
type FetchResult struct {
	// Source is the source configuration
	Source config.Source
	// Keys are the parsed SSH keys
	Keys []keyparser.ParsedKey
	// Error is set if the fetch failed
	Error error
	// StatusCode is the HTTP status code (0 if request failed)
	StatusCode int
	// DiscardedLines is the number of discarded lines during parsing
	DiscardedLines int
}

// Fetcher fetches SSH keys from remote sources
type Fetcher struct {
	client *http.Client
	logger *slog.Logger
}

// New creates a new Fetcher with the default HTTP client and a no-op logger
func New() *Fetcher {
	return &Fetcher{
		client: &http.Client{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// NewWithLogger creates a new Fetcher with the default HTTP client and a logger
func NewWithLogger(logger *slog.Logger) *Fetcher {
	return &Fetcher{
		client: &http.Client{},
		logger: logger,
	}
}

// NewWithClient creates a new Fetcher with a custom HTTP client
func NewWithClient(client *http.Client) *Fetcher {
	return &Fetcher{
		client: client,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// NewWithClientAndLogger creates a new Fetcher with a custom HTTP client and logger
func NewWithClientAndLogger(client *http.Client, logger *slog.Logger) *Fetcher {
	return &Fetcher{
		client: client,
		logger: logger,
	}
}

// Fetch fetches keys from a single source
func (f *Fetcher) Fetch(ctx context.Context, source config.Source) *FetchResult {
	result := &FetchResult{
		Source: source,
	}

	// Create context with timeout
	timeout := time.Duration(source.GetTimeoutSeconds()) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build request
	var bodyReader io.Reader
	if source.Body != "" {
		bodyReader = strings.NewReader(source.Body)
	}

	req, err := http.NewRequestWithContext(ctx, source.GetMethod(), source.URL, bodyReader)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	// Set default User-Agent if not provided
	hasUserAgent := false
	for key := range source.Headers {
		if strings.EqualFold(key, "User-Agent") {
			hasUserAgent = true
			break
		}
	}
	if !hasUserAgent {
		req.Header.Set("User-Agent", version.UserAgent())
	}

	// Set custom headers
	for key, value := range source.Headers {
		req.Header.Set(key, value)
	}

	// Log request details for debugging
	f.logger.Debug("executing HTTP request",
		"url", source.URL,
		"method", source.GetMethod(),
		"user_agent", req.Header.Get("User-Agent"),
		"timeout_seconds", source.GetTimeoutSeconds())

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode

	// Check status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return result
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		return result
	}

	// Parse keys
	parseResult, err := keyparser.ParseString(string(body))
	if err != nil {
		result.Error = fmt.Errorf("failed to parse keys: %w", err)
		return result
	}

	result.Keys = parseResult.Keys
	result.DiscardedLines = parseResult.DiscardedLines

	return result
}

// FetchAll fetches keys from multiple sources for a user.
// If any source fails, the function returns an error and stops processing.
func (f *Fetcher) FetchAll(ctx context.Context, sources []config.Source) ([]*FetchResult, error) {
	results := make([]*FetchResult, 0, len(sources))

	for _, source := range sources {
		result := f.Fetch(ctx, source)
		results = append(results, result)

		// If any source fails, abort for this user
		if result.Error != nil {
			return results, fmt.Errorf("source %s failed: %w", source.URL, result.Error)
		}
	}

	return results, nil
}

// FetcherProvider is an interface for fetching keys
type FetcherProvider interface {
	Fetch(ctx context.Context, source config.Source) *FetchResult
	FetchAll(ctx context.Context, sources []config.Source) ([]*FetchResult, error)
}
