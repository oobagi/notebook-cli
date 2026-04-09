package theme

import (
	"strings"

	"charm.land/lipgloss/v2"
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
	Marker      string // top-level marker, e.g. "  •  "
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
	LabelAlign string // "left" (default), "center", or "right"
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

// DefinitionStyle controls definition list rendering.
type DefinitionStyle struct {
	Marker      string // prefix before definition, e.g. "  : "
	MarkerColor string // hex; "" means theme.Muted
	TermBold    bool   // whether to bold the term
}

// EmbedStyle controls embedded note link rendering.
type EmbedStyle struct {
	Icon  string // prefix icon, e.g. "↗ "
	Color string // hex; "" means theme.Accent
}

// BlockStyles groups all per-block-type formatting.
type BlockStyles struct {
	Heading1   HeadingStyle
	Heading2   HeadingStyle
	Heading3   HeadingStyle
	Bullet     ListStyle
	Numbered   NumberedStyle
	Checklist  ChecklistStyle
	Code       CodeStyle
	Quote      QuoteStyle
	Divider    DividerStyle
	Definition DefinitionStyle
	Embed      EmbedStyle
}

// DefaultBlockStyles returns the baseline block styles that match the original
// hardcoded rendering. Presets override specific fields from this base.
func DefaultBlockStyles() BlockStyles {
	return BlockStyles{
		Heading1: HeadingStyle{Text: TextStyle{Bold: true}},             // Color "" = Accent
		Heading2: HeadingStyle{Text: TextStyle{Bold: true, Color: "-"}}, // "-" = no color
		Heading3: HeadingStyle{Text: TextStyle{Bold: true, Faint: true, Color: "-"}},
		Bullet: ListStyle{
			Marker: "  \u2022  ",
		},
		Numbered: NumberedStyle{Format: "  %d. "},
		Checklist: ChecklistStyle{
			Checked:          " [x] ",
			Unchecked:        " [ ] ",
			CheckedBold:      true,
			CheckedTextFaint: true,
		},
		Code:       CodeStyle{LabelAlign: "left"},
		Quote:      QuoteStyle{Bar: "\u2502 "},
		Divider:    DividerStyle{Char: "\u2500", MaxWidth: 40},
		Definition: DefinitionStyle{Marker: "  : ", TermBold: true},
		Embed:      EmbedStyle{Icon: "\u2197 "},
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
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u25B8  " // ▸ (small right triangle)
		bs.Checklist.Checked = " [\u2713] "
		bs.Checklist.Unchecked = " [\u2219] " // ∙
		bs.Quote.Bar = "\u258F " // ▏ (left one eighth block)
		bs.Divider.Char = "\u2508" // ┈ (light quadruple dash)
		return bs
	}(),
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
		bs.Numbered.Format = "  %d\u2502 " // │ separator
		bs.Checklist.Checked = " \u25C9  " // ◉
		bs.Checklist.Unchecked = " \u25CE  " // ◎
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2503 " // ┃ (heavy vertical)
		bs.Divider.Char = "\u2500\u2508" // ─┈ (alternating)
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
		bs.Numbered.Format = "  %d\u00BB " // » separator
		bs.Checklist.Checked = " [\u2713] "
		bs.Checklist.Unchecked = " [\u00B7] "
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2595 " // ▕ (right one eighth block)
		bs.Divider.Char = "\u2500\u253C" // ─┼ (cross pattern)
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
		bs.Bullet.Marker = "  \u2605  " // ★
		bs.Numbered.Format = "  %d) "
		bs.Checklist.Checked = " \u25C6  " // ◆
		bs.Checklist.Unchecked = " \u25C7  " // ◇
		bs.Code.LabelAlign = "right"
		bs.Quote.Bar = "\u2503 " // ┃
		bs.Divider.Char = "\u2053" // ⁓ (swung dash)
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
		bs.Code.LabelAlign = "right"
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
		bs.Heading1.Text = TextStyle{Bold: true, Italic: true}
		bs.Heading3.Text = TextStyle{Bold: true, Italic: true, Color: "-"}
		bs.Bullet.Marker = "  \u25E6  " // ◦
		bs.Numbered.Format = "  %d\u2502 "
		bs.Checklist.Checked = "  \u2764  " // ❤ (heart)
		bs.Checklist.Unchecked = "  \u25CB  " // ○
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2502 "
		bs.Divider.Char = "\u2022" // • (bullet dot)
		bs.Divider.MaxWidth = 30
		return bs
	}(),
}

// Cyberpunk uses neon colors on a dark background with sharp ASCII styling.
var Cyberpunk = Theme{
	Name:       "cyberpunk",
	Primary:    "#FF00FF", // magenta
	Success:    "#00FF41", // matrix green
	Error:      "#FF0040", // neon red
	Warning:    "#FFD300", // electric yellow
	Muted:      "#8B008B", // dark magenta
	Border:     "#6A0DAD", // purple
	Accent:     "#00FFFF", // cyan
	StatusBg:   "#1A0033",
	StatusFg:   "#FF00FF",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Underline: true}
		bs.Bullet.Marker = "  \u00BB  " // »
		bs.Numbered.Format = "  [%d] "
		bs.Checklist.Checked = " [\u2588] "  // █ (full block)
		bs.Checklist.Unchecked = " [\u2591] " // ░ (light shade)
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2551 " // ║ (double vertical)
		bs.Divider.Char = "\u2550" // ═ (double horizontal)
		return bs
	}(),
}

