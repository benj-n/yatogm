// Package worker orchestrates the fetch-and-forward pipeline for each Yahoo mailbox.
package worker

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/benj-n/yatogm/internal/config"
	"github.com/benj-n/yatogm/internal/pop3"
	smtpsender "github.com/benj-n/yatogm/internal/smtp"
	"github.com/benj-n/yatogm/internal/state"
)

// Worker processes email fetching and forwarding for all configured mailboxes.
type Worker struct {
	cfg     *config.Config
	tracker *state.Tracker
	sender  *smtpsender.Sender
	logger  *slog.Logger
}

// New creates a new Worker.
func New(cfg *config.Config, tracker *state.Tracker, logger *slog.Logger) *Worker {
	sender := smtpsender.NewSender(
		cfg.Gmail.SMTPHost,
		cfg.Gmail.SMTPPort,
		cfg.Gmail.Email,
		cfg.Gmail.AppPassword,
		cfg.Gmail.Email,
	)

	return &Worker{
		cfg:     cfg,
		tracker: tracker,
		sender:  sender,
		logger:  logger,
	}
}

// Run executes one full cycle: fetch from all Yahoo mailboxes and forward to Gmail.
func (w *Worker) Run() error {
	w.logger.Info("starting fetch cycle", "mailboxes", len(w.cfg.Yahoo))

	var totalFetched, totalErrors int

	for i, yahoo := range w.cfg.Yahoo {
		fetched, errors := w.processMailbox(i, yahoo)
		totalFetched += fetched
		totalErrors += errors
	}

	w.logger.Info("fetch cycle complete",
		"total_fetched", totalFetched,
		"total_errors", totalErrors,
	)

	// Log state stats.
	for mailbox, count := range w.tracker.Stats() {
		w.logger.Debug("state", "mailbox", mailbox, "tracked_uids", count)
	}

	if totalErrors > 0 {
		return fmt.Errorf("completed with %d errors", totalErrors)
	}
	return nil
}

// processMailbox fetches and forwards emails from a single Yahoo mailbox.
func (w *Worker) processMailbox(index int, yahoo config.YahooMailbox) (fetched, errors int) {
	log := w.logger.With("mailbox", yahoo.Email, "index", index)
	log.Info("processing mailbox")

	// Connect to POP3 server.
	client, err := pop3.Dial(yahoo.POP3Host, yahoo.POP3Port, 30*time.Second)
	if err != nil {
		log.Error("failed to connect", "error", err)
		return 0, 1
	}
	defer func() {
		if err := client.Quit(); err != nil {
			log.Warn("quit failed", "error", err)
		}
	}()

	// Login.
	if err := client.Login(yahoo.Email, yahoo.AppPassword); err != nil {
		log.Error("login failed", "error", err)
		return 0, 1
	}

	log.Debug("logged in successfully")

	// Get UID list.
	uidMap, err := client.UIDList()
	if err != nil {
		log.Error("UIDL failed", "error", err)
		return 0, 1
	}

	log.Info("found messages", "total", len(uidMap))

	// Sort message numbers for deterministic processing.
	msgNums := make([]int, 0, len(uidMap))
	for num := range uidMap {
		msgNums = append(msgNums, num)
	}
	sort.Ints(msgNums)

	// Process each message.
	for _, msgNum := range msgNums {
		uid := uidMap[msgNum]

		// Skip already-fetched messages.
		if w.tracker.IsFetched(yahoo.Email, uid) {
			log.Debug("skipping already-fetched message", "msg_num", msgNum, "uid", uid)
			continue
		}

		log.Info("fetching message", "msg_num", msgNum, "uid", uid)

		// Retrieve the message.
		rawMsg, err := client.Retrieve(msgNum)
		if err != nil {
			log.Error("retrieve failed", "msg_num", msgNum, "uid", uid, "error", err)
			errors++
			continue
		}

		// Forward to Gmail.
		if err := w.sender.Send(rawMsg, yahoo.Email); err != nil {
			log.Error("forward failed", "msg_num", msgNum, "uid", uid, "error", err)
			errors++
			continue
		}

		// Mark as fetched.
		if err := w.tracker.MarkFetched(yahoo.Email, uid); err != nil {
			log.Error("state update failed", "msg_num", msgNum, "uid", uid, "error", err)
			errors++
			continue
		}

		// Delete from Yahoo server (actual removal happens on QUIT).
		if err := client.Delete(msgNum); err != nil {
			log.Error("delete failed", "msg_num", msgNum, "uid", uid, "error", err)
			errors++
			continue
		}

		fetched++
		log.Info("message forwarded and deleted", "msg_num", msgNum, "uid", uid)
	}

	log.Info("mailbox processing complete", "fetched", fetched, "errors", errors)
	return fetched, errors
}
