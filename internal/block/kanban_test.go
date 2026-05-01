package block

import (
	"strings"
	"testing"
)

func TestParseKanbanBasic(t *testing.T) {
	body := "## Todo\n- A\n- B\n\n## Done\n- [x] C"
	cols := ParseKanban(body)
	if len(cols) != 2 {
		t.Fatalf("got %d columns, want 2", len(cols))
	}
	if cols[0].Title != "Todo" || cols[1].Title != "Done" {
		t.Errorf("titles = %q / %q", cols[0].Title, cols[1].Title)
	}
	if len(cols[0].Cards) != 2 || cols[0].Cards[0].Text != "A" || cols[0].Cards[1].Text != "B" {
		t.Errorf("Todo cards = %+v", cols[0].Cards)
	}
	if len(cols[1].Cards) != 1 || !cols[1].Cards[0].Done || cols[1].Cards[0].Text != "C" {
		t.Errorf("Done cards = %+v", cols[1].Cards)
	}
}

func TestParseKanbanPriority(t *testing.T) {
	body := "## Todo\n- !!! urgent\n- !! medium\n- ! low\n- plain"
	cols := ParseKanban(body)
	if len(cols) != 1 || len(cols[0].Cards) != 4 {
		t.Fatalf("unexpected: %+v", cols)
	}
	want := []Priority{PriorityHigh, PriorityMed, PriorityLow, PriorityNone}
	for i, p := range want {
		if cols[0].Cards[i].Priority != p {
			t.Errorf("card %d priority = %v, want %v", i, cols[0].Cards[i].Priority, p)
		}
	}
}

func TestParseKanbanTag(t *testing.T) {
	body := "## Todo\n- [bug] broken\n- [feature] new thing\n- [documentation] readme\n- [x] !! [question] maybe\n- [unknown] literal"
	cols := ParseKanban(body)
	if len(cols) != 1 || len(cols[0].Cards) != 5 {
		t.Fatalf("unexpected: %+v", cols)
	}
	tests := []struct {
		idx      int
		wantTag  KanbanTag
		wantText string
		wantPrio Priority
		wantDone bool
	}{
		{0, KanbanTagBug, "broken", PriorityNone, false},
		{1, KanbanTagFeature, "new thing", PriorityNone, false},
		{2, KanbanTagDocumentation, "readme", PriorityNone, false},
		{3, KanbanTagQuestion, "maybe", PriorityMed, true},
		{4, KanbanTagNone, "[unknown] literal", PriorityNone, false},
	}
	for _, tt := range tests {
		c := cols[0].Cards[tt.idx]
		if c.Tag != tt.wantTag || c.Text != tt.wantText || c.Priority != tt.wantPrio || c.Done != tt.wantDone {
			t.Errorf("card %d = %+v, want tag=%v text=%q prio=%v done=%v",
				tt.idx, c, tt.wantTag, tt.wantText, tt.wantPrio, tt.wantDone)
		}
	}
}

func TestParseKanbanCheckedAndPriority(t *testing.T) {
	body := "## Done\n- [x] !!! shipped"
	cols := ParseKanban(body)
	if len(cols) != 1 || len(cols[0].Cards) != 1 {
		t.Fatalf("unexpected shape")
	}
	c := cols[0].Cards[0]
	if !c.Done || c.Priority != PriorityHigh || c.Text != "shipped" {
		t.Errorf("card = %+v", c)
	}
}

func TestSerializeKanbanRoundTrip(t *testing.T) {
	cols := []KanbanColumn{
		{Title: "Todo", Cards: []KanbanCard{
			{Text: "Buy groceries", Priority: PriorityHigh, Tag: KanbanTagBug},
			{Text: "Read book", Tag: KanbanTagDocumentation},
		}},
		{Title: "In Progress", Cards: []KanbanCard{
			{Text: "Email", Priority: PriorityMed},
		}},
		{Title: "Done", Cards: []KanbanCard{
			{Text: "Shipped", Done: true},
		}},
	}
	md := SerializeKanban(cols)
	got := ParseKanban(md)
	if len(got) != len(cols) {
		t.Fatalf("col count = %d, want %d", len(got), len(cols))
	}
	for i := range cols {
		if got[i].Title != cols[i].Title {
			t.Errorf("col %d title = %q, want %q", i, got[i].Title, cols[i].Title)
		}
		if len(got[i].Cards) != len(cols[i].Cards) {
			t.Fatalf("col %d card count = %d, want %d", i, len(got[i].Cards), len(cols[i].Cards))
		}
		for j := range cols[i].Cards {
			if got[i].Cards[j] != cols[i].Cards[j] {
				t.Errorf("col %d card %d = %+v, want %+v", i, j, got[i].Cards[j], cols[i].Cards[j])
			}
		}
	}
}

