package browser

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oobagi/notebook/internal/storage"
)

// setupTestStore creates a temp directory with optional notebooks and notes.
func setupTestStore(t *testing.T, books map[string][]string) *storage.Store {
	t.Helper()
	dir := t.TempDir()
	s := storage.NewStore(dir)

	for book, notes := range books {
		if err := s.CreateNotebook(book); err != nil {
			t.Fatalf("create notebook %q: %v", book, err)
		}
		for _, note := range notes {
			if err := s.CreateNote(book, note, "test content for "+note); err != nil {
				t.Fatalf("create note %q/%q: %v", book, note, err)
			}
		}
	}
	return s
}

// initModel creates a Model and processes the Init command to load data.
func initModel(t *testing.T, s *storage.Store) Model {
	t.Helper()
	m := New(Config{
		Store:    s,
		EditNote: func(book, note string) error { return nil },
	})

	// Run Init and process the resulting message.
	cmd := m.Init()
	if cmd != nil {
		msg := cmd()
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}

	// Send a window size so the view renders properly.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	return m
}

func sendKey(t *testing.T, m Model, key tea.KeyType) Model {
	t.Helper()
	updated, cmd := m.Update(tea.KeyMsg{Type: key})
	m = updated.(Model)

	// Process any commands (like loadNotes).
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
	}
	return m
}

func sendRune(t *testing.T, m Model, r rune) Model {
	t.Helper()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	m = updated.(Model)

	if cmd != nil {
		msg := cmd()
		if msg != nil {
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
	}
	return m
}

func TestBrowserInitShowsNotebooks(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work":     {"meeting-notes", "todo"},
		"personal": {"journal"},
	})

	m := initModel(t, s)

	if m.level != 0 {
		t.Errorf("expected level 0, got %d", m.level)
	}

	if len(m.notebooks) != 2 {
		t.Fatalf("expected 2 notebooks, got %d", len(m.notebooks))
	}

	view := m.View()
	// Notebook names should appear in the view.
	if !containsStr(view, "work") {
		t.Errorf("view should contain 'work', got:\n%s", view)
	}
	if !containsStr(view, "personal") {
		t.Errorf("view should contain 'personal', got:\n%s", view)
	}
}

func TestBrowserEnterOpensNotebook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"meeting-notes", "todo"},
	})

	m := initModel(t, s)

	// Cursor is on the first notebook. Press Enter.
	m = sendKey(t, m, tea.KeyEnter)

	if m.level != 1 {
		t.Errorf("expected level 1 after Enter, got %d", m.level)
	}

	if m.currentBook == "" {
		t.Error("currentBook should be set after Enter")
	}

	if len(m.notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(m.notes))
	}

	view := m.View()
	if !containsStr(view, "meeting-notes") {
		t.Errorf("view should contain 'meeting-notes', got:\n%s", view)
	}
	if !containsStr(view, "todo") {
		t.Errorf("view should contain 'todo', got:\n%s", view)
	}
}

func TestBrowserEscGoesBack(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"meeting-notes"},
	})

	m := initModel(t, s)

	// Enter notebook.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press Esc to go back.
	m = sendKey(t, m, tea.KeyEsc)
	if m.level != 0 {
		t.Errorf("expected level 0 after Esc, got %d", m.level)
	}
}

func TestBrowserFilterReducesList(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work":     {"todo"},
		"personal": {"journal"},
		"recipes":  {"pasta"},
	})

	m := initModel(t, s)

	if len(m.filtered) != 3 {
		t.Fatalf("expected 3 filtered items, got %d", len(m.filtered))
	}

	// Press '/' to enter filter mode.
	m = sendRune(t, m, '/')

	if !m.filtering {
		t.Fatal("expected filtering to be true after '/'")
	}

	// Type "per" to filter.
	m = sendRune(t, m, 'p')
	m = sendRune(t, m, 'e')
	m = sendRune(t, m, 'r')

	if m.filter != "per" {
		t.Errorf("expected filter 'per', got %q", m.filter)
	}

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered item, got %d", len(m.filtered))
	}

	// The filtered item should be "personal".
	idx := m.filtered[0]
	if m.notebooks[idx].name != "personal" {
		t.Errorf("expected filtered notebook 'personal', got %q", m.notebooks[idx].name)
	}
}

func TestBrowserQuitOnQ(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	if !m.quitting {
		t.Error("expected quitting to be true after 'q'")
	}

	// cmd should be tea.Quit.
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestBrowserEmptyState(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	view := m.View()
	if !containsStr(view, "No notebooks yet") {
		t.Errorf("expected empty state message, got:\n%s", view)
	}
	if !containsStr(view, "notebook new") {
		t.Errorf("expected create hint, got:\n%s", view)
	}
}

func TestBrowserEmptyNotebook(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStore(dir)

	// Create a notebook directory with no notes.
	if err := os.MkdirAll(filepath.Join(dir, "empty-book"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := initModel(t, s)

	// Enter the empty notebook.
	m = sendKey(t, m, tea.KeyEnter)

	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	view := m.View()
	if !containsStr(view, "No notes in") {
		t.Errorf("expected empty notebook message, got:\n%s", view)
	}
}

func TestBrowserNoteSelection(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"meeting-notes", "todo"},
	})

	m := initModel(t, s)

	// Enter notebook.
	m = sendKey(t, m, tea.KeyEnter)

	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press Enter on first note -- should set selected and quit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	sel := m.Selected()
	if sel == nil {
		t.Fatal("expected a selection after Enter on a note")
	}

	if sel.Book == "" {
		t.Error("selection book should not be empty")
	}
	if sel.Note == "" {
		t.Error("selection note should not be empty")
	}

	if cmd == nil {
		t.Error("expected quit command after note selection")
	}
}

func TestBrowserArrowNavigation(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"alpha":   {"a"},
		"bravo":   {"b"},
		"charlie": {"c"},
	})

	m := initModel(t, s)

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	// Move down.
	m = sendKey(t, m, tea.KeyDown)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after down, got %d", m.cursor)
	}

	// Move down again.
	m = sendKey(t, m, tea.KeyDown)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after second down, got %d", m.cursor)
	}

	// Down at end should not move past last item.
	m = sendKey(t, m, tea.KeyDown)
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Move up.
	m = sendKey(t, m, tea.KeyUp)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after up, got %d", m.cursor)
	}
}

func TestBrowserFilterClearOnEsc(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work":     {"todo"},
		"personal": {"journal"},
	})

	m := initModel(t, s)

	// Start filtering.
	m = sendRune(t, m, '/')
	m = sendRune(t, m, 'w')

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(m.filtered))
	}

	// Esc should clear filter and show all items.
	m = sendKey(t, m, tea.KeyEsc)

	if m.filtering {
		t.Error("expected filtering to be false after Esc")
	}
	if m.filter != "" {
		t.Errorf("expected empty filter after Esc, got %q", m.filter)
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered items after Esc, got %d", len(m.filtered))
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
