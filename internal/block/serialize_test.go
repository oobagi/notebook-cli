package block

import (
	"testing"
)

func TestSerializeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		md   string
	}{
		{
			name: "empty document",
			md:   "",
		},
		{
			name: "single paragraph",
			md:   "Hello, world!",
		},
		{
			name: "multi-line paragraph",
			md:   "line one\nline two\nline three",
		},
		{
			name: "heading 1",
			md:   "# Title",
		},
		{
			name: "heading 2",
			md:   "## Subtitle",
		},
		{
			name: "heading 3",
			md:   "### Details",
		},
		{
			name: "all heading levels",
			md:   "# H1\n\n## H2\n\n### H3",
		},
		{
			name: "bullet list",
			md:   "- item one\n- item two\n- item three",
		},
		{
			name: "numbered list sequencing",
			md:   "1. first\n2. second\n3. third",
		},
		{
			name: "numbered list resets after non-numbered block",
			md:   "1. alpha\n2. beta\n\n1. one\n2. two",
		},
		{
			name: "checklist unchecked",
			md:   "- [ ] todo item",
		},
		{
			name: "checklist checked",
			md:   "- [x] done item",
		},
		{
			name: "mixed checklist",
			md:   "- [ ] pending\n- [x] complete\n- [ ] also pending",
		},
		{
			name: "code block with language",
			md:   "```go\nfmt.Println(\"hello\")\n```",
		},
		{
			name: "code block without language",
			md:   "```\nsome code\n```",
		},
		{
			name: "code block multiline",
			md:   "```python\ndef hello():\n    print(\"hi\")\n\nreturn 42\n```",
		},
		{
			name: "empty code block",
			md:   "```\n```",
		},
		{
			name: "code block with language no content",
			md:   "```go\n```",
		},
		{
			name: "single line quote",
			md:   "> a wise saying",
		},
		{
			name: "multi-line quote",
			md:   "> line one\n> line two\n> line three",
		},
		{
			name: "quote with empty line",
			md:   "> first\n>\n> third",
		},
		{
			name: "bare quote marker",
			md:   ">",
		},
		{
			name: "divider",
			md:   "---",
		},
		{
			name: "paragraph with blank line",
			md:   "text above\n\ntext below",
		},
		{
			name: "consecutive blank lines",
			md:   "top\n\n\nbottom",
		},
		{
			name: "complex mixed document",
			md:   "# Title\n\nSome intro text.\n\n## Section\n\n- bullet one\n- bullet two\n\n1. step one\n2. step two\n\n> a quote\n\n---\n\n```go\nfunc main() {}\n```\n\n- [ ] task\n- [x] done",
		},
		{
			name: "heading followed by paragraph",
			md:   "# Title\nSome text",
		},
		{
			name: "numbered list with five items",
			md:   "1. a\n2. b\n3. c\n4. d\n5. e",
		},
		{
			name: "code block preserves markdown syntax inside",
			md:   "```\n# not a heading\n- not a list\n> not a quote\n```",
		},
		{
			name: "divider between paragraphs",
			md:   "above\n\n---\n\nbelow",
		},
		{
			name: "definition list single item",
			md:   "Term\n: Definition text here",
		},
		{
			name: "definition list multiple items",
			md:   "Term One\n: First definition\n\nTerm Two\n: Second definition",
		},
		{
			name: "definition list in mixed document",
			md:   "# Glossary\n\nAPI\n: Application Programming Interface\n\nSDK\n: Software Development Kit",
		},
		{
			name: "embed block",
			md:   "![[notebook/note]]",
		},
		{
			name: "embed with spaces",
			md:   "![[my notebook/my note]]",
		},
		{
			name: "embed between paragraphs",
			md:   "text\n\n![[ref]]\n\nmore text",
		},
		{
			name: "simple table",
			md:   "| A   | B   |\n| --- | --- |\n| 1   | 2   |",
		},
		{
			name: "table with alignment markers",
			md:   "| Left | Center | Right |\n| :--- | :----: | ----: |\n| a    | b      | c     |",
		},
		{
			name: "table with no body rows",
			md:   "| Name | Age |\n| ---- | --- |",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := Parse(tt.md)
			got := Serialize(blocks)
			if got != tt.md {
				t.Errorf("round-trip mismatch:\n  input:      %q\n  serialized: %q", tt.md, got)
			}
		})
	}
}

