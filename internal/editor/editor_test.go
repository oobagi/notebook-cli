package editor

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewParsesBlocks(t *testing.T) {
	content := "# Title\n\nSome paragraph\n\n- bullet one"
	m := New(Config{Title: "test", Content: content})

	if m.BlockCount() < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", m.BlockCount())
	}
}

func TestNewEmptyContent(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})

	if m.BlockCount() < 1 {
		t.Fatal("empty content should produce at least one block")
	}
}

func TestContentReturnsSerializedMarkdown(t *testing.T) {
	content := "# Hello\n\nWorld"
	m := New(Config{Title: "test", Content: content})

	got := m.Content()
	if got != content {
		t.Fatalf("Content() round-trip failed:\nwant: %q\ngot:  %q", content, got)
	}
}

func TestRoundTripSimpleMarkdown(t *testing.T) {
	content := "# Heading\n\nA paragraph.\n\n- item one\n- item two"
	m := New(Config{Title: "test", Content: content})
	got := m.Content()
	if got != content {
		t.Fatalf("round-trip failed:\nwant: %q\ngot:  %q", content, got)
	}
}

func TestRoundTripChecklist(t *testing.T) {
	content := "- [ ] unchecked\n- [x] checked"
	m := New(Config{Title: "test", Content: content})
	got := m.Content()
	if got != content {
		t.Fatalf("round-trip failed:\nwant: %q\ngot:  %q", content, got)
	}
}

func TestRoundTripCodeBlock(t *testing.T) {
	content := "```go\nfmt.Println(\"hello\")\n```"
	m := New(Config{Title: "test", Content: content})
	got := m.Content()
	if got != content {
		t.Fatalf("round-trip failed:\nwant: %q\ngot:  %q", content, got)
	}
}

func TestInitialContentNotModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	if m.modified() {
		t.Fatal("new editor should not be marked as modified")
	}
}

func TestInitReturnsBlink(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil command for cursor blinking")
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

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+S should return a save command")
	}

	msg := cmd()
	if !saved {
		t.Fatal("save function should have been called")
	}
	if savedContent != "hello" {
		t.Fatalf("saved content should be %q, got %q", "hello", savedContent)
	}

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

func TestCtrlQQuitsWhenClean(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
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

func TestCtrlQShowsPromptWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Type a character to modify the active block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if !m.modified() {
		t.Fatal("editor should be modified after typing")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("Ctrl+Q on modified content should show prompt, not quit")
	}
	if !m.quitPrompt {
		t.Fatal("should show quit prompt")
	}
	if m.status == "" {
		t.Fatal("status should contain prompt message")
	}
}

func TestQuitPromptSaveAndQuit(t *testing.T) {
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

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("should show quit prompt")
	}

	// Press 'y' to save and quit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("pressing 'y' should return a save-then-quit command")
	}

	msg := cmd()
	if !saved {
		t.Fatal("save function should have been called")
	}
	if savedContent == "" {
		t.Fatal("saved content should not be empty")
	}

	updated, quitCmd := m.Update(msg)
	m = updated.(Model)

	if !m.quitting {
		t.Fatal("model should be in quitting state after save-then-quit")
	}
	if quitCmd == nil {
		t.Fatal("should return tea.Quit command")
	}
}

func TestQuitPromptDiscardAndQuit(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	// Press 'n' to quit without saving.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("pressing 'n' should return tea.Quit command")
	}
	if !m.quitting {
		t.Fatal("model should be in quitting state")
	}
}

func TestQuitPromptCancel(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("should show quit prompt")
	}

	// Press Esc to cancel.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.quitPrompt {
		t.Fatal("quit prompt should be dismissed after Esc")
	}
	if m.quitting {
		t.Fatal("should not be quitting after Esc")
	}
	if cmd != nil {
		t.Fatal("Esc should not return a command")
	}
	if m.status != "" {
		t.Fatal("status should be cleared after cancelling prompt")
	}
}

func TestQuitPromptIgnoresOtherKeys(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("quit prompt should be set")
	}

	// Type an unrelated character.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("quit prompt should remain active after pressing unrelated key")
	}
	if m.quitting {
		t.Fatal("should not be quitting")
	}
	if cmd != nil {
		t.Fatal("unrelated key in prompt should not return a command")
	}
}

func TestCtrlCForceQuitsWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+C should return a quit command")
	}
	if !m.quitting {
		t.Fatal("expected quitting to be true")
	}
}

func TestCtrlCQuitsWhenClean(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Ctrl+C should return tea.Quit command")
	}
	if !m.quitting {
		t.Fatal("model should be in quitting state")
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

func TestStatusBarContainsHelpHint(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(Model)

	view := m.View()
	if !containsPlainText(view, "Ctrl+G help") {
		t.Fatal("status bar should contain Ctrl+G help hint")
	}
}

func TestCtrlGTogglesHelp(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("help should not be visible initially")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("Ctrl+G should show help overlay")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("Ctrl+G should hide help overlay")
	}
}

