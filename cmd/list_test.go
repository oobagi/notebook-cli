package cmd

import (
	"strings"
	"testing"

	"github.com/oobagi/notebook/internal/storage"
)

// --- notebook list (top-level) ---

func TestListNotebooksWithCountsAndTimestamps(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("Personal")
	_ = st.CreateNote("Personal", "note1", "content one")
	_ = st.CreateNote("Personal", "note2", "content two")
	_ = st.CreateNote("Personal", "note3", "content three")
	_ = st.CreateNote("Personal", "note4", "content four")

	_ = st.CreateNotebook("Work")
	_ = st.CreateNote("Work", "todo", "stuff")

	out, err := executeCapture([]string{"--dir", dir, "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain notebook names.
	if !strings.Contains(out, "Personal") {
		t.Errorf("expected 'Personal' in output, got %q", out)
	}
	if !strings.Contains(out, "Work") {
		t.Errorf("expected 'Work' in output, got %q", out)
	}

	// Should contain note counts.
	if !strings.Contains(out, "4 notes") {
		t.Errorf("expected '4 notes' in output, got %q", out)
	}
	if !strings.Contains(out, "1 note") {
		t.Errorf("expected '1 note' (singular) in output, got %q", out)
	}

	// Should contain relative timestamps.
	if !strings.Contains(out, "just now") {
		t.Errorf("expected 'just now' in output, got %q", out)
	}

	// Each line should start with two-space indent.
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, "  ") {
			t.Errorf("line missing two-space indent: %q", line)
		}
	}
}

func TestListNotebooksEmpty(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No books yet.") {
		t.Errorf("expected empty state message, got %q", out)
	}
	if !strings.Contains(out, "notebook new") {
		t.Errorf("expected onboarding hint, got %q", out)
	}
}

func TestListNotebooksEmptyNotebookShowsEmpty(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("EmptyBook")

	out, err := executeCapture([]string{"--dir", dir, "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "EmptyBook") {
		t.Errorf("expected 'EmptyBook' in output, got %q", out)
	}
	if !strings.Contains(out, "0 notes") {
		t.Errorf("expected '0 notes' in output, got %q", out)
	}
	if !strings.Contains(out, "empty") {
		t.Errorf("expected 'empty' for timestamp of empty notebook, got %q", out)
	}
}

// --- notebook <book> list (notes in a book) ---

func TestListNotesWithSizesAndTimestamps(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "Meeting Notes", "# Meeting\nDiscussed the quarterly plan.")
	_ = st.CreateNote("work", "Project Roadmap", strings.Repeat("x", 8192))
	_ = st.CreateNote("work", "Weekly Standup", "short")

	out, err := executeCapture([]string{"--dir", dir, "work", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain note names.
	if !strings.Contains(out, "Meeting Notes") {
		t.Errorf("expected 'Meeting Notes' in output, got %q", out)
	}
	if !strings.Contains(out, "Project Roadmap") {
		t.Errorf("expected 'Project Roadmap' in output, got %q", out)
	}
	if !strings.Contains(out, "Weekly Standup") {
		t.Errorf("expected 'Weekly Standup' in output, got %q", out)
	}

	// Should contain sizes.
	if !strings.Contains(out, "KB") && !strings.Contains(out, "B") {
		t.Errorf("expected size units in output, got %q", out)
	}

	// Should contain relative timestamps.
	if !strings.Contains(out, "just now") {
		t.Errorf("expected 'just now' in output, got %q", out)
	}

	// Each line should start with two-space indent.
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, "  ") {
			t.Errorf("line missing two-space indent: %q", line)
		}
	}
}

func TestListNotesEmptyBook(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("empty")

	out, err := executeCapture([]string{"--dir", dir, "empty", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No notes yet.") {
		t.Errorf("expected empty state message, got %q", out)
	}
	if !strings.Contains(out, "notebook empty new") {
		t.Errorf("expected onboarding hint with book name, got %q", out)
	}
}

func TestListNotesNonExistentBook(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "ghost", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "doesn't exist") {
		t.Errorf("expected 'doesn't exist' in output, got %q", out)
	}
}

func TestListNotesBareBookCommand(t *testing.T) {
	// notebook <book> with no verb should list notes in that book.
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("ideas", "spark", "an idea")

	out, err := executeCapture([]string{"--dir", dir, "ideas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "spark") {
		t.Errorf("expected 'spark' in output, got %q", out)
	}
}

// --- storage: NoteCount and NotebookModTime ---

func TestNoteCount(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)

	// Empty notebook.
	_ = st.CreateNotebook("empty")
	count, err := st.NoteCount("empty")
	if err != nil {
		t.Fatalf("NoteCount: %v", err)
	}
	if count != 0 {
		t.Errorf("NoteCount = %d, want 0", count)
	}

	// Notebook with notes.
	_ = st.CreateNote("full", "a", "x")
	_ = st.CreateNote("full", "b", "y")
	_ = st.CreateNote("full", "c", "z")
	count, err = st.NoteCount("full")
	if err != nil {
		t.Fatalf("NoteCount: %v", err)
	}
	if count != 3 {
		t.Errorf("NoteCount = %d, want 3", count)
	}

	// Non-existent notebook.
	count, err = st.NoteCount("nonexistent")
	if err != nil {
		t.Fatalf("NoteCount for nonexistent: %v", err)
	}
	if count != 0 {
		t.Errorf("NoteCount for nonexistent = %d, want 0", count)
	}
}

func TestNotebookModTime(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)

	// Empty notebook returns zero time.
	_ = st.CreateNotebook("empty")
	mt, err := st.NotebookModTime("empty")
	if err != nil {
		t.Fatalf("NotebookModTime: %v", err)
	}
	if !mt.IsZero() {
		t.Errorf("expected zero time for empty notebook, got %v", mt)
	}

	// Notebook with notes returns a non-zero time.
	_ = st.CreateNote("active", "n1", "content")
	mt, err = st.NotebookModTime("active")
	if err != nil {
		t.Fatalf("NotebookModTime: %v", err)
	}
	if mt.IsZero() {
		t.Error("expected non-zero time for notebook with notes")
	}

	// Non-existent notebook returns zero time.
	mt, err = st.NotebookModTime("nonexistent")
	if err != nil {
		t.Fatalf("NotebookModTime for nonexistent: %v", err)
	}
	if !mt.IsZero() {
		t.Errorf("expected zero time for nonexistent notebook, got %v", mt)
	}
}
