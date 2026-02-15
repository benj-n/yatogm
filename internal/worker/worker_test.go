package worker

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/benj-n/yatogm/internal/config"
	"github.com/benj-n/yatogm/internal/state"
)

func TestNewWorker(t *testing.T) {
	cfg := &config.Config{
		Gmail: config.GmailConfig{
			Email:       "test@gmail.com",
			AppPassword: "secret",
			SMTPHost:    "smtp.gmail.com",
			SMTPPort:    587,
		},
		Yahoo: []config.YahooMailbox{
			{
				Email:       "test@yahoo.com",
				AppPassword: "yahoo-secret",
				POP3Host:    "pop.mail.yahoo.com",
				POP3Port:    995,
			},
		},
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	}

	tracker, err := state.NewTracker(cfg.StatePath)
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	w := New(cfg, tracker, logger)
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
	if w.cfg != cfg {
		t.Error("worker config mismatch")
	}
	if w.sender == nil {
		t.Error("expected non-nil sender")
	}
}
