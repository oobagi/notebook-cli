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
	t.Setenv("HOME", t.TempDir())
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
	if !containsStr(view, "meeting notes") {
		t.Errorf("view should contain 'meeting notes', got:\n%s", view)
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
	if !containsStr(view, "Press n") {
		t.Errorf("expected 'Press n' hint, got:\n%s", view)
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
	if !containsStr(view, "Press n") {
		t.Errorf("expected 'Press n' hint, got:\n%s", view)
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
		if r == ' ' {
			m = sendKey(t, m, tea.KeySpace)
		} else {
			m = sendRune(t, m, r)
		}
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

	// Type the correct note name (display name with spaces).
	m = sendString(t, m, storage.DisplayName(name))

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

func TestBrowserCopyNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter notebook to reach level 1.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 'c' to copy the note.
	m = sendRune(t, m, 'c')

	// Should have a status message about copying.
	if m.statusText == "" {
		t.Fatal("expected a status message after pressing 'c'")
	}
	if !containsStr(m.statusText, "Copied") && !containsStr(m.statusText, "copy") {
		t.Errorf("expected status to mention copying, got %q", m.statusText)
	}
}

func TestBrowserViewNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter notebook to reach level 1.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 'v' to view the note.
	m = sendRune(t, m, 'v')

	if !m.viewMode {
		t.Fatal("expected viewMode to be true after pressing 'v'")
	}

	if !containsStr(m.viewTitle, "todo") {
		t.Errorf("expected viewTitle to contain 'todo', got %q", m.viewTitle)
	}

	view := m.View()
	if !containsStr(view, "todo") {
		t.Errorf("view should contain the note title, got:\n%s", view)
	}

	// Press Esc to close the view.
	m = sendKey(t, m, tea.KeyEsc)

	if m.viewMode {
		t.Fatal("expected viewMode to be false after pressing Esc")
	}

	// Should still be at level 1 (not navigated back).
	if m.level != 1 {
		t.Errorf("expected level 1 after closing view, got %d", m.level)
	}
}

func TestBrowserViewScroll(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter notebook to reach level 1.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 'v' to view the note.
	m = sendRune(t, m, 'v')

	if !m.viewMode {
		t.Fatal("expected viewMode to be true after pressing 'v'")
	}

	// Scroll starts at 0.
	if m.viewScroll != 0 {
		t.Errorf("expected viewScroll 0, got %d", m.viewScroll)
	}

	// Press down to scroll.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	// viewScroll should have incremented (or stayed clamped if content is short).
	// We just verify it didn't go negative and the model is still in view mode.
	if !m.viewMode {
		t.Fatal("expected viewMode to still be true after scrolling down")
	}
	scrollAfterDown := m.viewScroll

	// Press up to scroll back.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.viewScroll > scrollAfterDown {
		t.Errorf("expected viewScroll to decrease or stay after up, got %d (was %d)", m.viewScroll, scrollAfterDown)
	}
	if !m.viewMode {
		t.Fatal("expected viewMode to still be true after scrolling up")
	}
}

func TestBrowserCopyAtL0IsNoop(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// At level 0, press 'c'.
	if m.level != 0 {
		t.Fatalf("expected level 0, got %d", m.level)
	}

	m = sendRune(t, m, 'c')

	// Should not have any status message (noop).
	if m.statusText != "" {
		t.Errorf("expected no status message at L0, got %q", m.statusText)
	}
}

// processAllCmds runs the cmd→msg→Update loop until no more cmds are returned.
func processAllCmds(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		updated, next := m.Update(msg)
		m = updated.(Model)
		cmd = next
	}
	return m
}

