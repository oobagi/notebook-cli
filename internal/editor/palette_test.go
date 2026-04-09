package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/block"
)

func TestSlashAtPos0OpensPalette(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The first block is an empty paragraph with cursor at pos 0.
	if m.palette.Visible {
		t.Fatal("palette should not be visible initially")
	}

	// Type "/" at position 0 of an empty block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible after typing / at position 0 of an empty block")
	}
}

func TestSlashMidLineDoesNotOpenPalette(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Cursor is at end of "hello" (not pos 0) because textarea places cursor at end.
	// Move cursor to end to be sure.
	m.textareas[m.active].CursorEnd()

	// Type "/".
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if m.palette.Visible {
		t.Fatal("palette should not open when / is typed mid-line")
	}
}

func TestPaletteFilterReducesList(t *testing.T) {
	p := newPalette()
	p.open(0)

	allCount := p.FilteredCount()
	if allCount != len(p.Items()) {
		t.Fatalf("expected all items visible initially, got %d", allCount)
	}

	// Type "hea" to filter.
	p.AddFilterRune('h')
	p.AddFilterRune('e')
	p.AddFilterRune('a')

	if p.FilteredCount() >= allCount {
		t.Fatalf("filtered list should be smaller after typing 'hea', got %d", p.FilteredCount())
	}

	// All filtered items should contain "hea" in their label.
	for _, idx := range p.FilteredIndices() {
		item := p.Items()[idx].(paletteItem)
		if !containsPlainText(item.Label, "Heading") {
			t.Fatalf("filtered item %q should contain 'Heading'", item.Label)
		}
	}
}

func TestPaletteEnterAppliesSelectedType(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible")
	}

	// Move cursor down to "Heading 1" (index 1 in the default list).
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	// Press Enter to select.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.palette.Visible {
		t.Fatal("palette should be closed after Enter")
	}

	if m.blocks[m.active].Type != block.Heading1 {
		t.Fatalf("expected block type Heading1 (%d), got %d", block.Heading1, m.blocks[m.active].Type)
	}
}

func TestPaletteEscClosesWithoutChanges(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	originalType := m.blocks[m.active].Type

	// Open palette.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible")
	}

	// Move cursor to a different item.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	// Press Esc to cancel.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(Model)

	if m.palette.Visible {
		t.Fatal("palette should be closed after Esc")
	}

	if m.blocks[m.active].Type != originalType {
		t.Fatalf("block type should be unchanged after Esc, expected %d, got %d", originalType, m.blocks[m.active].Type)
	}
}

func TestPaletteContainsAllBlockTypes(t *testing.T) {
	items := defaultPaletteItems(-1)

	expectedTypes := map[block.BlockType]bool{
		block.Paragraph:      true,
		block.Heading1:       true,
		block.Heading2:       true,
		block.Heading3:       true,
		block.BulletList:     true,
		block.NumberedList:   true,
		block.Checklist:      true,
		block.CodeBlock:      true,
		block.Quote:          true,
		block.Divider:        true,
		block.DefinitionList: true,
	}

	for _, item := range items {
		pi := item.(paletteItem)
		delete(expectedTypes, pi.Type)
	}

	if len(expectedTypes) != 0 {
		t.Fatalf("palette is missing block types: %v", expectedTypes)
	}
}

func TestPaletteUpDownNavigation(t *testing.T) {
	p := newPalette()
	p.open(0)

	if p.Cursor() != 0 {
		t.Fatalf("cursor should start at 0, got %d", p.Cursor())
	}

	p.MoveDown()
	if p.Cursor() != 1 {
		t.Fatalf("cursor should be 1 after MoveDown, got %d", p.Cursor())
	}

	p.MoveDown()
	if p.Cursor() != 2 {
		t.Fatalf("cursor should be 2 after second MoveDown, got %d", p.Cursor())
	}

	p.MoveUp()
	if p.Cursor() != 1 {
		t.Fatalf("cursor should be 1 after MoveUp, got %d", p.Cursor())
	}

	// Move up past the top.
	p.MoveUp()
	p.MoveUp()
	if p.Cursor() != 0 {
		t.Fatalf("cursor should not go below 0, got %d", p.Cursor())
	}
}

func TestPaletteDividerClearsContent(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	// Type "div" to filter to Divider.
	for _, r := range "div" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(Model)
	}

	// Select divider.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.blocks[m.active].Type != block.Divider {
		t.Fatalf("expected Divider type, got %d", m.blocks[m.active].Type)
	}

	if m.blocks[m.active].Content != "" {
		t.Fatalf("divider content should be empty, got %q", m.blocks[m.active].Content)
	}
}

func TestPaletteRenderNotEmpty(t *testing.T) {
	p := newPalette()
	p.open(0)

	rendered := p.RenderFooter(80)
	if rendered == "" {
		t.Fatal("palette render should not be empty when visible")
	}
}

