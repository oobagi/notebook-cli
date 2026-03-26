package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateNotebook(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNotebook("work"); err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}

	info, err := os.Stat(filepath.Join(store.Root, "work"))
	if err != nil {
		t.Fatalf("notebook dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("notebook path is not a directory")
	}
}

func TestCreateNotebookIdempotent(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNotebook("work"); err != nil {
		t.Fatalf("first CreateNotebook: %v", err)
	}
	if err := store.CreateNotebook("work"); err != nil {
		t.Fatalf("second CreateNotebook should not fail: %v", err)
	}
}

func TestCreateNoteAndGetNote(t *testing.T) {
	store := NewStore(t.TempDir())

	content := "# Hello\nThis is a test note."
	if err := store.CreateNote("journal", "day1", content); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	note, err := store.GetNote("journal", "day1")
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}

	if note.Name != "day1" {
		t.Errorf("Name = %q, want %q", note.Name, "day1")
	}
	if note.Notebook != "journal" {
		t.Errorf("Notebook = %q, want %q", note.Notebook, "journal")
	}
	if note.Content != content {
		t.Errorf("Content = %q, want %q", note.Content, content)
	}
	if note.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if note.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestCreateNoteAutoCreatesNotebook(t *testing.T) {
	store := NewStore(t.TempDir())

	// Notebook directory does not exist yet.
	if err := store.CreateNote("ideas", "spark", "an idea"); err != nil {
		t.Fatalf("CreateNote with auto-create: %v", err)
	}

	info, err := os.Stat(filepath.Join(store.Root, "ideas"))
	if err != nil {
		t.Fatalf("notebook dir not auto-created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("notebook path is not a directory")
	}
}

func TestNoteStoredAsMdFile(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("work", "todo", "buy milk"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	expected := filepath.Join(store.Root, "work", "todo.md")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected file %q not found: %v", expected, err)
	}
}

func TestGetNoteNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	_, err := store.GetNote("ghost", "missing")
	if err == nil {
		t.Fatal("expected error for non-existent note, got nil")
	}
}

func TestListNotebooks(t *testing.T) {
	store := NewStore(t.TempDir())

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := store.CreateNotebook(name); err != nil {
			t.Fatalf("CreateNotebook(%q): %v", name, err)
		}
	}

	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}

	if len(notebooks) != 3 {
		t.Fatalf("got %d notebooks, want 3", len(notebooks))
	}

	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, n := range notebooks {
		if !want[n] {
			t.Errorf("unexpected notebook %q", n)
		}
	}
}

func TestListNotebooksEmpty(t *testing.T) {
	store := NewStore(t.TempDir())

	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}
	if len(notebooks) != 0 {
		t.Fatalf("got %d notebooks, want 0", len(notebooks))
	}
}

func TestListNotes(t *testing.T) {
	store := NewStore(t.TempDir())

	for _, name := range []string{"a", "b", "c"} {
		if err := store.CreateNote("nb", name, "content of "+name); err != nil {
			t.Fatalf("CreateNote(%q): %v", name, err)
		}
	}

	notes, err := store.ListNotes("nb")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}

	found := map[string]bool{}
	for _, n := range notes {
		found[n.Name] = true
		if n.Notebook != "nb" {
			t.Errorf("note %q has Notebook=%q, want %q", n.Name, n.Notebook, "nb")
		}
	}
	for _, name := range []string{"a", "b", "c"} {
		if !found[name] {
			t.Errorf("note %q not found in list", name)
		}
	}
}

