package editor

import (
	"errors"
	"strings"
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

func TestCtrlQShowsPromptWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Type a character to modify the content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if !m.modified() {
		t.Fatal("editor should be modified after typing")
	}

	// Ctrl+Q should show prompt, not quit.
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

	// Execute the command to trigger save.
	msg := cmd()
	if !saved {
		t.Fatal("save function should have been called")
	}
	if savedContent == "" {
		t.Fatal("saved content should not be empty")
	}

	// Process the savedQuitMsg.
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

	// Type an unrelated character — should stay in prompt.
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

func TestPreviewVisibleByDefault(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	if !m.showPreview {
		t.Fatal("showPreview should be true by default")
	}
}

func TestCtrlPTogglesPreview(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if !m.showPreview {
		t.Fatal("preview should be visible initially")
	}

	// Toggle off.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(Model)

	if m.showPreview {
		t.Fatal("Ctrl+P should toggle preview off")
	}

	// Toggle back on.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(Model)

	if !m.showPreview {
		t.Fatal("Ctrl+P should toggle preview back on")
	}
	if cmd == nil {
		t.Fatal("toggling preview on should schedule a preview tick")
	}
}

func TestPreviewTickRendersContent(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Mark dirty and send tick with matching seq.
	m.previewDirty = true
	updated, _ = m.Update(previewTickMsg{seq: m.previewSeq})
	m = updated.(Model)

	if m.previewDirty {
		t.Fatal("previewDirty should be false after tick renders")
	}
	if m.preview == "" {
		t.Fatal("preview should contain rendered content after tick")
	}
}

func TestPreviewTickSkipsWhenNotDirty(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Ensure not dirty.
	m.previewDirty = false
	m.preview = "old"

	updated, _ = m.Update(previewTickMsg{seq: m.previewSeq})
	m = updated.(Model)

	if m.preview != "old" {
		t.Fatal("preview should not change when not dirty")
	}
}

func TestPreviewTickSkipsStaleSeq(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m.previewDirty = true
	staleSeq := m.previewSeq - 1

	updated, _ = m.Update(previewTickMsg{seq: staleSeq})
	m = updated.(Model)

	if !m.previewDirty {
		t.Fatal("stale tick should not trigger render")
	}
}

func TestViewContainsPreviewWhenVisible(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Trigger preview render.
	m.previewDirty = true
	updated, _ = m.Update(previewTickMsg{seq: m.previewSeq})
	m = updated.(Model)

	view := m.View()
	// The view should contain the border character when preview is visible.
	if !strings.Contains(view, "│") {
		t.Fatal("view should contain the vertical border when preview is visible")
	}
}

func TestViewFullWidthWhenPreviewHidden(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Toggle preview off.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(Model)

	view := m.View()
	// Without preview, there should be no border character.
	if strings.Contains(view, "│") {
		t.Fatal("view should not contain border when preview is hidden")
	}
}

func TestPreviewAutoHidesWhenNarrow(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello"})
	// Set width below minSplitWidth.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	m = updated.(Model)

	// Preview is logically on, but should not render in the view.
	if !m.showPreview {
		t.Fatal("showPreview should still be true (auto-hide is view-level)")
	}

	view := m.View()
	if strings.Contains(view, "│") {
		t.Fatal("view should not show preview border when terminal is too narrow")
	}
}

func TestStatusBarContainsPreviewHint(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(Model)

	view := m.View()
	if !containsPlainText(view, "Ctrl+E preview") {
		t.Fatal("status bar should contain Ctrl+E preview hint")
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

	// Press Ctrl+G to show help.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("Ctrl+G should show help overlay")
	}

	// Press Ctrl+G again to hide help.
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

	// Show help.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("help should be visible")
	}

	// Press Esc to dismiss.
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

	// Show help.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	view := m.View()

	keybindings := []string{
		"Ctrl+S", "Save",
		"Ctrl+Q", "Quit",
		"Ctrl+C", "Force quit",
		"Ctrl+P", "Toggle preview",
		"Ctrl+G", "Toggle this help",
		"Ctrl+K", "Cut line",
		"Ctrl+U", "Paste line",
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

	// Show help.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = updated.(Model)

	// Try to type — should be blocked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("typing should be blocked while help is showing")
	}
	if !m.showHelp {
		t.Fatal("help should still be showing")
	}
}

func TestCtrlKCutsLine(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nline2\nline3"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The textarea cursor starts at the end of the content (last line).
	// Cut the last line.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)

	if m.clipboard != "line3" {
		t.Fatalf("clipboard should be %q, got %q", "line3", m.clipboard)
	}

	content := m.Content()
	if strings.Contains(content, "line3") {
		t.Fatal("line3 should be removed from content after cut")
	}
	if !strings.Contains(content, "line1") || !strings.Contains(content, "line2") {
		t.Fatal("other lines should remain after cut")
	}
}

