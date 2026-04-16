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

		case Callout:
			lines = append(lines, "> [!"+b.Variant.String()+"]")
			if b.Content != "" {
				for _, cl := range strings.Split(b.Content, "\n") {
					if cl == "" {
						lines = append(lines, ">")
					} else {
						lines = append(lines, "> "+cl)
					}
				}
			}

		case Divider:
			lines = append(lines, "---")

		case DefinitionList:
			term, def := ExtractDefinition(b.Content)
			lines = append(lines, term)
			for _, d := range strings.Split(def, "\n") {
				lines = append(lines, ": "+d)
			}

		case Embed:
			lines = append(lines, "![["+b.Content+"]]")

		case Table:
			lines = append(lines, serializeTable(b.Content)...)

		default:
			if b.Content != "" {
				lines = append(lines, strings.Split(b.Content, "\n")...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// IsEmptyParagraph reports whether a block is a blank paragraph — the
// shape that Parse produces for a bare newline in the source file.
func IsEmptyParagraph(b Block) bool {
	return b.Type == Paragraph && b.Content == ""
}

// TrimEmptyEdges drops leading and trailing empty-paragraph blocks.
// Blank lines in the middle of the document are preserved. View-mode
// rendering uses this to hide accidental top/bottom whitespace without
// mutating the file on disk.
func TrimEmptyEdges(blocks []Block) []Block {
	start := 0
	for start < len(blocks) && IsEmptyParagraph(blocks[start]) {
		start++
	}
	end := len(blocks)
	for end > start && IsEmptyParagraph(blocks[end-1]) {
		end--
	}
	return blocks[start:end]
}

// isSeparatorCell reports whether a trimmed cell is a GFM separator
// (e.g. "---", ":---", "---:", ":---:").
func isSeparatorCell(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Strip optional leading/trailing colons.
	t := strings.TrimPrefix(s, ":")
	t = strings.TrimSuffix(t, ":")
	if len(t) == 0 {
		return false
	}
	for _, c := range t {
		if c != '-' {
			return false
		}
	}
	return true
}

// ParsePipeRow splits a pipe-delimited row into trimmed cells.
// Leading and trailing pipes are removed: "| a | b |" → ["a", "b"].
func ParsePipeRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

// IsSeparatorRow reports whether a full row is a GFM separator row.
func IsSeparatorRow(line string) bool {
	cells := ParsePipeRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !isSeparatorCell(c) {
			return false
		}
	}
	return true
}

// serializeTable formats pipe table content with auto-padded columns.
// If the content includes a separator row, its alignment markers are
// preserved. If no separator is found, one is auto-generated after the
// first (header) row.
func serializeTable(content string) []string {
	rows := strings.Split(content, "\n")
	if len(rows) == 0 {
		return nil
	}

	// Parse all rows into cells and detect separator row.
	type parsedRow struct {
		cells []string
		isSep bool
		raw   string // original line for separator alignment extraction
	}
	var parsed []parsedRow
	sepIdx := -1
	maxCols := 0
	for _, row := range rows {
		isSep := IsSeparatorRow(row)
		cells := ParsePipeRow(row)
		if len(cells) > maxCols {
			maxCols = len(cells)
		}
		if isSep && sepIdx == -1 {
			sepIdx = len(parsed)
		}
		parsed = append(parsed, parsedRow{cells: cells, isSep: isSep, raw: row})
	}

	// Normalize all rows to the same column count.
	for i := range parsed {
		for len(parsed[i].cells) < maxCols {
			parsed[i].cells = append(parsed[i].cells, "")
		}
	}

	// Find max width per column (excluding separator rows).
	colWidths := make([]int, maxCols)
	for _, pr := range parsed {
		if pr.isSep {
			continue
		}
		for j, cell := range pr.cells {
			if len(cell) > colWidths[j] {
				colWidths[j] = len(cell)
			}
		}
	}
	for j := range colWidths {
		if colWidths[j] < 3 {
			colWidths[j] = 3
		}
	}

	// Build output lines.
	var out []string
	hasSep := false
	for _, pr := range parsed {
		if pr.isSep {
			hasSep = true
			// Preserve alignment markers from the original separator.
			var parts []string
			for j, cell := range pr.cells {
				w := colWidths[j]
				cell = strings.TrimSpace(cell)
				leftColon := strings.HasPrefix(cell, ":")
				rightColon := strings.HasSuffix(cell, ":")
				dashes := w
				if leftColon {
					dashes--
				}
				if rightColon {
					dashes--
				}
				if dashes < 1 {
					dashes = 1
				}
				sep := strings.Repeat("-", dashes)
				if leftColon {
					sep = ":" + sep
				}
				if rightColon {
					sep = sep + ":"
				}
				parts = append(parts, " "+sep+" ")
			}
			out = append(out, "|"+strings.Join(parts, "|")+"|")
		} else {
			var parts []string
			for j, cell := range pr.cells {
				w := colWidths[j]
				pad := w - len(cell)
				if pad < 0 {
					pad = 0
				}
				parts = append(parts, " "+cell+strings.Repeat(" ", pad)+" ")
			}
			out = append(out, "|"+strings.Join(parts, "|")+"|")
		}
	}

	// If no separator was found, insert one after the first row.
	if !hasSep && len(out) > 0 {
		var sepParts []string
		for _, w := range colWidths {
			sepParts = append(sepParts, " "+strings.Repeat("-", w)+" ")
		}
		sepLine := "|" + strings.Join(sepParts, "|") + "|"
		// Insert after first line.
		newOut := make([]string, 0, len(out)+1)
		newOut = append(newOut, out[0])
		newOut = append(newOut, sepLine)
		newOut = append(newOut, out[1:]...)
		out = newOut
	}

	return out
}
