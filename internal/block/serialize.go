package block

import (
	"fmt"
	"strings"
)

// Serialize converts a slice of Blocks back into a markdown string.
// It is the inverse of Parse: for canonical markdown input,
// Serialize(Parse(md)) == md.
//
// Numbered lists are auto-sequenced: consecutive NumberedList blocks
// are numbered 1, 2, 3, etc. A non-NumberedList block resets the counter.
func Serialize(blocks []Block) string {
	// Special case: single empty paragraph is the empty document.
	if len(blocks) == 1 && blocks[0].Type == Paragraph && blocks[0].Content == "" {
		return ""
	}

	var lines []string
	numberedSeq := 0

	for _, b := range blocks {
		// Track numbered list sequencing.
		if b.Type == NumberedList {
			numberedSeq++
		} else {
			numberedSeq = 0
		}

		switch b.Type {
		case Paragraph:
			if b.Content == "" {
				// Empty paragraph = blank line.
				lines = append(lines, "")
			} else {
				// Paragraph content may contain newlines (merged lines).
				lines = append(lines, strings.Split(b.Content, "\n")...)
			}

		case Heading1:
			lines = append(lines, "# "+b.Content)

		case Heading2:
			lines = append(lines, "## "+b.Content)

		case Heading3:
			lines = append(lines, "### "+b.Content)

		case BulletList:
			lines = append(lines, "- "+b.Content)

		case NumberedList:
			lines = append(lines, fmt.Sprintf("%d. %s", numberedSeq, b.Content))

		case Checklist:
			if b.Checked {
				lines = append(lines, "- [x] "+b.Content)
			} else {
				lines = append(lines, "- [ ] "+b.Content)
			}

		case CodeBlock:
			lines = append(lines, "```"+b.Language)
			if b.Content != "" {
				lines = append(lines, strings.Split(b.Content, "\n")...)
			}
			lines = append(lines, "```")

		case Quote:
			if b.Content == "" {
				lines = append(lines, ">")
			} else {
				for _, ql := range strings.Split(b.Content, "\n") {
					if ql == "" {
						lines = append(lines, ">")
					} else {
						lines = append(lines, "> "+ql)
					}
				}
			}

		case Divider:
			lines = append(lines, "---")

		default:
			if b.Content != "" {
				lines = append(lines, strings.Split(b.Content, "\n")...)
			}
		}
	}

	return strings.Join(lines, "\n")
}
