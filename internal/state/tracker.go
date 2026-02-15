// Package state tracks which email UIDs have been fetched to avoid duplicates.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Tracker persists the set of fetched email UIDs per mailbox.
type Tracker struct {
	mu       sync.Mutex
	filePath string
	data     StateData
}

// StateData holds the fetched UIDs per mailbox (keyed by email address).
type StateData struct {
	Mailboxes map[string]*MailboxState `json:"mailboxes"`
}

// MailboxState holds the state for a single mailbox.
type MailboxState struct {
	FetchedUIDs map[string]bool `json:"fetched_uids"`
}

// NewTracker creates a new Tracker, loading existing state from disk if available.
func NewTracker(filePath string) (*Tracker, error) {
	t := &Tracker{
		filePath: filePath,
		data: StateData{
			Mailboxes: make(map[string]*MailboxState),
		},
	}

	if err := t.load(); err != nil {
		// If file doesn't exist, start fresh.
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading state from %s: %w", filePath, err)
		}
	}

	return t, nil
}

// IsFetched returns true if the given UID has been fetched for the given mailbox.
func (t *Tracker) IsFetched(mailbox, uid string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	ms, ok := t.data.Mailboxes[mailbox]
	if !ok {
		return false
	}
	return ms.FetchedUIDs[uid]
}

// MarkFetched marks the given UID as fetched for the given mailbox and persists to disk.
func (t *Tracker) MarkFetched(mailbox, uid string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	ms, ok := t.data.Mailboxes[mailbox]
	if !ok {
		ms = &MailboxState{
			FetchedUIDs: make(map[string]bool),
		}
		t.data.Mailboxes[mailbox] = ms
	}

	ms.FetchedUIDs[uid] = true

	return t.save()
}

// MarkBatchFetched marks multiple UIDs as fetched and persists once.
func (t *Tracker) MarkBatchFetched(mailbox string, uids []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	ms, ok := t.data.Mailboxes[mailbox]
	if !ok {
		ms = &MailboxState{
			FetchedUIDs: make(map[string]bool),
		}
		t.data.Mailboxes[mailbox] = ms
	}

	for _, uid := range uids {
		ms.FetchedUIDs[uid] = true
	}

	return t.save()
}

// Stats returns the number of tracked UIDs per mailbox.
func (t *Tracker) Stats() map[string]int {
	t.mu.Lock()
	defer t.mu.Unlock()

	stats := make(map[string]int)
	for k, v := range t.data.Mailboxes {
		stats[k] = len(v.FetchedUIDs)
	}
	return stats
}

// load reads the state from disk.
func (t *Tracker) load() error {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return err
	}

	var sd StateData
	if err := json.Unmarshal(data, &sd); err != nil {
		// If state file is corrupted, log and start fresh.
		return nil
	}

	if sd.Mailboxes != nil {
		t.data = sd
	}

	return nil
}

// save writes the state to disk atomically using a temp file + rename.
func (t *Tracker) save() error {
	dir := filepath.Dir(t.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(t.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Atomic write: write to temp file, then rename.
	tmpFile := t.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tmpFile, t.filePath); err != nil {
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}
