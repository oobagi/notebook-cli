package format

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderInlineMarkdown_Bold(t *testing.T) {
	input := "hello **world** end"
	result := RenderInlineMarkdown(input)

	bold := lipgloss.NewStyle().Bold(true).Render("world")
	expected := "hello " + bold + " end"
	if result != expected {
		t.Errorf("bold:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Italic(t *testing.T) {
	input := "hello *world* end"
	result := RenderInlineMarkdown(input)

	italic := lipgloss.NewStyle().Italic(true).Render("world")
	expected := "hello " + italic + " end"
	if result != expected {
		t.Errorf("italic:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Strikethrough(t *testing.T) {
	input := "hello ~~world~~ end"
	result := RenderInlineMarkdown(input)

	strike := lipgloss.NewStyle().Strikethrough(true).Render("world")
	expected := "hello " + strike + " end"
	if result != expected {
		t.Errorf("strikethrough:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_Underline(t *testing.T) {
	input := "hello __world__ end"
	result := RenderInlineMarkdown(input)

	underline := lipgloss.NewStyle().Underline(true).Render("world")
	expected := "hello " + underline + " end"
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
		{"empty italic", "hello ** end"},
		{"empty strikethrough", "hello ~~~~ end"},
		{"empty underline", "hello ____ end"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderInlineMarkdown(tt.input)
			// Should not contain ANSI escapes — everything rendered literally
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
	input := "hello ***bold italic*** end"
	result := RenderInlineMarkdown(input)

	// *** opens as ** (bold) then * (italic). Closing *** closes * (italic) then ** (bold).
	inner := lipgloss.NewStyle().Italic(true).Render("bold italic")
	outer := lipgloss.NewStyle().Bold(true).Render(inner)
	expected := "hello " + outer + " end"
	if result != expected {
		t.Errorf("nested bold+italic:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_BoldWithItalicInside(t *testing.T) {
	input := "**bold with *italic* inside**"
	result := RenderInlineMarkdown(input)

	italic := lipgloss.NewStyle().Italic(true).Render("italic")
	bold := lipgloss.NewStyle().Bold(true).Render("bold with " + italic + " inside")
	if result != bold {
		t.Errorf("bold with italic inside:\n got: %q\nwant: %q", result, bold)
	}
}

func TestRenderInlineMarkdown_MixedFormatting(t *testing.T) {
	input := "**bold** and *italic* and ~~strike~~"
	result := RenderInlineMarkdown(input)

	bold := lipgloss.NewStyle().Bold(true).Render("bold")
	italic := lipgloss.NewStyle().Italic(true).Render("italic")
	strike := lipgloss.NewStyle().Strikethrough(true).Render("strike")
	expected := bold + " and " + italic + " and " + strike
	if result != expected {
		t.Errorf("mixed:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_MultiLine(t *testing.T) {
	input := "**bold** line1\n*italic* line2"
	result := RenderInlineMarkdown(input)

	bold := lipgloss.NewStyle().Bold(true).Render("bold")
	italic := lipgloss.NewStyle().Italic(true).Render("italic")
	expected := bold + " line1\n" + italic + " line2"
	if result != expected {
		t.Errorf("multiline:\n got: %q\nwant: %q", result, expected)
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
	// __ at start/end of words should work
	input := "__underlined text__ here"
	result := RenderInlineMarkdown(input)

	underline := lipgloss.NewStyle().Underline(true).Render("underlined text")
	expected := underline + " here"
	if result != expected {
		t.Errorf("underline at word boundary:\n got: %q\nwant: %q", result, expected)
	}
}

func TestRenderInlineMarkdown_DoubleUnderscoreInsideWord(t *testing.T) {
	// foo__bar__baz — underscores inside a word should not trigger underline
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
