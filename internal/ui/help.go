package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// HelpBinding is a single key → description pair.
type HelpBinding struct {
	Key  string
	Desc string
}

// RenderHelpFooter builds a compact footer panel with bindings in a
// column-major grid, matching the picker footer style.
func RenderHelpFooter(bindings []HelpBinding, width int) string {
	if width <= 0 {
		width = 80
	}
	if len(bindings) == 0 {
		return ""
	}

	th := theme.Current()
	dim := lipgloss.NewStyle().Faint(true)
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))

	// Find max visual width of any single binding entry.
	maxEntry := 0
	for _, b := range bindings {
		w := lipgloss.Width(b.Key) + 1 + lipgloss.Width(b.Desc)
		if w > maxEntry {
			maxEntry = w
		}
	}

	colW := maxEntry + 3
	cols := (width - 2) / colW // -2 for left margin
	if cols < 1 {
		cols = 1
	}
	if cols > len(bindings) {
		cols = len(bindings)
	}

	rows := (len(bindings) + cols - 1) / cols

	border := dim.Render(strings.Repeat("\u2500", width))

	var sb strings.Builder
	sb.WriteString(border + "\n")

	for r := 0; r < rows; r++ {
		sb.WriteString("  ")
		for c := 0; c < cols; c++ {
			idx := r + c*rows // column-major order
			if idx < len(bindings) {
				k := bindings[idx].Key
				desc := bindings[idx].Desc
				sb.WriteString(accent.Render(k) + " " + desc)
				visual := lipgloss.Width(k) + 1 + lipgloss.Width(desc)
				if pad := colW - visual; pad > 0 {
					sb.WriteString(strings.Repeat(" ", pad))
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