func TestCtrlUPastesLine(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nline2\nline3"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Cursor is at the end (line3). Cut it.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)

	if m.clipboard != "line3" {
		t.Fatalf("clipboard should be %q, got %q", "line3", m.clipboard)
	}

	// Now paste — should re-insert line3.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)

	content := m.Content()
	if !strings.Contains(content, "line3") {
		t.Fatal("pasted line should appear in content")
	}
	if !strings.Contains(content, "line1") || !strings.Contains(content, "line2") {
		t.Fatal("other lines should remain after paste")
	}
}

func TestCtrlUNopWithEmptyClipboard(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nline2"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	// Paste with empty clipboard — should be a no-op.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("paste with empty clipboard should not change content")
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

func TestCtrlETogglesPreviewMode(t *testing.T) {
	content := "- [ ] task one\n- [x] task two"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.previewMode {
		t.Fatal("preview mode should be off initially")
	}

	// Press Ctrl+E to enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if !m.previewMode {
		t.Fatal("Ctrl+E should activate preview mode")
	}
	if len(m.elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(m.elements))
	}

	// Press Ctrl+E again to exit preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if m.previewMode {
		t.Fatal("Ctrl+E should deactivate preview mode")
	}
}

func TestCtrlENoElementsShowsWarning(t *testing.T) {
	content := "just plain text"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Ctrl+E with no interactive elements.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if m.previewMode {
		t.Fatal("preview mode should not activate with no elements")
	}
	if m.status == "" {
		t.Fatal("should show a warning status when no elements found")
	}
}

func TestPreviewModeNavigatesElements(t *testing.T) {
	content := "- [ ] first\n- [ ] second\n- [ ] third"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if m.focusedElement != 0 {
		t.Fatalf("focused element should start at 0, got %d", m.focusedElement)
	}

	// Down arrow.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.focusedElement != 1 {
		t.Fatalf("focused element should be 1 after down, got %d", m.focusedElement)
	}

	// Down again.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.focusedElement != 2 {
		t.Fatalf("focused element should be 2 after second down, got %d", m.focusedElement)
	}

	// Down at end should stay at 2.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.focusedElement != 2 {
		t.Fatalf("focused element should stay at 2 when at end, got %d", m.focusedElement)
	}

	// Up arrow.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.focusedElement != 1 {
		t.Fatalf("focused element should be 1 after up, got %d", m.focusedElement)
	}

	// Up at beginning should stay at 0.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.focusedElement != 0 {
		t.Fatalf("focused element should stay at 0 when at beginning, got %d", m.focusedElement)
	}
}

func TestPreviewModeTabNavigation(t *testing.T) {
	content := "- [ ] first\n- [ ] second"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	// Tab moves forward.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if m.focusedElement != 1 {
		t.Fatalf("Tab should move to next element, got %d", m.focusedElement)
	}

	// Shift+Tab moves backward.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)

	if m.focusedElement != 0 {
		t.Fatalf("Shift+Tab should move to previous element, got %d", m.focusedElement)
	}
}

func TestPreviewModeToggleCheckbox(t *testing.T) {
	content := "- [ ] unchecked\n- [x] checked"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	// Press Enter to toggle first checkbox (unchecked -> checked).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !strings.Contains(m.Content(), "- [x] unchecked") {
		t.Fatalf("checkbox should be toggled to checked, got: %q", m.Content())
	}

	// Navigate to second checkbox.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Press Space to toggle second checkbox (checked -> unchecked).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)

	if !strings.Contains(m.Content(), "- [ ] checked") {
		t.Fatalf("checkbox should be toggled to unchecked, got: %q", m.Content())
	}
}

func TestPreviewModeEscReturnsToEdit(t *testing.T) {
	content := "- [ ] task"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	if !m.previewMode {
		t.Fatal("should be in preview mode")
	}

	// Press Esc to exit.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.previewMode {
		t.Fatal("Esc should exit preview mode")
	}
}

func TestPreviewModeStatusBar(t *testing.T) {
	content := "- [ ] task"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(Model)

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	view := m.View()

	if !containsPlainText(view, "PREVIEW") {
		t.Fatal("status bar should contain PREVIEW indicator in preview mode")
	}
	if !containsPlainText(view, "navigate") {
		t.Fatal("status bar should contain navigate hint in preview mode")
	}
	if !containsPlainText(view, "toggle/open") {
		t.Fatal("status bar should contain toggle/open hint in preview mode")
	}
}

func TestPreviewModeBlocksTyping(t *testing.T) {
	content := "- [ ] task"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	// Enter preview mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	// Try to type — should be blocked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("typing should be blocked in preview mode")
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
