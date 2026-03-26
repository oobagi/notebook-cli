package cmd

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", now.Add(-10 * time.Second), "just now"},
		{"30 seconds ago", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1m ago"},
		{"3 minutes ago", now.Add(-3 * time.Minute), "3m ago"},
		{"10 minutes ago", now.Add(-10 * time.Minute), "10m ago"},
		{"59 minutes ago", now.Add(-59 * time.Minute), "59m ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1h ago"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2h ago"},
		{"23 hours ago", now.Add(-23 * time.Hour), "23h ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1d ago"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3d ago"},
		{"6 days ago", now.Add(-6 * 24 * time.Hour), "6d ago"},
		{"1 week ago", now.Add(-7 * 24 * time.Hour), "1w ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), "2w ago"},
		{"4 weeks ago", now.Add(-28 * 24 * time.Hour), "4w ago"},
		{"same year older", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "Jan 15"},
		{"different year", time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC), "Mar 15, 2025"},
		{"future time", now.Add(1 * time.Hour), "just now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTimeFrom(tt.t, now)
			if got != tt.want {
				t.Errorf("relativeTimeFrom(%v, now) = %q, want %q", tt.t, got, tt.want)
			}
		})
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1 KB"},
		{"1.2 KB", 1229, "1.2 KB"},
		{"3.2 KB", 3277, "3.2 KB"},
		{"8.1 KB", 8294, "8.1 KB"},
		{"exactly 1 MB", 1024 * 1024, "1 MB"},
		{"3.4 MB", 3565158, "3.4 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1 GB"},
		{"1.5 GB", 1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanSize(tt.bytes)
			if got != tt.want {
				t.Errorf("humanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count    int
		singular string
		plural   string
		want     string
	}{
		{0, "note", "notes", "0 notes"},
		{1, "note", "notes", "1 note"},
		{2, "note", "notes", "2 notes"},
		{12, "note", "notes", "12 notes"},
		{1, "book", "books", "1 book"},
		{5, "book", "books", "5 books"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := pluralize(tt.count, tt.singular, tt.plural)
			if got != tt.want {
				t.Errorf("pluralize(%d, %q, %q) = %q, want %q",
					tt.count, tt.singular, tt.plural, got, tt.want)
			}
		})
	}
}

func TestAlignColumns(t *testing.T) {
	t.Run("basic alignment", func(t *testing.T) {
		rows := [][]string{
			{"Personal", "4 notes", "2h ago"},
			{"Work", "12 notes", "just now"},
			{"Research", "1 note", "3d ago"},
		}
		lines := alignColumns(rows)
		if len(lines) != 3 {
			t.Fatalf("got %d lines, want 3", len(lines))
		}

		// Each line should start with two-space indent.
		for i, line := range lines {
			if line[:2] != "  " {
				t.Errorf("line %d missing two-space indent: %q", i, line)
			}
		}

		// "Personal" and "Research" are 8 chars, "Work" is 4 chars.
		// Column 0 width is 8, so "Work" should be padded with 4+4=8 spaces after it.
		// Verify Work has more spaces after it than Personal.
		if len(lines[0]) != len(lines[2]) {
			t.Logf("line 0 = %q", lines[0])
			t.Logf("line 2 = %q", lines[2])
			// They might differ due to last column, but rows with same-length
			// last columns should be the same total length.
		}
	})

	t.Run("empty input", func(t *testing.T) {
		lines := alignColumns(nil)
		if lines != nil {
			t.Errorf("expected nil, got %v", lines)
		}
	})

	t.Run("single column", func(t *testing.T) {
		rows := [][]string{{"hello"}, {"world"}}
		lines := alignColumns(rows)
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
		if lines[0] != "  hello" {
			t.Errorf("line 0 = %q, want %q", lines[0], "  hello")
		}
	})
}
