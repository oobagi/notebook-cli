package ui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/format"
)

// TextInput is a reusable single-line input model for footer/status prompts.
type TextInput struct {
	value  []rune
	cursor int
}

// TextInputProps controls how a TextInput is rendered.
type TextInputProps struct {
	Prompt        string
	Placeholder   string
	Hints         string
	Width         int
	CursorVisible bool
}

func NewTextInput(value string) TextInput {
	var t TextInput
	t.SetValue(value)
	return t
}

func (t *TextInput) SetValue(value string) {
	t.value = []rune(value)
	t.cursor = len(t.value)
}

func (t TextInput) Value() string { return string(t.value) }

func (t TextInput) Cursor() int { return t.cursor }

func (t *TextInput) SetCursor(cursor int) {
	t.cursor = cursor
	t.clamp()
}

func (t *TextInput) Reset() {
	t.value = nil
	t.cursor = 0
}

func (t *TextInput) InsertText(text string) {
	t.clamp()
	runes := []rune(SanitizeSingleLinePaste(text))
	t.value = append(t.value[:t.cursor], append(runes, t.value[t.cursor:]...)...)
	t.cursor += len(runes)
}

// HandleKey processes common single-line editing keys. Mode-specific keys
// such as Enter, Esc, and paste commands stay with the caller.
func (t *TextInput) HandleKey(msg tea.KeyPressMsg) bool {
	key := msg.String()
	superOrMeta := msg.Mod.Contains(tea.ModSuper) || msg.Mod.Contains(tea.ModMeta)
	switch {
	case key == "left":
		t.moveLeft()
	case key == "right":
		t.moveRight()
	case key == "alt+left" || key == "alt+b":
		t.moveWordLeft()
	case key == "alt+right" || key == "alt+f":
		t.moveWordRight()
	case key == "home" || key == "ctrl+a" || key == "super+left" || key == "meta+left" ||
		(msg.Code == tea.KeyLeft && superOrMeta):
		t.moveHome()
	case key == "end" || key == "ctrl+e" || key == "super+right" || key == "meta+right" ||
		(msg.Code == tea.KeyRight && superOrMeta):
		t.moveEnd()
	case key == "backspace":
		t.backspace()
	case key == "delete":
		t.deleteForward()
	case key == "alt+backspace" || key == "ctrl+w":
		t.deleteWordBackward()
	case key == "alt+delete" || key == "alt+d":
		t.deleteWordForward()
	case key == "ctrl+u" || key == "super+backspace" || key == "meta+backspace" ||
		(msg.Code == tea.KeyBackspace && superOrMeta):
		t.deleteBeforeCursor()
	case key == "ctrl+k" || key == "super+delete" || key == "meta+delete" ||
		(msg.Code == tea.KeyDelete && superOrMeta):
		t.deleteAfterCursor()
	case key == "space":
		t.InsertText(" ")
	default:
		if msg.Text == "" {
			return false
		}
		t.InsertText(msg.Text)
	}
	return true
}

func (t TextInput) RenderFooter(props TextInputProps) string {
	return format.FooterInputWithPlaceholder(props.Prompt, t.Value(), props.Placeholder, t.cursor, props.Hints, props.Width, props.CursorVisible)
}

func (t TextInput) RenderStatus(props TextInputProps) string {
	return format.StatusBarInputWithPlaceholder(props.Prompt, t.Value(), props.Placeholder, t.cursor, props.Hints, props.Width, props.CursorVisible)
}

// EditLine applies TextInput editing behavior to an existing string/cursor pair.
func EditLine(value string, cursor int, msg tea.KeyPressMsg) (string, int, bool) {
	t := NewTextInput(value)
	t.SetCursor(cursor)
	handled := t.HandleKey(msg)
	return t.Value(), t.Cursor(), handled
}

// InsertLineText inserts pasted text into an existing string/cursor pair.
func InsertLineText(value string, cursor int, text string) (string, int) {
	t := NewTextInput(value)
	t.SetCursor(cursor)
	t.InsertText(text)
	return t.Value(), t.Cursor()
}

// SanitizeSingleLinePaste makes pasted clipboard text suitable for one-line
// inputs while preserving normal typed text behavior.
func SanitizeSingleLinePaste(text string) string {
	text = strings.Trim(text, "\r\n")
	replacer := strings.NewReplacer(
		"\r\n", " ",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	return replacer.Replace(text)
}

func (t *TextInput) clamp() {
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor > len(t.value) {
		t.cursor = len(t.value)
	}
}

func (t *TextInput) backspace() {
	t.clamp()
	if t.cursor == 0 {
		return
	}
	t.value = append(t.value[:t.cursor-1], t.value[t.cursor:]...)
	t.cursor--
}

func (t *TextInput) deleteForward() {
	t.clamp()
	if t.cursor >= len(t.value) {
		return
	}
	t.value = append(t.value[:t.cursor], t.value[t.cursor+1:]...)
}

func (t *TextInput) deleteBeforeCursor() {
	t.clamp()
	t.value = t.value[t.cursor:]
	t.cursor = 0
}

func (t *TextInput) deleteAfterCursor() {
	t.clamp()
	t.value = t.value[:t.cursor]
}

func (t *TextInput) deleteWordBackward() {
	t.clamp()
	if t.cursor == 0 {
		return
	}
	start := t.wordLeft()
	t.value = append(t.value[:start], t.value[t.cursor:]...)
	t.cursor = start
}

func (t *TextInput) deleteWordForward() {
	t.clamp()
	if t.cursor >= len(t.value) {
		return
	}
	end := t.wordRight()
	t.value = append(t.value[:t.cursor], t.value[end:]...)
}

func (t *TextInput) moveLeft() {
	if t.cursor > 0 {
		t.cursor--
	}
}

func (t *TextInput) moveRight() {
	if t.cursor < len(t.value) {
		t.cursor++
	}
}

func (t *TextInput) moveHome() { t.cursor = 0 }
func (t *TextInput) moveEnd()  { t.cursor = len(t.value) }

func (t *TextInput) moveWordLeft()  { t.cursor = t.wordLeft() }
func (t *TextInput) moveWordRight() { t.cursor = t.wordRight() }

func (t *TextInput) wordLeft() int {
	t.clamp()
	pos := t.cursor
	for pos > 0 && unicode.IsSpace(t.value[pos-1]) {
		pos--
	}
	for pos > 0 && !unicode.IsSpace(t.value[pos-1]) {
		pos--
	}
	return pos
}

func (t *TextInput) wordRight() int {
	t.clamp()
	pos := t.cursor
	for pos < len(t.value) && unicode.IsSpace(t.value[pos]) {
		pos++
	}
	for pos < len(t.value) && !unicode.IsSpace(t.value[pos]) {
		pos++
	}
	return pos
}
