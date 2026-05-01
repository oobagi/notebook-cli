package editor

import "github.com/oobagi/notebook-cli/internal/ui"

// linkPromptField identifies which field a link prompt is editing.
type linkPromptField int

const (
	linkPromptURL linkPromptField = iota
	linkPromptTitle
)

// linkPromptState is a single-line input footer for editing one field of a
// Link block. Two prompts run sequentially when inserting (URL then title);
// keybinds re-open one field at a time for edits.
type linkPromptState struct {
	visible  bool
	field    linkPromptField
	chain    bool // true when committing the URL should auto-open the title prompt
	blockIdx int
	input    ui.TextInput
}

func (s *linkPromptState) open(idx int, field linkPromptField, initial string, chain bool) {
	s.visible = true
	s.field = field
	s.chain = chain
	s.blockIdx = idx
	s.input.SetValue(initial)
}

func (s *linkPromptState) close() {
	s.visible = false
	s.field = linkPromptURL
	s.chain = false
	s.blockIdx = -1
	s.input.Reset()
}

// label returns the prompt prefix for the current field.
func (s *linkPromptState) label() string {
	if s.field == linkPromptTitle {
		return "Title:"
	}
	return "URL:"
}

// linkPromptHeight is the line count of the prompt footer (border + 1 line).
const linkPromptHeight = 2

// renderLinkPrompt draws a single-line input footer styled like the embed
// picker's filter row.
func (m Model) renderLinkPrompt() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	placeholder := "https://"
	if m.linkPrompt.field == linkPromptTitle {
		placeholder = "Link title"
	}
	return m.linkPrompt.input.RenderFooter(ui.TextInputProps{
		Prompt:        m.linkPrompt.label(),
		Placeholder:   placeholder,
		Hints:         "Enter confirm · Esc cancel",
		Width:         w,
		CursorVisible: true,
	})
}
