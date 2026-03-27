package picker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitShowsItems(t *testing.T) {
	cfg := Config{
		Title: "Pick one",
		Items: []string{"alpha", "beta", "gamma"},
	}
	m := New(cfg)

	// Init should return nil (no initial command).
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}

	// View should contain all items.
	view := m.View()
	for _, item := range cfg.Items {
		if !containsStr(view, item) {
			t.Errorf("View() should contain item %q, got:\n%s", item, view)
		}
	}

	// View should contain the title.
	if !containsStr(view, cfg.Title) {
		t.Errorf("View() should contain title %q, got:\n%s", cfg.Title, view)
	}
}

func TestArrowNavigation(t *testing.T) {
	cfg := Config{
		Title: "Navigate",
		Items: []string{"first", "second", "third"},
	}
	m := New(cfg)

	// Initial cursor should be at 0.
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	// Move down again.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("after second down: cursor = %d, want 2", m.cursor)
	}

	// Move down at bottom (should not go past last item).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("after down at bottom: cursor = %d, want 2", m.cursor)
	}

	// Move up.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}

	// Move up to top.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("after up to top: cursor = %d, want 0", m.cursor)
	}

	// Move up at top (should stay at 0).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("after up at top: cursor = %d, want 0", m.cursor)
	}
}

func TestEnterSelects(t *testing.T) {
	cfg := Config{
		Title: "Select",
		Items: []string{"one", "two", "three"},
	}
	m := New(cfg)

	// Move to second item.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Press Enter.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.Selected() != "two" {
		t.Errorf("Selected() = %q, want %q", m.Selected(), "two")
	}
	if !m.quitting {
		t.Error("model should be quitting after Enter")
	}
	if cmd == nil {
		t.Error("Enter should return a quit command")
	}
}

func TestEnterSelectsFirstByDefault(t *testing.T) {
	cfg := Config{
		Title: "Select",
		Items: []string{"alpha", "beta"},
	}
	m := New(cfg)

	// Press Enter without moving (cursor at 0).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.Selected() != "alpha" {
		t.Errorf("Selected() = %q, want %q", m.Selected(), "alpha")
	}
}

func TestEscCancels(t *testing.T) {
	cfg := Config{
		Title: "Cancel",
		Items: []string{"a", "b"},
	}
	m := New(cfg)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.Selected() != "" {
		t.Errorf("Selected() = %q, want empty string after Esc", m.Selected())
	}
	if !m.quitting {
		t.Error("model should be quitting after Esc")
	}
	if cmd == nil {
		t.Error("Esc should return a quit command")
	}
}

func TestQKeyCancels(t *testing.T) {
	cfg := Config{
		Title: "Cancel",
		Items: []string{"x", "y"},
	}
	m := New(cfg)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	if m.Selected() != "" {
		t.Errorf("Selected() = %q, want empty string after q", m.Selected())
	}
	if !m.quitting {
		t.Error("model should be quitting after q key")
	}
	if cmd == nil {
		t.Error("q key should return a quit command")
	}
}

func TestEmptyItems(t *testing.T) {
	cfg := Config{
		Title: "Empty",
		Items: []string{},
	}
	m := New(cfg)

	// View should still render (title + hint) without panicking.
	view := m.View()
	if !containsStr(view, "Empty") {
		t.Errorf("View() should contain title even with no items, got:\n%s", view)
	}

	// Enter with no items should not select anything.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.Selected() != "" {
		t.Errorf("Selected() = %q, want empty with no items", m.Selected())
	}
}

func TestRunWithEmptyItems(t *testing.T) {
	result, err := Run(Config{Title: "Empty", Items: []string{}})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if result != "" {
		t.Errorf("Run() = %q, want empty string for empty items", result)
	}
}

func TestViewShowsHint(t *testing.T) {
	cfg := Config{
		Title: "Hint",
		Items: []string{"item"},
	}
	m := New(cfg)
	view := m.View()

	if !containsStr(view, "navigate") {
		t.Errorf("View() should contain hint text 'navigate', got:\n%s", view)
	}
	if !containsStr(view, "Enter") {
		t.Errorf("View() should contain hint text 'Enter', got:\n%s", view)
	}
	if !containsStr(view, "Esc") {
		t.Errorf("View() should contain hint text 'Esc', got:\n%s", view)
	}
}

func TestViewEmptyAfterQuit(t *testing.T) {
	cfg := Config{
		Title: "Quit",
		Items: []string{"a"},
	}
	m := New(cfg)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	view := m.View()
	if view != "" {
		t.Errorf("View() after quit should be empty, got %q", view)
	}
}

// containsStr is a simple helper to check for substrings.
func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
