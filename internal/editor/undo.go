package editor

import (
	"charm.land/bubbles/v2/textarea"
	"github.com/oobagi/notebook/internal/block"
)

const maxUndoDepth = 100

// editorState captures a snapshot of the editor for undo/redo.
type editorState struct {
	blocks []block.Block
	active int
	row    int
	col    int
}

// undoStack is a bounded stack of editor snapshots.
type undoStack struct {
	items []editorState
}

func (s *undoStack) push(state editorState) {
	s.items = append(s.items, state)
	if len(s.items) > maxUndoDepth {
		s.items = s.items[len(s.items)-maxUndoDepth:]
	}
}

func (s *undoStack) pop() (editorState, bool) {
	if len(s.items) == 0 {
		return editorState{}, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

func (s *undoStack) clear() {
	s.items = s.items[:0]
}

func (s *undoStack) len() int {
	return len(s.items)
}

// captureState snapshots the current editor state.
func (m *Model) captureState() editorState {
	row, col := 0, 0
	if m.active >= 0 && m.active < len(m.textareas) {
		row = m.textareas[m.active].Line()
		col = m.textareas[m.active].Column()
	}
	blocks := make([]block.Block, len(m.blocks))
	copy(blocks, m.blocks)
	if m.active >= 0 && m.active < len(m.textareas) {
		blocks[m.active].Content = m.textareas[m.active].Value()
	}
	return editorState{
		blocks: blocks,
		active: m.active,
		row:    row,
		col:    col,
	}
}

// pushUndo saves the current state onto the undo stack and clears the redo stack.
func (m *Model) pushUndo() {
	m.undo.push(m.captureState())
	m.redo.clear()
}

// restoreState rebuilds the editor from a snapshot.
func (m *Model) restoreState(state editorState) {
	m.blocks = make([]block.Block, len(state.blocks))
	copy(m.blocks, state.blocks)

	m.textareas = make([]textarea.Model, len(m.blocks))
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	for i, b := range m.blocks {
		m.textareas[i] = newTextareaForBlock(b, w)
	}

	active := state.active
	if active < 0 || active >= len(m.blocks) {
		active = 0
	}
	m.active = active
	if len(m.textareas) > 0 {
		m.cursorCmd = m.textareas[m.active].Focus()
		m.textareas[m.active].SetRow(state.row)
		m.textareas[m.active].SetCursorColumn(state.col)
	}
	m.undoDirty = false
}

// performUndo restores the previous state. Returns true if undo was performed.
func (m *Model) performUndo() bool {
	if m.undo.len() == 0 {
		return false
	}
	m.redo.push(m.captureState())
	state, _ := m.undo.pop()
	m.restoreState(state)
	return true
}

// performRedo re-applies the next state. Returns true if redo was performed.
func (m *Model) performRedo() bool {
	if m.redo.len() == 0 {
		return false
	}
	m.undo.push(m.captureState())
	state, _ := m.redo.pop()
	m.restoreState(state)
	return true
}
