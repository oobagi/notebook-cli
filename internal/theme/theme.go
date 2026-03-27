package theme

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds color values tuned for a specific terminal background.
type Theme struct {
	Name         string // "dark" or "light"
	Primary      string // main accent (cyan on dark, blue on light)
	Success      string // confirmations
	Error        string // errors
	Warning      string // cautions
	Muted        string // secondary/dim text
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
	GlamourStyle: "light",
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

// FromName returns a theme by name. Accepted values are "dark", "light", and
// "auto". Any other value falls back to Dark.
func FromName(name string) Theme {
	switch name {
	case "dark":
		return Dark
	case "light":
		return Light
	case "auto":
		return Detect()
	default:
		return Dark
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
