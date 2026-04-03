package recents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MaxEntries is the maximum number of recent entries to keep.
const MaxEntries = 50

// Entry represents a recently edited note or file.
type Entry struct {
	Type       string    `json:"type"`                 // "store" or "external"
	Notebook   string    `json:"notebook,omitempty"`    // store notes only
	Name       string    `json:"name,omitempty"`        // store notes only
	Path       string    `json:"path,omitempty"`        // external files only
	LastEdited time.Time `json:"last_edited"`
}

// DefaultPath returns the default path to the recents file:
// ~/.config/notebook/recent.json.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "notebook", "recent.json")
}

// Load reads the recents list from the given path.
// If the file does not exist, an empty slice is returned without error.
func Load(path string) ([]Entry, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read recents: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse recents: %w", err)
	}
	return entries, nil
}

// Save writes the recents list to the given path, creating directories
// as needed.
func Save(path string, entries []Entry) error {
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create recents directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode recents: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write recents: %w", err)
	}
	return nil
}

// Record upserts an entry into the recents list and saves to disk.
// If an entry with the same identity already exists, it is updated
// in place. The list is capped at MaxEntries.
func Record(path string, entry Entry) error {
	entries, err := Load(path)
	if err != nil {
		// Start fresh if the file is corrupted.
		entries = nil
	}

	entries = upsert(entries, entry)
	return Save(path, entries)
}

// Prune removes entries whose backing files no longer exist.
// For "store" entries, the file is <storeRoot>/<notebook>/<name>.md.
// For "external" entries, the file is entry.Path.
func Prune(entries []Entry, storeRoot string) []Entry {
	kept := make([]Entry, 0, len(entries))
	for _, e := range entries {
		switch e.Type {
		case "store":
			p := filepath.Join(storeRoot, e.Notebook, e.Name+".md")
			if _, err := os.Stat(p); err == nil {
				kept = append(kept, e)
			}
		case "external":
			if _, err := os.Stat(e.Path); err == nil {
				kept = append(kept, e)
			}
		default:
			// Unknown type — drop it.
		}
	}
	return kept
}

// Remove deletes a single entry from the list by identity match.
func Remove(entries []Entry, target Entry) []Entry {
	kept := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if !sameIdentity(e, target) {
			kept = append(kept, e)
		}
	}
	return kept
}

// upsert adds or updates an entry, keeping the list sorted by
// LastEdited descending (most recent first) and capped at MaxEntries.
func upsert(entries []Entry, entry Entry) []Entry {
	// Remove existing entry with same identity.
	result := make([]Entry, 0, len(entries)+1)
	for _, e := range entries {
		if !sameIdentity(e, entry) {
			result = append(result, e)
		}
	}

	// Prepend the new/updated entry (most recent first).
	result = append([]Entry{entry}, result...)

	// Cap at MaxEntries.
	if len(result) > MaxEntries {
		result = result[:MaxEntries]
	}
	return result
}

// sameIdentity returns true if two entries refer to the same note or file.
func sameIdentity(a, b Entry) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case "store":
		return a.Notebook == b.Notebook && a.Name == b.Name
	case "external":
		return a.Path == b.Path
	}
	return false
}
