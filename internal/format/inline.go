package format

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
)

// RenderInlineMarkdown applies inline markdown formatting to plain text.
// It supports **bold**, *italic*, ~~strikethrough~~, and __underline__.
// Nesting is supported (e.g. ***bold italic***). Empty delimiters and
// unmatched delimiters are rendered literally. snake_case words are NOT
// treated as underline markup.
//
// The input should be plain text (no ANSI escapes). The output contains
// lipgloss-styled ANSI sequences.
func RenderInlineMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = renderInlineLine(line)
	}
	return strings.Join(lines, "\n")
}

// delimKind represents the type of inline markdown delimiter.
type delimKind int

const (
	delimBold          delimKind = iota // **
	delimItalic                         // *
	delimStrikethrough                  // ~~
	delimUnderline                      // __
)

// stackEntry tracks an opening delimiter's position and kind.
type stackEntry struct {
	kind     delimKind
	pos      int // byte offset in buf where content starts (after delimiter text)
	snapshot int // byte offset in buf just before the delimiter text
}

func renderInlineLine(line string) string {
	runes := []rune(line)
	n := len(runes)

	// Use a byte slice so we can truncate freely.
	buf := make([]byte, 0, n*2)
	var stack []stackEntry

	i := 0
	for i < n {
		// --- ~~ strikethrough ---
		if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
			if idx := findStack(stack, delimStrikethrough); idx >= 0 {
				entry := stack[idx]
				content := string(buf[entry.pos:])
				if content == "" {
					// Empty delimiter — pop stack, leave literal delimiters
					stack = stack[:idx]
					buf = append(buf, '~', '~')
					i += 2
					continue
				}
				rendered := lipgloss.NewStyle().Strikethrough(true).Render(content)
				buf = append(buf[:entry.snapshot], rendered...)
				stack = stack[:idx]
			} else {
				stack = append(stack, stackEntry{
					kind:     delimStrikethrough,
					snapshot: len(buf),
					pos:      len(buf) + 2, // after "~~"
				})
				buf = append(buf, '~', '~')
			}
			i += 2
			continue
		}

		// --- __ underline (word-boundary only) ---
		if i+1 < n && runes[i] == '_' && runes[i+1] == '_' {
			if idx := findStack(stack, delimUnderline); idx >= 0 {
				// Potential closing __
				afterOk := (i+2 >= n) || !isWordChar(runes[i+2])
				if afterOk {
					entry := stack[idx]
					content := string(buf[entry.pos:])
					if content == "" {
						stack = stack[:idx]
						buf = append(buf, '_', '_')
						i += 2
						continue
					}
					rendered := lipgloss.NewStyle().Underline(true).Render(content)
					buf = append(buf[:entry.snapshot], rendered...)
					stack = stack[:idx]
					i += 2
					continue
				}
			}
			// Potential opening __
			beforeOk := (i == 0) || !isWordChar(runes[i-1])
			if beforeOk {
				stack = append(stack, stackEntry{
					kind:     delimUnderline,
					snapshot: len(buf),
					pos:      len(buf) + 2,
				})
				buf = append(buf, '_', '_')
				i += 2
				continue
			}
			// Not at word boundary — literal
			buf = append(buf, string(runes[i:i+2])...)
			i += 2
			continue
		}

		// --- * or ** stars (bold / italic) ---
		if runes[i] == '*' {
			starCount := 0
			j := i
			for j < n && runes[j] == '*' {
				starCount++
				j++
			}
			i += handleStars(&buf, &stack, starCount)
			continue
		}

		// Regular character
		buf = append(buf, string(runes[i:i+1])...)
		i++
	}

	return string(buf)
}

// handleStars processes a run of * characters for bold/italic.
// Returns number of runes consumed.
func handleStars(buf *[]byte, stack *[]stackEntry, starCount int) int {
	remaining := starCount

	for remaining > 0 {
		// Find the topmost (most recently opened) star-based delimiter.
		boldIdx := findStack(*stack, delimBold)
		italicIdx := findStack(*stack, delimItalic)

		// Determine which to close first — always close the topmost.
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
			content := string((*buf)[entry.pos:])
			if content == "" {
				*stack = (*stack)[:boldIdx]
				*buf = append(*buf, '*', '*')
				remaining -= 2
				continue
			}
			rendered := lipgloss.NewStyle().Bold(true).Render(content)
			*buf = append((*buf)[:entry.snapshot], rendered...)
			*stack = (*stack)[:boldIdx]
			remaining -= 2
			continue
		}

		if closeItalic {
			entry := (*stack)[italicIdx]
			content := string((*buf)[entry.pos:])
			if content == "" {
				*stack = (*stack)[:italicIdx]
				*buf = append(*buf, '*')
				remaining--
				continue
			}
			rendered := lipgloss.NewStyle().Italic(true).Render(content)
			*buf = append((*buf)[:entry.snapshot], rendered...)
			*stack = (*stack)[:italicIdx]
			remaining--
			continue
		}

		// No matching closer — open new delimiters.
		// Open bold (**) first if we have 2+, then italic (*) for remainder.
		if remaining >= 2 {
			*stack = append(*stack, stackEntry{
				kind:     delimBold,
				snapshot: len(*buf),
				pos:      len(*buf) + 2,
			})
			*buf = append(*buf, '*', '*')
			remaining -= 2
			continue
		}
		if remaining >= 1 {
			*stack = append(*stack, stackEntry{
				kind:     delimItalic,
				snapshot: len(*buf),
				pos:      len(*buf) + 1,
			})
			*buf = append(*buf, '*')
			remaining--
			continue
		}
	}

	return starCount
}

// findStack finds the last (most recent) stack entry with the given kind.
func findStack(stack []stackEntry, kind delimKind) int {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].kind == kind {
			return i
		}
	}
	return -1
}

// isWordChar returns true if r is a letter, digit, or underscore.
func isWordChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
