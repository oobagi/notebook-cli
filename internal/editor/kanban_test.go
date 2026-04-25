package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/block"
)

const sampleKanbanMD = "```kanban\n" +
	"## Todo\n" +
	"- A\n" +
	"- B\n" +
	"\n" +
	"## Doing\n" +
	"- C\n" +
	"\n" +
	"## Done\n" +
	"- [x] D\n" +
	"```"

// firstKanban returns the index of the first Kanban block in m, or -1.
func firstKanban(m Model) int {
	for i, b := range m.blocks {
		if b.Type == block.Kanban {
			return i
		}
	}
	return -1
}

// pressKey synthesizes a KeyPressMsg for tests via the model's Update path.
func pressKey(m Model, key string) Model {
	msg := keyMsgFromString(key)
	out, _ := m.Update(msg)
	return out.(Model)
}

// keyMsgFromString builds a tea.KeyPressMsg whose String() matches `key`.
// We map common test keys directly to KeyCode + ModMask so the resulting
// msg.String() matches the strings the editor's switch statements compare.
func keyMsgFromString(key string) tea.KeyPressMsg {
	switch key {
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "shift+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}
	case "shift+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	case "shift+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	case "shift+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "n":
		return tea.KeyPressMsg{Code: 'n', Text: "n"}
	case "p":
		return tea.KeyPressMsg{Code: 'p', Text: "p"}
	case "s":
		return tea.KeyPressMsg{Code: 's', Text: "s"}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "x":
		return tea.KeyPressMsg{Code: 'x', Text: "x"}
	case "z":
		return tea.KeyPressMsg{Code: 'z', Text: "z"}
	case "?":
		return tea.KeyPressMsg{Code: '?', Text: "?"}
	}
	// Generic single-char fallback.
	if len(key) == 1 {
		return tea.KeyPressMsg{Code: rune(key[0]), Text: key}
	}
	return tea.KeyPressMsg{}
}

func newKanbanEditor(t *testing.T) Model {
	t.Helper()
	m := New(Config{Title: "test", Content: sampleKanbanMD, Save: func(string) error { return nil }})
	// Window-size message wires up widths; tests exercise model methods that
	// don't require a real terminal, but column width math needs >0.
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no Kanban block parsed from sample")
	}
	if m.kanban == nil {
		t.Fatalf("kanban state not initialized")
	}
	return m
}

func TestKanbanInitialSelection(t *testing.T) {
	m := newKanbanEditor(t)
	if m.kanban.col != 0 {
		t.Errorf("initial col = %d, want 0", m.kanban.col)
	}
	if m.kanban.card != 0 {
		t.Errorf("initial card = %d, want 0", m.kanban.card)
	}
	if c := m.kanban.selectedCard(); c == nil || c.Text != "A" {
		t.Errorf("selected card = %+v, want A", c)
	}
}

func TestKanbanRightArrowMovesColumn(t *testing.T) {
	m := newKanbanEditor(t)
	m = pressKey(m, "right")
	if m.kanban.col != 1 {
		t.Errorf("col after right = %d, want 1", m.kanban.col)
	}
	if c := m.kanban.selectedCard(); c == nil || c.Text != "C" {
		t.Errorf("selected = %+v, want C", c)
	}
}

func TestKanbanShiftRightMovesCard(t *testing.T) {
	m := newKanbanEditor(t)
	// Move A to "Doing" column.
	m = pressKey(m, "shift+right")
	if m.kanban.col != 1 {
		t.Errorf("col after shift+right = %d, want 1", m.kanban.col)
	}
	doing := m.kanban.cols[1]
	if len(doing.Cards) != 2 {
		t.Errorf("Doing cards count = %d, want 2", len(doing.Cards))
	}
	if doing.Cards[len(doing.Cards)-1].Text != "A" {
		t.Errorf("last Doing card = %q, want A", doing.Cards[len(doing.Cards)-1].Text)
	}
	// Source column should now be missing A.
	for _, c := range m.kanban.cols[0].Cards {
		if c.Text == "A" {
			t.Errorf("A still in Todo column")
		}
	}
}

