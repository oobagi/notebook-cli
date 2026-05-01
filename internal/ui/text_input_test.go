package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTextInputEditingKeys(t *testing.T) {
	input := NewTextInput("alpha beta gamma")

	input.HandleKey(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModAlt})
	if got := input.Value(); got != "alpha beta " {
		t.Fatalf("alt+backspace value = %q, want previous word deleted", got)
	}

	input.HandleKey(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	if got := input.Cursor(); got != len([]rune("alpha ")) {
		t.Fatalf("alt+left cursor = %d, want start of previous word", got)
	}

	input.HandleKey(tea.KeyPressMsg{Code: tea.KeyDelete, Mod: tea.ModAlt})
	if got := input.Value(); got != "alpha  " {
		t.Fatalf("alt+delete value = %q, want next word deleted", got)
	}
}

func TestTextInputCommandStyleLineDeletes(t *testing.T) {
	input := NewTextInput("prefix suffix")
	input.SetCursor(len([]rune("prefix")))

	input.HandleKey(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModSuper})
	if got := input.Value(); got != " suffix" {
		t.Fatalf("super+backspace value = %q, want text before cursor deleted", got)
	}
	if got := input.Cursor(); got != 0 {
		t.Fatalf("super+backspace cursor = %d, want 0", got)
	}

	input.SetValue("prefix suffix")
	input.SetCursor(len([]rune("prefix")))
	input.HandleKey(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	if got := input.Value(); got != "prefix" {
		t.Fatalf("ctrl+k value = %q, want text after cursor deleted", got)
	}
}

func TestTextInputPasteSanitizesToSingleLine(t *testing.T) {
	input := NewTextInput("ab")
	input.SetCursor(1)

	input.InsertText("x\ny\tz\n")

	if got := input.Value(); got != "ax y zb" {
		t.Fatalf("value = %q, want sanitized paste inserted at cursor", got)
	}
	if got := input.Cursor(); got != len([]rune("ax y z")) {
		t.Fatalf("cursor = %d, want after inserted text", got)
	}
}
