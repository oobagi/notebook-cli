package block

import "strings"

// ParseTableCells parses pipe-delimited content into a 2D cell grid and
// alignment info. Separator rows are consumed for alignment and excluded
// from the returned cells.
func ParseTableCells(content string) (cells [][]string, aligns []string) {
	if content == "" {
		return nil, nil
	}

	rows := strings.Split(content, "\n")
	maxCols := 0

	for _, row := range rows {
		if IsSeparatorRow(row) {
			// Extract alignment from separator.
			sepCells := ParsePipeRow(row)
			aligns = make([]string, len(sepCells))
			for i, c := range sepCells {
				c = strings.TrimSpace(c)
				left := strings.HasPrefix(c, ":")
				right := strings.HasSuffix(c, ":")
				switch {
				case left && right:
					aligns[i] = "center"
				case right:
					aligns[i] = "right"
				default:
					aligns[i] = "left"
				}
			}
			continue
		}
		parsed := ParsePipeRow(row)
		if len(parsed) > maxCols {
			maxCols = len(parsed)
		}
		cells = append(cells, parsed)
	}

	// Normalize all rows to the same column count.
	for i := range cells {
		for len(cells[i]) < maxCols {
			cells[i] = append(cells[i], "")
		}
	}

	// Default alignment if no separator was found.
	if aligns == nil && maxCols > 0 {
		aligns = make([]string, maxCols)
		for i := range aligns {
			aligns[i] = "left"
		}
	}

	// Normalize aligns to match column count.
	for len(aligns) < maxCols {
		aligns = append(aligns, "left")
	}
	if len(aligns) > maxCols && maxCols > 0 {
		aligns = aligns[:maxCols]
	}

	return cells, aligns
}

// SerializeTableCells converts a 2D cell grid back to pipe-delimited content
// with a separator row after the header. Alignment markers are preserved.
func SerializeTableCells(cells [][]string, aligns []string) string {
	if len(cells) == 0 {
		return ""
	}

	maxCols := 0
	for _, row := range cells {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Normalize column counts.
	for i := range cells {
		for len(cells[i]) < maxCols {
			cells[i] = append(cells[i], "")
		}
	}
	for len(aligns) < maxCols {
		aligns = append(aligns, "left")
	}

	// Build pipe rows with the separator inlined as content.
	var contentRows []string
	for i, row := range cells {
		var parts []string
		for _, cell := range row {
			parts = append(parts, cell)
		}
		contentRows = append(contentRows, strings.Join(parts, "|"))
		// Insert separator after the header row.
		if i == 0 {
			var sepParts []string
			for j := 0; j < maxCols; j++ {
				a := "left"
				if j < len(aligns) {
					a = aligns[j]
				}
				switch a {
				case "center":
					sepParts = append(sepParts, ":---:")
				case "right":
					sepParts = append(sepParts, "---:")
				default:
					sepParts = append(sepParts, "---")
				}
			}
			contentRows = append(contentRows, strings.Join(sepParts, "|"))
		}
	}

	// Re-join as pipe content and let serializeTable handle padding.
	var pipeLines []string
	for _, row := range contentRows {
		pipeLines = append(pipeLines, "| "+strings.ReplaceAll(row, "|", " | ")+" |")
	}
	content := strings.Join(pipeLines, "\n")
	return content
}
