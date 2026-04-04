package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// HintsPath returns the path to the dismissed hints file.
func HintsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "notebook", "hints.json")
}

// LoadDismissedHints reads the set of dismissed hint IDs from disk.
func LoadDismissedHints() map[string]bool {
	path := HintsPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil
	}
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// DismissHint adds a hint ID to the dismissed set and persists it.
func DismissHint(id string) {
	dismissed := LoadDismissedHints()
	if dismissed == nil {
		dismissed = make(map[string]bool)
	}
	dismissed[id] = true
	saveDismissedHints(dismissed)
}

// ResetHints clears all dismissed hints.
func ResetHints() {
	path := HintsPath()
	if path != "" {
		os.Remove(path)
	}
}

func saveDismissedHints(m map[string]bool) {
	path := HintsPath()
	if path == "" {
		return
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(path, data, 0o644)
}
