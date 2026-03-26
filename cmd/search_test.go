package cmd

import (
	"strings"
	"testing"

	"github.com/oobagi/notebook/internal/storage"
)

func TestSearchCommand(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("Work", "Meeting Notes", "discussed the quarterly plan and budget")
	_ = st.CreateNote("Personal", "Journal", "reminded me of the quarterly review")

	out, err := executeCapture([]string{"--dir", dir, "search", "quarterly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain breadcrumb-style output.
	if !strings.Contains(out, "\u203A") {
		t.Errorf("expected breadcrumb separator in output, got %q", out)
	}

	// Should contain both notebook names.
	if !strings.Contains(out, "Work") {
		t.Errorf("expected 'Work' in output, got %q", out)
	}
	if !strings.Contains(out, "Personal") {
		t.Errorf("expected 'Personal' in output, got %q", out)
	}

	// Should contain match term (in bold ANSI).
	if !strings.Contains(out, "quarterly") {
		t.Errorf("expected 'quarterly' in output, got %q", out)
	}

	// Should contain summary line.
	if !strings.Contains(out, "2 matches") {
		t.Errorf("expected '2 matches' in output, got %q", out)
	}
	if !strings.Contains(out, "2 books") {
		t.Errorf("expected '2 books' in output, got %q", out)
	}
}

func TestSearchBookScoped(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "todo", "buy milk for the team")
	_ = st.CreateNote("personal", "list", "buy milk at the store")

	out, err := executeCapture([]string{"--dir", dir, "work", "search", "milk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only contain work results.
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' in output, got %q", out)
	}
	if !strings.Contains(out, "todo") {
		t.Errorf("expected 'todo' in output, got %q", out)
	}

	// Should NOT contain personal results.
	if strings.Contains(out, "personal") {
		t.Errorf("scoped search should not contain 'personal', got %q", out)
	}

	// Should show 1 match across 1 book.
	if !strings.Contains(out, "1 match across 1 book") {
		t.Errorf("expected '1 match across 1 book' in output, got %q", out)
	}
}

func TestSearchNoResults(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("nb", "note", "nothing relevant")

	out, err := executeCapture([]string{"--dir", dir, "search", "zzzzz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No matches") {
		t.Errorf("expected 'No matches' in output, got %q", out)
	}
}

func TestSearchCaseSensitiveFlag(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("nb", "note", "Hello World\nhello world")

	// Case-sensitive: only "Hello" should match line 1.
	out, err := executeCapture([]string{"--dir", dir, "search", "--case-sensitive", "Hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1 match") {
		t.Errorf("expected exactly '1 match' for case-sensitive search, got %q", out)
	}
}

func TestSearchOutputFormat(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("Work", "Notes", "the quarterly plan was discussed")

	out, err := executeCapture([]string{"--dir", dir, "search", "quarterly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// First line: breadcrumb with two-space indent.
	if len(lines) < 1 || !strings.HasPrefix(lines[0], "  ") {
		t.Errorf("breadcrumb line missing two-space indent: %q", lines[0])
	}
	if !strings.Contains(lines[0], "Work \u203A Notes") {
		t.Errorf("expected breadcrumb 'Work > Notes', got %q", lines[0])
	}

	// Second line: snippet with four-space indent.
	if len(lines) < 2 || !strings.HasPrefix(lines[1], "    ") {
		t.Errorf("snippet line missing four-space indent: %q", lines[1])
	}

	// Should contain bold ANSI around the match.
	if !strings.Contains(lines[1], "\x1b[1m") {
		t.Errorf("expected ANSI bold in snippet, got %q", lines[1])
	}
}
