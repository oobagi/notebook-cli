package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
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
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
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
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	// Block 0 should be focused.
	if m.active != 0 {
		t.Fatalf("expected active block 0, got %d", m.active)
	}

	// The textarea height should match the content line count.
	ta := &m.textareas[m.active]
	value := ta.Value()
	expectedLines := strings.Count(value, "\n") + 1

	// For paragraphs, minLines is 1; for code/quote, minLines is 3.
	minLines := 1
	if m.blocks[m.active].Type == block.CodeBlock || m.blocks[m.active].Type == block.Quote {
		minLines = 3
	}
	if expectedLines < minLines {
		expectedLines = minLines
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
