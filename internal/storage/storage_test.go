package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

func TestCreateAndListNotebooks(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNotebook("work"); err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}
	if err := s.CreateNotebook("personal"); err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}

	names, err := s.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 notebooks, got %d", len(names))
	}
}

func TestCreateNoteAutoCreatesNotebook(t *testing.T) {
	s := newTestStore(t)

	// Create a note in a notebook that doesn't exist yet
	err := s.CreateNote("ideas", "shower-thought", "# Great Idea\n\nThis is brilliant.")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	// Notebook should now exist
	if !s.NotebookExists("ideas") {
		t.Error("expected notebook 'ideas' to be auto-created")
	}

	// Note should exist as a .md file
	path := filepath.Join(s.Root, "ideas", "shower-thought.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected note file to exist at %s", path)
	}
}

func TestGetNote(t *testing.T) {
	s := newTestStore(t)

	content := "# Meeting Notes\n\nDiscussed the roadmap."
	if err := s.CreateNote("work", "meeting", content); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	note, err := s.GetNote("work", "meeting")
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if note.Name != "meeting" {
		t.Errorf("expected Name 'meeting', got %q", note.Name)
	}
	if note.Notebook != "work" {
		t.Errorf("expected Notebook 'work', got %q", note.Notebook)
	}
	if note.Content != content {
		t.Errorf("content mismatch: got %q", note.Content)
	}
	if note.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestUpdateNote(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNote("work", "draft", "initial"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := s.UpdateNote("work", "draft", "updated content"); err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}

	note, err := s.GetNote("work", "draft")
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if note.Content != "updated content" {
		t.Errorf("expected updated content, got %q", note.Content)
	}
}

func TestUpdateNoteNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.UpdateNote("work", "nonexistent", "content")
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
}

func TestDeleteNote(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNote("work", "temp", "temporary"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := s.DeleteNote("work", "temp"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	if s.NoteExists("work", "temp") {
		t.Error("expected note to be deleted")
	}
}

func TestDeleteNotebook(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNote("scratch", "note1", "content1"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := s.CreateNote("scratch", "note2", "content2"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := s.DeleteNotebook("scratch"); err != nil {
		t.Fatalf("DeleteNotebook: %v", err)
	}
	if s.NotebookExists("scratch") {
		t.Error("expected notebook to be deleted")
	}
}

func TestListNotes(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNote("journal", "day1", "Day 1"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := s.CreateNote("journal", "day2", "Day 2"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	notes, err := s.ListNotes("journal")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestNoteCount(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNotebook("empty"); err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}
	count, err := s.NoteCount("empty")
	if err != nil {
		t.Fatalf("NoteCount: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 notes, got %d", count)
	}

	if err := s.CreateNote("empty", "first", "content"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	count, err = s.NoteCount("empty")
	if err != nil {
		t.Fatalf("NoteCount: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 note, got %d", count)
	}
}

func TestNotebookAndNoteExist(t *testing.T) {
	s := newTestStore(t)

	if s.NotebookExists("nope") {
		t.Error("expected notebook to not exist")
	}
	if s.NoteExists("nope", "nope") {
		t.Error("expected note to not exist")
	}

	if err := s.CreateNote("real", "note", "content"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if !s.NotebookExists("real") {
		t.Error("expected notebook to exist")
	}
	if !s.NoteExists("real", "note") {
		t.Error("expected note to exist")
	}
}

func TestListNotesIgnoresNonMarkdown(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateNotebook("mixed"); err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}
	// Create a .md file and a non-.md file
	if err := s.CreateNote("mixed", "real-note", "content"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	nonMd := filepath.Join(s.Root, "mixed", "config.json")
	if err := os.WriteFile(nonMd, []byte("{}"), 0644); err != nil {
		t.Fatalf("write non-md file: %v", err)
	}

	notes, err := s.ListNotes("mixed")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note (ignoring .json), got %d", len(notes))
	}
	if notes[0] != "real-note" {
		t.Errorf("expected 'real-note', got %q", notes[0])
	}
}
