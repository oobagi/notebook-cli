package render

import (
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape sequences for easier assertion on visible text.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// containsANSI checks whether s contains the given ANSI color code parameter
// (e.g. "36" matches \x1b[36m).
func containsANSI(s, code string) bool {
	return strings.Contains(s, "\x1b["+code+"m")
}

func TestRenderAdmonitionNote(t *testing.T) {
	// Simulate Glamour-rendered blockquote with [!NOTE].
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!NOTE\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mThis is a note\x1b[0m"
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "\u2192 Note") {
		t.Errorf("expected Note header with arrow icon, got %q", stripped)
	}
	if strings.Contains(stripped, "[!NOTE]") {
		t.Errorf("expected [!NOTE] tag to be removed, got %q", stripped)
	}
	// Cyan color code.
	if !containsANSI(got, "36") {
		t.Errorf("expected cyan ANSI code (36) in output, got %q", got)
	}
}

func TestRenderAdmonitionTip(t *testing.T) {
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!TIP\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mHelpful tip\x1b[0m"
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "\u2713 Tip") {
		t.Errorf("expected Tip header with checkmark icon, got %q", stripped)
	}
	if !containsANSI(got, "32") {
		t.Errorf("expected green ANSI code (32) in output, got %q", got)
	}
}

func TestRenderAdmonitionWarning(t *testing.T) {
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!WARNING\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mBe careful\x1b[0m"
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "! Warning") {
		t.Errorf("expected Warning header with ! icon, got %q", stripped)
	}
	if !containsANSI(got, "33") {
		t.Errorf("expected yellow ANSI code (33) in output, got %q", got)
	}
}

func TestRenderAdmonitionCaution(t *testing.T) {
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!CAUTION\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mDangerous operation\x1b[0m"
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "\u2717 Caution") {
		t.Errorf("expected Caution header with X icon, got %q", stripped)
	}
	if !containsANSI(got, "31") {
		t.Errorf("expected red ANSI code (31) in output, got %q", got)
	}
}

func TestRenderAdmonitionImportant(t *testing.T) {
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!IMPORTANT\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mRead this\x1b[0m"
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "! Important") {
		t.Errorf("expected Important header with ! icon, got %q", stripped)
	}
	// Bold magenta: 1;35.
	if !containsANSI(got, "1;35") {
		t.Errorf("expected bold magenta ANSI code (1;35) in output, got %q", got)
	}
}

func TestRenderAdmonitionPreservesNonAdmonition(t *testing.T) {
	// A regular blockquote without [!TYPE] should pass through unchanged.
	input := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252mJust a normal quote\x1b[0m"
	got := RenderAdmonitions(input)

	if got != input {
		t.Errorf("expected non-admonition blockquote unchanged\n got: %q\nwant: %q", got, input)
	}
}

func TestRenderAdmonitionMultiple(t *testing.T) {
	// Two admonitions separated by a non-blockquote line.
	note := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!NOTE\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mFirst\x1b[0m"
	separator := "  \x1b[38;5;252mSome text\x1b[0m"
	warning := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!WARNING\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mSecond\x1b[0m"

	input := note + "\n" + separator + "\n" + warning
	got := RenderAdmonitions(input)

	stripped := stripANSI(got)
	if !strings.Contains(stripped, "\u2192 Note") {
		t.Errorf("expected Note header, got %q", stripped)
	}
	if !strings.Contains(stripped, "! Warning") {
		t.Errorf("expected Warning header, got %q", stripped)
	}
	if !strings.Contains(stripped, "Some text") {
		t.Errorf("expected separator text preserved, got %q", stripped)
	}
}

func TestRenderAdmonitionMultiLine(t *testing.T) {
	// An admonition block spanning multiple blockquote lines.
	header := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252m[\x1b[0m\x1b[38;5;252m!TIP\x1b[0m\x1b[38;5;252m] \x1b[0m\x1b[38;5;252mFirst line\x1b[0m"
	body := "\x1b[38;5;252m\x1b[0m  \x1b[38;5;252m\u2502 \x1b[0m\x1b[38;5;252mSecond line\x1b[0m"

	input := header + "\n" + body
	got := RenderAdmonitions(input)

	// Both lines should get the green color on the border.
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if line == "" {
			continue
		}
		if !containsANSI(line, "32") {
			t.Errorf("line %d: expected green ANSI code (32), got %q", i, line)
		}
	}
}

func TestRenderAdmonitionEmptyInput(t *testing.T) {
	got := RenderAdmonitions("")
	if got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestRenderAdmonitionNoBlockquote(t *testing.T) {
	// Plain text with [!NOTE] but not in a blockquote should pass through.
	input := "This has [!NOTE] but is not a blockquote"
	got := RenderAdmonitions(input)
	if got != input {
		t.Errorf("expected unchanged output for non-blockquote [!NOTE]\n got: %q\nwant: %q", got, input)
	}
}
