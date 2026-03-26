package render

import (
	"strings"
	"testing"
)

func TestRenderMarkdownHeader(t *testing.T) {
	out := RenderMarkdown("# Hello World", 80)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "Hello World") {
		t.Errorf("expected rendered output to contain 'Hello World', got %q", out)
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```"
	out := RenderMarkdown(md, 80)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "Println") {
		t.Errorf("expected rendered output to contain 'Println', got %q", out)
	}
}

func TestRenderMarkdownList(t *testing.T) {
	md := "- item one\n- item two\n- item three"
	out := RenderMarkdown(md, 80)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "item one") {
		t.Errorf("expected 'item one' in output, got %q", out)
	}
	if !strings.Contains(out, "item two") {
		t.Errorf("expected 'item two' in output, got %q", out)
	}
}

func TestRenderMarkdownProducesNonEmptyOutput(t *testing.T) {
	md := "Some **bold** and *italic* text with a [link](https://example.com)."
	out := RenderMarkdown(md, 80)
	if len(strings.TrimSpace(out)) == 0 {
		t.Fatal("expected non-empty output for non-empty markdown")
	}
}

func TestRenderMarkdownEmptyInput(t *testing.T) {
	// Empty input should not panic; output may be whitespace from Glamour.
	_ = RenderMarkdown("", 80)
}

func TestRenderMarkdownZeroWidth(t *testing.T) {
	md := "# Auto Width\n\nThis should use default terminal width."
	out := RenderMarkdown(md, 0)
	if !strings.Contains(out, "Auto Width") {
		t.Errorf("expected 'Auto Width' in output, got %q", out)
	}
}

func TestRenderMarkdownCustomWidth(t *testing.T) {
	md := "# Narrow\n\nText content here."
	out := RenderMarkdown(md, 40)
	if !strings.Contains(out, "Narrow") {
		t.Errorf("expected 'Narrow' in output, got %q", out)
	}
}
