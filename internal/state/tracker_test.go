package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTrackerFresh(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.IsFetched("user@yahoo.com", "uid1") {
		t.Error("expected uid1 to not be fetched")
	}
}

func TestMarkFetchedAndPersist(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := tracker.MarkFetched("user@yahoo.com", "uid1"); err != nil {
		t.Fatalf("MarkFetched failed: %v", err)
	}

	if !tracker.IsFetched("user@yahoo.com", "uid1") {
		t.Error("expected uid1 to be fetched")
	}
	if tracker.IsFetched("user@yahoo.com", "uid2") {
		t.Error("expected uid2 to not be fetched")
	}

	// Verify persistence: create a new tracker from the same file.
	tracker2, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error loading: %v", err)
	}

	if !tracker2.IsFetched("user@yahoo.com", "uid1") {
		t.Error("expected uid1 persisted across reloads")
	}
}

func TestMarkBatchFetched(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uids := []string{"uid1", "uid2", "uid3"}
	if err := tracker.MarkBatchFetched("user@yahoo.com", uids); err != nil {
		t.Fatalf("MarkBatchFetched failed: %v", err)
	}

	for _, uid := range uids {
		if !tracker.IsFetched("user@yahoo.com", uid) {
			t.Errorf("expected %s to be fetched", uid)
		}
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = tracker.MarkFetched("a@yahoo.com", "uid1")
	_ = tracker.MarkFetched("a@yahoo.com", "uid2")
	_ = tracker.MarkFetched("b@yahoo.com", "uid1")

	stats := tracker.Stats()
	if stats["a@yahoo.com"] != 2 {
		t.Errorf("expected 2 UIDs for a@yahoo.com, got %d", stats["a@yahoo.com"])
	}
	if stats["b@yahoo.com"] != 1 {
		t.Errorf("expected 1 UID for b@yahoo.com, got %d", stats["b@yahoo.com"])
	}
}

func TestCorruptedStateFile(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	// Write corrupted JSON.
	if err := os.WriteFile(stateFile, []byte("not json{{{"), 0600); err != nil {
		t.Fatal(err)
	}

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("should handle corrupted state gracefully, got: %v", err)
	}

	// Should start fresh.
	if tracker.IsFetched("user@yahoo.com", "uid1") {
		t.Error("expected fresh state after corruption")
	}
}

func TestMultipleMailboxes(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	tracker, err := NewTracker(stateFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = tracker.MarkFetched("a@yahoo.com", "uid1")
	_ = tracker.MarkFetched("b@yahoo.com", "uid1")

	if !tracker.IsFetched("a@yahoo.com", "uid1") {
		t.Error("expected uid1 fetched for a@yahoo.com")
	}
	if !tracker.IsFetched("b@yahoo.com", "uid1") {
		t.Error("expected uid1 fetched for b@yahoo.com")
	}
	// uid1 for a should not affect other UIDs for a.
	if tracker.IsFetched("a@yahoo.com", "uid2") {
		t.Error("expected uid2 not fetched for a@yahoo.com")
	}
}
