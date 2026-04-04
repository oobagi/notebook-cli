package cmd

import (
	"strings"
	"testing"
)

// --- isFilePath tests ---

func TestIsFilePathWithSlash(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"path/to/file.md", true},
		{"./notes.txt", true},
		{"../README.md", true},
		{"/absolute/path.md", true},
		{"relative/dir", true},
	}
	for _, tt := range tests {
		if got := isFilePath(tt.input); got != tt.want {
			t.Errorf("isFilePath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsFilePathWithExtension(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"notes.md", true},
		{"todo.txt", true},
		{"NOTES.MD", true},
		{"file.TXT", true},
	}
	for _, tt := range tests {
		if got := isFilePath(tt.input); got != tt.want {
			t.Errorf("isFilePath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsFilePathPlainNames(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"mynotebook", false},
		{"work", false},
		{"ideas", false},
		{"list", false},
		{"new", false},
	}
	for _, tt := range tests {
		if got := isFilePath(tt.input); got != tt.want {
			t.Errorf("isFilePath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- File not found tests ---

func TestOpenFileNotFound(t *testing.T) {
	setupTestStore(t)

	_, err := executeCapture([]string{"path/to/nonexistent.md"})
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' in error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "nonexistent.md") {
		t.Errorf("expected file path in error, got %q", err.Error())
	}
}

func TestOpenFileNotFoundWithExtension(t *testing.T) {
	setupTestStore(t)

	_, err := executeCapture([]string{"nonexistent.md"})
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' in error, got %q", err.Error())
	}
}

// --- Fallthrough to dispatch ---

func TestPlainNameStillDispatchesToNotebook(t *testing.T) {
	dir := setupTestStore(t)
	store.CreateNote("ideas", "spark", "content")

	out, err := executeCapture([]string{"--dir", dir, "ideas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should list notes, not try to open as file.
	if !strings.Contains(out, "spark") {
		t.Errorf("expected note 'spark' in output, got %q", out)
	}
}