func TestKanbanShiftDownReorders(t *testing.T) {
	m := newKanbanEditor(t)
	m = pressKey(m, "shift+down")
	if m.kanban.col != 0 || m.kanban.card != 1 {
		t.Errorf("after shift+down: col=%d card=%d, want 0/1", m.kanban.col, m.kanban.card)
	}
	if m.kanban.cols[0].Cards[0].Text != "B" || m.kanban.cols[0].Cards[1].Text != "A" {
		t.Errorf("reordered = %+v, want [B, A]", m.kanban.cols[0].Cards)
	}
}

func TestKanbanPriorityCycle(t *testing.T) {
	m := newKanbanEditor(t)
	m = pressKey(m, "p")
	if c := m.kanban.selectedCard(); c == nil || c.Priority != block.PriorityLow {
		t.Errorf("after one p: priority = %v, want Low", c.Priority)
	}
	m = pressKey(m, "p")
	m = pressKey(m, "p")
	if c := m.kanban.selectedCard(); c == nil || c.Priority != block.PriorityHigh {
		t.Errorf("after three p: priority = %v, want High", c.Priority)
	}
	m = pressKey(m, "p")
	if c := m.kanban.selectedCard(); c == nil || c.Priority != block.PriorityNone {
		t.Errorf("after four p: priority = %v, want None", c.Priority)
	}
}

func TestKanbanAddCard(t *testing.T) {
	m := newKanbanEditor(t)
	before := len(m.kanban.cols[0].Cards)
	m = pressKey(m, "n")
	after := len(m.kanban.cols[0].Cards)
	if after != before+1 {
		t.Errorf("card count: %d → %d, want +1", before, after)
	}
	if !m.kanban.edit {
		t.Errorf("addCard should enter edit mode")
	}
}

func TestKanbanCopyCardToClipboard(t *testing.T) {
	m := newKanbanEditor(t)
	c := m.kanban.selectedCard()
	if c == nil || c.Text == "" {
		t.Fatalf("expected a non-empty card selected, got %+v", c)
	}
	out, cmd := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModAlt})
	m = out.(Model)
	if cmd == nil {
		t.Errorf("expected status-dismiss cmd, got nil")
	}
	// Copy succeeds via OSC52 even when pbcopy/xclip aren't available, so
	// the status should be the success message regardless of CI host.
	if m.status != "Card copied to clipboard" {
		t.Errorf("status = %q, want %q", m.status, "Card copied to clipboard")
	}
}

func TestKanbanCopyCardOnEmptyColumn(t *testing.T) {
	// Move into "Doing" then delete its only card so the column is empty
	// and no card is selected — alt+k should report nothing to copy.
	m := newKanbanEditor(t)
	m = pressKey(m, "right")
	m = pressKey(m, "backspace")
	if m.kanban.selectedCard() != nil {
		t.Fatalf("expected no card selected after deleting last card")
	}
	out, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModAlt})
	m = out.(Model)
	if m.status != "No card to copy" {
		t.Errorf("status = %q, want %q", m.status, "No card to copy")
	}
}

func TestKanbanDeleteCard(t *testing.T) {
	m := newKanbanEditor(t)
	before := len(m.kanban.cols[0].Cards)
	m = pressKey(m, "backspace")
	after := len(m.kanban.cols[0].Cards)
	if after != before-1 {
		t.Errorf("card count: %d → %d, want -1", before, after)
	}
	// Selection should still be valid.
	if c := m.kanban.selectedCard(); c == nil {
		t.Errorf("selection invalid after delete")
	}
}

