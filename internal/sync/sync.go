// Package sync orchestrates the SSH key synchronization process.
package sync

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/eduardolat/authkeysync/internal/backup"
	"github.com/eduardolat/authkeysync/internal/config"
	"github.com/eduardolat/authkeysync/internal/keyfetcher"
	"github.com/eduardolat/authkeysync/internal/keyparser"
	"github.com/eduardolat/authkeysync/internal/sshfile"
	"github.com/eduardolat/authkeysync/internal/userinfo"
	"github.com/eduardolat/authkeysync/internal/version"
)

// Syncer handles the key synchronization process
type Syncer struct {
	cfg           *config.Config
	logger        *slog.Logger
	fetcher       *keyfetcher.Fetcher
	backupManager *backup.Manager
	fileWriter    *sshfile.Writer
	userLookup    userinfo.LookupProvider
	dryRun        bool
	timeNow       func() time.Time
}

// New creates a new Syncer
func New(cfg *config.Config, logger *slog.Logger, dryRun bool) *Syncer {
	return &Syncer{
		cfg:           cfg,
		logger:        logger,
		fetcher:       keyfetcher.NewWithLogger(logger),
		backupManager: backup.New(),
		fileWriter:    sshfile.New(),
		userLookup:    &userinfo.SystemLookupProvider{},
		dryRun:        dryRun,
		timeNow:       time.Now,
	}
}

// UserResult contains the result of syncing a single user
type UserResult struct {
	Username    string
	Skipped     bool
	SkipReason  string
	Error       error
	KeysWritten int
	LocalKeys   int
	Changed     bool
	BackupPath  string
}

// SyncResult contains the result of the entire sync operation
type SyncResult struct {
	Users     []UserResult
	HasErrors bool
}

// Run executes the synchronization for all configured users.
// Returns a SyncResult containing the outcome for each user.
func (s *Syncer) Run(ctx context.Context) *SyncResult {
	result := &SyncResult{
		Users: make([]UserResult, 0, len(s.cfg.Users)),
	}

	for _, user := range s.cfg.Users {
		userResult := s.syncUser(ctx, user)
		result.Users = append(result.Users, userResult)

		if userResult.Error != nil {
			result.HasErrors = true
		}
	}

	return result
}

// syncUser synchronizes keys for a single user
func (s *Syncer) syncUser(ctx context.Context, user config.User) UserResult {
	start := s.timeNow()
	defer func() {
		s.logger.Debug("user sync finished",
			"username", user.Username,
			"duration_ms", time.Since(start).Milliseconds())
	}()

	result := UserResult{Username: user.Username}

	s.logger.Info("processing user", "username", user.Username)

	// Look up user info
	info, err := s.userLookup.Lookup(user.Username)
	if err != nil {
		if errors.Is(err, userinfo.ErrUserNotFound) {
			s.logger.Warn("skipping user sync: system user lookup failed",
				"username", user.Username,
				"reason", "user does not exist in system")
			result.Skipped = true
			result.SkipReason = "user not found in system"
			return result
		}
		if errors.Is(err, userinfo.ErrSSHDirNotFound) {
			s.logger.Warn("skipping user sync: SSH directory not available",
				"username", user.Username,
				"reason", ".ssh directory does not exist")
			result.Skipped = true
			result.SkipReason = ".ssh directory not found"
			return result
		}
		if errors.Is(err, userinfo.ErrSSHDirNotDir) {
			s.logger.Warn("skipping user sync: SSH directory invalid",
				"username", user.Username,
				"reason", ".ssh exists but is not a directory")
			result.Skipped = true
			result.SkipReason = ".ssh exists but is not a directory"
			return result
		}
		result.Error = fmt.Errorf("failed to lookup user: %w", err)
		s.logger.Error("failed to lookup user",
			"username", user.Username,
			"error", err)
		return result
	}

	// Fetch keys from all sources
	fetchResults, err := s.fetcher.FetchAll(ctx, user.Sources)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch keys: %w", err)
		s.logger.Error("failed to fetch keys, aborting user sync",
			"username", user.Username,
			"error", err)
		return result
	}

	// Log fetch results
	for _, fr := range fetchResults {
		s.logger.Info("fetched keys from source",
			"username", user.Username,
			"url", fr.Source.URL,
			"keys", len(fr.Keys),
			"discarded_lines", fr.DiscardedLines)
	}

	// Build content with deduplication
	content, stats := s.buildContent(info, fetchResults)

	result.KeysWritten = stats.TotalKeys
	result.LocalKeys = stats.LocalKeys

	// Log deduplication info
	for _, dup := range stats.Duplicates {
		s.logger.Info("duplicate key found",
			"username", user.Username,
			"key_fingerprint", keyFingerprint(dup.Key),
			"first_source", dup.FirstSource,
			"duplicate_source", dup.DuplicateSource)
	}

	if s.dryRun {
		s.logger.Info("dry-run: would write authorized_keys",
			"username", user.Username,
			"keys", stats.TotalKeys,
			"local_keys", stats.LocalKeys)
		s.logger.Debug("dry-run: file content",
			"username", user.Username,
			"content", string(content))
		return result
	}

	// Create backup if enabled and content changed
	if s.cfg.Policy.IsBackupEnabled() {
		existingContent, _ := sshfile.ReadContent(info.SSHDir)
		if len(existingContent) > 0 && string(existingContent) != string(content) {
			backupPath, err := s.backupManager.CreateBackup(info.SSHDir, info.UID, info.GID)
			if err != nil {
				result.Error = fmt.Errorf("failed to create backup: %w", err)
				s.logger.Error("failed to create backup",
					"username", user.Username,
					"error", err)
				return result
			}
			if backupPath != "" {
				result.BackupPath = backupPath
				s.logger.Info("created backup",
					"username", user.Username,
					"path", backupPath)
			}

			// Rotate old backups
			deleted, err := s.backupManager.RotateBackups(info.SSHDir, s.cfg.Policy.GetBackupRetentionCount())
			if err != nil {
				s.logger.Warn("failed to rotate backups",
					"username", user.Username,
					"error", err)
			} else if len(deleted) > 0 {
				s.logger.Info("rotated old backups",
					"username", user.Username,
					"deleted_count", len(deleted))
			}
		}
	}

	// Write file atomically
	writeResult, err := s.fileWriter.WriteAtomic(info.SSHDir, content, info.UID, info.GID)
	if err != nil {
		result.Error = fmt.Errorf("failed to write authorized_keys: %w", err)
		s.logger.Error("failed to write authorized_keys",
			"username", user.Username,
			"error", err)
		return result
	}

	result.Changed = writeResult.Changed

	if writeResult.Changed {
		s.logger.Info("updated authorized_keys",
			"username", user.Username,
			"path", writeResult.Path,
			"keys", stats.TotalKeys)
	} else {
		s.logger.Info("authorized_keys unchanged",
			"username", user.Username)
	}

	return result
}

