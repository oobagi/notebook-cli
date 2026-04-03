package recents

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recent.json")

	entries := []Entry{
		{
			Type:       "store",
			Notebook:   "work",
			Name:       "standup",
			LastEdited: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			Type:       "external",
			Path:       "/tmp/todo.txt",
			LastEdited: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		},
	}

	if err := Save(path, entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(loaded), len(entries))
	}

	if loaded[0].Type != "store" || loaded[0].Notebook != "work" || loaded[0].Name != "standup" {
		t.Errorf("entry 0 mismatch: %+v", loaded[0])
	}
	if loaded[1].Type != "external" || loaded[1].Path != "/tmp/todo.txt" {
		t.Errorf("entry 1 mismatch: %+v", loaded[1])
	}
}

func TestLoadMissing(t *testing.T) {
	entries, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("Load nonexistent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(entries))
	}
}

func TestLoadEmptyPath(t *testing.T) {
	entries, err := Load("")
	if err != nil {
		t.Fatalf("Load empty path: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestUpsert(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Type: "store", Notebook: "work", Name: "standup", LastEdited: now.Add(-1 * time.Hour)},
		{Type: "store", Notebook: "journal", Name: "today", LastEdited: now.Add(-2 * time.Hour)},
	}

	// Update existing entry — should move to front with new timestamp.
	updated := Entry{Type: "store", Notebook: "work", Name: "standup", LastEdited: now}
	result := upsert(entries, updated)

	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].Notebook != "work" || result[0].Name != "standup" {
		t.Errorf("expected updated entry first, got %+v", result[0])
	}
	if !result[0].LastEdited.Equal(now) {
		t.Errorf("timestamp not updated: got %v, want %v", result[0].LastEdited, now)
	}
}

func TestUpsertNewEntry(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Type: "store", Notebook: "work", Name: "standup", LastEdited: now.Add(-1 * time.Hour)},
	}

	newEntry := Entry{Type: "external", Path: "/tmp/notes.md", LastEdited: now}
	result := upsert(entries, newEntry)

	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].Type != "external" || result[0].Path != "/tmp/notes.md" {
		t.Errorf("expected new entry first, got %+v", result[0])
	}
}

func TestCap(t *testing.T) {
	now := time.Now()
	var entries []Entry
	for i := 0; i < MaxEntries; i++ {
		entries = append(entries, Entry{
			Type:       "store",
			Notebook:   "book",
			Name:       "note-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			LastEdited: now.Add(-time.Duration(i) * time.Hour),
		})
	}

	// Adding one more should cap at MaxEntries.
	extra := Entry{Type: "external", Path: "/tmp/extra.txt", LastEdited: now}
	result := upsert(entries, extra)

	if len(result) != MaxEntries {
		t.Errorf("got %d entries, want %d", len(result), MaxEntries)
	}
	if result[0].Type != "external" {
		t.Errorf("expected new entry first, got %+v", result[0])
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()

	// Create a real file for one store entry.
	bookDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "standup.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a real external file.
	extFile := filepath.Join(dir, "todo.txt")
	if err := os.WriteFile(extFile, []byte("todo"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	entries := []Entry{
		{Type: "store", Notebook: "work", Name: "standup", LastEdited: now},
		{Type: "store", Notebook: "work", Name: "deleted-note", LastEdited: now},
		{Type: "external", Path: extFile, LastEdited: now},
		{Type: "external", Path: "/nonexistent/file.txt", LastEdited: now},
	}

	result := Prune(entries, dir)
	if len(result) != 2 {
		t.Fatalf("got %d entries after prune, want 2", len(result))
	}
	if result[0].Name != "standup" {
		t.Errorf("expected standup, got %+v", result[0])
	}
	if result[1].Path != extFile {
		t.Errorf("expected %s, got %+v", extFile, result[1])
	}
}

func TestRemove(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Type: "store", Notebook: "work", Name: "standup", LastEdited: now},
		{Type: "external", Path: "/tmp/todo.txt", LastEdited: now},
		{Type: "store", Notebook: "journal", Name: "today", LastEdited: now},
	}

	result := Remove(entries, Entry{Type: "store", Notebook: "work", Name: "standup"})
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].Type != "external" {
		t.Errorf("expected external first, got %+v", result[0])
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recent.json")

	now := time.Now()
	entry := Entry{Type: "store", Notebook: "work", Name: "standup", LastEdited: now}

	if err := Record(path, entry); err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Notebook != "work" || entries[0].Name != "standup" {
		t.Errorf("entry mismatch: %+v", entries[0])
	}
}

func TestSameIdentity(t *testing.T) {
	tests := []struct {
		name string
		a, b Entry
		want bool
	}{
		{
			"same store entry",
			Entry{Type: "store", Notebook: "work", Name: "standup"},
			Entry{Type: "store", Notebook: "work", Name: "standup"},
			true,
		},
		{
			"different store name",
			Entry{Type: "store", Notebook: "work", Name: "standup"},
			Entry{Type: "store", Notebook: "work", Name: "retro"},
			false,
		},
		{
			"different store notebook",
			Entry{Type: "store", Notebook: "work", Name: "standup"},
			Entry{Type: "store", Notebook: "journal", Name: "standup"},
			false,
		},
		{
			"same external path",
			Entry{Type: "external", Path: "/tmp/a.txt"},
			Entry{Type: "external", Path: "/tmp/a.txt"},
			true,
		},
		{
			"different external path",
			Entry{Type: "external", Path: "/tmp/a.txt"},
			Entry{Type: "external", Path: "/tmp/b.txt"},
			false,
		},
		{
			"different types",
			Entry{Type: "store", Notebook: "work", Name: "standup"},
			Entry{Type: "external", Path: "/tmp/a.txt"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameIdentity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("sameIdentity(%+v, %+v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
