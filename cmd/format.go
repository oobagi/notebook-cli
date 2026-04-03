package cmd

import (
	"fmt"
	"strings"
)

// pluralize returns "1 note" or "3 notes" style strings.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// alignColumns takes rows of string columns and pads them so each column
// is left-aligned with a minimum of 4 spaces between columns.
// Each returned string has a two-space left indent.
func alignColumns(rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}

	// Find the maximum number of columns.
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Find the maximum width for each column (except the last).
	widths := make([]int, maxCols)
	for _, row := range rows {
		for c := 0; c < len(row)-1; c++ {
			if len(row[c]) > widths[c] {
				widths[c] = len(row[c])
			}
		}
	}

	// Build each formatted line.
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		var b strings.Builder
		b.WriteString("  ") // two-space left indent
		for c, col := range row {
			b.WriteString(col)
			if c < len(row)-1 {
				// Pad to column width + 4 spaces gap.
				padding := widths[c] - len(col) + 4
				b.WriteString(strings.Repeat(" ", padding))
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}