func TestPaletteRenderEmptyState(t *testing.T) {
	p := newPalette()
	p.open(0)

	// Type a filter that matches nothing.
	for _, r := range "zzz" {
		p.AddFilterRune(r)
	}

	if p.FilteredCount() != 0 {
		t.Fatalf("expected no filtered items for 'zzz', got %d", p.FilteredCount())
	}

	rendered := p.RenderFooter(80)
	if rendered == "" {
		t.Fatal("palette should still render when filter has no matches")
	}

	if !strings.Contains(rendered, "No matches") {
		t.Fatalf("palette should show 'No matches' text, got %q", rendered)
	}

	if !strings.Contains(rendered, "zzz") {
		t.Fatalf("palette should still show the filter input 'zzz', got %q", rendered)
	}
}

func TestPaletteBackspaceOnEmptyFilterCloses(t *testing.T) {
	p := newPalette()
	p.open(0)

	// Filter is empty, backspace should return false (close signal).
	if p.DeleteFilterRune() {
		t.Fatal("DeleteFilterRune should return false when filter is empty")
	}
}

func TestPaletteBackspaceRemovesFilterChar(t *testing.T) {
	p := newPalette()
	p.open(0)

	p.AddFilterRune('h')
	p.AddFilterRune('e')

	if p.Filter() != "he" {
		t.Fatalf("filter should be 'he', got %q", p.Filter())
	}

	if !p.DeleteFilterRune() {
		t.Fatal("DeleteFilterRune should return true when filter is non-empty")
	}

	if p.Filter() != "h" {
		t.Fatalf("filter should be 'h' after backspace, got %q", p.Filter())
	}
}

func TestPaletteBlocksTypingToTextarea(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	// Type characters while palette is open.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m = updated.(Model)

	// Characters should go to the palette filter, not the textarea.
	if m.palette.Filter() != "ab" {
		t.Fatalf("palette filter should be 'ab', got %q", m.palette.Filter())
	}

	// Textarea should still be empty.
	if m.textareas[m.active].Value() != "" {
		t.Fatalf("textarea should remain empty while palette is open, got %q", m.textareas[m.active].Value())
	}
}

func TestSlashAtPos0OnNonEmptyBlockOpensPalette(t *testing.T) {
	m := New(Config{Title: "test", Content: "some text"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to position 0.
	m.textareas[m.active].CursorStart()

	// Type "/" at position 0 of a non-empty block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should open at position 0 even when block has content")
	}

	// The "/" character must not be inserted into the textarea.
	if strings.Contains(m.textareas[m.active].Value(), "/") {
		t.Fatal("the '/' character should not be inserted into the textarea when palette opens")
	}
}

func TestContentPreservedAfterTypeChange(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello world"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to position 0 and open palette.
	m.textareas[m.active].CursorStart()
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible")
	}

	// Move to Heading 1 (second item) and select it.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if m.blocks[m.active].Type != block.Heading1 {
		t.Fatalf("expected block type Heading1, got %v", m.blocks[m.active].Type)
	}

	if m.textareas[m.active].Value() != "hello world" {
		t.Fatalf("content should be preserved after type change, got %q", m.textareas[m.active].Value())
	}
}

func TestDividerHiddenWhenBlockHasContent(t *testing.T) {
	m := New(Config{Title: "test", Content: "some content"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to position 0 and open palette.
	m.textareas[m.active].CursorStart()
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible")
	}

	// Verify Divider is not in the filtered list.
	for _, idx := range m.palette.FilteredIndices() {
		if m.palette.Items()[idx].(paletteItem).Type == block.Divider {
			t.Fatal("Divider should be hidden when the block has content")
		}
	}
}

func TestDividerShownWhenBlockIsEmpty(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette on empty block.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if !m.palette.Visible {
		t.Fatal("palette should be visible")
	}

	// Verify Divider IS in the filtered list.
	found := false
	for _, idx := range m.palette.FilteredIndices() {
		if m.palette.Items()[idx].(paletteItem).Type == block.Divider {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Divider should be shown when the block is empty")
	}
}

func TestPaletteShowsCurrentTypeIndicator(t *testing.T) {
	p := newPalette()
	p.openForBlock(0, block.Paragraph, false)

	rendered := p.RenderFooter(80)
	if !strings.Contains(rendered, "\u2713") {
		t.Fatal("palette should show a checkmark next to the current block type")
	}
}

func TestSlashMidTextInsertsNormally(t *testing.T) {
	m := New(Config{Title: "test", Content: "hello"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to end so it's not at position 0.
	m.textareas[m.active].CursorEnd()

	// Type "/".
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)

	if m.palette.Visible {
		t.Fatal("palette should not open when / is typed mid-text")
	}

	if !strings.Contains(m.textareas[m.active].Value(), "/") {
		t.Fatal("/ should be inserted normally when typed mid-text")
	}
}
