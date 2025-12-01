// AuthKeySync is a lightweight CLI utility for synchronizing SSH public keys
// from remote URLs into local authorized_keys files.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/eduardolat/authkeysync/internal/config"
	"github.com/eduardolat/authkeysync/internal/sync"
	"github.com/eduardolat/authkeysync/internal/version"
)

// Exit codes
const (
	ExitSuccess = 0
	ExitFailure = 1
)

// ASCII art banner for the CLI
const banner = `
    _         _   _     _  __          ____                   
   / \  _   _| |_| |__ | |/ /___ _   _/ ___| _   _ _ __   ___ 
  / _ \| | | | __| '_ \| ' // _ \ | | \___ \| | | | '_ \ / __|
 / ___ \ |_| | |_| | | | . \  __/ |_| |___) | |_| | | | | (__ 
/_/   \_\__,_|\__|_| |_|_|\_\___|\__, |____/ \__, |_| |_|\___|
                                 |___/       |___/            
`

func main() {
	os.Exit(run())
}

func run() int {
	// Check platform - AuthKeySync only supports Linux and macOS
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		fmt.Fprintf(os.Stderr, "Error: AuthKeySync is only supported on Linux and macOS.\n")
		fmt.Fprintf(os.Stderr, "Current platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(os.Stderr, "Windows is not supported because this tool requires POSIX-specific\n")
		fmt.Fprintf(os.Stderr, "features such as file ownership (chown), Unix permissions, and\n")
		fmt.Fprintf(os.Stderr, "atomic file operations that are not available on Windows.\n")
		return ExitFailure
	}

	// Define CLI flags
	configPath := flag.String("config", config.DefaultConfigPath, "Path to the configuration file")
	dryRun := flag.Bool("dry-run", false, "Simulate sync without modifying files")
	showVersion := flag.Bool("version", false, "Show version information and exit")
	debug := flag.Bool("debug", false, "Enable debug logging (most verbose)")
	quiet := flag.Bool("quiet", false, "Show only warnings and errors (for cron/scheduled tasks)")
	silent := flag.Bool("silent", false, "Show only errors (most quiet)")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, banner)
		fmt.Fprintf(os.Stderr, "\nSSH Public Key Synchronization Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  authkeysync [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nLog Levels:\n")
		fmt.Fprintf(os.Stderr, "  (default)   Show info, warnings, and errors\n")
		fmt.Fprintf(os.Stderr, "  --debug     Show all messages including debug details\n")
		fmt.Fprintf(os.Stderr, "  --quiet     Show only warnings and errors (recommended for cron)\n")
		fmt.Fprintf(os.Stderr, "  --silent    Show only errors\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  authkeysync                           # Use default config path\n")
		fmt.Fprintf(os.Stderr, "  authkeysync --config /path/to/config  # Use custom config\n")
		fmt.Fprintf(os.Stderr, "  authkeysync --dry-run                 # Simulate without changes\n")
		fmt.Fprintf(os.Stderr, "  authkeysync --quiet                   # Run silently for cron jobs\n")
		fmt.Fprintf(os.Stderr, "\nExit Codes:\n")
		fmt.Fprintf(os.Stderr, "  0  Success (all users processed successfully or skipped)\n")
		fmt.Fprintf(os.Stderr, "  1  Failure (at least one user failed to synchronize)\n")
		fmt.Fprintf(os.Stderr, "\nMore info: https://github.com/eduardolat/authkeysync\n")
	}

	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Print(banner)
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("Commit:  %s\n", version.Commit)
		fmt.Printf("Built:   %s\n", version.Date)
		fmt.Println()
		return ExitSuccess
	}

	// Setup logger with hierarchy: debug > default > quiet > silent
	var logLevel slog.Level
	switch {
	case *debug:
		logLevel = slog.LevelDebug // Shows everything (-4)
	case *silent:
		logLevel = slog.LevelError // Only errors (8)
	case *quiet:
		logLevel = slog.LevelWarn // Warnings and errors (4)
	default:
		logLevel = slog.LevelInfo // Normal operation (0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("AuthKeySync starting",
		"version", version.Version,
		"config", *configPath,
		"dry_run", *dryRun)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load configuration",
			"path", *configPath,
			"error", err)
		return ExitFailure
	}

	logger.Info("configuration loaded",
		"users", len(cfg.Users),
		"backup_enabled", cfg.Policy.IsBackupEnabled(),
		"backup_retention", cfg.Policy.GetBackupRetentionCount(),
		"preserve_local_keys", cfg.Policy.IsPreserveLocalKeys())

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Warn("received signal, shutting down",
			"signal", sig.String())
		cancel()
	}()

	// Run synchronization
	syncer := sync.New(cfg, logger, *dryRun)
	result := syncer.Run(ctx)

	// Log summary
	successCount := 0
	skippedCount := 0
	failedCount := 0

	for _, userResult := range result.Users {
		if userResult.Error != nil {
			failedCount++
		} else if userResult.Skipped {
			skippedCount++
		} else {
			successCount++
		}
	}

	// Use appropriate log level for summary based on outcome
	if failedCount > 0 {
		logger.Warn("synchronization complete with failures",
			"success", successCount,
			"skipped", skippedCount,
			"failed", failedCount)
		logger.Error("some users failed to synchronize")
		return ExitFailure
	}

	logger.Info("synchronization complete",
		"success", successCount,
		"skipped", skippedCount,
		"failed", failedCount)
	logger.Info("all users processed successfully")
	return ExitSuccess
}