func TestBrowserRenameNotebook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"old-name": {"note1"},
	})

	m := initModel(t, s)

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}
	if m.inputValue != "old name" {
		t.Errorf("expected inputValue pre-populated with 'old name', got %q", m.inputValue)
	}

	// Clear the pre-populated value and type a new name.
	for range "old-name" {
		m = sendKey(t, m, tea.KeyBackspace)
	}
	m = sendString(t, m, "new-name")

	// Press Enter to confirm — process the full cmd chain (action → reload → loaded).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = processAllCmds(t, updated.(Model), cmd)

	// Store should reflect the rename.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, nb := range notebooks {
		found[nb] = true
	}
	if found["old-name"] {
		t.Error("old-name should no longer exist")
	}
	if !found["new-name"] {
		t.Error("new-name should exist after rename")
	}

	// Notes should be preserved.
	notes, err := s.ListNotes("new-name")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note in renamed notebook, got %d", len(notes))
	}

	// Cursor should be on the renamed item.
	if len(m.filtered) > 0 {
		idx := m.filtered[m.cursor]
		if m.notebooks[idx].name != "new-name" {
			t.Errorf("expected cursor on 'new-name', got %q", m.notebooks[idx].name)
		}
	}
}

func TestBrowserRenameNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"old-note"},
	})

	m := initModel(t, s)

	// Enter notebook.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'r'")
	}
	if m.inputValue != "old note" {
		t.Errorf("expected inputValue pre-populated with 'old note', got %q", m.inputValue)
	}

	// Clear and type new name.
	for range "old-note" {
		m = sendKey(t, m, tea.KeyBackspace)
	}
	m = sendString(t, m, "new-note")

	// Press Enter to confirm — process the full cmd chain.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = processAllCmds(t, updated.(Model), cmd)

	// Old note should be gone, new note should exist.
	_, err := s.GetNote("work", "old-note")
	if err == nil {
		t.Error("old-note should not exist after rename")
	}
	note, err := s.GetNote("work", "new-note")
	if err != nil {
		t.Fatalf("GetNote new-note: %v", err)
	}
	if note.Content != "test content for old-note" {
		t.Errorf("content not preserved, got %q", note.Content)
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
		t.Fatal("expected inputMode to be true")
	}

	// Press Esc to cancel.
	m = sendKey(t, m, tea.KeyEsc)

	if m.inputMode {
		t.Error("expected inputMode to be false after Esc")
	}

	// Notebook should be unchanged.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 || notebooks[0] != "work" {
		t.Errorf("expected unchanged notebooks, got %v", notebooks)
	}
}

func TestBrowserRenameEmptyList(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	// Press 'r' on empty list should do nothing.
	m = sendRune(t, m, 'r')

	if m.inputMode {
		t.Error("expected inputMode to be false when list is empty")
	}
}

func TestBrowserRenameEmptyName(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	// Clear the name entirely.
	for range "work" {
		m = sendKey(t, m, tea.KeyBackspace)
	}

	// Press Enter with empty name.
	m = sendKey(t, m, tea.KeyEnter)

	// Should have error status.
	if m.statusText == "" {
		t.Error("expected status message for empty name")
	}
	if !containsStr(m.statusText, "empty") {
		t.Errorf("expected status to mention 'empty', got %q", m.statusText)
	}

	// Notebook should be unchanged.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 {
		t.Errorf("expected 1 notebook unchanged, got %d", len(notebooks))
	}
}

func TestBrowserRenameSameName(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 'r' to start rename.
	m = sendRune(t, m, 'r')

	// Don't change the name, just press Enter.
	m = sendKey(t, m, tea.KeyEnter)

	// Should not be in input mode and should show "No change" status.
	if m.inputMode {
		t.Error("expected inputMode to be false")
	}
	if m.statusText != "No change" {
		t.Errorf("expected status 'No change', got %q", m.statusText)
	}

	// Notebook should still exist.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 {
		t.Errorf("expected 1 notebook, got %d", len(notebooks))
	}
}

func TestBrowserCreateNotebook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	// Press 'n' to start create.
	m = sendRune(t, m, 'n')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'n'")
	}
	if m.inputValue != "" {
		t.Errorf("expected empty inputValue, got %q", m.inputValue)
	}

	// Type a name and confirm.
	m = sendString(t, m, "my-book")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = processAllCmds(t, updated.(Model), cmd)

	// Notebook should exist.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 1 || notebooks[0] != "my-book" {
		t.Errorf("expected [my-book], got %v", notebooks)
	}
}

func TestBrowserCreateNote(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {},
	})

	m := initModel(t, s)

	// Enter the notebook.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 'n' to start create.
	m = sendRune(t, m, 'n')

	if !m.inputMode {
		t.Fatal("expected inputMode to be true after pressing 'n'")
	}

	// Type a name and confirm.
	m = sendString(t, m, "my-note")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = processAllCmds(t, updated.(Model), cmd)

	// Note should exist.
	notes, err := s.ListNotes("work")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Name != "my-note" {
		t.Errorf("expected [my-note], got %v", notes)
	}
}

