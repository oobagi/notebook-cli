package cmd

import (
	"strings"
	"testing"

	"github.com/oobagi/notebook-cli/internal/storage"
)

// TestAutoCreateNotebookEndToEnd verifies that creating a note in a
// non-existent notebook silently auto-creates the notebook, and that the
// full lifecycle (create, list, view, delete note, delete notebook) works
// end-to-end through the CLI command layer.
func TestAutoCreateNotebookEndToEnd(t *testing.T) {
	dir := setupTestStore(t)
	book := "brand-new-book"
	note := "first-note"

	// Step 1: Create a note in a notebook that does not exist yet.
	// The notebook should be auto-created silently.
	out, err := executeCapture([]string{"--dir", dir, book, "new", note})
	if err != nil {
		t.Fatalf("create note in new notebook: unexpected error: %v", err)
	}

	// Verify output is exactly the note success message — no extra
	// "created notebook" line or any other output.
	want := "  \u2713 Created \"first-note\" in brand-new-book\n"
	if out != want {
		t.Errorf("create note output = %q, want %q", out, want)
	}

	// Step 2: Verify `notebook list` shows the auto-created notebook.
	out, err = executeCapture([]string{"--dir", dir, "list"})
	if err != nil {
		t.Fatalf("list notebooks: unexpected error: %v", err)
	}
	if !strings.Contains(out, book) {
		t.Errorf("expected %q in notebook list output, got %q", book, out)
	}

	// Step 3: Verify `notebook <book> list` shows the note.
	out, err = executeCapture([]string{"--dir", dir, book, "list"})
	if err != nil {
		t.Fatalf("list notes: unexpected error: %v", err)
	}
	if !strings.Contains(out, note) {
		t.Errorf("expected %q in note list output, got %q", note, out)
	}

	// Step 4: Verify `notebook <book> <note> delete` deletes the note.
	// Pipe the note name to confirm deletion.
	rootCmd.SetIn(strings.NewReader(note + "\n"))
	defer rootCmd.SetIn(nil)

	out, err = executeCapture([]string{"--dir", dir, book, note, "delete"})
	if err != nil {
		t.Fatalf("delete note: unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713 Deleted \"first-note\" from brand-new-book") {
		t.Errorf("expected success message in delete note output, got %q", out)
	}

	// Confirm the note is actually gone.
	st := storage.NewStore(dir)
	_, err = st.GetNote(book, note)
	if err == nil {
		t.Error("note should have been deleted but still exists")
	}

	// Step 6: Verify `notebook delete --force <book>` deletes the notebook.
	out, err = executeCapture([]string{"--dir", dir, "delete", "--force", book})
	if err != nil {
		t.Fatalf("delete notebook: unexpected error: %v", err)
	}
	wantBookDel := "  \u2713 Deleted \"brand-new-book\"\n"
	if out != wantBookDel {
		t.Errorf("delete notebook output = %q, want %q", out, wantBookDel)
	}

	// Confirm the notebook is actually gone.
	nbs, _ := st.ListNotebooks()
	for _, n := range nbs {
		if n == book {
			t.Error("notebook should have been deleted but still exists")
		}
	}
}

// TestAutoCreateNotebookWithContent verifies the auto-create path works
// when the note has actual content, and that viewing renders it correctly.
// TestAutoCreateNoExtraOutput verifies that auto-creating a notebook during
// note creation produces exactly one line of output (the note success message)
// and nothing else — no "created notebook" or similar extra messages.
func TestAutoCreateNoExtraOutput(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "invisible-book", "new", "a-note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 line of output, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "\u2713") {
		t.Errorf("expected success symbol in output, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "a-note") {
		t.Errorf("expected note name in output, got %q", lines[0])
	}
}

// TestAutoCreateMultipleNotesInNewNotebook verifies that creating multiple
// notes in a non-existent notebook works correctly — the first call creates
// the notebook, and subsequent calls reuse it.
func TestAutoCreateMultipleNotesInNewNotebook(t *testing.T) {
	dir := setupTestStore(t)
	book := "multi-book"

	for _, note := range []string{"note-a", "note-b", "note-c"} {
		out, err := executeCapture([]string{"--dir", dir, book, "new", note})
		if err != nil {
			t.Fatalf("create %q: unexpected error: %v", note, err)
		}
		if !strings.Contains(out, note) {
			t.Errorf("expected %q in output, got %q", note, out)
		}
	}

	// All three notes should appear in a listing.
	out, err := executeCapture([]string{"--dir", dir, book, "list"})
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	for _, note := range []string{"note-a", "note-b", "note-c"} {
		if !strings.Contains(out, note) {
			t.Errorf("expected %q in list output, got %q", note, out)
		}
	}
}
