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


func TestBrowserHelpToggle(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Help should be hidden initially.
	if m.showHelp {
		t.Fatal("expected showHelp to be false initially")
	}

	// Press '?' to open help.
	m = sendRune(t, m, '?')

	if !m.showHelp {
		t.Fatal("expected showHelp to be true after pressing '?'")
	}

	view := m.View()
	if !containsStr(view, "Keybindings") {
		t.Errorf("help overlay should contain 'Keybindings', got:\n%s", view)
	}
	if !containsStr(view, "Navigate") {
		t.Errorf("help overlay should contain 'Navigate', got:\n%s", view)
	}

	// Press '?' again to dismiss.
	m = sendRune(t, m, '?')

	if m.showHelp {
		t.Fatal("expected showHelp to be false after pressing '?' again")
	}

	view = m.View()
	if containsStr(view, "Keybindings") {
		t.Error("help overlay should not be visible after dismissing")
	}
}

func TestBrowserHelpEscDismisses(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter the notebook so we're at level 1.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Open help.
	m = sendRune(t, m, '?')
	if !m.showHelp {
		t.Fatal("expected showHelp to be true")
	}

	// Press Esc to dismiss help.
	m = sendKey(t, m, tea.KeyEsc)

	if m.showHelp {
		t.Fatal("expected showHelp to be false after Esc")
	}

	// Esc should NOT navigate back -- still at level 1.
	if m.level != 1 {
		t.Errorf("expected level 1 after Esc dismisses help, got %d", m.level)
	}
}

// sendString types a sequence of runes into the model, processing commands after each.
func sendString(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = sendRune(t, m, r)
	}
	return m
}

func TestBrowserDeleteNotebook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work":     {"todo"},
		"personal": {"journal"},
	})

	m := initModel(t, s)

	// Find which notebook is at cursor 0.
	if len(m.filtered) == 0 {
		t.Fatal("expected notebooks in filtered list")
	}
	idx := m.filtered[m.cursor]
	name := m.notebooks[idx].name

	// Press 'd' to start delete.
	m = sendRune(t, m, 'd')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'd'")
	}

	// Type the correct name.
	m = sendString(t, m, name)

	// Press Enter to confirm.
	m = sendKey(t, m, tea.KeyEnter)

	// The notebook should be deleted from the store.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	for _, nb := range notebooks {
		if nb == name {
			t.Errorf("notebook %q should have been deleted", name)
		}
	}

	// Should NOT be in input mode anymore.
	if m.inputMode {
		t.Error("expected inputMode to be false after confirming delete")
	}
}

func TestBrowserDeleteNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"meeting-notes", "todo"},
	})

	m := initModel(t, s)

	// Enter the notebook.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Find which note is at cursor 0.
	if len(m.filtered) == 0 {
		t.Fatal("expected notes in filtered list")
	}
	idx := m.filtered[m.cursor]
	name := m.notes[idx].Name

	// Press 'd' to start delete.
	m = sendRune(t, m, 'd')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'd'")
	}

	// Type the correct note name.
	m = sendString(t, m, name)

	// Press Enter to confirm.
	m = sendKey(t, m, tea.KeyEnter)

	// The note should be deleted from the store.
	notes, err := s.ListNotes(m.currentBook)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if n.Name == name {
			t.Errorf("note %q should have been deleted", name)
		}
	}

	if m.inputMode {
		t.Error("expected inputMode to be false after confirming delete")
	}
}

func TestBrowserDeleteWrongName(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 'd' to start delete.
	m = sendRune(t, m, 'd')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'd'")
	}

	// Type the wrong name.
	m = sendString(t, m, "wrong-name")

	// Press Enter.
	m = sendKey(t, m, tea.KeyEnter)

	// The notebook should NOT have been deleted.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 {
		t.Errorf("expected 1 notebook (not deleted), got %d", len(notebooks))
	}

	// A status message should be set.
	if m.statusText == "" {
		t.Error("expected a status message after wrong name")
	}
	if !containsStr(m.statusText, "match") {
		t.Errorf("expected status to mention 'match', got %q", m.statusText)
	}

	// Should NOT be in input mode.
	if m.inputMode {
		t.Error("expected inputMode to be false after wrong name")
	}
}

func TestBrowserDeleteCancel(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 'd' to start delete.
	m = sendRune(t, m, 'd')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'd'")
	}

	// Press Esc to cancel.
	m = sendKey(t, m, tea.KeyEsc)

	// The notebook should NOT have been deleted.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 {
		t.Errorf("expected 1 notebook (not deleted), got %d", len(notebooks))
	}

	// Should NOT be in input mode.
	if m.inputMode {
		t.Error("expected inputMode to be false after Esc")
	}
}

