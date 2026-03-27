package theme

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// allPresets holds every named preset in display order.
// Populated in init() after all package-level vars are defined.
var allPresets []Theme

// Theme holds color values tuned for a specific terminal background.
type Theme struct {
	Name         string // preset name, e.g. "dark", "ocean"
	Primary      string // main accent (cyan on dark, blue on light)
	Success      string // confirmations
	Error        string // errors
	Warning      string // cautions
	Muted        string // secondary/dim text
	Border       string // border/separator color
	Accent       string // accent color for selections/highlights
	StatusBg     string // status bar background
	StatusFg     string // status bar foreground
	Background   string // metadata: "dark" or "light" — target background type
	GlamourStyle string // passed to glamour.WithStandardStyle
}

// Dark is the color scheme for terminals with a dark background.
var Dark = Theme{
	Name:         "dark",
	Primary:      "#00BFFF", // cyan
	Success:      "#00E676", // green
	Error:        "#FF5252", // red
	Warning:      "#FFD740", // yellow
	Muted:        "#777777", // dim gray
	Border:       "#555555",
	Accent:       "#00BFFF",
	StatusBg:     "#333333",
	StatusFg:     "#CCCCCC",
	Background:   "dark",
	GlamourStyle: "dark",
}

// Light is the color scheme for terminals with a light background.
var Light = Theme{
	Name:         "light",
	Primary:      "#0057B7", // blue
	Success:      "#2E7D32", // green
	Error:        "#C62828", // red
	Warning:      "#F57F17", // yellow/amber
	Muted:        "#999999", // mid gray
	Border:       "#CCCCCC",
	Accent:       "#0057B7",
	StatusBg:     "#E0E0E0",
	StatusFg:     "#333333",
	Background:   "light",
	GlamourStyle: "light",
}

// Ocean uses deep blue tones on a dark background.
var Ocean = Theme{
	Name:         "ocean",
	Primary:      "#4FC3F7", // light blue
	Success:      "#00E676", // green
	Error:        "#FF5252", // red
	Warning:      "#FFD740", // yellow
	Muted:        "#6688AA", // steel blue-gray
	Border:       "#1A5276",
	Accent:       "#81D4FA",
	StatusBg:     "#0D3B66",
	StatusFg:     "#B3E5FC",
	Background:   "dark",
	GlamourStyle: "dark",
}

// Forest uses green tones on a dark background.
var Forest = Theme{
	Name:         "forest",
	Primary:      "#66BB6A", // medium green
	Success:      "#A5D6A7", // light green
	Error:        "#EF5350", // red
	Warning:      "#FFD54F", // amber
	Muted:        "#6B8E6B", // sage
	Border:       "#2E5E2E",
	Accent:       "#81C784",
	StatusBg:     "#1B4332",
	StatusFg:     "#C8E6C9",
	Background:   "dark",
	GlamourStyle: "dark",
}

// Sunset uses warm orange and amber tones on a dark background.
var Sunset = Theme{
	Name:         "sunset",
	Primary:      "#FFB74D", // orange
	Success:      "#81C784", // green
	Error:        "#E57373", // soft red
	Warning:      "#FFF176", // yellow
	Muted:        "#A1887F", // brown-gray
	Border:       "#5D4037",
	Accent:       "#FFCC80",
	StatusBg:     "#4E342E",
	StatusFg:     "#FFECB3",
	Background:   "dark",
	GlamourStyle: "dark",
}

// Monochrome uses grayscale tones on a dark background.
var Monochrome = Theme{
	Name:         "monochrome",
	Primary:      "#E0E0E0", // near white
	Success:      "#BDBDBD", // light gray
	Error:        "#F5F5F5", // bright white
	Warning:      "#EEEEEE", // off-white
	Muted:        "#757575", // mid gray
	Border:       "#616161",
	Accent:       "#FAFAFA",
	StatusBg:     "#424242",
	StatusFg:     "#E0E0E0",
	Background:   "dark",
	GlamourStyle: "dark",
}

// Rose uses pink tones on a dark background.
var Rose = Theme{
	Name:         "rose",
	Primary:      "#F48FB1", // pink
	Success:      "#A5D6A7", // light green
	Error:        "#EF5350", // red
	Warning:      "#FFD54F", // amber
	Muted:        "#AD8B9E", // dusty rose
	Border:       "#6D3B55",
	Accent:       "#F8BBD0",
	StatusBg:     "#4A1942",
	StatusFg:     "#FCE4EC",
	Background:   "dark",
	GlamourStyle: "dark",
}

// current holds the active theme. It defaults to Dark and is set during
// initialization via SetTheme or the Detect/FromName helpers.
var current = Dark

// Current returns the active theme.
func Current() Theme {
	return current
}

// SetTheme sets the active theme.
func SetTheme(t Theme) {
	current = t
}

// Detect uses lipgloss to query the terminal background color and returns
// the appropriate theme. When detection is not possible (e.g. piped output),
// it defaults to Dark.
func Detect() Theme {
	if lipgloss.HasDarkBackground() {
		return Dark
	}
	return Light
}

// FromName returns a theme by name. Accepted values include "dark", "light",
// "auto", and any preset name (case-insensitive). Unknown values fall back to
// Dark.
func FromName(name string) Theme {
	if strings.EqualFold(name, "auto") {
		return Detect()
	}
	if t, ok := PresetByName(name); ok {
		return t
	}
	return Dark
}

// Presets returns all named presets in display order.
func Presets() []Theme {
	dst := make([]Theme, len(allPresets))
	copy(dst, allPresets)
	return dst
}

// PresetByName returns the preset with the given name (case-insensitive).
func PresetByName(name string) (Theme, bool) {
	for _, t := range allPresets {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return Theme{}, false
}

func init() {
	allPresets = []Theme{
		Dark,
		Light,
		Ocean,
		Forest,
		Sunset,
		Monochrome,
		Rose,
	}
}

// builtinGlamourStyles lists the style names that glamour ships with.
// "auto" is handled separately before this map is consulted.
var builtinGlamourStyles = map[string]bool{
	"dark":        true,
	"light":       true,
	"dracula":     true,
	"tokyo-night": true,
	"notty":       true,
	"ascii":       true,
	"pink":        true,
}

// ResolveGlamourStyle determines the glamour style to use based on the
// user's glamour_style config value and the active theme. The returned
// string is either a built-in style name or an absolute path to a JSON
// style file. The boolean indicates whether the result is a file path.
//
// When glamourCfg is empty or "auto", the theme's own GlamourStyle is used
// (which is "dark" or "light" depending on terminal detection).
func ResolveGlamourStyle(glamourCfg string) (style string, isFilePath bool) {
	glamourCfg = strings.TrimSpace(glamourCfg)

	// Empty or "auto" — defer to the active theme's default glamour style.
	if glamourCfg == "" || glamourCfg == "auto" {
		return current.GlamourStyle, false
	}

	// A known built-in style name.
	if builtinGlamourStyles[glamourCfg] {
		return glamourCfg, false
	}

	// Treat as a file path to a custom JSON style. If the file exists,
	// return the path; otherwise fall back to the theme default.
	if _, err := os.Stat(glamourCfg); err == nil {
		return glamourCfg, true
	}

	// Unknown value — fall back to theme default.
	return current.GlamourStyle, false
}
