package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook/internal/block"
)

// helper to create an editor with the given markdown content.
func newTestEditor(content string) Model {
	return New(Config{
		Title:   "test",
		Content: content,
	})
}

// helper to send a key message and return the updated model.
func sendKey(m Model, key string) Model {
	var msg tea.KeyPressMsg
	switch key {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+z":
		msg = tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	case "ctrl+y":
		msg = tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
	case "ctrl+k":
		msg = tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}
	case "ctrl+x":
		msg = tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}
	case "alt+up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModAlt}
	case "alt+down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModAlt}
	default:
		if len(key) == 1 {
			msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
		}
	}
	result, _ := m.Update(msg)
	return result.(Model)
}

func TestUndoAfterEnter(t *testing.T) {
	m := newTestEditor("hello world")
	if len(m.blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(m.blocks))
	}

	// Move cursor to middle and press Enter to split.
	m.textareas[0].SetCursorColumn(5)
	m = sendKey(m, "enter")

	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks after split, got %d", len(m.blocks))
	}

	// Undo should rejoin.
	m = sendKey(m, "ctrl+z")
	if len(m.blocks) != 1 {
		t.Fatalf("expected 1 block after undo, got %d", len(m.blocks))
	}
	if m.textareas[0].Value() != "hello world" {
		t.Errorf("expected 'hello world', got %q", m.textareas[0].Value())
	}
}

func TestUndoAfterBackspaceMerge(t *testing.T) {
	// Use two bullet items so they parse as separate blocks.
	m := newTestEditor("- alpha\n- beta")
	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.blocks))
	}

	// Focus second block and position cursor at start.
	m.focusBlock(1)
	m.textareas[1].CursorStart()

	// Backspace at position 0 of a bullet converts to paragraph.
	m = sendKey(m, "backspace")

	if m.blocks[1].Type != block.Paragraph {
		t.Fatalf("expected Paragraph after backspace, got %v", m.blocks[1].Type)
	}

	// Undo should restore the bullet list type.
	m = sendKey(m, "ctrl+z")
	if m.blocks[1].Type != block.BulletList {
		t.Errorf("expected BulletList after undo, got %v", m.blocks[1].Type)
	}
}

func TestUndoAfterCutBlock(t *testing.T) {
	m := newTestEditor("# Title\n\nparagraph")
	originalContent := m.textareas[0].Value()

	m = sendKey(m, "ctrl+k")

	// First block was cut — remaining blocks should be fewer.
	if len(m.blocks) >= 3 {
		t.Fatalf("expected fewer blocks after cut, got %d", len(m.blocks))
	}

	m = sendKey(m, "ctrl+z")
	if len(m.blocks) != 3 {
		t.Fatalf("expected 3 blocks after undo, got %d", len(m.blocks))
	}
	if m.textareas[0].Value() != originalContent {
		t.Errorf("expected %q, got %q", originalContent, m.textareas[0].Value())
	}
}

func TestUndoAfterSwap(t *testing.T) {
	m := newTestEditor("- first\n- second")
	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.blocks))
	}

	first := m.textareas[0].Value()
	second := m.textareas[1].Value()

	m = sendKey(m, "alt+down")

	// After swap, order should be reversed.
	if m.textareas[0].Value() != second || m.textareas[1].Value() != first {
		t.Fatalf("swap did not work: [0]=%q [1]=%q", m.textareas[0].Value(), m.textareas[1].Value())
	}

	m = sendKey(m, "ctrl+z")
	if m.textareas[0].Value() != first || m.textareas[1].Value() != second {
		t.Errorf("undo did not restore order: [0]=%q [1]=%q", m.textareas[0].Value(), m.textareas[1].Value())
	}
}

func TestUndoAfterChecklistToggle(t *testing.T) {
	m := newTestEditor("- [ ] task")
	if m.blocks[0].Type != block.Checklist {
		t.Fatalf("expected Checklist, got %v", m.blocks[0].Type)
	}
	if m.blocks[0].Checked {
		t.Fatal("expected unchecked")
	}

	m = sendKey(m, "ctrl+x")
	if !m.blocks[0].Checked {
		t.Fatal("expected checked after toggle")
	}

	m = sendKey(m, "ctrl+z")
	if m.blocks[0].Checked {
		t.Error("expected unchecked after undo")
	}
}