func TestKanbanBackspaceOnEmptyColumnDeletesBoard(t *testing.T) {
	// Non-empty kanban (Doing has only "C"). Navigate selection to the
	// "Doing" column, delete C, leaving Doing empty. Pressing backspace
	// again on the now-empty column should delete the whole kanban.
	md := "para\n\n" + sampleKanbanMD
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no kanban parsed")
	}
	m.focusBlock(idx)
	if m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	// Move to "Doing" column (col index 1) and delete its only card.
	m = pressKey(m, "right")
	if m.kanban.col != 1 {
		t.Fatalf("expected col 1 after right, got %d", m.kanban.col)
	}
	m = pressKey(m, "backspace")
	if c := m.kanban.selectedCard(); c != nil {
		t.Fatalf("col should now be empty, got selection %+v", c)
	}
	beforeBlocks := len(m.blocks)
	// Backspace on an empty column → delete the whole kanban.
	m = pressKey(m, "backspace")
	if len(m.blocks) != beforeBlocks-1 {
		t.Errorf("blocks count: %d → %d, want -1", beforeBlocks, len(m.blocks))
	}
	if m.kanban != nil {
		t.Errorf("kanban state should be cleared after empty-col backspace")
	}
	for _, b := range m.blocks {
		if b.Type == block.Kanban {
			t.Errorf("kanban block still present after empty-col backspace")
		}
	}
}

func TestKanbanEnterOnEmptyColumnInsertsParagraphBelow(t *testing.T) {
	md := "para\n\n```kanban\n## Todo\n## Doing\n## Done\n```"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no kanban parsed")
	}
	m.active = idx
	if m.kanban == nil {
		m.kanban = newKanbanState(m.blocks[idx].Content)
	}
	beforeBlocks := len(m.blocks)
	m = pressKey(m, "enter")
	if len(m.blocks) != beforeBlocks+1 {
		t.Errorf("blocks count: %d → %d, want +1", beforeBlocks, len(m.blocks))
	}
	if m.active != idx+1 {
		t.Errorf("active block = %d, want %d", m.active, idx+1)
	}
	if m.blocks[idx+1].Type != block.Paragraph {
		t.Errorf("inserted block type = %v, want Paragraph", m.blocks[idx+1].Type)
	}
}

func TestKanbanDownAtLastCardInsertsParagraphBelow(t *testing.T) {
	// Single kanban block, navigate to last card, then down should insert a
	// paragraph after.
	md := "```kanban\n## Todo\n- only\n```"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 || m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	beforeBlocks := len(m.blocks)
	m = pressKey(m, "down")
	if len(m.blocks) != beforeBlocks+1 {
		t.Errorf("blocks count: %d → %d, want +1", beforeBlocks, len(m.blocks))
	}
	if m.blocks[idx+1].Type != block.Paragraph {
		t.Errorf("inserted block type = %v, want Paragraph", m.blocks[idx+1].Type)
	}
}

func TestKanbanOffsetsShiftOnInsertAndDelete(t *testing.T) {
	// Two paragraphs sandwiching a kanban; verify the kanban's saved
	// horizontal scroll offset stays attached to the kanban after we
	// insert a block before it (shifting indexes) and after we delete
	// a block before it (shifting back).
	md := "first\n\n" + sampleKanbanMD + "\n\nlast"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 1 {
		t.Fatalf("expected paragraph before kanban")
	}
	m.focusBlock(idx)
	m.kanban.colOffset = 2
	m.kanbanOffsets = map[int]int{idx: 2}

	// Insert a paragraph before the kanban; the kanban moves to idx+1
	// and the offset map key should shift with it.
	m.insertBlockBefore(idx, block.Block{Type: block.Paragraph})
	if got, ok := m.kanbanOffsets[idx+1]; !ok || got != 2 {
		t.Errorf("after insert: kanbanOffsets[%d] = (%d, %v), want (2, true). Map: %+v",
			idx+1, got, ok, m.kanbanOffsets)
	}

	// Delete the paragraph we just inserted; offset should shift back.
	m.deleteBlock(idx)
	if got, ok := m.kanbanOffsets[idx]; !ok || got != 2 {
		t.Errorf("after delete: kanbanOffsets[%d] = (%d, %v), want (2, true). Map: %+v",
			idx, got, ok, m.kanbanOffsets)
	}
}