func TestBrowserCreateCancel(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	// Press 'n' then Esc.
	m = sendRune(t, m, 'n')
	m = sendKey(t, m, tea.KeyEsc)

	if m.inputMode {
		t.Error("expected inputMode to be false after Esc")
	}

	// Nothing should have been created.
	notebooks, err := s.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(notebooks) != 0 {
		t.Errorf("expected 0 notebooks, got %d", len(notebooks))
	}
}

func TestBrowserCreateEmptyName(t *testing.T) {
	s := setupTestStore(t, map[string][]string{})

	m := initModel(t, s)

	// Press 'n' then Enter with empty name.
	m = sendRune(t, m, 'n')
	m = sendKey(t, m, tea.KeyEnter)

	if m.statusText == "" {
		t.Error("expected error status for empty name")
	}
	if !containsStr(m.statusText, "empty") {
		t.Errorf("expected status to mention 'empty', got %q", m.statusText)
	}
}

func TestBrowserCreateDuplicate(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter the notebook.
	m = sendKey(t, m, tea.KeyEnter)

	// Try to create a duplicate note.
	m = sendRune(t, m, 'n')
	m = sendString(t, m, "todo")
	m = sendKey(t, m, tea.KeyEnter)

	if m.statusText == "" {
		t.Error("expected error status for duplicate name")
	}
	if !containsStr(m.statusText, "already exists") {
		t.Errorf("expected status to mention 'already exists', got %q", m.statusText)
	}
}

func TestBrowserInitialBook(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo", "meeting-notes"},
	})

	m := New(Config{
		Store:       s,
		EditNote:    func(book, note string) error { return nil },
		InitialBook: "work",
	})

	cmd := m.Init()
	if cmd != nil {
		msg := cmd()
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.level != 1 {
		t.Errorf("expected level 1 with InitialBook, got %d", m.level)
	}
	if m.currentBook != "work" {
		t.Errorf("expected currentBook 'work', got %q", m.currentBook)
	}
	if len(m.notes) != 2 {
		t.Errorf("expected 2 notes loaded, got %d", len(m.notes))
	}

	// Pressing Esc should go back to L0.
	m = sendKey(t, m, tea.KeyEsc)
	if m.level != 0 {
		t.Errorf("expected level 0 after Esc, got %d", m.level)
	}
}

func TestBrowserViewAtL0IsNoop(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// At level 0, press 'v'.
	if m.level != 0 {
		t.Fatalf("expected level 0, got %d", m.level)
	}

	m = sendRune(t, m, 'v')

	// Should not enter view mode (noop).
	if m.viewMode {
		t.Error("expected viewMode to be false at L0")
	}
}

func TestBrowserThemePickerOpen(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Press 't' to open theme picker at L0.
	m = sendRune(t, m, 't')

	if !m.themeMode {
		t.Fatal("expected themeMode to be true after pressing 't'")
	}
	if len(m.themeStyles) == 0 {
		t.Fatal("expected themeStyles to be populated")
	}
	if m.themeCursor != 0 {
		t.Errorf("expected themeCursor at 0, got %d", m.themeCursor)
	}
	if m.themePreview == "" {
		t.Error("expected themePreview to be non-empty")
	}

	view := m.View()
	if !containsStr(view, "Glamour Style") {
		t.Errorf("view should contain 'Glamour Style', got:\n%s", view)
	}
}

func TestBrowserThemePickerOpenAtL1(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Enter notebook to reach L1.
	m = sendKey(t, m, tea.KeyEnter)
	if m.level != 1 {
		t.Fatalf("expected level 1, got %d", m.level)
	}

	// Press 't' to open theme picker at L1.
	m = sendRune(t, m, 't')

	if !m.themeMode {
		t.Fatal("expected themeMode to be true after pressing 't' at L1")
	}
}