func TestRedoAfterUndo(t *testing.T) {
	m := newTestEditor("hello world")
	m.textareas[0].SetCursorColumn(5)
	m = sendKey(m, "enter")

	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.blocks))
	}

	m = sendKey(m, "ctrl+z")
	if len(m.blocks) != 1 {
		t.Fatalf("expected 1 block after undo, got %d", len(m.blocks))
	}

	m = sendKey(m, "ctrl+y")
	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks after redo, got %d", len(m.blocks))
	}
}

func TestNewEditClearsRedoStack(t *testing.T) {
	m := newTestEditor("hello world")
	m.textareas[0].SetCursorColumn(5)
	m = sendKey(m, "enter")
	m = sendKey(m, "ctrl+z")

	// Now type something — should clear redo.
	m = sendKey(m, "a")
	m = sendKey(m, "ctrl+y")

	// Redo should show "Nothing to redo" since new edit cleared it.
	if m.redo.len() != 0 {
		t.Error("expected redo stack to be empty after new edit")
	}
}

func TestUndoEmptyStackShowsStatus(t *testing.T) {
	m := newTestEditor("hello")
	m = sendKey(m, "ctrl+z")

	if m.status != "Nothing to undo" {
		t.Errorf("expected 'Nothing to undo' status, got %q", m.status)
	}
}

func TestRedoEmptyStackShowsStatus(t *testing.T) {
	m := newTestEditor("hello")
	m = sendKey(m, "ctrl+y")

	if m.status != "Nothing to redo" {
		t.Errorf("expected 'Nothing to redo' status, got %q", m.status)
	}
}

func TestUndoMaxDepth(t *testing.T) {
	m := newTestEditor("start")

	// Push 101 mutations — should cap at 100.
	for i := 0; i < maxUndoDepth+1; i++ {
		m.textareas[0].SetCursorColumn(len(m.textareas[0].Value()))
		m = sendKey(m, "enter")
	}

	if m.undo.len() > maxUndoDepth {
		t.Errorf("expected max %d undo entries, got %d", maxUndoDepth, m.undo.len())
	}
}

func TestCharacterBatching(t *testing.T) {
	m := newTestEditor("- first\n- second")
	original := m.textareas[0].Value()

	// Type several characters in the first block.
	for _, ch := range "abc" {
		m = sendKey(m, string(ch))
	}

	// Check the undo stack has a pre-typing snapshot.
	if m.undo.len() != 1 {
		t.Fatalf("expected 1 undo entry (pre-typing), got %d", m.undo.len())
	}
	preTyping := m.undo.items[0]
	if preTyping.blocks[0].Content != original {
		t.Fatalf("pre-typing snapshot has wrong content: %q (expected %q)", preTyping.blocks[0].Content, original)
	}

	// Undo the typing directly (no structural mutation needed).
	m = sendKey(m, "ctrl+z")

	if m.textareas[0].Value() != original {
		t.Errorf("expected %q after undoing typing, got %q", original, m.textareas[0].Value())
	}
}

func TestUndoStackStruct(t *testing.T) {
	var s undoStack

	// Push and pop.
	s.push(editorState{active: 1})
	s.push(editorState{active: 2})
	if s.len() != 2 {
		t.Fatalf("expected len 2, got %d", s.len())
	}

	state, ok := s.pop()
	if !ok || state.active != 2 {
		t.Errorf("expected active=2, got %d", state.active)
	}

	// Clear.
	s.clear()
	if s.len() != 0 {
		t.Errorf("expected empty after clear, got %d", s.len())
	}

	// Pop from empty.
	_, ok = s.pop()
	if ok {
		t.Error("expected pop from empty to return false")
	}
}

func TestCursorRestoredAfterUndo(t *testing.T) {
	m := newTestEditor("hello world")

	// Place cursor at position 5.
	m.textareas[0].SetCursorColumn(5)

	// Split with Enter.
	m = sendKey(m, "enter")

	// Undo — cursor should be back at column 5, row 0.
	m = sendKey(m, "ctrl+z")

	row := m.textareas[m.active].Line()
	col := m.textareas[m.active].Column()
	if row != 0 || col != 5 {
		t.Errorf("expected cursor at (0, 5), got (%d, %d)", row, col)
	}
}
