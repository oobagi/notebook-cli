package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oobagi/notebook-cli/internal/block"
)

func TestEnsureWelcomeFirstLaunch(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.EnsureWelcome(); err != nil {
		t.Fatalf("EnsureWelcome: %v", err)
	}

	// Welcome note should exist.
	note, err := store.GetNote("getting-started", "welcome")
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if note.Content != welcomeContent {
		t.Error("welcome note content does not match expected content")
	}

	// Marker file should exist.
	marker := filepath.Join(store.Root, markerFileName)
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker file missing: %v", err)
	}
}

func TestEnsureWelcomeSubsequentLaunch(t *testing.T) {
	store := NewStore(t.TempDir())

	// First launch — seeds the note.
	if err := store.EnsureWelcome(); err != nil {
		t.Fatalf("first EnsureWelcome: %v", err)
	}

	// Record notebook count after first launch.
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}
	countBefore := len(notebooks)

	// Second launch — should be a no-op.
	if err := store.EnsureWelcome(); err != nil {
		t.Fatalf("second EnsureWelcome: %v", err)
	}

	notebooks, err = store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks after second call: %v", err)
	}
	if len(notebooks) != countBefore {
		t.Errorf("notebook count changed: got %d, want %d", len(notebooks), countBefore)
	}
}

func TestEnsureWelcomeDeletedNoteDoesNotReappear(t *testing.T) {
	store := NewStore(t.TempDir())

	// First launch seeds the welcome note.
	if err := store.EnsureWelcome(); err != nil {
		t.Fatalf("EnsureWelcome: %v", err)
	}

	// User deletes the welcome note and the notebook.
	if err := store.DeleteNotebook("getting-started"); err != nil {
		t.Fatalf("DeleteNotebook: %v", err)
	}

	// Subsequent launch should NOT recreate it.
	if err := store.EnsureWelcome(); err != nil {
		t.Fatalf("EnsureWelcome after delete: %v", err)
	}

	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}
	for _, nb := range notebooks {
		if nb == "getting-started" {
			t.Error("getting-started notebook reappeared after deletion")
		}
	}
}

func TestWelcomeNoteRoundTrips(t *testing.T) {
	blocks := block.Parse(welcomeContent)
	serialized := block.Serialize(blocks)

	if serialized != welcomeContent {
		t.Errorf("welcome content does not round-trip through Parse/Serialize\n--- got ---\n%s\n--- want ---\n%s", serialized, welcomeContent)
	}
}
