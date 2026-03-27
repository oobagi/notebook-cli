package styles

import (
	"encoding/json"
	"testing"
)

func TestList(t *testing.T) {
	names := List()

	want := []string{"catppuccin-mocha", "gruvbox", "nord", "solarized-dark"}
	if len(names) != len(want) {
		t.Fatalf("List() returned %d names, want %d: %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestLoad(t *testing.T) {
	for _, name := range List() {
		t.Run(name, func(t *testing.T) {
			data, err := Load(name)
			if err != nil {
				t.Fatalf("Load(%q) error: %v", name, err)
			}
			if len(data) == 0 {
				t.Fatalf("Load(%q) returned empty data", name)
			}

			// Verify it is valid JSON.
			var obj map[string]interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatalf("Load(%q) returned invalid JSON: %v", name, err)
			}

			// Verify expected top-level keys exist.
			for _, key := range []string{"document", "heading", "h1", "code_block"} {
				if _, ok := obj[key]; !ok {
					t.Errorf("Load(%q) JSON missing expected key %q", name, key)
				}
			}
		})
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("nonexistent-style")
	if err == nil {
		t.Fatal("Load(\"nonexistent-style\") should return an error")
	}
}

func TestHas(t *testing.T) {
	// Existing styles should return true.
	for _, name := range List() {
		if !Has(name) {
			t.Errorf("Has(%q) = false, want true", name)
		}
	}

	// Nonexistent style should return false.
	if Has("nonexistent-style") {
		t.Error("Has(\"nonexistent-style\") = true, want false")
	}
}
