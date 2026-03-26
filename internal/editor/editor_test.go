package editor

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitReturnsBlink(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil command for cursor blinking")
	}
}

func TestInitialContentNotModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	if m.modified() {
		t.Fatal("new editor should not be marked as modified")
	}
}

func TestCtrlQQuitsWhenClean(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	// Send a window size so the textarea is usable.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+Q on clean content should return tea.Quit command")
	}
	if !m.quitting {
		t.Fatal("model should be in quitting state")
	}
}

func TestCtrlQShowsWarningWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Type a character to modify the content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if !m.modified() {
		t.Fatal("editor should be modified after typing")
	}

	// First Ctrl+Q should show warning, not quit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("first Ctrl+Q on modified content should not quit")
	}
	if !m.quitWarning {
		t.Fatal("should show quit warning")
	}
	if m.status == "" {
		t.Fatal("status should contain warning message")
	}
}

func TestCtrlQTwiceQuitsWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// First Ctrl+Q: warning.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	// Second Ctrl+Q: quit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("second Ctrl+Q should return tea.Quit command")
	}
	if !m.quitting {
		t.Fatal("model should be in quitting state")
	}
}

func TestCtrlCForcesQuit(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+C should quit without warning.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+C should return tea.Quit command")
	}
	if !m.quitting {
		t.Fatal("model should be in quitting state")
	}
}

func TestCtrlSTriggersSave(t *testing.T) {
	saved := false
	var savedContent string
	saveFn := func(content string) error {
		saved = true
		savedContent = content
		return nil
	}

	m := New(Config{Title: "test", Content: "hello", Save: saveFn})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Ctrl+S.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+S should return a save command")
	}

	// Execute the command to trigger the save function.
	msg := cmd()
	if !saved {
		t.Fatal("save function should have been called")
	}
	if savedContent != "hello" {
		t.Fatalf("saved content should be %q, got %q", "hello", savedContent)
	}

	// Process the savedMsg.
	updated, _ = m.Update(msg)
	m = updated.(Model)

	if m.status != "Saved" {
		t.Fatalf("status should be %q, got %q", "Saved", m.status)
	}
}

func TestCtrlSSaveError(t *testing.T) {
	saveFn := func(content string) error {
		return errors.New("disk full")
	}

	m := New(Config{Title: "test", Content: "hello", Save: saveFn})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)

	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)

	if m.statusStyle != statusError {
		t.Fatal("status should indicate error after failed save")
	}
	if m.status == "" {
		t.Fatal("status message should describe the error")
	}
}

func TestOtherKeyResetsQuitWarning(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// First Ctrl+Q: warning.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if !m.quitWarning {
		t.Fatal("quit warning should be set")
	}

	// Type another character — should reset the warning.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if m.quitWarning {
		t.Fatal("quit warning should be reset after typing")
	}
}

func TestWindowSizeMsgSetsSize(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	if m.width != 120 {
		t.Fatalf("width should be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Fatalf("height should be 40, got %d", m.height)
	}
}

func TestViewNotEmptyWhenNotQuitting(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	view := m.View()
	if view == "" {
		t.Fatal("view should not be empty when not quitting")
	}
}

func TestViewEmptyWhenQuitting(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	view := m.View()
	if view != "" {
		t.Fatalf("view should be empty when quitting, got %q", view)
	}
}

func TestStatusBarContainsTitle(t *testing.T) {
	m := New(Config{Title: "work \u203A notes", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	view := m.View()
	if !containsPlainText(view, "work \u203A notes") {
		t.Fatal("view should contain the title in the status bar")
	}
}

func TestStatusBarShowsModifiedIndicator(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Before modification.
	view := m.View()
	if containsPlainText(view, "[modified]") {
		t.Fatal("should not show [modified] when content is unchanged")
	}

	// After modification.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	view = m.View()
	if !containsPlainText(view, "[modified]") {
		t.Fatal("should show [modified] after content is changed")
	}
}

// containsPlainText checks if a string contains the target text,
// ignoring any ANSI escape sequences.
func containsPlainText(s, target string) bool {
	// Strip ANSI escape sequences for comparison.
	clean := stripAnsi(s)
	return containsStr(clean, target)
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func stripAnsi(s string) string {
	var out []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until 'm' or end of string.
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++ // skip 'm'
			}
			i = j
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}
