package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectReturnsTheme(t *testing.T) {
	got := Detect()
	if got.Name != "dark" && got.Name != "light" {
		t.Errorf("Detect() returned theme with Name=%q, want \"dark\" or \"light\"", got.Name)
	}
}

func TestFromNameDark(t *testing.T) {
	got := FromName("dark")
	if got.Name != "dark" {
		t.Errorf("FromName(\"dark\") returned Name=%q, want \"dark\"", got.Name)
	}
}

func TestFromNameLight(t *testing.T) {
	got := FromName("light")
	if got.Name != "light" {
		t.Errorf("FromName(\"light\") returned Name=%q, want \"light\"", got.Name)
	}
}

func TestFromNameAuto(t *testing.T) {
	got := FromName("auto")
	if got.Name != "dark" && got.Name != "light" {
		t.Errorf("FromName(\"auto\") returned Name=%q, want \"dark\" or \"light\"", got.Name)
	}
}

func TestFromNameInvalid(t *testing.T) {
	got := FromName("invalid")
	if got.Name != "dark" {
		t.Errorf("FromName(\"invalid\") returned Name=%q, want \"dark\"", got.Name)
	}
}

func TestThemeHasAllColors(t *testing.T) {
	for _, th := range []Theme{Dark, Light} {
		t.Run(th.Name, func(t *testing.T) {
			if th.Primary == "" {
				t.Error("Primary is empty")
			}
			if th.Success == "" {
				t.Error("Success is empty")
			}
			if th.Error == "" {
				t.Error("Error is empty")
			}
			if th.Warning == "" {
				t.Error("Warning is empty")
			}
			if th.Muted == "" {
				t.Error("Muted is empty")
			}
			if th.GlamourStyle == "" {
				t.Error("GlamourStyle is empty")
			}
		})
	}
}

func TestResolveGlamourStyleAuto(t *testing.T) {
	SetTheme(Dark)
	style, isFile := ResolveGlamourStyle("auto")
	if style != "dark" {
		t.Errorf("ResolveGlamourStyle(\"auto\") = %q, want %q", style, "dark")
	}
	if isFile {
		t.Error("expected isFile=false for auto")
	}
}

func TestResolveGlamourStyleEmpty(t *testing.T) {
	SetTheme(Light)
	style, isFile := ResolveGlamourStyle("")
	if style != "light" {
		t.Errorf("ResolveGlamourStyle(\"\") = %q, want %q", style, "light")
	}
	if isFile {
		t.Error("expected isFile=false for empty")
	}
}

func TestResolveGlamourStyleBuiltin(t *testing.T) {
	SetTheme(Dark)

	builtins := []string{"dark", "light", "dracula", "tokyo-night", "notty", "ascii", "pink"}
	for _, name := range builtins {
		t.Run(name, func(t *testing.T) {
			style, isFile := ResolveGlamourStyle(name)
			if style != name {
				t.Errorf("ResolveGlamourStyle(%q) = %q, want %q", name, style, name)
			}
			if isFile {
				t.Errorf("expected isFile=false for built-in %q", name)
			}
		})
	}
}

func TestResolveGlamourStyleCustomFile(t *testing.T) {
	SetTheme(Dark)

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(jsonPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write custom style: %v", err)
	}

	style, isFile := ResolveGlamourStyle(jsonPath)
	if style != jsonPath {
		t.Errorf("ResolveGlamourStyle(%q) = %q, want file path", jsonPath, style)
	}
	if !isFile {
		t.Error("expected isFile=true for custom JSON file")
	}
}

func TestResolveGlamourStyleMissingFile(t *testing.T) {
	SetTheme(Dark)

	style, isFile := ResolveGlamourStyle("/nonexistent/style.json")
	if style != "dark" {
		t.Errorf("ResolveGlamourStyle with missing file = %q, want theme default %q", style, "dark")
	}
	if isFile {
		t.Error("expected isFile=false for missing file")
	}
}

func TestResolveGlamourStyleUnknownValue(t *testing.T) {
	SetTheme(Light)

	style, isFile := ResolveGlamourStyle("nonexistent-style")
	if style != "light" {
		t.Errorf("ResolveGlamourStyle(\"nonexistent-style\") = %q, want theme default %q", style, "light")
	}
	if isFile {
		t.Error("expected isFile=false for unknown value")
	}
}