func TestBrowserThemePickerNavigate(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open theme picker.
	m = sendRune(t, m, 't')
	if !m.themeMode {
		t.Fatal("expected themeMode to be true")
	}

	initialPreview := m.themePreview

	// Move down.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.themeCursor != 1 {
		t.Errorf("expected themeCursor at 1, got %d", m.themeCursor)
	}

	// Preview should update when cursor moves.
	if m.themePreview == "" {
		t.Error("expected themePreview to be non-empty after moving down")
	}
	// The preview for a different style may differ from the initial one.
	_ = initialPreview // used to verify preview exists

	// Move up back to first.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.themeCursor != 0 {
		t.Errorf("expected themeCursor at 0, got %d", m.themeCursor)
	}

	// Move up at top should not go negative.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.themeCursor != 0 {
		t.Errorf("expected themeCursor to stay at 0, got %d", m.themeCursor)
	}
}

func TestBrowserThemePickerEscCancels(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open theme picker.
	m = sendRune(t, m, 't')
	if !m.themeMode {
		t.Fatal("expected themeMode to be true")
	}

	// Press Esc to cancel.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.themeMode {
		t.Fatal("expected themeMode to be false after Esc")
	}

	// Should still be at same level (no navigation).
	if m.level != 0 {
		t.Errorf("expected level 0 after Esc, got %d", m.level)
	}
}

func TestBrowserThemePickerQDismisses(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open theme picker.
	m = sendRune(t, m, 't')
	if !m.themeMode {
		t.Fatal("expected themeMode to be true")
	}

	// Press 'q' to dismiss (should not quit the app).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	if m.themeMode {
		t.Fatal("expected themeMode to be false after 'q'")
	}
	if m.quitting {
		t.Error("'q' in theme picker should not quit the app")
	}
	if cmd != nil {
		t.Error("expected no command from 'q' in theme picker")
	}
}

func TestBrowserThemePickerTDismisses(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open theme picker.
	m = sendRune(t, m, 't')
	if !m.themeMode {
		t.Fatal("expected themeMode to be true")
	}

	// Press 't' again to dismiss (toggle).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(Model)

	if m.themeMode {
		t.Fatal("expected themeMode to be false after 't' toggle")
	}
}

func TestBrowserThemePickerDownClamps(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open theme picker.
	m = sendRune(t, m, 't')

	// Move to last item.
	for i := 0; i < len(m.themeStyles)+5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}

	if m.themeCursor != len(m.themeStyles)-1 {
		t.Errorf("expected themeCursor at %d, got %d", len(m.themeStyles)-1, m.themeCursor)
	}
}

func TestBrowserThemePickerStyles(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)
	m = sendRune(t, m, 't')

	expected := []string{"auto", "dark", "light", "dracula", "tokyo-night", "pink"}
	if len(m.themeStyles) != len(expected) {
		t.Fatalf("expected %d styles, got %d", len(expected), len(m.themeStyles))
	}
	for i, s := range expected {
		if m.themeStyles[i] != s {
			t.Errorf("expected style[%d] = %q, got %q", i, s, m.themeStyles[i])
		}
	}
}

func TestBrowserThemePickerViewContent(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)
	m = sendRune(t, m, 't')

	view := m.View()
	// Should contain the overlay with style names.
	if !containsStr(view, "dark") {
		t.Errorf("view should list 'dark' style, got:\n%s", view)
	}
	if !containsStr(view, "dracula") {
		t.Errorf("view should list 'dracula' style, got:\n%s", view)
	}
	// Should contain navigation hint.
	if !containsStr(view, "navigate") {
		t.Errorf("view should contain navigation hint, got:\n%s", view)
	}
}

func TestBrowserHelpShowsThemeKey(t *testing.T) {
	s := setupTestStore(t, map[string][]string{
		"work": {"todo"},
	})

	m := initModel(t, s)

	// Open help at L0.
	m = sendRune(t, m, '?')
	view := m.View()
	if !containsStr(view, "Theme picker") {
		t.Errorf("L0 help should mention 'Theme picker', got:\n%s", view)
	}

	// Close help, enter notebook, open help at L1.
	m = sendRune(t, m, '?')
	m = sendKey(t, m, tea.KeyEnter)
	m = sendRune(t, m, '?')
	view = m.View()
	if !containsStr(view, "Theme picker") {
		t.Errorf("L1 help should mention 'Theme picker', got:\n%s", view)
	}
}