func TestHelpDismissedByEsc(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("help should be visible")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("Esc should dismiss help overlay")
	}
}

func TestHelpViewContainsKeybindings(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	view := m.View()

	keybindings := []string{
		"Ctrl+S", "Save",
		"Ctrl+Q", "Quit",
		"Ctrl+C", "Force quit",
		"Ctrl+G", "Toggle this help",
		"Ctrl+K", "Cut block",
		"Ctrl+Y", "Paste line",
		"Ctrl+U", "Delete to line start",
		"Ctrl+D", "Toggle checkbox",
	}
	for _, kb := range keybindings {
		if !containsPlainText(view, kb) {
			t.Fatalf("help overlay should contain %q", kb)
		}
	}
}

func TestHelpBlocksOtherKeys(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	// Try to type while help is showing.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("typing should be blocked while help is showing")
	}
	if !m.showHelp {
		t.Fatal("help should still be showing")
	}
}

func TestCtrlDTogglesCheckbox(t *testing.T) {
	m := New(Config{Title: "test", Content: "- [ ] buy milk"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Toggle unchecked to checked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content := m.Content()
	if content != "- [x] buy milk" {
		t.Fatalf("expected %q, got %q", "- [x] buy milk", content)
	}

	// Toggle checked back to unchecked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content = m.Content()
	if content != "- [ ] buy milk" {
		t.Fatalf("expected %q, got %q", "- [ ] buy milk", content)
	}
}

func TestCtrlDNoOpOnNonCheckboxBlock(t *testing.T) {
	m := New(Config{Title: "test", Content: "just a normal line"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatalf("Ctrl+D on non-checkbox block should be no-op, got %q", m.Content())
	}
}

func TestNavigationBetweenBlocks(t *testing.T) {
	// Create content with multiple blocks.
	content := "# Title\n\nParagraph text"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Should start at block 0.
	if m.active != 0 {
		t.Fatalf("should start at block 0, got %d", m.active)
	}

	// Press down at last line of first block to move to next block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.active != 1 {
		t.Fatalf("down arrow at last line should move to block 1, got %d", m.active)
	}

	// Press up at first line of current block to move back.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.active != 0 {
		t.Fatalf("up arrow at first line should move to block 0, got %d", m.active)
	}
}

func TestNavigationDoesNotGoPastBounds(t *testing.T) {
	m := New(Config{Title: "test", Content: "single block"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Try to go up from block 0.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.active != 0 {
		t.Fatalf("should stay at block 0, got %d", m.active)
	}

	// Try to go down from the only block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.active != 0 {
		t.Fatalf("should stay at block 0, got %d", m.active)
	}
}

func TestCtrlKCutsBlock(t *testing.T) {
	content := "# Title\n\nParagraph\n\n- bullet"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()

	// Cut the first block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)

	if m.blockClip == nil {
		t.Fatal("block clipboard should not be nil after cut")
	}

	if m.BlockCount() >= blocksBefore {
		t.Fatalf("block count should decrease after cut: was %d, now %d", blocksBefore, m.BlockCount())
	}
}

func TestStatusBarShowsModifiedIndicator(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	view := m.View()
	if containsPlainText(view, "[modified]") {
		t.Fatal("should not show [modified] when content is unchanged")
	}

	// Modify content by typing.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	view = m.View()
	if !containsPlainText(view, "[modified]") {
		t.Fatal("should show [modified] after content is changed")
	}
}

func TestNewSetsInitialDimensions(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})

	if m.width != defaultWidth {
		t.Fatalf("initial width should be %d, got %d", defaultWidth, m.width)
	}
	if m.height != defaultHeight {
		t.Fatalf("initial height should be %d, got %d", defaultHeight, m.height)
	}
}

func TestCtrlEIsNoOp(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if m.quitting {
		t.Fatal("Ctrl+E should not cause quitting")
	}
	if m.showHelp {
		t.Fatal("Ctrl+E should not show help")
	}
	if m.quitPrompt {
		t.Fatal("Ctrl+E should not show quit prompt")
	}
}

func TestHelpContainsToggleCheckbox(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	view := m.View()

	if !containsPlainText(view, "Ctrl+D") {
		t.Fatal("help overlay should contain Ctrl+D keybinding")
	}
	if !containsPlainText(view, "Toggle checkbox") {
		t.Fatal("help overlay should contain 'Toggle checkbox' description")
	}
}

// containsPlainText checks if a string contains the target text,
// ignoring any ANSI escape sequences.
func containsPlainText(s, target string) bool {
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
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}

// Verify strings import is used.
var _ = strings.Contains
