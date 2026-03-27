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

func TestFromNamePresets(t *testing.T) {
	names := []string{"ocean", "forest", "sunset", "monochrome", "rose"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got := FromName(name)
			if got.Name != name {
				t.Errorf("FromName(%q) returned Name=%q, want %q", name, got.Name, name)
			}
		})
	}
}

func TestFromNameCaseInsensitive(t *testing.T) {
	got := FromName("Ocean")
	if got.Name != "ocean" {
		t.Errorf("FromName(\"Ocean\") returned Name=%q, want \"ocean\"", got.Name)
	}
}

func TestThemeHasAllColors(t *testing.T) {
	for _, th := range Presets() {
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
			if th.Border == "" {
				t.Error("Border is empty")
			}
			if th.Accent == "" {
				t.Error("Accent is empty")
			}
			if th.StatusBg == "" {
				t.Error("StatusBg is empty")
			}
			if th.StatusFg == "" {
				t.Error("StatusFg is empty")
			}
			if th.Background == "" {
				t.Error("Background is empty")
			}
			if th.GlamourStyle == "" {
				t.Error("GlamourStyle is empty")
			}
		})
	}
}

func TestPresetsOrder(t *testing.T) {
	presets := Presets()
	if len(presets) < 7 {
		t.Fatalf("Presets() returned %d presets, want at least 7", len(presets))
	}
	if presets[0].Name != "dark" {
		t.Errorf("Presets()[0].Name = %q, want \"dark\"", presets[0].Name)
	}
	if presets[1].Name != "light" {
		t.Errorf("Presets()[1].Name = %q, want \"light\"", presets[1].Name)
	}
}

func TestPresetsUniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range Presets() {
		if seen[p.Name] {
			t.Errorf("duplicate preset name: %q", p.Name)
		}
		seen[p.Name] = true
	}
}

func TestPresetByName(t *testing.T) {
	for _, p := range Presets() {
		t.Run(p.Name, func(t *testing.T) {
			got, ok := PresetByName(p.Name)
			if !ok {
				t.Fatalf("PresetByName(%q) returned ok=false", p.Name)
			}
			if got.Name != p.Name {
				t.Errorf("PresetByName(%q).Name = %q", p.Name, got.Name)
			}
		})
	}
}

func TestPresetByNameNotFound(t *testing.T) {
	_, ok := PresetByName("nonexistent")
	if ok {
		t.Error("PresetByName(\"nonexistent\") returned ok=true, want false")
	}
}

func TestResolveGlamourStyleAuto(t *testing.T) {
	SetTheme(Dark)
	style, source := ResolveGlamourStyle("auto")
	if style != "dark" {
		t.Errorf("ResolveGlamourStyle(\"auto\") = %q, want %q", style, "dark")
	}
	if source != StyleBuiltin {
		t.Errorf("expected source=StyleBuiltin for auto, got %d", source)
	}
}

func TestResolveGlamourStyleEmpty(t *testing.T) {
	SetTheme(Light)
	style, source := ResolveGlamourStyle("")
	if style != "light" {
		t.Errorf("ResolveGlamourStyle(\"\") = %q, want %q", style, "light")
	}
	if source != StyleBuiltin {
		t.Errorf("expected source=StyleBuiltin for empty, got %d", source)
	}
}

func TestResolveGlamourStyleBuiltin(t *testing.T) {
	SetTheme(Dark)

	builtins := []string{"dark", "light", "dracula", "tokyo-night", "notty", "ascii", "pink"}
	for _, name := range builtins {
		t.Run(name, func(t *testing.T) {
			style, source := ResolveGlamourStyle(name)
			if style != name {
				t.Errorf("ResolveGlamourStyle(%q) = %q, want %q", name, style, name)
			}
			if source != StyleBuiltin {
				t.Errorf("expected source=StyleBuiltin for built-in %q, got %d", name, source)
			}
		})
	}
}

func TestResolveGlamourStyleCommunity(t *testing.T) {
	SetTheme(Dark)

	style, source := ResolveGlamourStyle("gruvbox")
	if style != "gruvbox" {
		t.Errorf("ResolveGlamourStyle(\"gruvbox\") = %q, want %q", style, "gruvbox")
	}
	if source != StyleCommunity {
		t.Errorf("expected source=StyleCommunity for gruvbox, got %d", source)
	}
}

func TestResolveGlamourStyleCustomFile(t *testing.T) {
	SetTheme(Dark)

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(jsonPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write custom style: %v", err)
	}

	style, source := ResolveGlamourStyle(jsonPath)
	if style != jsonPath {
		t.Errorf("ResolveGlamourStyle(%q) = %q, want file path", jsonPath, style)
	}
	if source != StyleFile {
		t.Errorf("expected source=StyleFile for custom JSON file, got %d", source)
	}
}

func TestResolveGlamourStyleMissingFile(t *testing.T) {
	SetTheme(Dark)

	style, source := ResolveGlamourStyle("/nonexistent/style.json")
	if style != "dark" {
		t.Errorf("ResolveGlamourStyle with missing file = %q, want theme default %q", style, "dark")
	}
	if source != StyleBuiltin {
		t.Errorf("expected source=StyleBuiltin for missing file, got %d", source)
	}
}

func TestResolveGlamourStyleUnknownValue(t *testing.T) {
	SetTheme(Light)

	style, source := ResolveGlamourStyle("nonexistent-style")
	if style != "light" {
		t.Errorf("ResolveGlamourStyle(\"nonexistent-style\") = %q, want theme default %q", style, "light")
	}
	if source != StyleBuiltin {
		t.Errorf("expected source=StyleBuiltin for unknown value, got %d", source)
	}
}
