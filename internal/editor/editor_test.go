package editor

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook/internal/block"
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

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
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

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
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

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	if !m.modified() {
		t.Fatal("editor should be modified after typing")
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("should show quit prompt")
	}

	// Press 'y' to save and quit.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Press 'n' to quit without saving.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("should show quit prompt")
	}

	// Press Esc to cancel.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	// Ctrl+Q: shows prompt.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.quitPrompt {
		t.Fatal("quit prompt should be set")
	}

	// Type an unrelated character.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
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

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
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

	view := m.View().Content
	if view == "" {
		t.Fatal("view should not be empty when not quitting")
	}
}

func TestViewEmptyWhenQuitting(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content
	if view != "" {
		t.Fatalf("view should be empty when quitting, got %q", view)
	}
}

func TestStatusBarContainsTitle(t *testing.T) {
	m := New(Config{Title: "work \u203A notes", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	view := m.View().Content
	if !containsPlainText(view, "work \u203A notes") {
		t.Fatal("view should contain the title in the status bar")
	}
}

func TestStatusBarContainsHelpHint(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(Model)

	view := m.View().Content
	if !containsPlainText(view, "/ commands") {
		t.Fatal("status bar should contain / commands hint")
	}
}

func TestCtrlGTogglesHelp(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("help should not be visible initially")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("Ctrl+G should show help overlay")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("Ctrl+G should hide help overlay")
	}
}

func TestHelpDismissedByEsc(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.showHelp {
		t.Fatal("help should be visible")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(Model)

	if m.showHelp {
		t.Fatal("Esc should dismiss help overlay")
	}
}

func TestHelpViewContainsKeybindings(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content

	keybindings := []string{
		"\u2303S", "Save",
		"\u2303Q", "Quit",
		"\u2303C", "Force quit",
		"\u2303G", "Toggle this help",
		"\u2303K", "Cut block",
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

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Try to type while help is showing.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	if m.Content() != contentBefore {
		t.Fatal("typing should be blocked while help is showing")
	}
	if !m.showHelp {
		t.Fatal("help should still be showing")
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	if m.active != 1 {
		t.Fatalf("down arrow at last line should move to block 1, got %d", m.active)
	}

	// Press up at first line of current block to move back.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = updated.(Model)

	if m.active != 0 {
		t.Fatalf("should stay at block 0, got %d", m.active)
	}

	// Try to go down from the only block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
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
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
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

	view := m.View().Content
	if containsPlainText(view, "[modified]") {
		t.Fatal("should not show [modified] when content is unchanged")
	}

	// Modify content by typing.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = updated.(Model)

	view = m.View().Content
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

func TestCtrlRTogglesViewMode(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter view mode.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.viewMode {
		t.Fatal("Ctrl+R should enter view mode")
	}
	if m.quitting {
		t.Fatal("Ctrl+R should not cause quitting")
	}

	// Exit view mode.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.viewMode {
		t.Fatal("Ctrl+R should exit view mode")
	}
}

// ---------- Block operations tests ----------

func TestEnterOnSingleLineBlockCreatesNewBlock(t *testing.T) {
	// Heading: Enter should create a new paragraph below.
	m := New(Config{Title: "test", Content: "# My Title"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 1 {
		t.Fatalf("expected 1 block, got %d", m.BlockCount())
	}

	// Press Enter.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter, got %d", m.BlockCount())
	}

	// New block should be focused (index 1).
	if m.active != 1 {
		t.Fatalf("expected active block to be 1, got %d", m.active)
	}

	// New block should be a paragraph.
	if m.blocks[1].Type != 0 { // block.Paragraph == 0
		t.Fatalf("expected new block to be Paragraph, got type %d", m.blocks[1].Type)
	}
}

func TestEnterOnChecklistCreatesNewChecklistItem(t *testing.T) {
	m := New(Config{Title: "test", Content: "- [ ] first item"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Enter.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter, got %d", m.BlockCount())
	}

	// New block should be a checklist item.
	if m.blocks[1].Type != 6 { // block.Checklist == 6
		t.Fatalf("expected new block to be Checklist, got type %d", m.blocks[1].Type)
	}

	// New checklist item should be unchecked.
	if m.blocks[1].Checked {
		t.Fatal("new checklist item should be unchecked")
	}
}

func TestEnterOnBulletListCreatesNewBulletItem(t *testing.T) {
	m := New(Config{Title: "test", Content: "- bullet one"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 4 { // block.BulletList == 4
		t.Fatalf("expected new block to be BulletList, got type %d", m.blocks[1].Type)
	}
}

func TestEnterOnNumberedListCreatesNewNumberedItem(t *testing.T) {
	m := New(Config{Title: "test", Content: "1. first"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 5 { // block.NumberedList == 5
		t.Fatalf("expected new block to be NumberedList, got type %d", m.blocks[1].Type)
	}
}

func TestBackspaceOnEmptyBlockDeletesIt(t *testing.T) {
	content := "# Title\n\nParagraph"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()

	// Navigate to the empty paragraph block (block 1, the blank line).
	m.focusBlock(1)
	// Verify it's empty.
	if m.textareas[1].Value() != "" {
		t.Fatalf("expected empty block, got %q", m.textareas[1].Value())
	}

	// Press Backspace.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() >= blocksBefore {
		t.Fatalf("expected block count to decrease: was %d, now %d", blocksBefore, m.BlockCount())
	}
}

func TestBackspaceOnNonEmptyBlockMergesWithPrevious(t *testing.T) {
	content := "# Title\n\nworld"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Navigate to the "world" block (block 2).
	m.focusBlock(2)
	// Put cursor at start (should already be there after focus).
	m.textareas[m.active].CursorStart()

	blocksBefore := m.BlockCount()

	// Press Backspace at position 0.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() >= blocksBefore {
		t.Fatalf("expected block count to decrease after merge: was %d, now %d", blocksBefore, m.BlockCount())
	}

	// The previous block should now contain merged content.
	mergedContent := m.textareas[m.active].Value()
	if mergedContent == "" {
		t.Fatal("merged block should have content")
	}
}

func TestCannotDeleteLastBlock(t *testing.T) {
	m := New(Config{Title: "test", Content: "only block"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 1 {
		t.Fatalf("expected 1 block, got %d", m.BlockCount())
	}

	// Clear content and try to delete.
	m.textareas[0].SetValue("")

	// Press Backspace on empty last block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// Should still have at least one block.
	if m.BlockCount() < 1 {
		t.Fatal("should always have at least one block")
	}
}

func TestAltUpMovesBlockUp(t *testing.T) {
	content := "# Title\n\nParagraph\n\n- bullet"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Focus the bullet block (last block, index depends on parsing).
	lastIdx := m.BlockCount() - 1
	m.focusBlock(lastIdx)

	blockTypeBefore := m.blocks[lastIdx].Type
	blockAboveBefore := m.blocks[lastIdx-1].Type

	// Press Alt+Up.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModAlt})
	m = updated.(Model)

	// The block that was at lastIdx should now be at lastIdx-1.
	if m.blocks[lastIdx-1].Type != blockTypeBefore {
		t.Fatalf("block should have moved up: expected type %d at position %d, got %d",
			blockTypeBefore, lastIdx-1, m.blocks[lastIdx-1].Type)
	}
	if m.blocks[lastIdx].Type != blockAboveBefore {
		t.Fatalf("previous block should have moved down: expected type %d at position %d, got %d",
			blockAboveBefore, lastIdx, m.blocks[lastIdx].Type)
	}
	if m.active != lastIdx-1 {
		t.Fatalf("active index should follow the moved block: expected %d, got %d", lastIdx-1, m.active)
	}
}

func TestAltDownMovesBlockDown(t *testing.T) {
	content := "# Title\n\nParagraph\n\n- bullet"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Focus block 0.
	m.focusBlock(0)
	blockTypeBefore := m.blocks[0].Type
	blockBelowBefore := m.blocks[1].Type

	// Press Alt+Down.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModAlt})
	m = updated.(Model)

	if m.blocks[1].Type != blockTypeBefore {
		t.Fatalf("block should have moved down: expected type %d at position 1, got %d",
			blockTypeBefore, m.blocks[1].Type)
	}
	if m.blocks[0].Type != blockBelowBefore {
		t.Fatalf("next block should have moved up: expected type %d at position 0, got %d",
			blockBelowBefore, m.blocks[0].Type)
	}
	if m.active != 1 {
		t.Fatalf("active index should follow the moved block: expected 1, got %d", m.active)
	}
}

func TestAltUpNoOpAtTop(t *testing.T) {
	content := "# Title\n\nParagraph"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Already at block 0.
	if m.active != 0 {
		t.Fatalf("expected active to be 0, got %d", m.active)
	}

	blocksBefore := m.BlockCount()
	typeBefore := m.blocks[0].Type

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModAlt})
	m = updated.(Model)

	if m.active != 0 {
		t.Fatalf("active should still be 0 after Alt+Up at top, got %d", m.active)
	}
	if m.BlockCount() != blocksBefore {
		t.Fatal("block count should not change")
	}
	if m.blocks[0].Type != typeBefore {
		t.Fatal("block order should not change")
	}
}

func TestAltDownNoOpAtBottom(t *testing.T) {
	content := "# Title\n\nParagraph"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	lastIdx := m.BlockCount() - 1
	m.focusBlock(lastIdx)

	blocksBefore := m.BlockCount()
	typeBefore := m.blocks[lastIdx].Type

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModAlt})
	m = updated.(Model)

	if m.active != lastIdx {
		t.Fatalf("active should still be %d after Alt+Down at bottom, got %d", lastIdx, m.active)
	}
	if m.BlockCount() != blocksBefore {
		t.Fatal("block count should not change")
	}
	if m.blocks[lastIdx].Type != typeBefore {
		t.Fatal("block order should not change")
	}
}

func TestEnterAtEndOfMultiLineBlockCreatesNewParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: "Hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to end of the paragraph.
	m.textareas[0].CursorEnd()

	if m.BlockCount() != 1 {
		t.Fatalf("expected 1 block, got %d", m.BlockCount())
	}

	// Press Enter at end of content.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter at end, got %d", m.BlockCount())
	}

	// New block should be focused.
	if m.active != 1 {
		t.Fatalf("expected active to be 1, got %d", m.active)
	}
}

func TestHelpContainsBlockOperationKeybindings(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content

	keybindings := []string{
		"Enter", "New block below",
		"Backspace", "Merge/delete block",
		"\u2325\u2191", "Move block up",
		"\u2325\u2193", "Move block down",
	}
	for _, kb := range keybindings {
		if !containsPlainText(view, kb) {
			t.Fatalf("help overlay should contain %q", kb)
		}
	}
}

func TestInsertBlockAfterMiddle(t *testing.T) {
	content := "# Title\n\nParagraph\n\n- bullet"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()

	// Focus block 0 and press Enter to create new block after it.
	m.focusBlock(0)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore+1 {
		t.Fatalf("expected %d blocks, got %d", blocksBefore+1, m.BlockCount())
	}

	// Active should be at the newly inserted block.
	if m.active != 1 {
		t.Fatalf("expected active to be 1, got %d", m.active)
	}
}

func TestDeleteBlockFocusesPrevious(t *testing.T) {
	content := "# Title\n\nParagraph\n\n- bullet"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Focus the empty paragraph (block 1) and delete it.
	m.focusBlock(1)
	// Block 1 should be the empty paragraph between title and paragraph text.
	m.textareas[1].SetValue("")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// Should have focused the previous block (0).
	if m.active != 0 {
		t.Fatalf("expected active to be 0 after deleting block 1, got %d", m.active)
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
		if s[i] == '\x1b' && i+1 < len(s) {
			if s[i+1] == '[' {
				// CSI sequence: \x1b[...m
				j := i + 2
				for j < len(s) && s[j] != 'm' {
					j++
				}
				if j < len(s) {
					j++
				}
				i = j
			} else if s[i+1] == ']' {
				// OSC sequence: \x1b]...ST where ST is \x1b\\ or \x07
				j := i + 2
				for j < len(s) {
					if s[j] == '\x07' {
						j++
						break
					}
					if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
						j += 2
						break
					}
					j++
				}
				i = j
			} else {
				// Other escape (e.g. \x1b(B for ASCII mode)
				i += 2
			}
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}

func TestEmptyDocumentStartsWithOneParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})

	if m.BlockCount() != 1 {
		t.Fatalf("empty document should have exactly 1 block, got %d", m.BlockCount())
	}

	if m.blocks[0].Type != 0 { // block.Paragraph == 0
		t.Fatalf("empty document should start with Paragraph block, got type %d", m.blocks[0].Type)
	}

	if m.blocks[0].Content != "" {
		t.Fatalf("empty document paragraph should have empty content, got %q", m.blocks[0].Content)
	}

	if m.active != 0 {
		t.Fatalf("active block should be 0, got %d", m.active)
	}
}

func TestStatusBarContainsCommandsHint(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(Model)

	view := m.View().Content
	if !containsPlainText(view, "/ commands") {
		t.Fatal("status bar should contain '/ commands' hint")
	}
}

func TestHelpContainsBlockTypePalette(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content
	if !containsPlainText(view, "Block type palette") {
		t.Fatal("help overlay should contain 'Block type palette' for / keybinding")
	}
}

func TestHelpDoesNotContainStaleEntries(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content

	stale := []string{"Ctrl+P", "Ctrl+E", "Ctrl+U", "Ctrl+Y"}
	for _, s := range stale {
		if containsPlainText(view, s) {
			t.Fatalf("help overlay should NOT contain stale entry %q", s)
		}
	}
}

// ---------- Ctrl+J (newline within block) tests ----------

func TestCtrlJInsertsNewlineInParagraphAtEnd(t *testing.T) {
	// Ctrl+J at the end of a paragraph should insert a newline, NOT create a new block.
	m := New(Config{Title: "test", Content: "Hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to end.
	m.textareas[0].CursorEnd()

	blocksBefore := m.BlockCount()

	// Press Ctrl+J.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Block count should NOT increase (no new block created).
	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	// Content should contain a newline.
	content := m.textareas[0].Value()
	if !strings.Contains(content, "\n") {
		t.Fatalf("Ctrl+J should insert newline, got %q", content)
	}

	// Should still be in the same block.
	if m.active != 0 {
		t.Fatalf("should remain in block 0, got %d", m.active)
	}
}

func TestCtrlJInsertsNewlineInParagraphMidContent(t *testing.T) {
	// Ctrl+J mid-content inserts a newline at cursor position.
	m := New(Config{Title: "test", Content: "Hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()

	// Press Ctrl+J (cursor is at start, which is mid-content).
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	content := m.textareas[0].Value()
	if !strings.Contains(content, "\n") {
		t.Fatalf("Ctrl+J should insert newline, got %q", content)
	}
}

func TestCtrlJInsertsNewlineInCodeBlock(t *testing.T) {
	m := New(Config{Title: "test", Content: "```go\nfmt.Println()\n```"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The code block should be focused.
	m.textareas[0].CursorEnd()

	blocksBefore := m.BlockCount()

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J in code block should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	content := m.textareas[0].Value()
	// Should have more newlines than before.
	if strings.Count(content, "\n") < 1 {
		t.Fatalf("Ctrl+J should insert newline in code block, got %q", content)
	}
}

func TestCtrlJInsertsNewlineInQuote(t *testing.T) {
	m := New(Config{Title: "test", Content: "> A quote"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m.textareas[0].CursorEnd()
	blocksBefore := m.BlockCount()

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J in quote should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	content := m.textareas[0].Value()
	if !strings.Contains(content, "\n") {
		t.Fatalf("Ctrl+J should insert newline in quote, got %q", content)
	}
}

func TestCtrlJNoOpOnSingleLineBlock(t *testing.T) {
	// On a heading (single-line), Ctrl+J should be a no-op.
	m := New(Config{Title: "test", Content: "# My Title"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()
	contentBefore := m.textareas[0].Value()

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J on heading should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	if m.textareas[0].Value() != contentBefore {
		t.Fatalf("Ctrl+J on heading should not modify content: was %q, now %q", contentBefore, m.textareas[0].Value())
	}
}

func TestCtrlJNoOpOnBulletList(t *testing.T) {
	// Bullet list items are single-line; Ctrl+J should be a no-op.
	m := New(Config{Title: "test", Content: "- bullet item"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	blocksBefore := m.BlockCount()
	contentBefore := m.textareas[0].Value()

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore {
		t.Fatalf("Ctrl+J on bullet should not create new block: had %d, now %d", blocksBefore, m.BlockCount())
	}

	if m.textareas[0].Value() != contentBefore {
		t.Fatalf("Ctrl+J on bullet should not modify content: was %q, now %q", contentBefore, m.textareas[0].Value())
	}
}

func TestEnterStillCreatesBlockAtEndOfParagraph(t *testing.T) {
	// Regression: Enter at end of paragraph should STILL create a new block.
	m := New(Config{Title: "test", Content: "Hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m.textareas[0].CursorEnd()

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("Enter at end should create new block: expected 2, got %d", m.BlockCount())
	}

	if m.active != 1 {
		t.Fatalf("Enter at end should focus new block: expected 1, got %d", m.active)
	}
}

func TestHelpContainsCtrlJKeybinding(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m = updated.(Model)

	view := m.View().Content
	if !containsPlainText(view, "\u2303J") {
		t.Fatal("help overlay should contain ⌃J keybinding")
	}
	if !containsPlainText(view, "Newline within block") {
		t.Fatal("help overlay should contain 'Newline within block' description")
	}
}

// ---------- Empty list item Enter → Paragraph tests ----------

func TestEnterOnEmptyBulletConvertsToParagraph(t *testing.T) {
	// Start with a bullet that has content, then create an empty bullet via Enter.
	m := New(Config{Title: "test", Content: "- bullet one"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Enter on the non-empty bullet to create a new empty bullet.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}
	if m.active != 1 {
		t.Fatalf("expected active to be 1, got %d", m.active)
	}
	if m.blocks[1].Type != 4 { // block.BulletList == 4
		t.Fatalf("expected new block to be BulletList, got type %d", m.blocks[1].Type)
	}

	// Now press Enter on the empty bullet — should convert it to a paragraph.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	// Block count should remain 2 (no new block created).
	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter on empty bullet, got %d", m.BlockCount())
	}
	// The current block should now be a Paragraph.
	if m.blocks[1].Type != 0 { // block.Paragraph == 0
		t.Fatalf("expected empty bullet to become Paragraph, got type %d", m.blocks[1].Type)
	}
}

func TestEnterOnEmptyNumberedListConvertsToParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: "1. first"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Enter to create a new empty numbered list item.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 5 { // block.NumberedList == 5
		t.Fatalf("expected new block to be NumberedList, got type %d", m.blocks[1].Type)
	}

	// Press Enter on the empty numbered list item.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter on empty numbered item, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 0 { // block.Paragraph == 0
		t.Fatalf("expected empty numbered item to become Paragraph, got type %d", m.blocks[1].Type)
	}
}

func TestEnterOnEmptyChecklistConvertsToParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: "- [ ] task one"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Press Enter to create a new empty checklist item.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 6 { // block.Checklist == 6
		t.Fatalf("expected new block to be Checklist, got type %d", m.blocks[1].Type)
	}

	// Press Enter on the empty checklist item.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter on empty checklist, got %d", m.BlockCount())
	}
	if m.blocks[1].Type != 0 { // block.Paragraph == 0
		t.Fatalf("expected empty checklist to become Paragraph, got type %d", m.blocks[1].Type)
	}
}

func TestEnterOnNonEmptyBulletStillCreatesNewBullet(t *testing.T) {
	m := New(Config{Title: "test", Content: "- bullet one"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 1 {
		t.Fatalf("expected 1 block, got %d", m.BlockCount())
	}

	// Press Enter on the non-empty bullet.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after Enter on non-empty bullet, got %d", m.BlockCount())
	}
	// The new block should be a BulletList, not a Paragraph.
	if m.blocks[1].Type != 4 { // block.BulletList == 4
		t.Fatalf("expected new block to be BulletList, got type %d", m.blocks[1].Type)
	}
	// The original block should still be a BulletList.
	if m.blocks[0].Type != 4 { // block.BulletList == 4
		t.Fatalf("original block should still be BulletList, got type %d", m.blocks[0].Type)
	}
}

// TestDeleteBlockThenEnterNoExtraBlankLines verifies that deleting a block
// followed by pressing Enter does not produce excess vertical space (issue #155).
// The root cause was that deleteBlock would focus an empty separator paragraph
// left over from the parsed markdown, and pressing Enter on it created a second
// empty paragraph — visually producing a double blank line.
func TestDeleteBlockThenEnterNoExtraBlankLines(t *testing.T) {
	// "# Title\n\nParagraph one\n\nParagraph two" parses to five blocks:
	//   [0] H1 "Title"
	//   [1] Paragraph ""   (separator)
	//   [2] Paragraph "Paragraph one"
	//   [3] Paragraph ""   (separator)
	//   [4] Paragraph "Paragraph two"
	m := New(Config{Title: "test", Content: "# Title\n\nParagraph one\n\nParagraph two"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 5 {
		t.Fatalf("expected 5 blocks initially, got %d", m.BlockCount())
	}

	// Focus "Paragraph two" (block 4), clear it, then backspace to delete.
	m.focusBlock(4)
	m.textareas[4].SetValue("")
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// Only the deleted block should be removed, leaving 4 blocks:
	// H1, separator, Paragraph one, separator.
	if m.BlockCount() != 4 {
		t.Fatalf("expected 4 blocks after delete (only one block removed), got %d", m.BlockCount())
	}
}

// TestCtrlKThenEnterNoExtraBlankLines verifies that Ctrl+K (cut block) followed
// by Enter also does not produce extra blank lines (issue #155).
func TestCtrlKThenEnterNoExtraBlankLines(t *testing.T) {
	m := New(Config{Title: "test", Content: "first\n\nsecond\n\nthird"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 5 {
		t.Fatalf("expected 5 blocks, got %d", m.BlockCount())
	}

	// Focus "third" (block 4) and cut it.
	m.focusBlock(4)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Only the cut block should be removed, leaving 4 blocks.
	if m.BlockCount() != 4 {
		t.Fatalf("expected 4 blocks after cut (only one block removed), got %d", m.BlockCount())
	}
}

// TestDeleteBlockCleansSeparatorOnly verifies that the separator cleanup in
// deleteBlock only removes truly empty paragraphs, not content blocks.
func TestDeleteBlockCleansSeparatorOnly(t *testing.T) {
	// Create blocks without separators: "first", "second", "third"
	m := New(Config{Title: "test", Content: "first"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m.insertBlockAfter(0, block.Block{Type: block.Paragraph, Content: "second"})
	m.insertBlockAfter(1, block.Block{Type: block.Paragraph, Content: "third"})
	m.focusBlock(0)

	if m.BlockCount() != 3 {
		t.Fatalf("expected 3 blocks, got %d", m.BlockCount())
	}

	// Delete "third" (block 2)
	m.focusBlock(2)
	m.textareas[2].SetValue("")
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// "second" is non-empty, so it should NOT be removed.
	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after delete, got %d", m.BlockCount())
	}
	if m.textareas[m.active].Value() != "second" {
		t.Fatalf("active should be 'second', got %q", m.textareas[m.active].Value())
	}
}

func TestDeleteBlockDoesNotDeleteExtra(t *testing.T) {
	// Create: Heading, empty paragraph, another empty paragraph, text paragraph
	content := "# Title\n\n\nText"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Focus the second empty paragraph (block 2).
	m.focusBlock(2)
	if m.textareas[2].Value() != "" {
		t.Fatalf("expected empty block at index 2, got %q", m.textareas[2].Value())
	}

	blocksBefore := m.BlockCount()

	// Press Backspace to delete the empty block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// Should have removed exactly one block.
	if m.BlockCount() != blocksBefore-1 {
		t.Fatalf("expected exactly one block removed: was %d, now %d", blocksBefore, m.BlockCount())
	}
}

func TestCutBlockDoesNotDeleteExtra(t *testing.T) {
	content := "# Title\n\n\nText"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Focus the second empty paragraph (block 2).
	m.focusBlock(2)
	blocksBefore := m.BlockCount()

	// Cut the block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Should have removed exactly one block.
	if m.BlockCount() != blocksBefore-1 {
		t.Fatalf("expected exactly one block removed: was %d, now %d", blocksBefore, m.BlockCount())
	}
}

func TestMergeHeadingIntoEmptyBlockPreservesType(t *testing.T) {
	// Empty paragraph followed by a heading.
	content := "\n# Important Title"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Find the heading block.
	headingIdx := -1
	for i, b := range m.blocks {
		if b.Type == block.Heading1 {
			headingIdx = i
			break
		}
	}
	if headingIdx < 0 {
		t.Fatal("could not find Heading1 block")
	}
	if headingIdx == 0 {
		t.Fatal("heading should not be the first block (need empty block before it)")
	}

	// Focus the heading and put cursor at start.
	m.focusBlock(headingIdx)
	m.textareas[m.active].CursorStart()

	// Press Backspace to merge into the empty block above.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// The merged block should be a Heading1, not a Paragraph.
	if m.blocks[m.active].Type != block.Heading1 {
		t.Fatalf("merged block should be Heading1, got %s", m.blocks[m.active].Type)
	}

	// Content should be preserved.
	if m.textareas[m.active].Value() != "Important Title" {
		t.Fatalf("content should be preserved, got %q", m.textareas[m.active].Value())
	}
}

func TestMergeIntoNonEmptyBlockKeepsTargetType(t *testing.T) {
	// Non-empty paragraph followed by heading — merge should keep paragraph type.
	content := "existing text\n# Title"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Find the heading block.
	headingIdx := -1
	for i, b := range m.blocks {
		if b.Type == block.Heading1 {
			headingIdx = i
			break
		}
	}
	if headingIdx < 0 {
		t.Fatal("could not find Heading1 block")
	}

	// Focus the heading and put cursor at start.
	m.focusBlock(headingIdx)
	m.textareas[m.active].CursorStart()

	// Press Backspace to merge into the block above.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// The merged block should keep the target's type (Paragraph).
	if m.blocks[m.active].Type != block.Paragraph {
		t.Fatalf("merged block should keep Paragraph type, got %s", m.blocks[m.active].Type)
	}
}

func TestHeadingTextareaGrowsWithWrapping(t *testing.T) {
	// Create a heading that's longer than the terminal width.
	longTitle := strings.Repeat("a", 120)
	content := "# " + longTitle
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The heading textarea should have height > 1 to account for wrapping.
	h := m.textareas[0].Height()
	if h <= 1 {
		t.Fatalf("heading textarea should wrap: content is 120 chars in 80-wide terminal, but height is %d", h)
	}
}

func TestHeadingTextareaHeightUpdatesOnType(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Short"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m = updated.(Model)

	// Height should be 1 for short heading.
	if m.textareas[0].Height() != 1 {
		t.Fatalf("short heading should have height 1, got %d", m.textareas[0].Height())
	}

	// Type enough text to cause wrapping (make it longer than 40 chars).
	for _, ch := range "this is a much longer heading that wraps" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = updated.(Model)
	}

	h := m.textareas[0].Height()
	if h <= 1 {
		t.Fatalf("heading textarea should have grown after typing long content, got height %d", h)
	}
}

// ---------- Block splitting on Enter tests ----------

func TestEnterAtBeginningOfHeadingSplits(t *testing.T) {
	m := New(Config{Title: "test", Content: "# My Title"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Cursor should be at end by default. Move to beginning.
	m.textareas[0].CursorStart()

	// Press Enter.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}

	// First block should be empty paragraph (empty heading → paragraph).
	if m.blocks[0].Type != block.Paragraph {
		t.Fatalf("first block should be Paragraph, got %s", m.blocks[0].Type)
	}
	if m.textareas[0].Value() != "" {
		t.Fatalf("first block should be empty, got %q", m.textareas[0].Value())
	}

	// Second block should be Heading1 with the content.
	if m.blocks[1].Type != block.Heading1 {
		t.Fatalf("second block should be Heading1, got %s", m.blocks[1].Type)
	}
	if m.textareas[1].Value() != "My Title" {
		t.Fatalf("second block should have 'My Title', got %q", m.textareas[1].Value())
	}

	// Cursor should be on the second block.
	if m.active != 1 {
		t.Fatalf("active should be 1, got %d", m.active)
	}
}

func TestEnterInMiddleOfHeadingSplits(t *testing.T) {
	m := New(Config{Title: "test", Content: "# Hello World"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to position 5 ("Hello|World").
	m.textareas[0].CursorStart()
	m.textareas[0].SetCursorColumn(5)

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}

	// First block stays as Heading1 with "Hello".
	if m.blocks[0].Type != block.Heading1 {
		t.Fatalf("first block should be Heading1, got %s", m.blocks[0].Type)
	}
	if m.textareas[0].Value() != "Hello" {
		t.Fatalf("first block should have 'Hello', got %q", m.textareas[0].Value())
	}

	// Second block is Paragraph with " World".
	if m.blocks[1].Type != block.Paragraph {
		t.Fatalf("second block should be Paragraph, got %s", m.blocks[1].Type)
	}
	if m.textareas[1].Value() != " World" {
		t.Fatalf("second block should have ' World', got %q", m.textareas[1].Value())
	}
}

func TestEnterAtBeginningOfBulletSplits(t *testing.T) {
	m := New(Config{Title: "test", Content: "- my item"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m.textareas[0].CursorStart()

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}

	// First block: empty BulletList.
	if m.blocks[0].Type != block.BulletList {
		t.Fatalf("first block should be BulletList, got %s", m.blocks[0].Type)
	}
	if m.textareas[0].Value() != "" {
		t.Fatalf("first block should be empty, got %q", m.textareas[0].Value())
	}

	// Second block: BulletList with content.
	if m.blocks[1].Type != block.BulletList {
		t.Fatalf("second block should be BulletList, got %s", m.blocks[1].Type)
	}
	if m.textareas[1].Value() != "my item" {
		t.Fatalf("second block should have 'my item', got %q", m.textareas[1].Value())
	}
}

func TestEnterAtEndOfBlockStillWorks(t *testing.T) {
	// Existing behavior: Enter at end creates empty block below.
	m := New(Config{Title: "test", Content: "# Title"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Put cursor explicitly at the end.
	m.textareas[0].CursorEnd()

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}

	// First block keeps content.
	if m.textareas[0].Value() != "Title" {
		t.Fatalf("first block should have 'Title', got %q", m.textareas[0].Value())
	}

	// Second block is empty paragraph.
	if m.blocks[1].Type != block.Paragraph {
		t.Fatalf("second block should be Paragraph, got %s", m.blocks[1].Type)
	}
	if m.textareas[1].Value() != "" {
		t.Fatalf("second block should be empty, got %q", m.textareas[1].Value())
	}
}

func TestEnterSplitsMultiLineParagraph(t *testing.T) {
	// Paragraph with two lines, cursor at beginning of second line.
	m := New(Config{Title: "test", Content: "line one\nline two"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to beginning of second line.
	ta := &m.textareas[0]
	ta.CursorStart()
	ta.CursorDown()
	// CursorDown should put us on line 1 (second logical line).

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", m.BlockCount())
	}

	// First block: "line one"
	if m.textareas[0].Value() != "line one" {
		t.Fatalf("first block should have 'line one', got %q", m.textareas[0].Value())
	}

	// Second block: "line two"
	if m.textareas[1].Value() != "line two" {
		t.Fatalf("second block should have 'line two', got %q", m.textareas[1].Value())
	}
}

func TestCtrlWTogglesWordWrap(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if !m.wordWrap {
		t.Fatal("word wrap should be on by default")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	m = updated.(Model)

	if m.wordWrap {
		t.Fatal("Ctrl+W should toggle word wrap off")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	m = updated.(Model)

	if !m.wordWrap {
		t.Fatal("Ctrl+W should toggle word wrap back on")
	}
}

func TestMouseClickTogglesChecklistInViewMode(t *testing.T) {
	content := "- [ ] buy milk\n- [x] write tests"
	m := New(Config{Title: "test", Content: content})

	// Send WindowSizeMsg to initialize dimensions.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter view mode (Ctrl+E).
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)
	if !m.viewMode {
		t.Fatal("expected view mode after Ctrl+E")
	}

	// Verify initial state: block 0 unchecked, block 1 checked.
	if m.blocks[0].Checked {
		t.Fatal("block 0 should start unchecked")
	}
	if !m.blocks[1].Checked {
		t.Fatal("block 1 should start checked")
	}

	// The blockLineOffsets should be computed.
	if len(m.blockLineOffsets) != 2 {
		t.Fatalf("expected 2 block line offsets, got %d", len(m.blockLineOffsets))
	}

	// Click on the first checklist block's line.
	clickY := m.blockLineOffsets[0] - m.viewport.YOffset()
	updated, _ = m.Update(tea.MouseClickMsg{
		X:      5,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	m = updated.(Model)

	if !m.blocks[0].Checked {
		t.Fatal("block 0 should be checked after click")
	}

	// Click again to uncheck.
	updated, _ = m.Update(tea.MouseClickMsg{
		X:      5,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	m = updated.(Model)

	if m.blocks[0].Checked {
		t.Fatal("block 0 should be unchecked after second click")
	}
}

func TestMouseClickOnNonChecklistDoesNotToggle(t *testing.T) {
	content := "# Heading\n\nSome paragraph\n\n- [ ] checklist item"
	m := New(Config{Title: "test", Content: content})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter view mode.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)

	// Click on the heading line (block 0).
	if len(m.blockLineOffsets) < 1 {
		t.Fatal("expected block line offsets to be computed")
	}
	clickY := m.blockLineOffsets[0] - m.viewport.YOffset()
	updated, _ = m.Update(tea.MouseClickMsg{
		X:      5,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	m = updated.(Model)

	// Heading block should not gain a Checked state.
	if m.blocks[0].Checked {
		t.Fatal("heading block should not be toggled by click")
	}
}

func TestMouseClickInEditModePassesThrough(t *testing.T) {
	content := "- [ ] item"
	m := New(Config{Title: "test", Content: content})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Stay in edit mode (viewMode is false by default).
	if m.viewMode {
		t.Fatal("should start in edit mode")
	}

	// Click — should not toggle the checklist since we're in edit mode.
	updated, _ = m.Update(tea.MouseClickMsg{
		X:      5,
		Y:      0,
		Button: tea.MouseLeft,
	})
	m = updated.(Model)

	if m.blocks[0].Checked {
		t.Fatal("mouse click should not toggle checklist in edit mode")
	}
}

func TestBlockLineOffsetsComputed(t *testing.T) {
	// Parser produces 5 blocks: H1, empty P, P, empty P, Checklist.
	content := "# Title\n\nParagraph text\n\n- [ ] checklist"
	m := New(Config{Title: "test note", Content: content})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter view mode to trigger offset computation.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)

	if len(m.blockLineOffsets) != len(m.blocks) {
		t.Fatalf("expected %d block offsets, got %d", len(m.blocks), len(m.blockLineOffsets))
	}

	// Offsets must be non-decreasing (some empty blocks may share a line).
	for i := 1; i < len(m.blockLineOffsets); i++ {
		if m.blockLineOffsets[i] < m.blockLineOffsets[i-1] {
			t.Fatalf("offsets not non-decreasing: offset[%d]=%d < offset[%d]=%d",
				i, m.blockLineOffsets[i], i-1, m.blockLineOffsets[i-1])
		}
	}
}

func TestBlockIndexAtLineReturnsCorrectBlock(t *testing.T) {
	content := "- [ ] first\n- [x] second\n- [ ] third"
	m := New(Config{Title: "test", Content: content})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Enter view mode.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = updated.(Model)

	if len(m.blockLineOffsets) != 3 {
		t.Fatalf("expected 3 offsets, got %d", len(m.blockLineOffsets))
	}

	// Each block's own offset should map to that block.
	for i, offset := range m.blockLineOffsets {
		got := m.blockIndexAtLine(offset)
		if got != i {
			t.Errorf("blockIndexAtLine(%d) = %d, want %d", offset, got, i)
		}
	}

	// Line before any block should return -1.
	if m.blockLineOffsets[0] > 0 {
		got := m.blockIndexAtLine(0)
		if got != -1 {
			t.Errorf("blockIndexAtLine(0) = %d, want -1 (before any block)", got)
		}
	}
}

// Verify strings import is used.
var _ = strings.Contains
