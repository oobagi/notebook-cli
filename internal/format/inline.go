package format

import (
	"strings"
	"unicode"
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

// RenderInlineMarkdown applies inline markdown formatting to plain text.
// Supports **bold**, *italic*, ~~strikethrough~~, and __underline__ with
// correct nesting. Uses targeted ANSI SGR codes (not full resets) so that
// outer block-level styles (e.g. heading bold) are preserved.
func RenderInlineMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = renderInlineLine(line)
	}
	return strings.Join(lines, "\n")
}

type delimKind int

const (
	delimBold          delimKind = iota
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
}

func renderInlineLine(line string) string {
	runes := []rune(line)
	n := len(runes)
	if n == 0 {
		return ""
	}

	// Pass 1: find all matched delimiter pairs.
	pairs := findDelimiterPairs(runes)
	if len(pairs) == 0 {
		return line
	}

	// Mark delimiter runes for removal.
	skip := make([]bool, n)
	for _, p := range pairs {
		for j := p.openStart; j < p.openEnd; j++ {
			skip[j] = true
		}
		for j := p.closeStart; j < p.closeEnd; j++ {
			skip[j] = true
		}
	}

	// Pass 2: emit text with targeted ANSI style transitions.
	var buf strings.Builder
	buf.Grow(n * 2)
	var active styleFlags

	for i := 0; i < n; i++ {
		if skip[i] {
			continue
		}

		// Determine which styles are active at this rune.
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

		if flags != active {
			emitTransition(&buf, active, flags)
			active = flags
		}
		buf.WriteRune(runes[i])
	}

	// Disable any styles still open.
	emitTransition(&buf, active, styleFlags{})
	return buf.String()
}

// emitTransition writes only the ANSI codes needed to move from one style set to another.
func emitTransition(buf *strings.Builder, from, to styleFlags) {
	if from.bold && !to.bold {
		buf.WriteString(boldOff)
	} else if !from.bold && to.bold {
		buf.WriteString(boldOn)
	}
	if from.italic && !to.italic {
		buf.WriteString(italicOff)
	} else if !from.italic && to.italic {
		buf.WriteString(italicOn)
	}
	if from.underline && !to.underline {
		buf.WriteString(underlineOff)
	} else if !from.underline && to.underline {
		buf.WriteString(underlineOn)
	}
	if from.strikethrough && !to.strikethrough {
		buf.WriteString(strikethroughOff)
	} else if !from.strikethrough && to.strikethrough {
		buf.WriteString(strikethroughOn)
	}
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
