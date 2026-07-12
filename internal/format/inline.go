package format

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Targeted ANSI SGR codes — no full resets, so outer block styles survive.
const (
	boldOn           = "\x1b[1m"
	boldOff          = "\x1b[22m"
	italicOn         = "\x1b[3m"
	italicOff        = "\x1b[23m"
	underlineOn      = "\x1b[4m"
	underlineOff     = "\x1b[24m"
	strikethroughOn  = "\x1b[9m"
	strikethroughOff = "\x1b[29m"
)

var inlineMarkdownLinkRe = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^\s)]+)\)`)

// InlineLinkSpan records a markdown link's display range after
// [label](url) has been rewritten for terminal display.
type InlineLinkSpan struct {
	Start    int
	End      int
	URLStart int
	URLEnd   int
	URL      string
}

// InlineLinkRenderStyle controls optional color styling for rendered inline
// links. Empty colors preserve the plain formatter behavior.
type InlineLinkRenderStyle struct {
	LabelColor string
	URLColor   string
	HoverColor string
	BaseColor  string
}

// InlineLinkDisplay replaces inline markdown links with readable terminal text
// and returns the link ranges in that display text. The URL stays visible so
// terminal URL detection can still support Cmd-click/open-link behavior.
func InlineLinkDisplay(raw string) (string, []InlineLinkSpan) {
	raw = StripControlChars(raw)
	matches := inlineMarkdownLinkRe.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 0 {
		return raw, nil
	}

	var out strings.Builder
	var spans []InlineLinkSpan
	last := 0
	runeCount := 0

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		before := raw[last:match[0]]
		out.WriteString(before)
		runeCount += utf8.RuneCountInString(before)

		label := raw[match[2]:match[3]]
		url := raw[match[4]:match[5]]
		displayURL := InlineLinkDisplayURL(url)
		display := displayURL
		urlStart := runeCount
		if label != "" && label != url && label != displayURL {
			display = label + "  " + displayURL
			urlStart += utf8.RuneCountInString(label + "  ")
		}
		start := runeCount
		out.WriteString(display)
		runeCount += utf8.RuneCountInString(display)
		spans = append(spans, InlineLinkSpan{
			Start:    start,
			End:      runeCount,
			URLStart: urlStart,
			URLEnd:   runeCount,
			URL:      url,
		})
		last = match[1]
	}

	after := raw[last:]
	out.WriteString(after)
	return out.String(), spans
}

// InlineLinkDisplayURL keeps rendered links compact while preserving a
// terminal-recognizable URL-like string.
func InlineLinkDisplayURL(raw string) string {
	display := strings.TrimPrefix(raw, "https://")
	display = strings.TrimPrefix(display, "http://")
	if strings.HasSuffix(display, "/") && strings.Count(display, "/") == 1 {
		display = strings.TrimSuffix(display, "/")
	}
	return display
}

// StripControlChars removes C0 (0x00–0x1F except \n \r \t) and DEL (0x7F)
// from the input. This defangs terminal escape injection (OSC title
// rewrites, DCS, palette resets, hyperlinks) — all of which start with
// ESC (0x1B) on modern terminals — coming from untrusted markdown
// bodies before the renderer composes its own SGR codes around the text.
//
// Operates byte-by-byte so 0x80–0xBF UTF-8 continuation bytes pass through
// untouched. Bare 8-bit C1 controls are not stripped.
func StripControlChars(s string) string {
	if !needsStrip(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\n' || c == '\r' || c == '\t':
			b.WriteByte(c)
		case c < 0x20 || c == 0x7F:
			// drop C0 + DEL
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func needsStrip(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		if c < 0x20 || c == 0x7F {
			return true
		}
	}
	return false
}

// RenderInlineMarkdown applies inline markdown formatting to plain text.
// Supports **bold**, *italic*, ~~strikethrough~~, and __underline__ with
// correct nesting. Uses targeted ANSI SGR codes (not full resets) so that
// outer block-level styles (e.g. heading bold) are preserved. Delimiter
// pairs may span newlines — useful when the caller has already
// word-wrapped text, which can split a **foo bar** marker across lines.
//
// Control characters (C0/C1) in the input are stripped first to defang
// terminal escape injection from untrusted markdown content (e.g. a
// pasted card body containing OSC sequences that retitle the terminal).
func RenderInlineMarkdown(text string) string {
	display, links := InlineLinkDisplay(text)
	return RenderInlineMarkdownWithLinks(display, links, "")
}

// RenderInlineMarkdownWithLinks applies inline markdown formatting to text
// that has already had inline links rewritten by InlineLinkDisplay. Link URLs
// stay visible for terminal URL detection; hovered links get a light emphasis
// for mouse feedback in view mode.
func RenderInlineMarkdownWithLinks(text string, links []InlineLinkSpan, hoverURL string) string {
	return RenderInlineMarkdownWithLinksStyled(text, links, hoverURL, InlineLinkRenderStyle{})
}

// RenderInlineMarkdownWithLinksStyled is RenderInlineMarkdownWithLinks with
// optional foreground colors for inline-link labels and visible URLs.
func RenderInlineMarkdownWithLinksStyled(text string, links []InlineLinkSpan, hoverURL string, linkStyle InlineLinkRenderStyle) string {
	text = StripControlChars(text)
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}

	pairs := findDelimiterPairs(runes)
	if len(pairs) == 0 && len(links) == 0 {
		return text
	}

	skip := make([]bool, n)
	for _, p := range pairs {
		for j := p.openStart; j < p.openEnd; j++ {
			skip[j] = true
		}
		for j := p.closeStart; j < p.closeEnd; j++ {
			skip[j] = true
		}
	}

	var buf strings.Builder
	buf.Grow(n * 2)
	var active styleFlags

	for i := 0; i < n; i++ {
		if skip[i] {
			continue
		}

		// Close active styles around newlines so styles don't bleed
		// into prefixes/indents added by the caller on the next line.
		// Reopen after the newline to continue the style visually.
		if runes[i] == '\n' {
			if active != (styleFlags{}) {
				saved := active
				emitTransitionWithBase(&buf, active, styleFlags{}, linkStyle.BaseColor)
				active = styleFlags{}
				buf.WriteRune('\n')
				emitTransitionWithBase(&buf, active, saved, linkStyle.BaseColor)
				active = saved
			} else {
				buf.WriteRune('\n')
			}
			continue
		}

		var flags styleFlags
		for _, p := range pairs {
			if i >= p.openEnd && i < p.closeStart {
				switch p.kind {
				case delimBold:
					flags.bold = true
				case delimItalic:
					flags.italic = true
				case delimStrikethrough:
					flags.strikethrough = true
				case delimUnderline:
					flags.underline = true
				}
			}
		}
		for _, link := range links {
			if i >= link.Start && i < link.End {
				if i >= link.URLStart && i < link.URLEnd {
					flags.color = linkStyle.URLColor
				} else {
					flags.color = linkStyle.LabelColor
				}
				if hoverURL != "" && link.URL == hoverURL && i >= link.URLStart && i < link.URLEnd {
					flags.linkHover = true
					if linkStyle.HoverColor != "" {
						flags.color = linkStyle.HoverColor
					}
				}
				break
			}
		}

		if flags != active {
			emitTransitionWithBase(&buf, active, flags, linkStyle.BaseColor)
			active = flags
		}
		buf.WriteRune(runes[i])
	}

	emitTransitionWithBase(&buf, active, styleFlags{}, linkStyle.BaseColor)
	return buf.String()
}

type delimKind int

const (
	delimBold delimKind = iota
	delimItalic
	delimStrikethrough
	delimUnderline
)

// delimPair records a matched open/close delimiter and the rune ranges it covers.
type delimPair struct {
	kind       delimKind
	openStart  int // first rune of opening delimiter
	openEnd    int // rune after opening delimiter (content starts here)
	closeStart int // first rune of closing delimiter
	closeEnd   int // rune after closing delimiter
}

type styleFlags struct {
	bold, italic, strikethrough, underline bool
	linkHover                              bool
	color                                  string
}

// emitTransition writes only the ANSI codes needed to move from one style set to another.
func emitTransition(buf *strings.Builder, from, to styleFlags) {
	emitTransitionWithBase(buf, from, to, "")
}

func emitTransitionWithBase(buf *strings.Builder, from, to styleFlags, baseColor string) {
	fromBold := from.bold || from.linkHover
	toBold := to.bold || to.linkHover
	if fromBold && !toBold {
		buf.WriteString(boldOff)
	} else if !fromBold && toBold {
		buf.WriteString(boldOn)
	}
	if from.italic && !to.italic {
		buf.WriteString(italicOff)
	} else if !from.italic && to.italic {
		buf.WriteString(italicOn)
	}
	fromUnderline := from.underline || from.linkHover
	toUnderline := to.underline || to.linkHover
	if fromUnderline && !toUnderline {
		buf.WriteString(underlineOff)
	} else if !fromUnderline && toUnderline {
		buf.WriteString(underlineOn)
	}
	if from.strikethrough && !to.strikethrough {
		buf.WriteString(strikethroughOff)
	} else if !from.strikethrough && to.strikethrough {
		buf.WriteString(strikethroughOn)
	}
	if from.color != to.color {
		buf.WriteString(colorTransition(to.color, baseColor))
	}
}

func colorTransition(color, fallback string) string {
	if color == "" || color == "-" {
		color = fallback
	}
	if color == "" || color == "-" {
		return "\x1b[39m"
	}
	return colorSGR(color)
}

func colorSGR(color string) string {
	color = strings.TrimSpace(color)
	if len(color) == 7 && color[0] == '#' {
		r, errR := strconv.ParseUint(color[1:3], 16, 8)
		g, errG := strconv.ParseUint(color[3:5], 16, 8)
		b, errB := strconv.ParseUint(color[5:7], 16, 8)
		if errR == nil && errG == nil && errB == nil {
			return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
		}
	}
	return "\x1b[39m"
}

// --- Pass 1: delimiter pair matching ---

type stackEntry struct {
	kind      delimKind
	openStart int
	openEnd   int
}

func findDelimiterPairs(runes []rune) []delimPair {
	n := len(runes)
	var pairs []delimPair
	var stack []stackEntry

	i := 0
	for i < n {
		// ~~ strikethrough
		if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
			if idx := findStack(stack, delimStrikethrough); idx >= 0 {
				entry := stack[idx]
				if i > entry.openEnd { // non-empty content
					pairs = append(pairs, delimPair{
						kind:       delimStrikethrough,
						openStart:  entry.openStart,
						openEnd:    entry.openEnd,
						closeStart: i,
						closeEnd:   i + 2,
					})
				}
				stack = stack[:idx]
			} else {
				stack = append(stack, stackEntry{delimStrikethrough, i, i + 2})
			}
			i += 2
			continue
		}

		// __ underline (word-boundary only)
		if i+1 < n && runes[i] == '_' && runes[i+1] == '_' {
			if idx := findStack(stack, delimUnderline); idx >= 0 {
				afterOk := (i+2 >= n) || !isWordChar(runes[i+2])
				if afterOk {
					entry := stack[idx]
					if i > entry.openEnd {
						pairs = append(pairs, delimPair{
							kind:       delimUnderline,
							openStart:  entry.openStart,
							openEnd:    entry.openEnd,
							closeStart: i,
							closeEnd:   i + 2,
						})
					}
					stack = stack[:idx]
					i += 2
					continue
				}
			}
			beforeOk := (i == 0) || !isWordChar(runes[i-1])
			if beforeOk {
				stack = append(stack, stackEntry{delimUnderline, i, i + 2})
				i += 2
				continue
			}
			i += 2
			continue
		}

		// * or ** (italic / bold)
		if runes[i] == '*' {
			starCount := 0
			j := i
			for j < n && runes[j] == '*' {
				starCount++
				j++
			}
			processStars(i, starCount, &stack, &pairs)
			i += starCount
			continue
		}

		i++
	}
	return pairs
}

func processStars(pos, starCount int, stack *[]stackEntry, pairs *[]delimPair) {
	remaining := starCount

	for remaining > 0 {
		boldIdx := findStack(*stack, delimBold)
		italicIdx := findStack(*stack, delimItalic)

		closeBold := false
		closeItalic := false
		if boldIdx >= 0 && italicIdx >= 0 {
			if italicIdx > boldIdx {
				closeItalic = remaining >= 1
			} else {
				closeBold = remaining >= 2
			}
		} else if boldIdx >= 0 {
			closeBold = remaining >= 2
		} else if italicIdx >= 0 {
			closeItalic = remaining >= 1
		}

		if closeBold {
			entry := (*stack)[boldIdx]
			closeStart := pos + (starCount - remaining)
			if closeStart > entry.openEnd {
				*pairs = append(*pairs, delimPair{
					kind:       delimBold,
					openStart:  entry.openStart,
					openEnd:    entry.openEnd,
					closeStart: closeStart,
					closeEnd:   closeStart + 2,
				})
			}
			*stack = (*stack)[:boldIdx]
			remaining -= 2
			continue
		}

		if closeItalic {
			entry := (*stack)[italicIdx]
			closeStart := pos + (starCount - remaining)
			if closeStart > entry.openEnd {
				*pairs = append(*pairs, delimPair{
					kind:       delimItalic,
					openStart:  entry.openStart,
					openEnd:    entry.openEnd,
					closeStart: closeStart,
					closeEnd:   closeStart + 1,
				})
			}
			*stack = (*stack)[:italicIdx]
			remaining--
			continue
		}

		// Open new delimiters.
		openPos := pos + (starCount - remaining)
		if remaining >= 2 {
			*stack = append(*stack, stackEntry{delimBold, openPos, openPos + 2})
			remaining -= 2
			continue
		}
		*stack = append(*stack, stackEntry{delimItalic, openPos, openPos + 1})
		remaining--
	}
}

func findStack(stack []stackEntry, kind delimKind) int {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].kind == kind {
			return i
		}
	}
	return -1
}

func isWordChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
