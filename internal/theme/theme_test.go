package theme

import "testing"

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
