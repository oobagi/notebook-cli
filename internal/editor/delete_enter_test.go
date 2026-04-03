package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook/internal/block"
)

// TestDeleteBlockThenEnterCreatesNewBlock verifies that after deleting a block,
// pressing Enter on the focused block creates a new block (not a newline
// insertion). This is the core regression test for issue #155.
func TestDeleteBlockThenEnterCreatesNewBlock(t *testing.T) {
	// Set up: heading, empty paragraph, another paragraph.
	content := "# Title\n\nSome text"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() != 3 {
		t.Fatalf("expected 3 blocks, got %d", m.BlockCount())
	}

	// Focus the empty paragraph (block 1) and delete it via Backspace.
	m.focusBlock(1)
	m.textareas[1].SetValue("")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() != 2 {
		t.Fatalf("expected 2 blocks after delete, got %d", m.BlockCount())
	}

	// Now the heading (block 0) should be focused. Move cursor to end.
	if m.active != 0 {
		t.Fatalf("expected active block 0, got %d", m.active)
	}

	// Press Enter — should create a new block, not insert a newline.
	blocksBefore := m.BlockCount()
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore+1 {
		t.Fatalf("Enter after delete should create a new block: expected %d, got %d",
			blocksBefore+1, m.BlockCount())
	}
}

// TestDeleteBlockResetsStaleRowCursor verifies that when a multi-line paragraph
// has its cursor on row 0 from a prior editing session, deleting an adjacent
// block resets the cursor to the last line. A subsequent Enter should then
// create a new block rather than inserting a newline.
func TestDeleteBlockResetsStaleRowCursor(t *testing.T) {
	// Set up: a multi-line paragraph followed by an empty paragraph.
	content := "line one\nline two\nline three\n\n"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", m.BlockCount())
	}

	// Focus the second block (empty paragraph) and delete it via Backspace.
	m.focusBlock(1)
	m.textareas[1].SetValue("")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// After delete, block 0 (the multi-line paragraph) should be focused.
	if m.active != 0 {
		t.Fatalf("expected active block 0, got %d", m.active)
	}

	// The cursor should be on the last line after deleteBlock resets state.
	ta := &m.textareas[m.active]
	value := ta.Value()
	lineCount := strings.Count(value, "\n") + 1
	cursorLine := ta.Line()

	if cursorLine < lineCount-1 {
		t.Fatalf("cursor should be on last line (%d), but is on line %d",
			lineCount-1, cursorLine)
	}

	// Press Enter — should create a new block.
	blocksBefore := m.BlockCount()
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.BlockCount() != blocksBefore+1 {
		t.Fatalf("Enter should create a new block: expected %d, got %d",
			blocksBefore+1, m.BlockCount())
	}
}

// TestDeleteBlockRecalculatesHeight verifies that after deleting a block, the
// textarea height of the newly focused block matches its content.
func TestDeleteBlockRecalculatesHeight(t *testing.T) {
	// Set up: a 3-line paragraph followed by an empty paragraph.
	content := "line one\nline two\nline three\n\n"
	m := New(Config{Title: "test", Content: content})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	if m.BlockCount() < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", m.BlockCount())
	}

	// Focus the second block (empty paragraph) and delete it.
	m.focusBlock(1)
	m.textareas[1].SetValue("")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	// Block 0 should be focused.
	if m.active != 0 {
		t.Fatalf("expected active block 0, got %d", m.active)
	}

	// The textarea height should match the content line count.
	ta := &m.textareas[m.active]
	value := ta.Value()
	expectedLines := strings.Count(value, "\n") + 1

	// All block types use minLines = 1.
	if expectedLines < 1 {
		expectedLines = 1
	}

	// Read the height by checking the textarea's view line count.
	// We can verify via the rendered output that it has the correct number of lines.
	taView := ta.View()
	renderedLines := strings.Count(taView, "\n") + 1

	// The rendered view should have at least expectedLines lines (textarea may
	// include prompt/chrome lines, so we check >= rather than ==).
	if renderedLines < expectedLines {
		t.Fatalf("textarea should have at least %d rendered lines, got %d",
			expectedLines, renderedLines)
	}
}

// TestDeleteLastNonParagraphBlockRevertsToParagraph verifies that pressing
// backspace on the only block when it is a non-paragraph type (e.g., numbered
// list) converts it to an empty paragraph rather than doing nothing.
func TestDeleteLastNonParagraphBlockRevertsToParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Manually set block 0 to a NumberedList with empty content.
	m.blocks[0] = block.Block{Type: block.NumberedList, Content: ""}
	m.textareas[0].SetValue("")
	m.focusBlock(0)

	if m.blocks[0].Type != block.NumberedList {
		t.Fatalf("expected NumberedList, got %s", m.blocks[0].Type)
	}

	// Press Backspace on the empty numbered list block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() != 1 {
		t.Fatalf("expected 1 block, got %d", m.BlockCount())
	}

	if m.blocks[0].Type != block.Paragraph {
		t.Fatalf("expected Paragraph after backspace, got %s", m.blocks[0].Type)
	}
}
