package block

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []Block
	}{
		{
			name:  "empty document",
			input: "",
			expect: []Block{
				{Type: Paragraph, Content: ""},
			},
		},
		{
			name:  "heading 1",
			input: "# Hello",
			expect: []Block{
				{Type: Heading1, Content: "Hello"},
			},
		},
		{
			name:  "heading 2",
			input: "## Subtitle",
			expect: []Block{
				{Type: Heading2, Content: "Subtitle"},
			},
		},
		{
			name:  "heading 3",
			input: "### Details",
			expect: []Block{
				{Type: Heading3, Content: "Details"},
			},
		},
		{
			name:  "bullet list with dash",
			input: "- item one\n- item two",
			expect: []Block{
				{Type: BulletList, Content: "item one"},
				{Type: BulletList, Content: "item two"},
			},
		},
		{
			name:  "bullet list with asterisk",
			input: "* first\n* second",
			expect: []Block{
				{Type: BulletList, Content: "first"},
				{Type: BulletList, Content: "second"},
			},
		},
		{
			name:  "numbered list",
			input: "1. first\n2. second\n10. tenth",
			expect: []Block{
				{Type: NumberedList, Content: "first"},
				{Type: NumberedList, Content: "second"},
				{Type: NumberedList, Content: "tenth"},
			},
		},
		{
			name:  "checklist unchecked",
			input: "- [ ] todo item",
			expect: []Block{
				{Type: Checklist, Content: "todo item", Checked: false},
			},
		},
		{
			name:  "checklist checked",
			input: "- [x] done item",
			expect: []Block{
				{Type: Checklist, Content: "done item", Checked: true},
			},
		},
		{
			name:  "checklist checked uppercase X",
			input: "- [X] also done",
			expect: []Block{
				{Type: Checklist, Content: "also done", Checked: true},
			},
		},
		{
			name:  "code block with language",
			input: "```go\nfmt.Println(\"hello\")\n```",
			expect: []Block{
				{Type: CodeBlock, Content: "go\nfmt.Println(\"hello\")"},
			},
		},
		{
			name:  "code block without language",
			input: "```\nsome code\n```",
			expect: []Block{
				{Type: CodeBlock, Content: "\nsome code"},
			},
		},
		{
			name:  "code block preserves markdown syntax inside",
			input: "```\n# not a heading\n- not a list\n> not a quote\n```",
			expect: []Block{
				{Type: CodeBlock, Content: "\n# not a heading\n- not a list\n> not a quote"},
			},
		},
		{
			name:  "code block multiline",
			input: "```python\ndef hello():\n    print(\"hi\")\n\nreturn 42\n```",
			expect: []Block{
				{Type: CodeBlock, Content: "python\ndef hello():\n    print(\"hi\")\n\nreturn 42"},
			},
		},
		{
			name:  "single line quote",
			input: "> a wise saying",
			expect: []Block{
				{Type: Quote, Content: "a wise saying"},
			},
		},
		{
			name:  "multi-line quote merges",
			input: "> line one\n> line two\n> line three",
			expect: []Block{
				{Type: Quote, Content: "line one\nline two\nline three"},
			},
		},
		{
			name:  "divider with dashes",
			input: "---",
			expect: []Block{
				{Type: Divider},
			},
		},
		{
			name:  "divider with asterisks",
			input: "***",
			expect: []Block{
				{Type: Divider},
			},
		},
		{
			name:  "divider with underscores",
			input: "___",
			expect: []Block{
				{Type: Divider},
			},
		},
		{
			name:  "paragraph merges consecutive lines",
			input: "hello world\nthis is a paragraph\nwith three lines",
			expect: []Block{
				{Type: Paragraph, Content: "hello world\nthis is a paragraph\nwith three lines"},
			},
		},
		{
			name:  "blank line produces empty paragraph",
			input: "text above\n\ntext below",
			expect: []Block{
				{Type: Paragraph, Content: "text above"},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: "text below"},
			},
		},
		{
			name:  "consecutive blank lines preserved",
			input: "top\n\n\nbottom",
			expect: []Block{
				{Type: Paragraph, Content: "top"},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: "bottom"},
			},
		},
		{
			name:  "mixed content document",
			input: "# Title\n\nSome intro text.\n\n## Section\n\n- bullet one\n- bullet two\n\n1. step one\n2. step two\n\n> a quote\n\n---\n\n```go\nfunc main() {}\n```\n\n- [ ] task\n- [x] done",
			expect: []Block{
				{Type: Heading1, Content: "Title"},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: "Some intro text."},
				{Type: Paragraph, Content: ""},
				{Type: Heading2, Content: "Section"},
				{Type: Paragraph, Content: ""},
				{Type: BulletList, Content: "bullet one"},
				{Type: BulletList, Content: "bullet two"},
				{Type: Paragraph, Content: ""},
				{Type: NumberedList, Content: "step one"},
				{Type: NumberedList, Content: "step two"},
				{Type: Paragraph, Content: ""},
				{Type: Quote, Content: "a quote"},
				{Type: Paragraph, Content: ""},
				{Type: Divider},
				{Type: Paragraph, Content: ""},
				{Type: CodeBlock, Content: "go\nfunc main() {}"},
				{Type: Paragraph, Content: ""},
				{Type: Checklist, Content: "task", Checked: false},
				{Type: Checklist, Content: "done", Checked: true},
			},
		},
		{
			name:  "long divider",
			input: "-----",
			expect: []Block{
				{Type: Divider},
			},
		},
		{
			name:  "bare quote marker",
			input: ">",
			expect: []Block{
				{Type: Quote, Content: ""},
			},
		},
		{
			name:  "unclosed code fence treats rest as code",
			input: "```go\nfunc main() {}\nmore code",
			expect: []Block{
				{Type: CodeBlock, Content: "go\nfunc main() {}\nmore code"},
			},
		},
		{
			name:  "definition list single item",
			input: "Term\n: Definition text here",
			expect: []Block{
				{Type: DefinitionList, Content: "Term\nDefinition text here"},
			},
		},
		{
			name:  "definition list multiple items",
			input: "Term One\n: First definition\n\nTerm Two\n: Second definition",
			expect: []Block{
				{Type: DefinitionList, Content: "Term One\nFirst definition"},
				{Type: Paragraph, Content: ""},
				{Type: DefinitionList, Content: "Term Two\nSecond definition"},
			},
		},
		{
			name:  "definition list in mixed document",
			input: "# Glossary\n\nAPI\n: Application Programming Interface\n\nSDK\n: Software Development Kit",
			expect: []Block{
				{Type: Heading1, Content: "Glossary"},
				{Type: Paragraph, Content: ""},
				{Type: DefinitionList, Content: "API\nApplication Programming Interface"},
				{Type: Paragraph, Content: ""},
				{Type: DefinitionList, Content: "SDK\nSoftware Development Kit"},
			},
		},
		{
			name:  "orphan definition line parsed as paragraph",
			input: ": orphan definition",
			expect: []Block{
				{Type: Paragraph, Content: ": orphan definition"},
			},
		},
		{
			name:  "embed block",
			input: "![[notebook/note]]",
			expect: []Block{
				{Type: Embed, Content: "notebook/note"},
			},
		},
		{
			name:  "embed in mixed content",
			input: "# Title\n\n![[notebook/note]]\n\nSome text.",
			expect: []Block{
				{Type: Heading1, Content: "Title"},
				{Type: Paragraph, Content: ""},
				{Type: Embed, Content: "notebook/note"},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: "Some text."},
			},
		},
		{
			name:  "embed with spaces in path",
			input: "![[my notebook/my note]]",
			expect: []Block{
				{Type: Embed, Content: "my notebook/my note"},
			},
		},
		{
			name:  "simple 2x2 table",
			input: "| A | B |\n| --- | --- |\n| 1 | 2 |",
			expect: []Block{
				{Type: Table, Content: "| A | B |\n| --- | --- |\n| 1 | 2 |"},
			},
		},
		{
			name:  "table with alignment markers",
			input: "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |",
			expect: []Block{
				{Type: Table, Content: "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |"},
			},
		},
		{
			name:  "table with no body rows",
			input: "| Name | Age |\n| --- | --- |",
			expect: []Block{
				{Type: Table, Content: "| Name | Age |\n| --- | --- |"},
			},
		},
		{
			name:  "table between other block types",
			input: "# Title\n\n| X | Y |\n| - | - |\n| 1 | 2 |\n\nSome text",
			expect: []Block{
				{Type: Heading1, Content: "Title"},
				{Type: Paragraph, Content: ""},
				{Type: Table, Content: "| X | Y |\n| - | - |\n| 1 | 2 |"},
				{Type: Paragraph, Content: ""},
				{Type: Paragraph, Content: "Some text"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("block count: got %d, want %d\ngot:  %s\nwant: %s",
					len(got), len(tt.expect), formatBlocks(got), formatBlocks(tt.expect))
			}
			for i := range tt.expect {
				if got[i].Type != tt.expect[i].Type {
					t.Errorf("block[%d].Type: got %d, want %d", i, got[i].Type, tt.expect[i].Type)
				}
				if got[i].Content != tt.expect[i].Content {
					t.Errorf("block[%d].Content: got %q, want %q", i, got[i].Content, tt.expect[i].Content)
				}
				if got[i].Checked != tt.expect[i].Checked {
					t.Errorf("block[%d].Checked: got %v, want %v", i, got[i].Checked, tt.expect[i].Checked)
				}
			}
		})
	}
}

// formatBlocks is a test helper that formats blocks for readable failure output.
func formatBlocks(blocks []Block) string {
	var b strings.Builder
	for i, bl := range blocks {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("{")
		switch bl.Type {
		case Paragraph:
			b.WriteString("Paragraph")
		case Heading1:
			b.WriteString("Heading1")
		case Heading2:
			b.WriteString("Heading2")
		case Heading3:
			b.WriteString("Heading3")
		case BulletList:
			b.WriteString("BulletList")
		case NumberedList:
			b.WriteString("NumberedList")
		case Checklist:
			b.WriteString("Checklist")
		case CodeBlock:
			b.WriteString("CodeBlock")
		case Quote:
			b.WriteString("Quote")
		case Divider:
			b.WriteString("Divider")
		case DefinitionList:
			b.WriteString("DefinitionList")
		case Embed:
			b.WriteString("Embed")
		case Table:
			b.WriteString("Table")
		}
		if bl.Content != "" {
			b.WriteString(" " + bl.Content)
		}
		b.WriteString("}")
	}
	return b.String()
}
