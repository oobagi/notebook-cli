package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	storageDir = dir
	return dir
}

func TestNewNotebook(t *testing.T) {
	dir := setupTestDir(t)
	rootCmd.SetArgs([]string{"new", "work"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new notebook: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "work")); err != nil {
		t.Error("expected work directory to exist")
	}
}

func TestNewNotebookAlreadyExists(t *testing.T) {
	setupTestDir(t)
	rootCmd.SetArgs([]string{"new", "work"})
	_ = rootCmd.Execute()

	rootCmd.SetArgs([]string{"new", "work"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for duplicate notebook")
	}
}

func TestNewNotebookInvalidName(t *testing.T) {
	setupTestDir(t)
	rootCmd.SetArgs([]string{"new", "../escape"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid name")
	}
}

func TestNewNoteViaNounVerb(t *testing.T) {
	dir := setupTestDir(t)
	// First create the notebook
	rootCmd.SetArgs([]string{"new", "ideas"})
	_ = rootCmd.Execute()

	// Then create a note via noun-verb: notebook ideas new "shower thought"
	rootCmd.SetArgs([]string{"ideas", "new", "shower thought"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new note via noun-verb: %v", err)
	}
	path := filepath.Join(dir, "ideas", "shower thought.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected note file at %s", path)
	}
}

func TestNewNoteAutoCreatesNotebook(t *testing.T) {
	dir := setupTestDir(t)
	// Create note in non-existent notebook
	rootCmd.SetArgs([]string{"fresh-book", "new", "first note"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new note with auto-create: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fresh-book")); err != nil {
		t.Error("expected auto-created notebook directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "fresh-book", "first note.md")); err != nil {
		t.Error("expected note file to exist")
	}
}