func TestListNotesIgnoresNonMdFiles(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("nb", "real", "content"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	// Write a non-.md file directly.
	bogus := filepath.Join(store.Root, "nb", "stray.txt")
	if err := os.WriteFile(bogus, []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	notes, err := store.ListNotes("nb")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1 (non-.md should be ignored)", len(notes))
	}
	if notes[0].Name != "real" {
		t.Errorf("Name = %q, want %q", notes[0].Name, "real")
	}
}

func TestDeleteNote(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("nb", "doomed", "bye"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := store.DeleteNote("nb", "doomed"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, err := store.GetNote("nb", "doomed")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestDeleteNoteNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	err := store.DeleteNote("ghost", "missing")
	if err == nil {
		t.Fatal("expected error for deleting non-existent note, got nil")
	}
}

func TestDeleteNotebook(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("nb", "n1", "one"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := store.CreateNote("nb", "n2", "two"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := store.DeleteNotebook("nb"); err != nil {
		t.Fatalf("DeleteNotebook: %v", err)
	}

	_, err := os.Stat(filepath.Join(store.Root, "nb"))
	if !os.IsNotExist(err) {
		t.Fatalf("notebook dir should be gone, stat err = %v", err)
	}
}

func TestDeleteNotebookNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	// os.RemoveAll does not error on a missing path, so this should succeed.
	if err := store.DeleteNotebook("nope"); err != nil {
		t.Fatalf("DeleteNotebook on missing dir: %v", err)
	}
}

func TestUpdateNote(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("nb", "note", "v1"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := store.UpdateNote("nb", "note", "v2"); err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}

	note, err := store.GetNote("nb", "note")
	if err != nil {
		t.Fatalf("GetNote after update: %v", err)
	}
	if note.Content != "v2" {
		t.Errorf("Content = %q, want %q", note.Content, "v2")
	}
}

func TestUpdateNoteNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	err := store.UpdateNote("ghost", "missing", "data")
	if err == nil {
		t.Fatal("expected error updating non-existent note, got nil")
	}
}

// --- Path traversal rejection tests ---

func TestValidNameRejectsTraversal(t *testing.T) {
	badNames := []string{
		"",
		".",
		"..",
		"../etc",
		"foo/bar",
		"foo\\bar",
		"a..b",
	}

	for _, name := range badNames {
		t.Run("name="+name, func(t *testing.T) {
			err := validName(name)
			if err == nil {
				t.Fatalf("validName(%q) should have returned an error", name)
			}
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("error should wrap ErrInvalidName, got: %v", err)
			}
		})
	}
}

func TestValidNameAcceptsGoodNames(t *testing.T) {
	goodNames := []string{
		"work",
		"my-notebook",
		"day1",
		"UPPER",
		"with spaces",
	}

	for _, name := range goodNames {
		t.Run("name="+name, func(t *testing.T) {
			if err := validName(name); err != nil {
				t.Fatalf("validName(%q) should succeed, got: %v", name, err)
			}
		})
	}
}

func TestCreateNotebookRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	for _, name := range []string{"..", "../escape", "a/b"} {
		if err := store.CreateNotebook(name); err == nil {
			t.Errorf("CreateNotebook(%q) should have failed", name)
		}
	}
}

func TestCreateNoteRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	// Bad notebook name
	if err := store.CreateNote("../evil", "note", "x"); err == nil {
		t.Error("CreateNote with traversal notebook name should fail")
	}
	// Bad note name
	if err := store.CreateNote("good", "../evil", "x"); err == nil {
		t.Error("CreateNote with traversal note name should fail")
	}
}

func TestGetNoteRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.GetNote("../evil", "note"); err == nil {
		t.Error("GetNote with traversal notebook name should fail")
	}
	if _, err := store.GetNote("good", "../evil"); err == nil {
		t.Error("GetNote with traversal note name should fail")
	}
}

func TestDeleteNoteRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.DeleteNote("../evil", "note"); err == nil {
		t.Error("DeleteNote with traversal notebook name should fail")
	}
	if err := store.DeleteNote("good", "../evil"); err == nil {
		t.Error("DeleteNote with traversal note name should fail")
	}
}

func TestDeleteNotebookRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.DeleteNotebook("../evil"); err == nil {
		t.Error("DeleteNotebook with traversal name should fail")
	}
}

func TestUpdateNoteRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.UpdateNote("../evil", "note", "x"); err == nil {
		t.Error("UpdateNote with traversal notebook name should fail")
	}
	if err := store.UpdateNote("good", "../evil", "x"); err == nil {
		t.Error("UpdateNote with traversal note name should fail")
	}
}

func TestListNotesRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.ListNotes("../evil"); err == nil {
		t.Error("ListNotes with traversal name should fail")
	}
}

// --- Duplicate note rejection test ---

func TestCreateNoteDuplicate(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.CreateNote("nb", "dup", "first"); err != nil {
		t.Fatalf("first CreateNote: %v", err)
	}

	err := store.CreateNote("nb", "dup", "second")
	if err == nil {
		t.Fatal("second CreateNote should fail for duplicate name")
	}

	// Verify original content is preserved.
	note, err := store.GetNote("nb", "dup")
	if err != nil {
		t.Fatalf("GetNote after rejected duplicate: %v", err)
	}
	if note.Content != "first" {
		t.Errorf("Content = %q, want %q (original should be preserved)", note.Content, "first")
	}
}

// --- ListNotebooks with non-existent root ---

func TestListNotebooksNonExistentRoot(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "does-not-exist"))

	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks should return empty slice for missing root, got error: %v", err)
	}
	if notebooks == nil {
		t.Fatal("ListNotebooks should return non-nil empty slice, got nil")
	}
	if len(notebooks) != 0 {
		t.Fatalf("got %d notebooks, want 0", len(notebooks))
	}
}

// --- ListNotes on non-existent notebook ---

func TestListNotesNonExistentNotebook(t *testing.T) {
	store := NewStore(t.TempDir())

	_, err := store.ListNotes("nonexistent")
	if err == nil {
		t.Fatal("ListNotes on non-existent notebook should return an error")
	}
}

// --- SearchNotes tests ---

func TestSearchNotesFindsMatch(t *testing.T) {
	store := NewStore(t.TempDir())

	_ = store.CreateNote("work", "meeting", "discussed the quarterly plan\nother stuff")
	_ = store.CreateNote("personal", "journal", "reminded me of the quarterly review")

	results, err := store.SearchNotes("quarterly", "", false)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Results should be sorted by notebook name.
	if results[0].Notebook != "personal" {
		t.Errorf("first result notebook = %q, want %q", results[0].Notebook, "personal")
	}
	if results[1].Notebook != "work" {
		t.Errorf("second result notebook = %q, want %q", results[1].Notebook, "work")
	}
}

func TestSearchNotesCaseInsensitive(t *testing.T) {
	store := NewStore(t.TempDir())

	_ = store.CreateNote("nb", "note", "Hello World\ngoodbye world")

	results, err := store.SearchNotes("HELLO", "", false)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (case-insensitive match)", len(results))
	}
	if results[0].Line != 1 {
		t.Errorf("Line = %d, want 1", results[0].Line)
	}
	if results[0].Text != "Hello World" {
		t.Errorf("Text = %q, want %q", results[0].Text, "Hello World")
	}
}

func TestSearchNotesCaseSensitive(t *testing.T) {
	store := NewStore(t.TempDir())

	_ = store.CreateNote("nb", "note", "Hello World\nhello world")

	results, err := store.SearchNotes("Hello", "", true)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (case-sensitive, only first line)", len(results))
	}
	if results[0].Line != 1 {
		t.Errorf("Line = %d, want 1", results[0].Line)
	}
}

func TestSearchNotesScoped(t *testing.T) {
	store := NewStore(t.TempDir())

	_ = store.CreateNote("work", "todo", "buy milk")
	_ = store.CreateNote("personal", "list", "buy milk")

	results, err := store.SearchNotes("milk", "work", false)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (scoped to work)", len(results))
	}
	if results[0].Notebook != "work" {
		t.Errorf("Notebook = %q, want %q", results[0].Notebook, "work")
	}
}

func TestSearchNotesNoResults(t *testing.T) {
	store := NewStore(t.TempDir())

	_ = store.CreateNote("nb", "note", "nothing relevant here")

	results, err := store.SearchNotes("zzzzz", "", false)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}
