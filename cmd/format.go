package cmd

import (
	"fmt"
	"strings"
	"time"
)

// relativeTime returns a human-friendly relative timestamp string.
// It follows the design system rules from docs/design.md.
func relativeTime(t time.Time) string {
	return relativeTimeFrom(t, time.Now())
}

// relativeTimeFrom returns a relative timestamp using the given reference time.
// Exported logic for testing.
func relativeTimeFrom(t time.Time, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	default:
		if t.Year() == now.Year() {
			return t.Format("Jan 2")
		}
		return t.Format("Jan 2, 2006")
	}
}

// humanSize formats a byte count into a human-readable string.
func humanSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		kb := float64(bytes) / 1024
		return formatFloat(kb) + " KB"
	case bytes < 1024*1024*1024:
		mb := float64(bytes) / (1024 * 1024)
		return formatFloat(mb) + " MB"
	default:
		gb := float64(bytes) / (1024 * 1024 * 1024)
		return formatFloat(gb) + " GB"
	}
}

// formatFloat formats a float to one decimal place, dropping the ".0" suffix.
func formatFloat(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	if strings.HasSuffix(s, ".0") {
		return s[:len(s)-2]
	}
	return s
}

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
