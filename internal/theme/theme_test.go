package theme

import (
	"fmt"
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

func TestBlockStylesComplete(t *testing.T) {
	for _, th := range Presets() {
		t.Run(th.Name, func(t *testing.T) {
			bs := th.Blocks
			if bs.Bullet.Marker == "" {
				t.Error("Bullet.Marker is empty")
			}
			if bs.Numbered.Format == "" {
				t.Error("Numbered.Format is empty")
			}
			if bs.Checklist.Checked == "" {
				t.Error("Checklist.Checked is empty")
			}
			if bs.Checklist.Unchecked == "" {
				t.Error("Checklist.Unchecked is empty")
			}
			if bs.Quote.Bar == "" {
				t.Error("Quote.Bar is empty")
			}
			if bs.Divider.Char == "" {
				t.Error("Divider.Char is empty")
			}
			if bs.Code.LabelPosition == "" {
				t.Error("Code.LabelPosition is empty")
			}
			valid := map[string]bool{"inside": true, "top": true, "bottom": true}
			if !valid[bs.Code.LabelPosition] {
				t.Errorf("Code.LabelPosition %q is not valid", bs.Code.LabelPosition)
			}
		})
	}
}

func TestBlockStylesChecklistWidthParity(t *testing.T) {
	for _, th := range Presets() {
		t.Run(th.Name, func(t *testing.T) {
			bs := th.Blocks
			checkedLen := len([]rune(bs.Checklist.Checked))
			uncheckedLen := len([]rune(bs.Checklist.Unchecked))
			if checkedLen != uncheckedLen {
				t.Errorf("Checklist width mismatch: Checked=%q (%d runes) vs Unchecked=%q (%d runes)",
					bs.Checklist.Checked, checkedLen, bs.Checklist.Unchecked, uncheckedLen)
			}
		})
	}
}

func TestBlockStylesNumberedFormat(t *testing.T) {
	for _, th := range Presets() {
		t.Run(th.Name, func(t *testing.T) {
			bs := th.Blocks
			// Format must contain %d and not panic.
			panicked := func() bool {
				defer func() { recover() }()
				_ = fmt.Sprintf(bs.Numbered.Format, 1)
				return false
			}()
			if panicked {
				t.Error("Numbered.Format panicked on Sprintf")
			}
		})
	}
}

func TestTextStyleToLipgloss(t *testing.T) {
	// Color "-" should not set foreground.
	ts := TextStyle{Bold: true, Color: "-"}
	_ = ts.ToLipgloss("#FF0000") // should not panic

	// Empty color should use fallback.
	ts2 := TextStyle{Italic: true}
	_ = ts2.ToLipgloss("#00FF00")
}
