package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/block"
)

func TestLinkPromptAcceptsBracketedPaste(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	m.blocks[0] = block.Block{Type: block.Link}
	m.textareas[0] = newTextareaForBlock(m.blocks[0], m.width)
	m.linkPrompt.open(0, linkPromptURL, "", false)

	updated, _ := m.Update(tea.PasteMsg{Content: "https://example.com\n"})
	m = updated.(Model)

	if got := m.linkPrompt.input.Value(); got != "https://example.com" {
		t.Fatalf("prompt value = %q, want pasted URL", got)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if got := m.blocks[0].Content; got != "https://example.com" {
		t.Fatalf("committed link content = %q, want pasted URL", got)
	}
}

func TestPickerAcceptsBracketedPaste(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	m.palette.open(0)

	updated, _ := m.Update(tea.PasteMsg{Content: "hea\n"})
	m = updated.(Model)

	if got := m.palette.Filter(); got != "hea" {
		t.Fatalf("palette filter = %q, want pasted filter", got)
	}
}

func TestBackspaceDeletesSelectedLinkBlock(t *testing.T) {
	m := New(Config{Title: "test", Content: "above\n\n[Example](https://example.com)\n\nbelow"})
	m.focusBlock(2)
	m.textareas[m.active].MoveToEnd()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() != 4 {
		t.Fatalf("block count = %d, want link block removed", m.BlockCount())
	}
	for _, b := range m.blocks {
		if b.Type == block.Link {
			t.Fatalf("link block was not deleted: %+v", m.blocks)
		}
	}
	if m.active != 1 {
		t.Fatalf("active block = %d, want previous block focused", m.active)
	}
}

func TestDeleteKeyDeletesSelectedLinkBlock(t *testing.T) {
	m := New(Config{Title: "test", Content: "above\n\n[Example](https://example.com)\n\nbelow"})
	m.focusBlock(2)
	m.textareas[m.active].MoveToEnd()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	m = updated.(Model)

	for _, b := range m.blocks {
		if b.Type == block.Link {
			t.Fatalf("link block was not deleted: %+v", m.blocks)
		}
	}
}

func TestBackspaceDeletesOnlyLinkToEmptyParagraph(t *testing.T) {
	m := New(Config{Title: "test", Content: "[Example](https://example.com)"})

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)

	if m.BlockCount() != 1 {
		t.Fatalf("block count = %d, want one empty paragraph", m.BlockCount())
	}
	if m.blocks[0].Type != block.Paragraph || m.blocks[0].Content != "" {
		t.Fatalf("block = %+v, want empty paragraph", m.blocks[0])
	}
}

func TestLinkPromptSupportsWordAndLineEditingKeys(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	m.linkPrompt.open(0, linkPromptURL, "alpha beta gamma", false)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModAlt})
	m = updated.(Model)
	if got := m.linkPrompt.input.Value(); got != "alpha beta " {
		t.Fatalf("alt+backspace value = %q, want previous word deleted", got)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	m = updated.(Model)
	if m.linkPrompt.input.Cursor() != len([]rune("alpha ")) {
		t.Fatalf("alt+left cursor = %d, want start of previous word", m.linkPrompt.input.Cursor())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
	m = updated.(Model)
	if m.linkPrompt.input.Cursor() != len([]rune("alpha beta")) {
		t.Fatalf("alt+right cursor = %d, want end of next word", m.linkPrompt.input.Cursor())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDelete, Mod: tea.ModAlt})
	m = updated.(Model)
	if got := m.linkPrompt.input.Value(); got != "alpha beta" {
		t.Fatalf("alt+delete value = %q, want next word deleted", got)
	}
}

func TestLinkPromptSupportsCommandStyleLineDeletes(t *testing.T) {
	m := New(Config{Title: "test", Content: ""})
	m.linkPrompt.open(0, linkPromptURL, "prefix suffix", false)
	m.linkPrompt.input.SetCursor(len([]rune("prefix")))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModSuper})
	m = updated.(Model)
	if got := m.linkPrompt.input.Value(); got != " suffix" {
		t.Fatalf("super+backspace value = %q, want text before cursor deleted", got)
	}
	if m.linkPrompt.input.Cursor() != 0 {
		t.Fatalf("super+backspace cursor = %d, want 0", m.linkPrompt.input.Cursor())
	}

	m.linkPrompt.open(0, linkPromptURL, "prefix suffix", false)
	m.linkPrompt.input.SetCursor(len([]rune("prefix")))
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(Model)
	if got := m.linkPrompt.input.Value(); got != "prefix" {
		t.Fatalf("ctrl+k value = %q, want text after cursor deleted", got)
	}

	m.linkPrompt.open(0, linkPromptURL, "prefix suffix", false)
	m.linkPrompt.input.SetCursor(len([]rune("prefix")))
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = updated.(Model)
	if got := m.linkPrompt.input.Value(); got != " suffix" {
		t.Fatalf("ctrl+u value = %q, want text before cursor deleted", got)
	}
}