func TestSerializeIdempotent(t *testing.T) {
	// Parse(Serialize(Parse(md))) == Parse(md)
	// This checks that even if the first round-trip changes formatting,
	// subsequent round-trips are stable.
	inputs := []string{
		"",
		"# Hello\n\nWorld",
		"1. first\n2. second\n3. third",
		"- [ ] task\n- [x] done",
		"```go\ncode\n```",
		"> quote\n> continues",
		"---",
		"# Title\n\nSome intro text.\n\n## Section\n\n- bullet one\n- bullet two\n\n1. step one\n2. step two\n\n> a quote\n\n---\n\n```go\nfunc main() {}\n```\n\n- [ ] task\n- [x] done",
		"Term\n: Definition text here",
		"API\n: Application Programming Interface\n\nSDK\n: Software Development Kit",
		"![[notebook/note]]",
		"text\n\n![[ref]]\n\nmore text",
		"| A   | B   |\n| --- | --- |\n| 1   | 2   |",
		"| Left | Center | Right |\n| :--- | :----: | ----: |\n| a    | b      | c     |",
	}

	for _, md := range inputs {
		t.Run("", func(t *testing.T) {
			blocks1 := Parse(md)
			serialized := Serialize(blocks1)
			blocks2 := Parse(serialized)

			if len(blocks1) != len(blocks2) {
				t.Fatalf("idempotency failed: block count %d vs %d\n  input: %q\n  serialized: %q",
					len(blocks1), len(blocks2), md, serialized)
			}
			for i := range blocks1 {
				if blocks1[i].Type != blocks2[i].Type {
					t.Errorf("block[%d].Type: %d vs %d", i, blocks1[i].Type, blocks2[i].Type)
				}
				if blocks1[i].Content != blocks2[i].Content {
					t.Errorf("block[%d].Content: %q vs %q", i, blocks1[i].Content, blocks2[i].Content)
				}
				if blocks1[i].Checked != blocks2[i].Checked {
					t.Errorf("block[%d].Checked: %v vs %v", i, blocks1[i].Checked, blocks2[i].Checked)
				}
			}
		})
	}
}

func TestSerializeNumberedListSequencing(t *testing.T) {
	blocks := []Block{
		{Type: NumberedList, Content: "alpha"},
		{Type: NumberedList, Content: "beta"},
		{Type: NumberedList, Content: "gamma"},
	}
	got := Serialize(blocks)
	want := "1. alpha\n2. beta\n3. gamma"
	if got != want {
		t.Errorf("numbered sequencing:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestSerializeTablePadding(t *testing.T) {
	// Uneven column widths should get padded.
	blocks := []Block{
		{Type: Table, Content: "| A | LongHeader |\n| --- | --- |\n| short | x |"},
	}
	got := Serialize(blocks)
	want := "| A     | LongHeader |\n| ----- | ---------- |\n| short | x          |"
	if got != want {
		t.Errorf("table padding:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestSerializeTableAlignmentPreserved(t *testing.T) {
	blocks := []Block{
		{Type: Table, Content: "| L | C | R |\n| :--- | :---: | ---: |\n| a | b | c |"},
	}
	got := Serialize(blocks)
	want := "| L   | C   | R   |\n| :-- | :-: | --: |\n| a   | b   | c   |"
	if got != want {
		t.Errorf("table alignment:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestTrimEmptyEdges(t *testing.T) {
	blocks := []Block{
		{Type: Paragraph, Content: ""},
		{Type: Paragraph, Content: ""},
		{Type: Heading1, Content: "Title"},
		{Type: Paragraph, Content: "body"},
		{Type: Paragraph, Content: ""},
		{Type: Paragraph, Content: ""},
	}
	trimmed := TrimEmptyEdges(blocks)
	if len(trimmed) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(trimmed))
	}
	if trimmed[0].Type != Heading1 || trimmed[1].Type != Paragraph {
		t.Fatalf("wrong block types after trim: %+v", trimmed)
	}
}

// Internal blanks must survive — they're how paragraphs get separated.
func TestTrimEmptyEdgesPreservesInternal(t *testing.T) {
	blocks := []Block{
		{Type: Paragraph, Content: "one"},
		{Type: Paragraph, Content: ""},
		{Type: Paragraph, Content: "two"},
	}
	trimmed := TrimEmptyEdges(blocks)
	if len(trimmed) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(trimmed))
	}
}

func TestSerializeNumberedListResets(t *testing.T) {
	blocks := []Block{
		{Type: NumberedList, Content: "one"},
		{Type: NumberedList, Content: "two"},
		{Type: Paragraph, Content: ""},
		{Type: NumberedList, Content: "again one"},
		{Type: NumberedList, Content: "again two"},
	}
	got := Serialize(blocks)
	want := "1. one\n2. two\n\n1. again one\n2. again two"
	if got != want {
		t.Errorf("numbered list reset:\n  got:  %q\n  want: %q", got, want)
	}
}