func TestKanbanCancelEditAfterAddRestoresSelection(t *testing.T) {
	// Backlog has "A" and "B". Select "A", press n (adds new card after
	// A, enters edit), press Esc with no input. Selection should land
	// back on A, not on B.
	md := "```kanban\n## Todo\n- A\n- B\n```"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	if m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	m.kanban.col = 0
	m.kanban.card = 0 // on "A"
	m = pressKey(m, "n")
	if !m.kanban.edit {
		t.Fatalf("n should enter edit mode")
	}
	if len(m.kanban.cols[0].Cards) != 3 {
		t.Fatalf("addCard should add a card, got %d", len(m.kanban.cols[0].Cards))
	}
	m = pressKey(m, "esc")
	if m.kanban.edit {
		t.Errorf("esc should exit edit mode")
	}
	if len(m.kanban.cols[0].Cards) != 2 {
		t.Errorf("esc on empty new card should drop it; got %d cards", len(m.kanban.cols[0].Cards))
	}
	// Selection should be back on "A" (index 0), not "B" (index 1).
	if c := m.kanban.selectedCard(); c == nil || c.Text != "A" {
		t.Errorf("selection after cancel = %+v, want A", c)
	}
}

func TestKanbanShiftUpDownBlockedWhileAutoSort(t *testing.T) {
	// When auto-sort is on, manual within-column reorder is blocked
	// so the list doesn't appear to snap back mysteriously after the
	// next sort runs.
	md := "```kanban\n## Todo\n- !!! high\n- ! low\n- !! mid\n```"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = out.(Model)
	if m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	// Turn on sort.
	m = pressKey(m, "s")
	if !m.kanbanSortByPrio {
		t.Fatalf("s should enable sort")
	}
	// After sort, order is high, mid, low. Select first card (high).
	if c := m.kanban.cols[0].Cards[0]; c.Priority != block.PriorityHigh {
		t.Fatalf("first card should be high after sort, got %v", c.Priority)
	}
	beforeOrder := []block.Priority{}
	for _, c := range m.kanban.cols[0].Cards {
		beforeOrder = append(beforeOrder, c.Priority)
	}
	// Try shift+down — should be a no-op (status hint set instead).
	m = pressKey(m, "shift+down")
	for i, c := range m.kanban.cols[0].Cards {
		if c.Priority != beforeOrder[i] {
			t.Errorf("shift+down should be blocked while sort on; order changed at %d", i)
		}
	}
	if m.status == "" {
		t.Errorf("expected a status hint explaining the block")
	}
}

func TestKanbanPasteIntoCardEdit(t *testing.T) {
	// Bracketed-paste arrives as tea.PasteMsg. While editing a card,
	// the pasted content must land in the card's editTA, not be dropped.
	m := newKanbanEditor(t)
	m.kanban.col = 0
	m.kanban.card = 0
	m = pressKey(m, "enter") // enter edit mode
	if !m.kanban.edit {
		t.Fatalf("enter should put us in edit mode")
	}
	// Send a paste message.
	out, _ := m.Update(tea.PasteMsg{Content: "PASTED"})
	m = out.(Model)
	got := m.kanban.editTA.Value()
	if !strings.Contains(got, "PASTED") {
		t.Errorf("paste content not in editTA: %q", got)
	}
	// Commit and verify the card text picked up the paste.
	m = pressKey(m, "enter")
	if c := m.kanban.selectedCard(); c == nil || !strings.Contains(c.Text, "PASTED") {
		t.Errorf("after commit, card text = %+v, want it to contain PASTED", c)
	}
}