// Minimal uses sparse, subtle markers for a distraction-free feel.
var Minimal = Theme{
	Name:       "minimal",
	Primary:    "#B0B0B0",
	Success:    "#A0C4A0",
	Error:      "#C49090",
	Warning:    "#C4C490",
	Muted:      "#606060",
	Border:     "#484848",
	Accent:     "#D0D0D0",
	StatusBg:   "#282828",
	StatusFg:   "#A0A0A0",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Color: "-"}
		bs.Heading2.Text = TextStyle{Bold: true, Color: "-"}
		bs.Heading3.Text = TextStyle{Faint: true, Color: "-"}
		bs.Bullet.Marker = "  \u2013  " // – (en dash)
		bs.Numbered.Format = "  %d\u2502 " // │ separator
		bs.Checklist.Checked = "  \u2714  "  // ✔
		bs.Checklist.Unchecked = "  \u00B7  " // ·
		bs.Checklist.CheckedBold = false
		bs.Code.LabelAlign = "right"
		bs.Quote.Bar = "\u2502 "
		bs.Divider.Char = "\u00B7" // · (middle dot)
		bs.Divider.MaxWidth = 20
		return bs
	}(),
}

// Retro uses old-school ASCII characters for a terminal nostalgia feel.
var Retro = Theme{
	Name:       "retro",
	Primary:    "#33FF33", // phosphor green
	Success:    "#33FF33",
	Error:      "#FF3333",
	Warning:    "#FFFF33",
	Muted:      "#338833",
	Border:     "#226622",
	Accent:     "#66FF66",
	StatusBg:   "#003300",
	StatusFg:   "#33FF33",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Underline: true}
		bs.Bullet.Marker = "  >  "
		bs.Numbered.Format = "  %d) "
		bs.Checklist.Checked = " [X] "
		bs.Checklist.Unchecked = " [_] "
		bs.Code.LabelAlign = "left"
		bs.Quote.Bar = ":: "
		bs.Divider.Char = "="
		return bs
	}(),
}

// Nord uses the Nord color palette with cool, muted tones.
var Nord = Theme{
	Name:       "nord",
	Primary:    "#88C0D0", // frost blue
	Success:    "#A3BE8C", // aurora green
	Error:      "#BF616A", // aurora red
	Warning:    "#EBCB8B", // aurora yellow
	Muted:      "#4C566A", // polar night
	Border:     "#3B4252",
	Accent:     "#81A1C1", // frost
	StatusBg:   "#2E3440",
	StatusFg:   "#D8DEE9",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u2043  " // ⁃ (hyphen bullet)
		bs.Checklist.Checked = " [\u2713] "  // ✓
		bs.Checklist.Unchecked = " [\u2219] " // ∙ (bullet operator)
		bs.Code.LabelAlign = "left"
		bs.Quote.Bar = "\u2502 "
		bs.Divider.Char = "\u2500"
		return bs
	}(),
}

// Solarized uses Ethan Schoonover's Solarized palette.
var Solarized = Theme{
	Name:       "solarized",
	Primary:    "#268BD2", // blue
	Success:    "#859900", // green
	Error:      "#DC322F", // red
	Warning:    "#B58900", // yellow
	Muted:      "#586E75", // base01
	Border:     "#073642", // base02
	Accent:     "#2AA198", // cyan
	StatusBg:   "#002B36", // base03
	StatusFg:   "#93A1A1", // base1
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Underline: true}
		bs.Bullet.Marker = "  \u2022  " // •
		bs.Checklist.Checked = " (\u2713) "
		bs.Checklist.Unchecked = " ( ) "
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2503 " // ┃ (heavy)
		bs.Divider.Char = "\u2504" // ┄ (dashed)
		return bs
	}(),
}

// Dracula uses the popular Dracula color palette.
var Dracula = Theme{
	Name:       "dracula",
	Primary:    "#BD93F9", // purple
	Success:    "#50FA7B", // green
	Error:      "#FF5555", // red
	Warning:    "#F1FA8C", // yellow
	Muted:      "#6272A4", // comment
	Border:     "#44475A", // current line
	Accent:     "#FF79C6", // pink
	StatusBg:   "#282A36", // background
	StatusFg:   "#F8F8F2", // foreground
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Italic: true}
		bs.Bullet.Marker = "  \u2766  " // ❦ (floral heart)
		bs.Numbered.Format = "  %d\u2502 "
		bs.Checklist.Checked = " \u25A3  " // ▣ (filled square)
		bs.Checklist.Unchecked = " \u25A2  " // ▢ (empty square)
		bs.Code.LabelAlign = "left"
		bs.Quote.Bar = "\u2590 " // ▐ (right half block)
		bs.Divider.Char = "\u2500\u2504" // ─┄ (solid-dashed)
		return bs
	}(),
}

