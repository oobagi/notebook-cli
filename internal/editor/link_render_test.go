package editor

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oobagi/notebook-cli/internal/block"
)

// stripANSI returns the visible text with ANSI escapes removed so width
// assertions are stable across themes.
func stripANSI(s string) string {
	return ansi.Strip(s)
}

func TestRenderLinkCardWrapsLongTitleWhenWordWrapOn(t *testing.T) {
	content := "https://example.com/path\nA really quite long bookmark title that will not fit on one line"
	width := 30
	// Rendered card width = icon (2) + width.
	maxWidth := width + lipgloss.Width("↗ ")

	out := renderLinkCard(content, width, false, true)
	t.Logf("rendered:\n%s\n--- plain ---\n%s", out, stripANSI(out))

	lines := strings.Split(stripANSI(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapped output across multiple lines, got %d:\n%s", len(lines), out)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > maxWidth {
			t.Errorf("line %d width %d exceeds max %d: %q", i, w, maxWidth, l)
		}
	}
}

func TestRenderInactiveLinkBlockWrapsWithGutter(t *testing.T) {
	b := block.Block{Type: block.Link, Content: "https://example.com/path\nA really quite long bookmark title that will not fit on one line"}
	const totalWidth = 40
	out := renderInactiveBlock(b, b.Content, totalWidth, true, []block.Block{b}, 0)
	t.Logf("rendered:\n%s\n--- plain ---\n%s", out, stripANSI(out))
	plainLines := strings.Split(stripANSI(out), "\n")
	if len(plainLines) < 2 {
		t.Fatalf("expected wrapped output across multiple lines, got %d:\n%s", len(plainLines), stripANSI(out))
	}
	for i, l := range plainLines {
		if w := lipgloss.Width(l); w > totalWidth {
			t.Errorf("line %d width %d exceeds total width %d: %q", i, w, totalWidth, l)
		}
	}
}

func TestRenderLinkCardTruncatesWhenWordWrapOff(t *testing.T) {
	content := "https://example.com/path\nA really quite long bookmark title that will not fit on one line"
	width := 30

	out := renderLinkCard(content, width, false, false)

	if strings.Contains(out, "\n") {
		t.Fatalf("expected single-line output when wordWrap is off, got:\n%s", out)
	}
	if !strings.Contains(stripANSI(out), "…") {
		t.Errorf("expected ellipsis truncation in no-wrap mode, got: %q", stripANSI(out))
	}
}
