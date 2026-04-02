package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oobagi/notebook/internal/block"
)

func TestSlashAtPos0OpensPalette(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// The first block is an empty paragraph with cursor at pos 0.
	if m.palette.visible {
		t.Fatal("palette should not be visible initially")
	}

	// Type "/" at position 0 of an empty block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if !m.palette.visible {
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
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if m.palette.visible {
		t.Fatal("palette should not open when / is typed mid-line")
	}
}

func TestPaletteFilterReducesList(t *testing.T) {
	p := newPalette()
	p.open(0)

	allCount := len(p.filtered)
	if allCount != len(p.items) {
		t.Fatalf("expected all items visible initially, got %d", allCount)
	}

	// Type "hea" to filter.
	p.addFilterRune('h')
	p.addFilterRune('e')
	p.addFilterRune('a')

	if len(p.filtered) >= allCount {
		t.Fatalf("filtered list should be smaller after typing 'hea', got %d", len(p.filtered))
	}

	// All filtered items should contain "hea" in their label.
	for _, idx := range p.filtered {
		item := p.items[idx]
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
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if !m.palette.visible {
		t.Fatal("palette should be visible")
	}

	// Move cursor down to "Heading 1" (index 1 in the default list).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Press Enter to select.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.palette.visible {
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
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if !m.palette.visible {
		t.Fatal("palette should be visible")
	}

	// Move cursor to a different item.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Press Esc to cancel.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.palette.visible {
		t.Fatal("palette should be closed after Esc")
	}

	if m.blocks[m.active].Type != originalType {
		t.Fatalf("block type should be unchanged after Esc, expected %d, got %d", originalType, m.blocks[m.active].Type)
	}
}

func TestPaletteContainsAllBlockTypes(t *testing.T) {
	items := defaultPaletteItems()

	expectedTypes := map[block.BlockType]bool{
		block.Paragraph:    true,
		block.Heading1:     true,
		block.Heading2:     true,
		block.Heading3:     true,
		block.BulletList:   true,
		block.NumberedList: true,
		block.Checklist:    true,
		block.CodeBlock:    true,
		block.Quote:        true,
		block.Divider:      true,
	}

	for _, item := range items {
		delete(expectedTypes, item.Type)
	}

	if len(expectedTypes) != 0 {
		t.Fatalf("palette is missing block types: %v", expectedTypes)
	}
}

func TestPaletteUpDownNavigation(t *testing.T) {
	p := newPalette()
	p.open(0)

	if p.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", p.cursor)
	}

	p.moveDown()
	if p.cursor != 1 {
		t.Fatalf("cursor should be 1 after moveDown, got %d", p.cursor)
	}

	p.moveDown()
	if p.cursor != 2 {
		t.Fatalf("cursor should be 2 after second moveDown, got %d", p.cursor)
	}

	p.moveUp()
	if p.cursor != 1 {
		t.Fatalf("cursor should be 1 after moveUp, got %d", p.cursor)
	}

	// Move up past the top.
	p.moveUp()
	p.moveUp()
	if p.cursor != 0 {
		t.Fatalf("cursor should not go below 0, got %d", p.cursor)
	}
}

func TestPaletteDividerClearsContent(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	// Type "div" to filter to Divider.
	for _, r := range "div" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	// Select divider.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	rendered := p.render(80)
	if rendered == "" {
		t.Fatal("palette render should not be empty when visible")
	}
}

func TestPaletteRenderEmptyState(t *testing.T) {
	p := newPalette()
	p.open(0)

	// Type a filter that matches nothing.
	for _, r := range "zzz" {
		p.addFilterRune(r)
	}

	if len(p.filtered) != 0 {
		t.Fatalf("expected no filtered items for 'zzz', got %d", len(p.filtered))
	}

	rendered := p.render(80)
	if rendered == "" {
		t.Fatal("palette should still render when filter has no matches")
	}

	if !strings.Contains(rendered, "No matches") {
		t.Fatalf("palette should show 'No matches' text, got %q", rendered)
	}

	if !strings.Contains(rendered, "/zzz") {
		t.Fatalf("palette should still show the filter input '/zzz', got %q", rendered)
	}
}

func TestPaletteBackspaceOnEmptyFilterCloses(t *testing.T) {
	p := newPalette()
	p.open(0)

	// Filter is empty, backspace should return false (close signal).
	if p.deleteFilterRune() {
		t.Fatal("deleteFilterRune should return false when filter is empty")
	}
}

func TestPaletteBackspaceRemovesFilterChar(t *testing.T) {
	p := newPalette()
	p.open(0)

	p.addFilterRune('h')
	p.addFilterRune('e')

	if p.filter != "he" {
		t.Fatalf("filter should be 'he', got %q", p.filter)
	}

	if !p.deleteFilterRune() {
		t.Fatal("deleteFilterRune should return true when filter is non-empty")
	}

	if p.filter != "h" {
		t.Fatalf("filter should be 'h' after backspace, got %q", p.filter)
	}
}

func TestPaletteBlocksTypingToTextarea(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Open palette.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	// Type characters while palette is open.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)

	// Characters should go to the palette filter, not the textarea.
	if m.palette.filter != "ab" {
		t.Fatalf("palette filter should be 'ab', got %q", m.palette.filter)
	}

	// Textarea should still be empty.
	if m.textareas[m.active].Value() != "" {
		t.Fatalf("textarea should remain empty while palette is open, got %q", m.textareas[m.active].Value())
	}
}

func TestSlashOnNonEmptyBlockDoesNotOpenPalette(t *testing.T) {
	m := New(Config{Title: "test", Content: "some text"})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Move cursor to position 0.
	m.textareas[m.active].CursorStart()

	// Type "/" at position 0 of a non-empty block.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if m.palette.visible {
		t.Fatal("palette should not open when block has content, even at position 0")
	}
}