func TestBrowserDeleteEmptyList(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	// Press 'd' on empty list should do nothing.
	m = sendRune(t, m, 'd')

	if m.inputMode {
		t.Error("expected inputMode to be false when list is empty")
	}
}

func TestBrowserDeleteCursorAdjust(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"alpha": {"a"},
		"bravo": {"b"},
	})

	m := initModel(t, s)

	// Move cursor to last item.
	m = sendKey(t, m, tea.KeyDown)
	if m.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", m.cursor)
	}

	// Get the name at cursor position.
	idx := m.filtered[m.cursor]
	name := m.notebooks[idx].name

	// Delete it.
	m = sendRune(t, m, 'd')
	m = sendString(t, m, name)
	m = sendKey(t, m, tea.KeyEnter)

	// After reload, cursor should be clamped (only 1 item left).
	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor %d is out of bounds for %d items", m.cursor, len(m.filtered))
	}
}

func TestBrowserRenameNotebook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"old-name": {"todo"},
	})

	m := initModel(t, s)

	// Find which notebook is at cursor 0.
	if len(m.filtered) == 0 {
		t.Fatal("expected notebooks in filtered list")
	}
	idx := m.filtered[m.cursor]
	oldName := m.notebooks[idx].name

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}

	// Input should be pre-populated with the current name.
	if m.inputValue != oldName {
		t.Errorf("expected inputValue %q, got %q", oldName, m.inputValue)
	}

	// Clear the input and type new name.
	for range m.inputValue {
		m = sendKey(t, m, tea.KeyBackspace)
	}
	m = sendString(t, m, "new-name")

	// Press Enter to confirm.
	m = sendKey(t, m, tea.KeyEnter)

	// The old notebook should be gone and new one should exist.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, nb := range notebooks {
		found[nb] = true
	}
	if found[oldName] {
		t.Errorf("old notebook %q should have been renamed", oldName)
	}
	if !found["new-name"] {
		t.Error("new notebook 'new-name' should exist after rename")
	}

	if m.inputMode {
		t.Error("expected inputMode to be false after confirming rename")
	}
}

func TestBrowserRenameNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"old-note", "other"},
	})

	m := initModel(t, s)

	// Enter the notebook.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Find which note is at cursor 0.
	if len(m.filtered) == 0 {
		t.Fatal("expected notes in filtered list")
	}
	idx := m.filtered[m.cursor]
	oldName := m.notes[idx].Name

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}

	// Input should be pre-populated with the current name.
	if m.inputValue != oldName {
		t.Errorf("expected inputValue %q, got %q", oldName, m.inputValue)
	}

	// Clear the input and type new name.
	for range m.inputValue {
		m = sendKey(t, m, tea.KeyBackspace)
	}
	m = sendString(t, m, "new-note")

	// Press Enter to confirm.
	m = sendKey(t, m, tea.KeyEnter)

	// The old note should be gone and new one should exist.
	notes, err := s.ListNotes(m.currentBook)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, n := range notes {
		found[n.Name] = true
	}
	if found[oldName] {
		t.Errorf("old note %q should have been renamed", oldName)
	}
	if !found["new-note"] {
		t.Error("new note 'new-note' should exist after rename")
	}

	if m.inputMode {
		t.Error("expected inputMode to be false after confirming rename")
	}
}

func TestBrowserRenameCancel(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}

	// Press Esc to cancel.
	m = sendKey(t, m, tea.KeyEsc)

	// The notebook should NOT have been renamed.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 || notebooks[0] != "work" {
		t.Errorf("expected notebook 'work' unchanged, got %v", notebooks)
	}

	if m.inputMode {
		t.Error("expected inputMode to be false after Esc")
	}
}

func TestBrowserRenameSameName(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Find which notebook is at cursor 0.
	idx := m.filtered[m.cursor]
	name := m.notebooks[idx].name

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}

	// Input is pre-populated with current name. Press Enter without changing.
	m = sendKey(t, m, tea.KeyEnter)

	// Should show "No change" status.
	if m.statusText != "No change" {
		t.Errorf("expected status 'No change', got %q", m.statusText)
	}

	// Notebook should still exist.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 || notebooks[0] != name {
		t.Errorf("expected notebook %q unchanged, got %v", name, notebooks)
	}

	if m.inputMode {
		t.Error("expected inputMode to be false")
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
