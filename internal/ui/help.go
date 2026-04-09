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

// HelpSection is a titled group of keybindings.
type HelpSection struct {
	Title    string
	Bindings []HelpBinding
}

// RenderHelpOverlay builds a centered help panel.
func RenderHelpOverlay(sections []HelpSection, closeHint string, w, h int) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	th := theme.Current()
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	sep := dim.Render("\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")
	s := dim.Render("/") // dimmed slash separator

	var help strings.Builder
	for si, section := range sections {
		if si > 0 {
			help.WriteString("\n")
		}
		help.WriteString("  " + accent.Render(section.Title) + "\n")
		help.WriteString("  " + sep + "\n")

		for _, b := range section.Bindings {
			if b.Key == "" {
				// Blank separator line.
				help.WriteString("\n")
				continue
			}
			// Replace "/" in desc with dimmed slash for visual consistency.
			desc := strings.ReplaceAll(b.Desc, " / ", " "+s+" ")
			key := strings.ReplaceAll(b.Key, "/", s)
			help.WriteString("  " + key + desc + "\n")
		}
	}

	// Trim trailing newline so the box doesn't have extra space.
	content := strings.TrimRight(help.String(), "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.Border)).
		Padding(1, 2).
		Width(36).
		Align(lipgloss.Left)

	rendered := box.Render(content)
	statusHint := dim.Render(closeHint)
	full := rendered + "\n" + statusHint

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, full)
}