func TestKanbanSwallowsPlainPrintableKeys(t *testing.T) {
	// In selection mode, typing a plain printable character (one with no
	// modifier) must NOT mutate the underlying textarea — that would
	// corrupt the kanban's serialized content via undo and silently
	// discard the keystroke on next focusBlock.
	m := newKanbanEditor(t)
	idx := firstKanban(m)
	beforeContent := m.blocks[idx].Content
	beforeTAValue := m.textareas[idx].Value()
	// Press a key that has no kanban handler (e.g. 'z' or '?').
	m = pressKey(m, "z")
	m = pressKey(m, "?")
	if m.blocks[idx].Content != beforeContent {
		t.Errorf("kanban block content changed: %q → %q", beforeContent, m.blocks[idx].Content)
	}
	if m.textareas[idx].Value() != beforeTAValue {
		t.Errorf("kanban textarea was mutated by stray keys")
	}
}

func TestKanbanUpAtTopFirstBlockInsertsCleanParagraph(t *testing.T) {
	// Repro: kanban is the first (and only) block, user presses up.
	// A paragraph should be inserted before, but the paragraph must be
	// EMPTY — not contain the kanban's serialized content.
	m := New(Config{Title: "k", Content: sampleKanbanMD, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx != 0 {
		t.Fatalf("expected kanban at index 0, got %d", idx)
	}
	// Move selection to the top-left card so up triggers exit-up.
	if m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	m.kanban.col = 0
	m.kanban.card = 0

	beforeBlocks := len(m.blocks)
	m = pressKey(m, "up")
	if len(m.blocks) != beforeBlocks+1 {
		t.Fatalf("blocks count: %d → %d, want +1", beforeBlocks, len(m.blocks))
	}
	if m.blocks[0].Type != block.Paragraph {
		t.Errorf("inserted block type = %v, want Paragraph", m.blocks[0].Type)
	}
	if m.blocks[0].Content != "" {
		t.Errorf("inserted paragraph should be empty, got %q", m.blocks[0].Content)
	}
	if m.blocks[1].Type != block.Kanban {
		t.Errorf("kanban should now be at index 1, got %v", m.blocks[1].Type)
	}
	if !strings.Contains(m.blocks[1].Content, "## Todo") {
		t.Errorf("kanban content corrupted: %q", m.blocks[1].Content)
	}
}

func TestKanbanEntrySymmetric(t *testing.T) {
	// Both entries should land in the leftmost non-empty column,
	// only the card position differs (top → first, bottom → last).
	md := "para above\n\n" + sampleKanbanMD + "\n\npara below"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 || idx == 0 || idx == len(m.blocks)-1 {
		t.Fatalf("expected paragraphs on both sides of kanban, got idx=%d blocks=%d", idx, len(m.blocks))
	}

	// Enter from above: focus paragraph above, navigateDown.
	m.focusBlock(idx - 1)
	m.navigateDown()
	if m.active != idx {
		t.Fatalf("navigateDown active = %d, want %d", m.active, idx)
	}
	colTop, cardTop := m.kanban.col, m.kanban.card
	if cardTop != 0 {
		t.Errorf("entry from above card = %d, want 0", cardTop)
	}

	// Enter from below: focus paragraph below, navigateUp.
	m.focusBlock(idx + 1)
	m.navigateUp()
	if m.active != idx {
		t.Fatalf("navigateUp active = %d, want %d", m.active, idx)
	}
	colBot, cardBot := m.kanban.col, m.kanban.card
	if colBot != colTop {
		t.Errorf("inconsistent column on entry: top=%d bot=%d", colTop, colBot)
	}
	wantLast := len(m.kanban.cols[colBot].Cards) - 1
	if cardBot != wantLast {
		t.Errorf("entry from below card = %d, want %d (last)", cardBot, wantLast)
	}
}

func TestKanbanDeletingBlockBelowKeepsBoardRendered(t *testing.T) {
	// Repro: kanban followed by an empty paragraph. Focus paragraph,
	// press backspace to delete it. After deletion, active is the kanban
	// and m.kanban must be initialized so the board renders as a board,
	// not collapsed to a single raw line.
	md := sampleKanbanMD + "\n\n"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no kanban parsed")
	}
	// Focus the paragraph after the kanban.
	if idx+1 >= len(m.blocks) || m.blocks[idx+1].Type != block.Paragraph {
		t.Fatalf("expected empty paragraph after kanban, got blocks: %+v", m.blocks)
	}
	m.focusBlock(idx + 1)
	// Backspace on empty paragraph deletes it.
	m = pressKey(m, "backspace")
	if m.active != idx {
		t.Fatalf("active = %d, want %d (kanban)", m.active, idx)
	}
	if m.kanban == nil {
		t.Fatalf("kanban state should be initialized after delete-and-focus")
	}
	if len(m.kanban.cols) == 0 {
		t.Errorf("kanban should have columns after re-focus")
	}
	// Ensure block content is intact.
	if !strings.Contains(m.blocks[idx].Content, "## Todo") {
		t.Errorf("kanban content corrupted: %q", m.blocks[idx].Content)
	}
}

func TestKanbanParagraphBelowDoesNotMergeIntoBoard(t *testing.T) {
	// Non-empty paragraph below the kanban: backspace at column 0 should
	// NOT merge the paragraph text into the kanban content.
	md := sampleKanbanMD + "\n\nhello"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no kanban parsed")
	}
	// Find the "hello" paragraph after the kanban.
	helloIdx := -1
	for i := idx + 1; i < len(m.blocks); i++ {
		if m.blocks[i].Type == block.Paragraph && m.blocks[i].Content == "hello" {
			helloIdx = i
			break
		}
	}
	if helloIdx < 0 {
		t.Fatalf("hello paragraph not found, blocks: %+v", m.blocks)
	}
	m.focusBlock(helloIdx)
	// Move cursor to column 0 so backspace triggers the merge path.
	m.textareas[m.active].SetCursorColumn(0)
	beforeBlocks := len(m.blocks)
	m = pressKey(m, "backspace")
	// Block count should be unchanged — backspace at col 0 of a paragraph
	// preceding a Kanban should be a no-op (not merge into kanban).
	if len(m.blocks) > beforeBlocks {
		t.Errorf("blocks count grew: %d → %d", beforeBlocks, len(m.blocks))
	}
	// Kanban content must remain intact (no merged paragraph text).
	for _, b := range m.blocks {
		if b.Type == block.Kanban {
			if !strings.Contains(b.Content, "## Todo") {
				t.Errorf("kanban content corrupted: %q", b.Content)
			}
			if strings.Contains(b.Content, "hello") {
				t.Errorf("paragraph text leaked into kanban: %q", b.Content)
			}
		}
	}
}

