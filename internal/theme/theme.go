package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// allPresets holds every named preset in display order.
// Populated in init() after all package-level vars are defined.
var allPresets []Theme

// TextStyle holds lipgloss text formatting attributes.
// Color is a hex string; "" means inherit a fallback color from the parent theme.
type TextStyle struct {
	Bold      bool
	Italic    bool
	Faint     bool
	Underline bool
	Color     string
}

// ToLipgloss converts a TextStyle to a lipgloss.Style.
// If ts.Color is empty, fallbackColor is used instead.
func (ts TextStyle) ToLipgloss(fallbackColor string) lipgloss.Style {
	s := lipgloss.NewStyle()
	if ts.Bold {
		s = s.Bold(true)
	}
	if ts.Italic {
		s = s.Italic(true)
	}
	if ts.Faint {
		s = s.Faint(true)
	}
	if ts.Underline {
		s = s.Underline(true)
	}
	color := ts.Color
	if color == "" {
		color = fallbackColor
	}
	if color != "" && color != "-" {
		s = s.Foreground(lipgloss.Color(color))
	}
	return s
}

// HeadingStyle controls how a heading level renders.
type HeadingStyle struct {
	Text TextStyle
}

// ListStyle controls bullet list rendering.
type ListStyle struct {
	Marker      string // e.g. "  •  ", "  ‣  ", "  →  "
	MarkerColor string // hex; "" means theme.Muted
}

// NumberedStyle controls numbered list rendering.
type NumberedStyle struct {
	Format      string // fmt.Sprintf template with one %d, e.g. "  %d. "
	MarkerColor string // hex; "" means theme.Muted
}

// ChecklistStyle controls checklist item rendering.
type ChecklistStyle struct {
	Checked          string // e.g. " [x] ", " <x> "
	Unchecked        string // e.g. " [ ] ", " < > "
	CheckedColor     string // hex; "" means theme.Accent
	UncheckedColor   string // hex; "" means theme.Muted
	CheckedBold      bool
	CheckedTextFaint bool
}

// CodeStyle controls code block rendering.
type CodeStyle struct {
	LabelPosition string // "inside", "top", or "bottom"
}

// QuoteStyle controls block quote rendering.
type QuoteStyle struct {
	Bar      string // e.g. "│ ", "┃ ", "| "
	BarColor string // hex; "" means theme.Muted
}

// DividerStyle controls horizontal rule rendering.
type DividerStyle struct {
	Char     string // e.g. "─", "━", "-"
	MaxWidth int    // 0 means 40
	Color    string // hex; "" means theme.Accent (active) / theme.Muted (inactive)
}

// BlockStyles groups all per-block-type formatting.
type BlockStyles struct {
	Heading1  HeadingStyle
	Heading2  HeadingStyle
	Heading3  HeadingStyle
	Bullet    ListStyle
	Numbered  NumberedStyle
	Checklist ChecklistStyle
	Code      CodeStyle
	Quote     QuoteStyle
	Divider   DividerStyle
}

// DefaultBlockStyles returns the baseline block styles that match the original
// hardcoded rendering. Presets override specific fields from this base.
func DefaultBlockStyles() BlockStyles {
	return BlockStyles{
		Heading1: HeadingStyle{Text: TextStyle{Bold: true}},             // Color "" = Accent
		Heading2: HeadingStyle{Text: TextStyle{Bold: true, Color: "-"}}, // "-" = no color
		Heading3: HeadingStyle{Text: TextStyle{Bold: true, Faint: true, Color: "-"}},
		Bullet:   ListStyle{Marker: "  \u2022  "},
		Numbered: NumberedStyle{Format: "  %d. "},
		Checklist: ChecklistStyle{
			Checked:          " [x] ",
			Unchecked:        " [ ] ",
			CheckedBold:      true,
			CheckedTextFaint: true,
		},
		Code:    CodeStyle{LabelPosition: "inside"},
		Quote:   QuoteStyle{Bar: "\u2502 "},
		Divider: DividerStyle{Char: "\u2500", MaxWidth: 40},
	}
}

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
	StatusFg   string // status bar foreground
	Background string // metadata: "dark" or "light" — target background type
	Blocks     BlockStyles
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
	Background: "dark",
	Blocks:     DefaultBlockStyles(),
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
	Background: "light",
	Blocks:     DefaultBlockStyles(),
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
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u2023  " // ‣
		bs.Quote.Bar = "\u2503 "         // ┃ (heavy vertical)
		return bs
	}(),
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
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u2192  " // →
		bs.Checklist.Checked = " [\u2713] "
		bs.Checklist.Unchecked = " [\u00B7] "
		bs.Code.LabelPosition = "top"
		return bs
	}(),
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
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Italic: true}
		bs.Numbered.Format = "  %d) "
		bs.Checklist.Checked = " <x> "
		bs.Checklist.Unchecked = " < > "
		return bs
	}(),
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
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Color: "-"} // no accent color
		bs.Bullet.Marker = "  *  "
		bs.Code.LabelPosition = "bottom"
		bs.Quote.Bar = "| "
		bs.Divider.Char = "-"
		return bs
	}(),
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
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading3.Text = TextStyle{Bold: true, Italic: true, Color: "-"}
		bs.Bullet.Marker = "  \u25E6  " // ◦
		bs.Checklist.Checked = "  \u2713  "
		bs.Checklist.Unchecked = "  \u25CB  "
		return bs
	}(),
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

