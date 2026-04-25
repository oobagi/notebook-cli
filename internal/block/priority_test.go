package block

import "testing"

func TestPriorityString(t *testing.T) {
	tests := []struct {
		p    Priority
		want string
	}{
		{PriorityNone, ""},
		{PriorityLow, "LOW"},
		{PriorityMed, "MED"},
		{PriorityHigh, "HIGH"},
	}
	for _, tt := range tests {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("Priority(%d).String() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestPriorityMarker(t *testing.T) {
	tests := []struct {
		p    Priority
		want string
	}{
		{PriorityNone, ""},
		{PriorityLow, "!"},
		{PriorityMed, "!!"},
		{PriorityHigh, "!!!"},
	}
	for _, tt := range tests {
		if got := tt.p.Marker(); got != tt.want {
			t.Errorf("Priority(%d).Marker() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestPriorityNext(t *testing.T) {
	got := PriorityNone
	want := []Priority{PriorityLow, PriorityMed, PriorityHigh, PriorityNone}
	for i, w := range want {
		got = got.Next()
		if got != w {
			t.Errorf("step %d: got %v, want %v", i, got, w)
		}
	}
}

func TestParsePriorityMarker(t *testing.T) {
	tests := []struct {
		input    string
		wantPrio Priority
		wantBody string
	}{
		{"plain task", PriorityNone, "plain task"},
		{"! low task", PriorityLow, "low task"},
		{"!! medium", PriorityMed, "medium"},
		{"!!! high stakes", PriorityHigh, "high stakes"},
		{"!!!! too many bangs", PriorityNone, "!!!! too many bangs"},
		{"!no space", PriorityNone, "!no space"},
		{"! ", PriorityLow, ""},
		{"", PriorityNone, ""},
		{"hello! world", PriorityNone, "hello! world"},
	}
	for _, tt := range tests {
		gotP, gotBody := ParsePriorityMarker(tt.input)
		if gotP != tt.wantPrio || gotBody != tt.wantBody {
			t.Errorf("ParsePriorityMarker(%q) = (%v, %q), want (%v, %q)",
				tt.input, gotP, gotBody, tt.wantPrio, tt.wantBody)
		}
	}
}

func TestParseChecklistPriority(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPrio Priority
		wantBody string
		wantDone bool
	}{
		{"unchecked no priority", "- [ ] task", PriorityNone, "task", false},
		{"unchecked low", "- [ ] ! low task", PriorityLow, "low task", false},
		{"unchecked med", "- [ ] !! mid task", PriorityMed, "mid task", false},
		{"unchecked high", "- [ ] !!! urgent", PriorityHigh, "urgent", false},
		{"checked high", "- [x] !!! shipped", PriorityHigh, "shipped", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := Parse(tt.input)
			if len(blocks) != 1 {
				t.Fatalf("got %d blocks, want 1", len(blocks))
			}
			b := blocks[0]
			if b.Type != Checklist {
				t.Errorf("type = %v, want Checklist", b.Type)
			}
			if b.Priority != tt.wantPrio {
				t.Errorf("priority = %v, want %v", b.Priority, tt.wantPrio)
			}
			if b.Content != tt.wantBody {
				t.Errorf("content = %q, want %q", b.Content, tt.wantBody)
			}
			if b.Checked != tt.wantDone {
				t.Errorf("checked = %v, want %v", b.Checked, tt.wantDone)
			}
		})
	}
}

func TestSerializeChecklistPriority(t *testing.T) {
	tests := []struct {
		name string
		b    Block
		want string
	}{
		{"none", Block{Type: Checklist, Content: "task"}, "- [ ] task"},
		{"low", Block{Type: Checklist, Content: "low", Priority: PriorityLow}, "- [ ] ! low"},
		{"med", Block{Type: Checklist, Content: "mid", Priority: PriorityMed}, "- [ ] !! mid"},
		{"high checked", Block{Type: Checklist, Content: "done", Checked: true, Priority: PriorityHigh}, "- [x] !!! done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Serialize([]Block{tt.b})
			if got != tt.want {
				t.Errorf("Serialize = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChecklistPriorityRoundTrip(t *testing.T) {
	priorities := []Priority{PriorityNone, PriorityLow, PriorityMed, PriorityHigh}
	for _, p := range priorities {
		t.Run(p.String(), func(t *testing.T) {
			original := Block{Type: Checklist, Content: "round trip", Priority: p}
			md := Serialize([]Block{original})
			parsed := Parse(md)
			if len(parsed) != 1 {
				t.Fatalf("parse produced %d blocks, want 1", len(parsed))
			}
			if parsed[0].Priority != p {
				t.Errorf("priority round trip lost: got %v, want %v", parsed[0].Priority, p)
			}
			if parsed[0].Content != "round trip" {
				t.Errorf("content round trip lost: got %q", parsed[0].Content)
			}
		})
	}
}