func TestKanbanBackspaceOnEmptyDeletesBlock(t *testing.T) {
	// Build an editor where a paragraph precedes an empty kanban so
	// that deleting the kanban has somewhere to focus back to.
	md := "para text\n\n```kanban\n## Todo\n## Doing\n## Done\n```"
	m := New(Config{Title: "k", Content: md, Save: func(string) error { return nil }})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = out.(Model)
	idx := firstKanban(m)
	if idx < 0 {
		t.Fatalf("no kanban parsed")
	}
	// Focus the kanban block.
	m.active = idx
	if m.kanban == nil {
		// focusBlock initializes kanban; simulate by re-init.
		m.kanban = newKanbanState(m.blocks[idx].Content)
	}
	if !m.kanban.isEmpty() {
		t.Fatalf("expected empty board, got cards in some column")
	}
	before := len(m.blocks)
	m = pressKey(m, "backspace")
	if len(m.blocks) != before-1 {
		t.Errorf("blocks count: %d → %d, want -1", before, len(m.blocks))
	}
	if m.kanban != nil {
		t.Errorf("kanban state should be cleared after delete")
	}
}

func TestKanbanRoundTripThroughEditor(t *testing.T) {
	m := newKanbanEditor(t)
	got := m.Content()
	if !strings.Contains(got, "```kanban") || !strings.Contains(got, "## Todo") {
		t.Errorf("Content() lost kanban structure:\n%s", got)
	}
}