func TestSerializeEmptyCardIsDropped(t *testing.T) {
	// Fully-empty cards have no value — and serializing one as "- "
	// would round-trip to "no card" anyway (the parser strips trailing
	// whitespace). Make this explicit: empty cards drop on save.
	// Real cards alongside an empty one survive.
	cols := []KanbanColumn{{
		Title: "Todo",
		Cards: []KanbanCard{
			{Text: "", Priority: PriorityMed}, // empty + priority
			{Text: "real"},                    // real
			{Text: ""},                        // empty no priority
		},
	}}
	out := SerializeKanban(cols)
	round := ParseKanban(out)
	if len(round) != 1 {
		t.Fatalf("round-trip cols: got %d, want 1", len(round))
	}
	if len(round[0].Cards) != 1 {
		t.Fatalf("round-trip cards: got %d, want 1 (empty cards dropped)", len(round[0].Cards))
	}
	if round[0].Cards[0].Text != "real" {
		t.Errorf("kept card text = %q, want \"real\"", round[0].Cards[0].Text)
	}
}

func TestParseKanbanEmpty(t *testing.T) {
	if cols := ParseKanban(""); len(cols) != 0 {
		t.Errorf("empty body: got %d cols", len(cols))
	}
	if cols := ParseKanban("   \n\n  "); len(cols) != 0 {
		t.Errorf("whitespace body: got %d cols", len(cols))
	}
}

func TestKanbanFenceParse(t *testing.T) {
	input := "```kanban\n## Todo\n- task one\n## Done\n- [x] task two\n```"
	blocks := Parse(input)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].Type != Kanban {
		t.Errorf("type = %v, want Kanban", blocks[0].Type)
	}
	if !strings.Contains(blocks[0].Content, "## Todo") {
		t.Errorf("content missing Todo: %q", blocks[0].Content)
	}
}

func TestKanbanFenceRoundTrip(t *testing.T) {
	original := "```kanban\n## Todo\n- A\n- B\n\n## Done\n- [x] C\n```"
	blocks := Parse(original)
	if blocks[0].Type != Kanban {
		t.Fatalf("type = %v", blocks[0].Type)
	}
	got := Serialize(blocks)
	if got != original {
		t.Errorf("round trip:\nwant: %q\ngot:  %q", original, got)
	}
}

func TestKanbanMultilineCardRoundTrip(t *testing.T) {
	cols := []KanbanColumn{{Title: "Todo", Cards: []KanbanCard{
		{Text: "First line\nSecond line\nThird line"},
		{Text: "single line", Priority: PriorityHigh},
	}}}
	md := SerializeKanban(cols)
	got := ParseKanban(md)
	if len(got) != 1 || len(got[0].Cards) != 2 {
		t.Fatalf("shape lost: %+v", got)
	}
	if got[0].Cards[0].Text != "First line\nSecond line\nThird line" {
		t.Errorf("multi-line text lost: %q", got[0].Cards[0].Text)
	}
	if got[0].Cards[1].Text != "single line" || got[0].Cards[1].Priority != PriorityHigh {
		t.Errorf("second card = %+v", got[0].Cards[1])
	}
}

func TestKanbanMultilineCardWithBlankLineRoundTrip(t *testing.T) {
	cols := []KanbanColumn{{Title: "Todo", Cards: []KanbanCard{
		{Text: "First line\n\nThird line"},
		{Text: "next card"},
	}}}
	md := SerializeKanban(cols)
	got := ParseKanban(md)
	if len(got) != 1 || len(got[0].Cards) != 2 {
		t.Fatalf("shape lost: %+v", got)
	}
	if got[0].Cards[0].Text != "First line\n\nThird line" {
		t.Errorf("blank-line text lost: %q", got[0].Cards[0].Text)
	}
	if got[0].Cards[1].Text != "next card" {
		t.Errorf("second card = %+v", got[0].Cards[1])
	}
}

func TestKanbanDoneColumnAutoMarks(t *testing.T) {
	cols := []KanbanColumn{
		{Title: "Done", Cards: []KanbanCard{{Text: "shipped", Tag: KanbanTagFeature}}},
		{Title: "Todo", Cards: []KanbanCard{{Text: "next"}}},
	}
	md := SerializeKanban(cols)
	if !strings.Contains(md, "## Done\n- [x] [feature] shipped") {
		t.Errorf("Done column should auto-mark cards: %q", md)
	}
	if !strings.Contains(md, "## Todo\n- next") {
		t.Errorf("non-Done column should not mark: %q", md)
	}
}

func TestKanbanDefaultContent(t *testing.T) {
	cols := ParseKanban(DefaultKanbanContent)
	if len(cols) != 4 {
		t.Errorf("default has %d cols, want 4", len(cols))
	}
	titles := []string{"Backlog", "Todo", "In Progress", "Done"}
	for i, want := range titles {
		if cols[i].Title != want {
			t.Errorf("col %d title = %q, want %q", i, cols[i].Title, want)
		}
	}
}
