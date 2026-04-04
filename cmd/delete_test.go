package cmd

import (
	"strings"
	"testing"

	"github.com/oobagi/notebook-cli/internal/storage"
)

// --- Notebook deletion tests ---

func TestDeleteNotebookWithForceFlag(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("ideas")
	_ = st.CreateNote("ideas", "spark", "bright idea")
	_ = st.CreateNote("ideas", "plan", "a plan")

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "ideas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Deleted \"ideas\" and 2 notes\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}

	// Verify notebook is gone.
	nbs, _ := st.ListNotebooks()
	for _, n := range nbs {
		if n == "ideas" {
			t.Error("notebook 'ideas' should be deleted")
		}
	}
}

func TestDeleteNotebookWithForceShortFlag(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("temp")

	out, err := executeCapture([]string{"--dir", dir, "delete", "-f", "temp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Deleted \"temp\"\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestDeleteEmptyNotebookWithForce(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("empty")

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "empty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Deleted \"empty\"\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestDeleteNotebookConfirmYes(t *testing.T) {
	dir := setupTestStore(t)
	forceFlag = false
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("ideas")
	_ = st.CreateNote("ideas", "note1", "content")
	_ = st.CreateNote("ideas", "note2", "content")
	_ = st.CreateNote("ideas", "note3", "content")

	// Simulate user typing the notebook name to confirm deletion.
	rootCmd.SetIn(strings.NewReader("ideas\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "delete", "ideas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `Delete "ideas" and 3 notes?`) {
		t.Errorf("expected confirmation prompt in output, got %q", out)
	}
	if !strings.Contains(out, "Type the name to confirm:") {
		t.Errorf("expected name-confirmation prompt in output, got %q", out)
	}
	if !strings.Contains(out, "\u2713 Deleted \"ideas\" and 3 notes") {
		t.Errorf("expected success message in output, got %q", out)
	}

	// Verify notebook is gone.
	nbs, _ := st.ListNotebooks()
	for _, n := range nbs {
		if n == "ideas" {
			t.Error("notebook 'ideas' should be deleted")
		}
	}
}

func TestDeleteNotebookConfirmExactName(t *testing.T) {
	dir := setupTestStore(t)
	forceFlag = false
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("work")
	_ = st.CreateNote("work", "task", "do stuff")

	rootCmd.SetIn(strings.NewReader("work\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "delete", "work"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713 Deleted \"work\" and 1 note\n") {
		t.Errorf("expected success message in output, got %q", out)
	}
}

func TestDeleteNotebookConfirmNo(t *testing.T) {
	dir := setupTestStore(t)
	forceFlag = false
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("keep")
	_ = st.CreateNote("keep", "important", "do not delete")

	rootCmd.SetIn(strings.NewReader("n\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "delete", "keep"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Cancelled") {
		t.Errorf("expected 'Cancelled' in output, got %q", out)
	}

	// Verify notebook still exists.
	nbs, _ := st.ListNotebooks()
	found := false
	for _, n := range nbs {
		if n == "keep" {
			found = true
		}
	}
	if !found {
		t.Error("notebook 'keep' should still exist after cancelling delete")
	}
}

func TestDeleteNotebookConfirmEmpty(t *testing.T) {
	dir := setupTestStore(t)
	forceFlag = false
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("keep2")

	// Pressing enter without typing anything (default is N).
	rootCmd.SetIn(strings.NewReader("\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "delete", "keep2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Cancelled") {
		t.Errorf("expected 'Cancelled' in output, got %q", out)
	}

	// Verify notebook still exists.
	nbs, _ := st.ListNotebooks()
	found := false
	for _, n := range nbs {
		if n == "keep2" {
			found = true
		}
	}
	if !found {
		t.Error("notebook 'keep2' should still exist after empty input")
	}
}

func TestDeleteEmptyNotebookPrompt(t *testing.T) {
	dir := setupTestStore(t)
	forceFlag = false
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("vacant")

	rootCmd.SetIn(strings.NewReader("vacant\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "delete", "vacant"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty notebook prompt should NOT mention note count.
	if !strings.Contains(out, `Delete "vacant"? This cannot be undone.`) {
		t.Errorf("expected simple prompt for empty notebook, got %q", out)
	}
	if strings.Contains(out, "note") && !strings.Contains(out, "notebook") && !strings.Contains(out, "Deleted") {
		t.Errorf("empty notebook prompt should not mention notes, got %q", out)
	}
	if !strings.Contains(out, "\u2713 Deleted \"vacant\"\n") {
		t.Errorf("expected success message, got %q", out)
	}
}

func TestDeleteNotebookSingularNote(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("solo")
	_ = st.CreateNote("solo", "only", "the one")

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "solo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should say "1 note" not "1 notes".
	want := "  \u2713 Deleted \"solo\" and 1 note\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestDeleteNotebookPluralNotes(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("multi")
	_ = st.CreateNote("multi", "a", "x")
	_ = st.CreateNote("multi", "b", "y")
	_ = st.CreateNote("multi", "c", "z")

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "multi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Deleted \"multi\" and 3 notes\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// --- Not-found tests ---

func TestDeleteNonExistentNotebook(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "ghost"})
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

// --- Note deletion tests ---

func TestDeleteNoteConfirmByName(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "temp", "throwaway")

	// Simulate user typing the note name to confirm deletion.
	rootCmd.SetIn(strings.NewReader("temp\n"))
	defer rootCmd.SetIn(nil)

	// Delete via: notebook <book> delete <note>
	out, err := executeCapture([]string{"--dir", dir, "work", "delete", "temp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `Delete "temp" from work? This cannot be undone.`) {
		t.Errorf("expected confirmation prompt in output, got %q", out)
	}
	if !strings.Contains(out, "Type the name to confirm:") {
		t.Errorf("expected name-confirmation prompt in output, got %q", out)
	}
	if !strings.Contains(out, "\u2713 Deleted \"temp\" from work") {
		t.Errorf("expected success message in output, got %q", out)
	}

	// Verify note is gone.
	_, err = st.GetNote("work", "temp")
	if err == nil {
		t.Fatal("note should have been deleted")
	}
}

func TestDeleteNoteWrongNameCancels(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "important", "keep this")

	// Typing the wrong name should cancel.
	rootCmd.SetIn(strings.NewReader("wrong\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "work", "delete", "important"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Cancelled") {
		t.Errorf("expected 'Cancelled' in output, got %q", out)
	}

	// Verify note still exists.
	_, err = st.GetNote("work", "important")
	if err != nil {
		t.Fatal("note should still exist after cancelled delete")
	}
}

func TestDeleteNoteViaVerbSuffix(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("journal", "entry", "dear diary")

	// Simulate user typing the note name to confirm deletion.
	rootCmd.SetIn(strings.NewReader("entry\n"))
	defer rootCmd.SetIn(nil)

	// Delete via: notebook <book> <note> delete
	out, err := executeCapture([]string{"--dir", dir, "journal", "entry", "delete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713 Deleted \"entry\" from journal") {
		t.Errorf("expected success message in output, got %q", out)
	}

	_, err = st.GetNote("journal", "entry")
	if err == nil {
		t.Fatal("note should have been deleted")
	}
}

func TestDeleteNonExistentNote(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("work")

	out, err := executeCapture([]string{"--dir", dir, "work", "delete", "ghost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got %q", out)
	}
}

func TestDeleteNonExistentNoteViaVerbSuffix(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("work")

	out, err := executeCapture([]string{"--dir", dir, "work", "phantom", "delete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got %q", out)
	}
}