func TestKanbanContentReflectsMutation(t *testing.T) {
	m := newKanbanEditor(t)
	// Move A across to "Doing" and verify Content() shows the move.
	m = pressKey(m, "shift+right")
	out := m.Content()
	doingSection := strings.SplitN(strings.SplitN(out, "## Doing\n", 2)[1], "## Done", 2)[0]
	if !strings.Contains(doingSection, "- A") {
		t.Errorf("Content() missing moved A in Doing:\n%s", doingSection)
	}
}

func TestKanbanHorizontalScrollFollowsFocus(t *testing.T) {
	// 4 columns, narrow terminal — only some columns should be visible
	// at a time and the offset should slide as we navigate right.
	mdLines := []string{"```kanban"}
	for _, t := range []string{"Backlog", "Todo", "In Progress", "Done"} {
		mdLines = append(mdLines, "## "+t, "- card")
	}
	mdLines = append(mdLines, "```")
	md := strings.Join(mdLines, "\n")

	m := New(Config{Title: "k", Content: md})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = out.(Model)

	// Force a render so colOffset is computed.
	idx := firstKanban(m)
	_ = m.renderBlock(idx)

	startOffset := m.kanban.colOffset

	// Navigate right several times.
	m = pressKey(m, "right")
	m = pressKey(m, "right")
	m = pressKey(m, "right") // last column
	_ = m.renderBlock(idx)

	if m.kanban.col != 3 {
		t.Fatalf("col should be 3 after 3 rights, got %d", m.kanban.col)
	}
	if m.kanban.colOffset <= startOffset {
		t.Errorf("colOffset should advance to keep focus visible: start=%d after=%d",
			startOffset, m.kanban.colOffset)
	}

	// Navigate back to first.
	m = pressKey(m, "left")
	m = pressKey(m, "left")
	m = pressKey(m, "left")
	_ = m.renderBlock(idx)
	if m.kanban.colOffset != 0 {
		t.Errorf("colOffset should be 0 after returning to first column, got %d", m.kanban.colOffset)
	}
}

func TestKanbanSortByPriority(t *testing.T) {
	md := "```kanban\n## Todo\n- !! mid one\n- urgent later\n- !!! urgent first\n- ! low\n```"
	m := New(Config{Title: "k", Content: md})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = out.(Model)
	if m.kanban == nil {
		t.Fatalf("kanban not initialized")
	}
	// Toggle the sort.
	m = pressKey(m, "s")
	if !m.kanbanSortByPrio {
		t.Fatalf("toggle should enable sort")
	}
	cards := m.kanban.cols[0].Cards
	wantOrder := []block.Priority{block.PriorityHigh, block.PriorityMed, block.PriorityLow, block.PriorityNone}
	for i, want := range wantOrder {
		if cards[i].Priority != want {
			t.Errorf("card %d priority = %v, want %v (texts: %q)", i, cards[i].Priority, want, cards[i].Text)
		}
	}
}

// TestKanbanRenderDoesNotPanic exercises the active and inactive render
// paths to catch obvious layout / nil-deref bugs.
func TestKanbanRenderDoesNotPanic(t *testing.T) {
	m := newKanbanEditor(t)
	idx := firstKanban(m)
	_ = m.renderBlock(idx)
	// Force inactive render of the same block by switching the active
	// pointer somewhere else (simulate it).
	m.active = -1
	_ = m.renderBlock(idx)
}