// Tokyo uses Tokyo Night color palette — cool blues and purples.
var Tokyo = Theme{
	Name:       "tokyo",
	Primary:    "#7AA2F7", // blue
	Success:    "#9ECE6A", // green
	Error:      "#F7768E", // red
	Warning:    "#E0AF68", // yellow
	Muted:      "#565F89", // comment
	Border:     "#3B4261",
	Accent:     "#BB9AF7", // purple
	StatusBg:   "#1A1B26",
	StatusFg:   "#C0CAF5",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u2514  " // └
		bs.Numbered.Format = "  [%d] "
		bs.Checklist.Checked = " \u2611  " // ☑ (ballot box checked)
		bs.Checklist.Unchecked = " \u2610  " // ☐ (ballot box)
		bs.Code.LabelAlign = "right"
		bs.Quote.Bar = "\u258E " // ▎ (left one quarter block)
		bs.Divider.Char = "\u2500"
		return bs
	}(),
}

// Lavender uses soft purple and violet tones.
var Lavender = Theme{
	Name:       "lavender",
	Primary:    "#C4B5FD", // violet-300
	Success:    "#86EFAC", // green-300
	Error:      "#FCA5A5", // red-300
	Warning:    "#FDE68A", // amber-200
	Muted:      "#7C6F9C",
	Border:     "#4C3D6B",
	Accent:     "#DDD6FE", // violet-200
	StatusBg:   "#2E1065",
	StatusFg:   "#E9D5FF",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Italic: true}
		bs.Heading3.Text = TextStyle{Italic: true, Color: "-"}
		bs.Bullet.Marker = "  \u2727  " // ✧
		bs.Checklist.Checked = " \u2726  " // ✦
		bs.Checklist.Unchecked = " \u2727  " // ✧
		bs.Checklist.CheckedTextFaint = false
		bs.Code.LabelAlign = "center"
		bs.Quote.Bar = "\u2502 "
		bs.Divider.Char = "\u2025" // ‥ (two dot leader)
		bs.Divider.MaxWidth = 25
		return bs
	}(),
}

// Ember uses deep warm reds and oranges.
var Ember = Theme{
	Name:       "ember",
	Primary:    "#F97316", // orange-500
	Success:    "#84CC16", // lime-500
	Error:      "#EF4444", // red-500
	Warning:    "#EAB308", // yellow-500
	Muted:      "#92400E", // amber-800
	Border:     "#7C2D12", // orange-900
	Accent:     "#FB923C", // orange-400
	StatusBg:   "#431407",
	StatusFg:   "#FED7AA",
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Heading1.Text = TextStyle{Bold: true, Underline: true}
		bs.Bullet.Marker = "  \u25B6  " // ▶ (right triangle)
		bs.Numbered.Format = "  %d\u25B8 " // ▸ separator
		bs.Checklist.Checked = " \u25CF  " // ● (filled circle)
		bs.Checklist.Unchecked = " \u25CB  " // ○ (empty circle)
		bs.Code.LabelAlign = "left"
		bs.Quote.Bar = "\u2588 " // █ (full block)
		bs.Divider.Char = "\u2501" // ━ (heavy horizontal)
		return bs
	}(),
}

// Catppuccin uses the Catppuccin Mocha flavor palette.
var Catppuccin = Theme{
	Name:       "catppuccin",
	Primary:    "#89B4FA", // blue
	Success:    "#A6E3A1", // green
	Error:      "#F38BA8", // red
	Warning:    "#F9E2AF", // yellow
	Muted:      "#585B70", // surface2
	Border:     "#45475A", // surface1
	Accent:     "#CBA6F7", // mauve
	StatusBg:   "#1E1E2E", // base
	StatusFg:   "#CDD6F4", // text
	Background: "dark",
	Blocks: func() BlockStyles {
		bs := DefaultBlockStyles()
		bs.Bullet.Marker = "  \u2219  " // ∙ (bullet operator)
		bs.Numbered.Format = "  %d) "
		bs.Checklist.Checked = " ~\u2713~ " // ~✓~
		bs.Checklist.Unchecked = " ~ ~ "
		bs.Code.LabelAlign = "left"
		bs.Quote.Bar = "\u2502 "
		bs.Divider.Char = "\u2509" // ┉ (heavy quadruple dash)
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

// FromName returns a theme by name (case-insensitive). Unknown values fall
// back to Dark.
func FromName(name string) Theme {
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
		Ocean,
		Forest,
		Sunset,
		Monochrome,
		Rose,
		Cyberpunk,
		Minimal,
		Retro,
		Nord,
		Solarized,
		Dracula,
		Tokyo,
		Lavender,
		Ember,
		Catppuccin,
	}
}

