package format

import (
	"strings"
	"testing"
)

// Helper: wrap text in targeted ANSI on/off codes.
func bold(s string) string          { return boldOn + s + boldOff }
func italic(s string) string        { return italicOn + s + italicOff }
func underline(s string) string     { return underlineOn + s + underlineOff }
func strikethrough(s string) string { return strikethroughOn + s + strikethroughOff }

func TestRenderInlineMarkdown_Bold(t *testing.T) {
	result := RenderInlineMarkdown("hello **world** end")
	expected := "hello " + bold("world") + " end"
	if result != expected {
		t.Errorf("bold:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Italic(t *testing.T) {
	result := RenderInlineMarkdown("hello *world* end")
	expected := "hello " + italic("world") + " end"
	if result != expected {
		t.Errorf("italic:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Strikethrough(t *testing.T) {
	result := RenderInlineMarkdown("hello ~~world~~ end")
	expected := "hello " + strikethrough("world") + " end"
	if result != expected {
		t.Errorf("strikethrough:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Underline(t *testing.T) {
	result := RenderInlineMarkdown("hello __world__ end")
	expected := "hello " + underline("world") + " end"
	if result != expected {
		t.Errorf("underline:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_SnakeCaseNotUnderline(t *testing.T) {
	input := "use snake_case_variable here"
	result := RenderInlineMarkdown(input)
	if result != input {
		t.Errorf("snake_case should not be treated as underline:\n got: %q\nwant: %q", result, input)
	}
}

func TestRenderInlineMarkdown_EmptyDelimiters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty bold", "hello **** end"},
		{"unmatched double star", "hello ** end"},
		{"empty strikethrough", "hello ~~~~ end"},
		{"empty underline", "hello ____ end"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderInlineMarkdown(tt.input)
			if strings.Contains(result, "\x1b[") {
				t.Errorf("%s: expected literal output, got ANSI: %q", tt.name, result)
			}
		})
	}
}

func TestRenderInlineMarkdown_UnmatchedDelimiters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unmatched bold", "hello **world end"},
		{"unmatched italic", "hello *world end"},
		{"unmatched strikethrough", "hello ~~world end"},
		{"unmatched underline", "hello __world end"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderInlineMarkdown(tt.input)
			if result != tt.input {
				t.Errorf("%s: unmatched delimiters should render literally:\n got: %q\nwant: %q", tt.name, result, tt.input)
			}
		})
	}
}

func TestRenderInlineMarkdown_Nested_BoldItalic(t *testing.T) {
	result := RenderInlineMarkdown("hello ***bold italic*** end")
	// Both turn off at same boundary; emitTransition disables bold before italic.
	expected := "hello " + boldOn + italicOn + "bold italic" + boldOff + italicOff + " end"
	if result != expected {
		t.Errorf("nested bold+italic:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_BoldWithItalicInside(t *testing.T) {
	result := RenderInlineMarkdown("**bold with *italic* inside**")
	// bold on → "bold with " → +italic "italic" -italic → " inside" → bold off
	expected := boldOn + "bold with " + italicOn + "italic" + italicOff + " inside" + boldOff
	if result != expected {
		t.Errorf("bold with italic inside:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_MixedFormatting(t *testing.T) {
	result := RenderInlineMarkdown("**bold** and *italic* and ~~strike~~")
	expected := bold("bold") + " and " + italic("italic") + " and " + strikethrough("strike")
	if result != expected {
		t.Errorf("mixed:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_MultiLine(t *testing.T) {
	result := RenderInlineMarkdown("**bold** line1\n*italic* line2")
	expected := bold("bold") + " line1\n" + italic("italic") + " line2"
	if result != expected {
		t.Errorf("multiline:\n got: %q\nwant: %q", result, expected)
	}
}

// When the caller has already word-wrapped the text, a **bold** pair can
// span across a newline. The renderer should still style both lines.
func TestRenderInlineMarkdown_DelimiterSpansWrappedLines(t *testing.T) {
	result := RenderInlineMarkdown("**add the correct version\nto the settings.**")
	expected := boldOn + "add the correct version" + boldOff + "\n" + boldOn + "to the settings." + boldOff
	if result != expected {
		t.Errorf("wrapped bold:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_PlainText(t *testing.T) {
	input := "no formatting here"
	result := RenderInlineMarkdown(input)
	if result != input {
		t.Errorf("plain text should be unchanged:\n got: %q\nwant: %q", result, input)
	}
}

func TestRenderInlineMarkdown_UnderlineAtWordBoundary(t *testing.T) {
	result := RenderInlineMarkdown("__underlined text__ here")
	expected := underline("underlined text") + " here"
	if result != expected {
		t.Errorf("underline at word boundary:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_DoubleUnderscoreInsideWord(t *testing.T) {
	input := "foo__bar__baz"
	result := RenderInlineMarkdown(input)
	if result != input {
		t.Errorf("double underscore inside word should be literal:\n got: %q\nwant: %q", result, input)
	}
}

func TestRenderInlineMarkdown_EmptyInput(t *testing.T) {
	result := RenderInlineMarkdown("")
	if result != "" {
		t.Errorf("empty input should return empty:\n got: %q", result)
	}
}

// --- New: cross-type nesting (the bugs this rewrite fixes) ---

func TestRenderInlineMarkdown_ItalicWithUnderline(t *testing.T) {
	result := RenderInlineMarkdown("*__test__*")
	expected := italicOn + underlineOn + "test" + italicOff + underlineOff
	if result != expected {
		t.Errorf("italic+underline:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_BoldWithUnderline(t *testing.T) {
	result := RenderInlineMarkdown("**__test__**")
	expected := boldOn + underlineOn + "test" + boldOff + underlineOff
	if result != expected {
		t.Errorf("bold+underline:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_BoldWithStrikethrough(t *testing.T) {
	result := RenderInlineMarkdown("**~~test~~**")
	expected := boldOn + strikethroughOn + "test" + boldOff + strikethroughOff
	if result != expected {
		t.Errorf("bold+strikethrough:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_ItalicWithStrikethrough(t *testing.T) {
	result := RenderInlineMarkdown("*~~test~~*")
	expected := italicOn + strikethroughOn + "test" + italicOff + strikethroughOff
	if result != expected {
		t.Errorf("italic+strikethrough:\n got: %q\nwant: %q", result, expected)
	}
}

func TestStripControlChars(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello world", "hello world"},
		{"keeps newline tab CR", "a\nb\tc\rd", "a\nb\tc\rd"},
		{"strips ESC and BEL", "title\x1b]0;EVIL\x07rest", "title]0;EVILrest"},
		{"strips C0", "\x01\x02hello\x03", "hello"},
		{"strips DEL", "abc\x7fdef", "abcdef"},
		{"keeps utf8 multibyte", "héllo", "héllo"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		got := StripControlChars(tc.in)
		if got != tc.want {
			t.Errorf("%s: StripControlChars(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestRenderInlineMarkdown_StripsControlChars(t *testing.T) {
	// An OSC sequence in untrusted card text must not pass through.
	in := "card\x1b]0;EVIL\x07text"
	got := RenderInlineMarkdown(in)
	if strings.Contains(got, "\x1b]") {
		t.Errorf("output should not contain OSC introducer: %q", got)
	}
	if strings.Contains(got, "\x07") {
		t.Errorf("output should not contain BEL: %q", got)
	}
}

func TestRenderInlineMarkdown_NoFullReset(t *testing.T) {
	// The key property: output must never contain \x1b[0m (full reset)
	// which would kill outer block-level styles like heading bold.
	inputs := []string{
		"**bold**",
		"*italic*",
		"__underline__",
		"~~strike~~",
		"***bold italic***",
		"*__nested__*",
		"**~~nested~~**",
	}
	for _, input := range inputs {
		result := RenderInlineMarkdown(input)
		if strings.Contains(result, "\x1b[0m") {
			t.Errorf("output for %q contains full reset \\x1b[0m: %q", input, result)
		}
	}
}
