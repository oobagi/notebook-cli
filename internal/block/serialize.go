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

	for i, b := range blocks {
		pad := strings.Repeat("    ", b.Indent)

		switch b.Type {
		case Paragraph:
			if b.Content == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, strings.Split(b.Content, "\n")...)
			}

		case Heading1:
			lines = append(lines, "# "+b.Content)

		case Heading2:
			lines = append(lines, "## "+b.Content)

		case Heading3:
			lines = append(lines, "### "+b.Content)

		case BulletList:
			lines = append(lines, pad+"- "+b.Content)

		case NumberedList:
			seq := CountNumberedPosition(blocks, i)
			lines = append(lines, pad+fmt.Sprintf("%d. %s", seq, b.Content))

		case Checklist:
			if b.Checked {
				lines = append(lines, pad+"- [x] "+b.Content)
			} else {
				lines = append(lines, pad+"- [ ] "+b.Content)
			}

		case CodeBlock:
			lang, body := ExtractCodeLanguage(b.Content)
			lines = append(lines, "```"+lang)
			if body != "" {
				lines = append(lines, strings.Split(body, "\n")...)
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
