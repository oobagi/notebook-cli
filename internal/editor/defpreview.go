package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// defPreviewPadTop is a single blank line above the preview content.
const defPreviewPadTop = 1

// wrappedLines returns the preview's wrapped definition body split by line,
// shared between rendering and height accounting so both agree.
func (d *defPreviewState) wrappedLines(width int) []string {
	innerWidth := width - 4 // 2-col left pad + 2-col right breathing room
	if innerWidth < 20 {
		innerWidth = 20
	}
	// Normalize internal newlines into paragraph breaks.
	clean := strings.ReplaceAll(d.definition, "\r\n", "\n")
	paragraphs := strings.Split(clean, "\n")
	var out []string
	for i, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			if i > 0 && i < len(paragraphs)-1 {
				out = append(out, "")
			}
			continue
		}
		wrapped := ansi.Wordwrap(p, innerWidth, "")
		for _, line := range strings.Split(wrapped, "\n") {
			out = append(out, strings.TrimRight(line, " "))
		}
	}
	return out
}

// Height returns the number of terminal lines the preview occupies. Layout:
//
//	1 separator + padTop + 1 term + 1 blank + N body lines + 1 blank
//
// This stays bounded by the content so short defs stay short.
func (d *defPreviewState) Height(width int) int {
	if !d.visible {
		return 0
	}
	body := d.wrappedLines(width)
	// 1 border + padTop + 1 term + 1 gap + body + 1 trailing gap
	return 1 + defPreviewPadTop + 1 + 1 + len(body) + 1
}

// RenderFooter draws the preview panel as a footer pane above the status
// bar. It mirrors the picker-footer visual contract (top border + padded
// content) but with definition-focused styling.
func (d *defPreviewState) RenderFooter(width int) string {
	if !d.visible {
		return ""
	}
	th := theme.Current()
	border := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).
		Render(strings.Repeat("\u2500", width))
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true).
		Render(d.term)
	body := d.wrappedLines(width)

	var out strings.Builder
	out.WriteString(border)
	out.WriteString("\n")
	for i := 0; i < defPreviewPadTop; i++ {
		out.WriteString("\n")
	}
	out.WriteString("  ")
	out.WriteString(title)
	out.WriteString("\n\n")
	for _, line := range body {
		out.WriteString("  ")
		out.WriteString(line)
		out.WriteString("\n")
	}
	out.WriteString("\n")
	return out.String()
}
