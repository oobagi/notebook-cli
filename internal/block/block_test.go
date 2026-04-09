package block

import (
	"strings"
	"testing"
)

func TestCalloutVariantString(t *testing.T) {
	tests := []struct {
		v    CalloutVariant
		want string
	}{
		{CalloutNote, "NOTE"},
		{CalloutTip, "TIP"},
		{CalloutWarning, "WARNING"},
		{CalloutCaution, "CAUTION"},
		{CalloutImportant, "IMPORTANT"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("CalloutVariant(%d).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestCalloutVariantNext(t *testing.T) {
	// Cycles: Note → Tip → Warning → Caution → Important → Note
	got := CalloutNote
	want := []CalloutVariant{CalloutTip, CalloutWarning, CalloutCaution, CalloutImportant, CalloutNote}
	for i, w := range want {
		got = got.Next()
		if got != w {
			t.Errorf("step %d: got %v, want %v", i, got, w)
		}
	}
}

func TestParseCalloutVariant(t *testing.T) {
	tests := []struct {
		input string
		want  CalloutVariant
		ok    bool
	}{
		{"NOTE", CalloutNote, true},
		{"note", CalloutNote, true},
		{"Note", CalloutNote, true},
		{"TIP", CalloutTip, true},
		{"tip", CalloutTip, true},
		{"WARNING", CalloutWarning, true},
		{"warning", CalloutWarning, true},
		{"CAUTION", CalloutCaution, true},
		{"caution", CalloutCaution, true},
		{"IMPORTANT", CalloutImportant, true},
		{"important", CalloutImportant, true},
		{"unknown", CalloutNote, false},
		{"", CalloutNote, false},
	}
	for _, tt := range tests {
		got, ok := ParseCalloutVariant(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseCalloutVariant(%q) = (%v, %v), want (%v, %v)",
				tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCalloutBlockType(t *testing.T) {
	if Callout.String() != "Callout" {
		t.Errorf("Callout.String() = %q, want %q", Callout.String(), "Callout")
	}
	if Callout.Short() != "co" {
		t.Errorf("Callout.Short() = %q, want %q", Callout.Short(), "co")
	}
	if Callout.IsListItem() {
		t.Error("Callout.IsListItem() = true, want false")
	}
	if Callout.IsHeading() {
		t.Error("Callout.IsHeading() = true, want false")
	}
}

func TestParseCallout(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		variant CalloutVariant
		content string
	}{
		{
			name:    "note with content",
			input:   "> [!NOTE]\n> This is a note.",
			variant: CalloutNote,
			content: "This is a note.",
		},
		{
			name:    "tip with content",
			input:   "> [!TIP]\n> Helpful advice here.",
			variant: CalloutTip,
			content: "Helpful advice here.",
		},
		{
			name:    "warning with content",
			input:   "> [!WARNING]\n> Watch out!",
			variant: CalloutWarning,
			content: "Watch out!",
		},
		{
			name:    "caution with content",
			input:   "> [!CAUTION]\n> Be careful.",
			variant: CalloutCaution,
			content: "Be careful.",
		},
		{
			name:    "important with content",
			input:   "> [!IMPORTANT]\n> Critical info.",
			variant: CalloutImportant,
			content: "Critical info.",
		},
		{
			name:    "empty callout",
			input:   "> [!NOTE]",
			variant: CalloutNote,
			content: "",
		},
		{
			name:    "multi-line callout",
			input:   "> [!TIP]\n> Line one.\n> Line two.\n> Line three.",
			variant: CalloutTip,
			content: "Line one.\nLine two.\nLine three.",
		},
		{
			name:    "case insensitive variant",
			input:   "> [!note]\n> Lower case note.",
			variant: CalloutNote,
			content: "Lower case note.",
		},
		{
			name:    "mixed case variant",
			input:   "> [!Warning]\n> Mixed case.",
			variant: CalloutWarning,
			content: "Mixed case.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := Parse(tt.input)
			if len(blocks) != 1 {
				t.Fatalf("got %d blocks, want 1", len(blocks))
			}
			b := blocks[0]
			if b.Type != Callout {
				t.Errorf("type = %v, want Callout", b.Type)
			}
			if b.Variant != tt.variant {
				t.Errorf("variant = %v, want %v", b.Variant, tt.variant)
			}
			if b.Content != tt.content {
				t.Errorf("content = %q, want %q", b.Content, tt.content)
			}
		})
	}
}

func TestParseQuoteNotCallout(t *testing.T) {
	input := "> This is a regular quote.\n> Second line."
	blocks := Parse(input)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].Type != Quote {
		t.Errorf("type = %v, want Quote", blocks[0].Type)
	}
	if blocks[0].Content != "This is a regular quote.\nSecond line." {
		t.Errorf("content = %q", blocks[0].Content)
	}
}

func TestSerializeCallout(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  string
	}{
		{
			name:  "note with content",
			block: Block{Type: Callout, Variant: CalloutNote, Content: "This is a note."},
			want:  "> [!NOTE]\n> This is a note.",
		},
		{
			name:  "empty callout",
			block: Block{Type: Callout, Variant: CalloutWarning, Content: ""},
			want:  "> [!WARNING]",
		},
		{
			name:  "multi-line",
			block: Block{Type: Callout, Variant: CalloutTip, Content: "Line one.\nLine two."},
			want:  "> [!TIP]\n> Line one.\n> Line two.",
		},
		{
			name:  "content with empty line",
			block: Block{Type: Callout, Variant: CalloutImportant, Content: "Before.\n\nAfter."},
			want:  "> [!IMPORTANT]\n> Before.\n>\n> After.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Serialize([]Block{tt.block})
			if got != tt.want {
				t.Errorf("Serialize = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSerializeQuote(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  string
	}{
		{
			name:  "plain quote",
			block: Block{Type: Quote, Content: "Just a quote."},
			want:  "> Just a quote.",
		},
		{
			name:  "empty quote",
			block: Block{Type: Quote, Content: ""},
			want:  ">",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Serialize([]Block{tt.block})
			if got != tt.want {
				t.Errorf("Serialize = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalloutRoundTrip(t *testing.T) {
	variants := []CalloutVariant{
		CalloutNote, CalloutTip, CalloutWarning, CalloutCaution, CalloutImportant,
	}

	for _, v := range variants {
		t.Run(v.String(), func(t *testing.T) {
			original := "> [!" + v.String() + "]\n> Some content here.\n> And another line."
			blocks := Parse(original)
			if len(blocks) != 1 {
				t.Fatalf("parse produced %d blocks, want 1", len(blocks))
			}
			if blocks[0].Type != Callout {
				t.Errorf("type = %v, want Callout", blocks[0].Type)
			}
			if blocks[0].Variant != v {
				t.Errorf("variant = %v, want %v", blocks[0].Variant, v)
			}
			serialized := Serialize(blocks)
			if serialized != original {
				t.Errorf("round trip failed:\n  got:  %q\n  want: %q", serialized, original)
			}

			// Parse again and compare.
			blocks2 := Parse(serialized)
			if len(blocks2) != 1 {
				t.Fatalf("second parse produced %d blocks, want 1", len(blocks2))
			}
			if blocks2[0].Variant != blocks[0].Variant {
				t.Errorf("variant mismatch: %v vs %v", blocks2[0].Variant, blocks[0].Variant)
			}
			if blocks2[0].Content != blocks[0].Content {
				t.Errorf("content mismatch: %q vs %q", blocks2[0].Content, blocks[0].Content)
			}
		})
	}
}

func TestPlainQuoteRoundTrip(t *testing.T) {
	original := "> Just a plain quote.\n> Second line."
	blocks := Parse(original)
	if len(blocks) != 1 {
		t.Fatalf("parse produced %d blocks, want 1", len(blocks))
	}
	if blocks[0].Type != Quote {
		t.Errorf("type = %v, want Quote", blocks[0].Type)
	}
	serialized := Serialize(blocks)
	if serialized != original {
		t.Errorf("round trip failed: got %q, want %q", serialized, original)
	}
}

func TestMixedBlocksWithCallout(t *testing.T) {
	input := strings.Join([]string{
		"# Title",
		"",
		"> [!WARNING]",
		"> Be careful here.",
		"",
		"> This is a regular quote.",
		"",
		"A paragraph.",
	}, "\n")

	blocks := Parse(input)
	// Expected: Heading1, empty Paragraph, Callout, empty Paragraph, Quote, empty Paragraph, Paragraph
	if len(blocks) != 7 {
		t.Fatalf("got %d blocks, want 7. Blocks: %v", len(blocks), blocks)
	}
	if blocks[0].Type != Heading1 {
		t.Errorf("block 0: type = %v, want Heading1", blocks[0].Type)
	}
	if blocks[2].Type != Callout {
		t.Errorf("block 2: type = %v, want Callout", blocks[2].Type)
	}
	if blocks[2].Variant != CalloutWarning {
		t.Errorf("block 2: variant = %v, want CalloutWarning", blocks[2].Variant)
	}
	if blocks[4].Type != Quote {
		t.Errorf("block 4: type = %v, want Quote", blocks[4].Type)
	}
	if blocks[6].Type != Paragraph {
		t.Errorf("block 6: type = %v, want Paragraph", blocks[6].Type)
	}
}
