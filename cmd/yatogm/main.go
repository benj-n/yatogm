// YaToGm - Yahoo To Gmail
// Fetches emails from Yahoo mailboxes via POP3S and forwards them to Gmail via SMTP.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/benj-n/yatogm/internal/config"
	"github.com/benj-n/yatogm/internal/state"
	"github.com/benj-n/yatogm/internal/worker"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "/etc/yatogm/config.yml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("yatogm %s\n", version)
		os.Exit(0)
	}

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up structured logging.
	logLevel := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("yatogm starting",
		"version", version,
		"yahoo_mailboxes", len(cfg.Yahoo),
		"gmail", cfg.Gmail.Email,
	)

	// Initialize state tracker.
	tracker, err := state.NewTracker(cfg.StatePath)
	if err != nil {
		logger.Error("failed to initialize state tracker", "error", err)
		os.Exit(1)
	}

	// Run the worker.
	w := worker.New(cfg, tracker, logger)
	if err := w.Run(); err != nil {
		logger.Error("run completed with errors", "error", err)
		os.Exit(1)
	}

	logger.Info("yatogm finished successfully")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
