package render

import (
	"regexp"
	"strings"
)

// admonitionConfig holds the display properties for an admonition type.
type admonitionConfig struct {
	Icon  string
	Label string
	Color string // ANSI color parameter (e.g. "36" for cyan)
}

// admonitions maps GitHub-style admonition keywords to their display config.
var admonitions = map[string]admonitionConfig{
	"NOTE":      {Icon: "\u2192", Label: "Note", Color: "36"},      // cyan
	"TIP":       {Icon: "\u2713", Label: "Tip", Color: "32"},       // green
	"IMPORTANT": {Icon: "!", Label: "Important", Color: "1;35"},    // bold magenta
	"WARNING":   {Icon: "!", Label: "Warning", Color: "33"},        // yellow
	"CAUTION":   {Icon: "\u2717", Label: "Caution", Color: "31"},   // red
}

// ansiStripRe matches ANSI escape sequences so they can be removed for
// pattern matching while preserving the original string for output.
var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// admonitionTagRe matches [!TYPE] in ANSI-stripped text, where TYPE is one of
// the recognised admonition keywords. The match is case-insensitive to be
// forgiving, but the canonical GitHub syntax is uppercase.
var admonitionTagRe = regexp.MustCompile(`\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)

// isBlockquoteLine checks whether a raw (ANSI-containing) line is a Glamour
// blockquote line by looking for the │ border character in the stripped text.
func isBlockquoteLine(line string) bool {
	stripped := ansiStripRe.ReplaceAllString(line, "")
	trimmed := strings.TrimLeft(stripped, " ")
	return strings.HasPrefix(trimmed, "\u2502 ") || strings.HasPrefix(trimmed, "\u2502")
}

// RenderAdmonitions post-processes Glamour-rendered markdown to detect
// GitHub-style admonition blocks ([!NOTE], [!TIP], etc.) and restyle them
// with colored borders and header labels.
//
// The function scans each line for the [!TYPE] pattern. When found it:
//  1. Replaces the [!TYPE] tag with a colored header showing the icon and label.
//  2. Recolors the │ border on the header line and all subsequent blockquote
//     lines that belong to the same block.
func RenderAdmonitions(content string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	i := 0
	for i < len(lines) {
		line := lines[i]
		stripped := ansiStripRe.ReplaceAllString(line, "")

		match := admonitionTagRe.FindStringSubmatch(stripped)
		if match != nil && isBlockquoteLine(line) {
			adType := strings.ToUpper(match[1])
			cfg, ok := admonitions[adType]
			if !ok {
				result = append(result, line)
				i++
				continue
			}

			// Restyle the header line: replace the [!TYPE] tag text and
			// recolor the border.
			headerLine := restyleAdmonitionHeader(line, match[0], cfg)
			result = append(result, headerLine)
			i++

			// Recolor subsequent blockquote lines in the same block.
			for i < len(lines) && isBlockquoteLine(lines[i]) {
				result = append(result, recolorBorder(lines[i], cfg.Color))
				i++
			}
			continue
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// restyleAdmonitionHeader replaces the [!TYPE] tag in a rendered blockquote
// line with a colored "ICON Label" header and recolors the border.
func restyleAdmonitionHeader(line, tag string, cfg admonitionConfig) string {
	// Build the colored header text: "ICON Label"
	header := "\x1b[" + cfg.Color + "m" + cfg.Icon + " " + cfg.Label + "\x1b[0m"

	// We need to replace the [!TYPE] portion in the raw line. Because ANSI
	// codes are interspersed between the characters of [!TYPE], we build a
	// regex that matches the tag characters with optional ANSI codes between
	// them. We also consume a trailing space/ANSI if present.
	tagPattern := buildAnsiInterspersedPattern(tag)
	re := regexp.MustCompile(tagPattern + `(\x1b\[[0-9;]*m)?(\s?)`)

	line = re.ReplaceAllString(line, header+" ")

	// Recolor the border character.
	line = recolorBorder(line, cfg.Color)

	return line
}

// buildAnsiInterspersedPattern builds a regex pattern that matches the given
// literal text with optional ANSI escape sequences between each character.
// For example, "[!NOTE]" becomes a pattern matching "[", then optional ANSI,
// then "!", then optional ANSI, etc.
func buildAnsiInterspersedPattern(text string) string {
	ansiOpt := `(?:\x1b\[[0-9;]*m)*`
	var b strings.Builder
	for i, ch := range text {
		if i > 0 {
			b.WriteString(ansiOpt)
		}
		b.WriteString(regexp.QuoteMeta(string(ch)))
	}
	return b.String()
}

// recolorBorder replaces the default Glamour blockquote border color with
// the admonition color. The border character │ (U+2502) is preceded by
// a color code like \x1b[38;5;252m — we replace that with the admonition color.
func recolorBorder(line, color string) string {
	// Match: ANSI-color then "│ " then ANSI-reset, and replace the color.
	borderRe := regexp.MustCompile(`\x1b\[[\d;]*m(` + "\u2502" + ` )\x1b\[0m`)
	return borderRe.ReplaceAllString(line, "\x1b["+color+"m${1}\x1b[0m")
}