// ContentStats contains statistics about built content
type ContentStats struct {
	TotalKeys  int
	LocalKeys  int
	Duplicates []DuplicateInfo
}

// DuplicateInfo contains information about a duplicate key
type DuplicateInfo struct {
	Key             string
	FirstSource     string
	DuplicateSource string
}

// buildContent builds the authorized_keys file content with proper formatting and deduplication
func (s *Syncer) buildContent(info *userinfo.UserInfo, fetchResults []*keyfetcher.FetchResult) ([]byte, *ContentStats) {
	stats := &ContentStats{
		Duplicates: make([]DuplicateInfo, 0),
	}

	// Track seen keys for deduplication
	// Key: trimmed line, Value: source URL where first seen
	seenKeys := make(map[string]string)

	// Track keys per source
	type sourceKeys struct {
		url  string
		keys []string
	}
	sources := make([]sourceKeys, 0, len(fetchResults)+1)

	// Process remote sources in order
	for _, fr := range fetchResults {
		sk := sourceKeys{url: fr.Source.URL}
		for _, key := range fr.Keys {
			if firstSource, exists := seenKeys[key.Line]; exists {
				stats.Duplicates = append(stats.Duplicates, DuplicateInfo{
					Key:             key.Line,
					FirstSource:     firstSource,
					DuplicateSource: fr.Source.URL,
				})
				continue
			}
			seenKeys[key.Line] = fr.Source.URL
			sk.keys = append(sk.keys, key.Line)
		}
		if len(sk.keys) > 0 {
			sources = append(sources, sk)
		}
	}

	// Process local keys if preserve_local_keys is enabled
	var localKeys []string
	if s.cfg.Policy.IsPreserveLocalKeys() {
		existingContent, err := sshfile.ReadContent(info.SSHDir)
		if err == nil && len(existingContent) > 0 {
			parseResult, err := keyparser.ParseString(string(existingContent))
			if err == nil {
				for _, key := range parseResult.Keys {
					if firstSource, exists := seenKeys[key.Line]; exists {
						stats.Duplicates = append(stats.Duplicates, DuplicateInfo{
							Key:             key.Line,
							FirstSource:     firstSource,
							DuplicateSource: "Local",
						})
						continue
					}
					seenKeys[key.Line] = "Local"
					localKeys = append(localKeys, key.Line)
				}
			}
		}
	}

	// Build the output
	var builder strings.Builder

	// Header
	timestamp := s.timeNow().UTC().Format("2006-01-02T15:04:05Z")
	builder.WriteString("# ──────────────────────────────────────────────────────────────────\n")
	builder.WriteString("# Generated by AuthKeySync\n")
	builder.WriteString(fmt.Sprintf("# Version:   %s\n", version.Version))
	builder.WriteString(fmt.Sprintf("# Commit:    %s\n", version.Commit))
	builder.WriteString(fmt.Sprintf("# Built:     %s\n", version.Date))
	builder.WriteString(fmt.Sprintf("# Last sync: %s\n", timestamp))
	builder.WriteString("# More info: https://github.com/eduardolat/authkeysync\n")
	builder.WriteString("# ──────────────────────────────────────────────────────────────────\n")

	// Remote sources
	for _, src := range sources {
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("# Source: %s\n", src.url))
		for _, key := range src.keys {
			builder.WriteString(key)
			builder.WriteString("\n")
			stats.TotalKeys++
		}
	}

	// Local keys
	if len(localKeys) > 0 {
		builder.WriteString("\n")
		builder.WriteString("# Local (preserved)\n")
		for _, key := range localKeys {
			builder.WriteString(key)
			builder.WriteString("\n")
			stats.TotalKeys++
			stats.LocalKeys++
		}
	}

	return []byte(builder.String()), stats
}

// keyFingerprint computes a SHA256 fingerprint of an SSH key line for visual identification.
// Returns a short fingerprint like "SHA256:a1b2c3d4e5f6a7b8" based on the entire line.
func keyFingerprint(line string) string {
	if line == "" {
		return "(empty)"
	}
	hash := sha256.Sum256([]byte(line))
	return fmt.Sprintf("SHA256:%x", hash[:8])
}
