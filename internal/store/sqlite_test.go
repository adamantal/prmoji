package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteStore_slotIndexRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	ctx := context.Background()
	const prURL = "https://github.com/a/b/pull/1"
	if err := s.InsertPRMessage(ctx, prURL, "C1", "123.456", 2); err != nil {
		t.Fatal(err)
	}

	msgs, err := s.ListMessagesByPRURL(ctx, prURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message got %d", len(msgs))
	}
	if msgs[0].SlotIndex != 2 {
		t.Fatalf("slot_index: got %d want 2", msgs[0].SlotIndex)
	}
}

func TestSQLiteStore_insertRejectsInvalidSlot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	err = s.InsertPRMessage(context.Background(), "https://github.com/a/b/pull/1", "C1", "1.0", 9)
	if err == nil {
		t.Fatal("expected error for slot_index 9")
	}
}
