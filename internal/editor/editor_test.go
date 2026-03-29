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

func TestCtrlCForceQuitsWhenModified(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Modify content.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Ctrl+C should force quit even with unsaved changes.
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

	// Ctrl+C with no changes should quit immediately.
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

func TestCtrlYPastesLine(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nline2\nline3"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Cursor is at the end (line3). Cut it.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)

	if m.clipboard != "line3" {
		t.Fatalf("clipboard should be %q, got %q", "line3", m.clipboard)
	}

	// Now paste with Ctrl+Y — should re-insert line3.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	m = updated.(Model)

	content := m.Content()
	if !strings.Contains(content, "line3") {
		t.Fatal("pasted line should appear in content")
	}
	if !strings.Contains(content, "line1") || !strings.Contains(content, "line2") {
		t.Fatal("other lines should remain after paste")
	}
}

func TestCtrlYNopWithEmptyClipboard(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nline2"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	// Paste with empty clipboard — should be a no-op.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("paste with empty clipboard should not change content")
	}
}

func TestCtrlUDeletesToLineStart(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The textarea cursor starts at the end of the content.
	// Ctrl+U should delete everything before the cursor on the current line.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)

	content := m.Content()
	if content != "" {
		t.Fatalf("Ctrl+U at end of line should delete entire line content, got %q", content)
	}
}

func TestCtrlUDeletesToLineStartMultiline(t *testing.T) {
	m := New(Config{Title: "test", Content: "line1\nhello world\nline3"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to line 1 (middle line). The cursor starts at the end
	// of the last line. Move up twice to get to line 0, then down once
	// to get to line 1.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	// Now we should be on line 1 ("hello world"). Ctrl+U should delete
	// everything before cursor on this line, leaving other lines intact.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)

	content := m.Content()
	if !strings.Contains(content, "line1") {
		t.Fatal("line1 should remain after Ctrl+U on a different line")
	}
	if !strings.Contains(content, "line3") {
		t.Fatal("line3 should remain after Ctrl+U on a different line")
	}
}

func TestCtrlUNoOpAtLineStart(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to the start of the line.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = updated.(Model)

	contentBefore := m.Content()

	// Ctrl+U at start of line should be a no-op.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatalf("Ctrl+U at start of line should be no-op, got %q", m.Content())
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

func TestNewSetsInitialDimensions(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})

	if m.width != defaultWidth {
		t.Fatalf("initial width should be %d, got %d", defaultWidth, m.width)
	}
	if m.height != defaultHeight {
		t.Fatalf("initial height should be %d, got %d", defaultHeight, m.height)
	}
}

func TestCursorVisibleBeforeWindowSizeMsg(t *testing.T) {
	// Create an editor with multi-line content. The cursor should be within the
	// visible area even before a WindowSizeMsg arrives.
	lines := make([]byte, 0, 500)
	for i := 0; i < 30; i++ {
		lines = append(lines, []byte("line\n")...)
	}
	m := New(Config{Title: "test", Content: string(lines)})

	// The view should not be empty — the initial dimensions make it renderable.
	view := m.View()
	if view == "" {
		t.Fatal("view should not be empty before WindowSizeMsg thanks to default dimensions")
	}
}

func TestCtrlEIsNoOp(t *testing.T) {
	// After removing preview mode, Ctrl+E should be a no-op (key falls through
	// to the textarea).
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)

	// Model should still be usable — not in any special mode.
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

func TestCtrlDTogglesCheckboxUncheckedToChecked(t *testing.T) {
	m := New(Config{Title: "test", Content: "- [ ] buy milk"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content := m.Content()
	if content != "- [x] buy milk" {
		t.Fatalf("expected %q, got %q", "- [x] buy milk", content)
	}
}

func TestCtrlDTogglesCheckboxCheckedToUnchecked(t *testing.T) {
	m := New(Config{Title: "test", Content: "- [x] buy milk"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content := m.Content()
	if content != "- [ ] buy milk" {
		t.Fatalf("expected %q, got %q", "- [ ] buy milk", content)
	}
}

func TestCtrlDNoOpOnNonCheckboxLine(t *testing.T) {
	m := New(Config{Title: "test", Content: "just a normal line"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	contentBefore := m.Content()

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatalf("Ctrl+D on non-checkbox line should be no-op, got %q", m.Content())
	}
}

func TestCtrlDTogglesAsteriskCheckbox(t *testing.T) {
	m := New(Config{Title: "test", Content: "* [ ] task one"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Toggle unchecked to checked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content := m.Content()
	if content != "* [x] task one" {
		t.Fatalf("expected %q, got %q", "* [x] task one", content)
	}

	// Toggle checked back to unchecked.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	content = m.Content()
	if content != "* [ ] task one" {
		t.Fatalf("expected %q, got %q", "* [ ] task one", content)
	}
}

func TestHelpContainsToggleCheckbox(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Show help.
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
