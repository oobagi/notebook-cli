package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/block"
	"github.com/oobagi/notebook/internal/theme"
)

// renderBlock renders a single block. The active block shows its textarea;
// inactive blocks show styled static text.
func (m Model) renderBlock(idx int) string {
	if idx < 0 || idx >= len(m.blocks) || idx >= len(m.textareas) {
		return ""
	}
	b := m.blocks[idx]
	isActive := idx == m.active

	// For active blocks, sync content from textarea.
	content := b.Content
	if isActive && idx < len(m.textareas) {
		content = m.textareas[idx].Value()
	}

	if isActive {
		return m.renderActiveBlock(idx, b, content)
	}
	return renderInactiveBlock(b, content, m.width)
}

// renderActiveBlock renders the block that currently has focus, showing
// the textarea for editing with a left-margin accent indicator.
func (m Model) renderActiveBlock(idx int, b block.Block, _ string) string {
	if idx < 0 || idx >= len(m.textareas) {
		return ""
	}

	ta := m.textareas[idx]
	taView := ta.View()

	th := theme.Current()
	indicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Accent)).
		Render("\u258e")

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(th.Accent))
		rendered = style.Render(taView)
		// Add blank line above H1 unless it's the first block.
		if idx > 0 {
			rendered = "\n" + rendered
		}

	case block.Heading2:
		style := lipgloss.NewStyle().Bold(true)
		rendered = style.Render(taView)

	case block.Heading3:
		style := lipgloss.NewStyle().Bold(true).Faint(true)
		rendered = style.Render(taView)

	case block.BulletList:
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render("  \u2022  ")
		rendered = prefix + taView

	case block.NumberedList:
		num := 1
		for i := idx - 1; i >= 0; i-- {
			if m.blocks[i].Type == block.NumberedList {
				num++
			} else {
				break
			}
		}
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render(fmt.Sprintf("  %d. ", num))
		rendered = prefix + taView

	case block.Checklist:
		var marker string
		if b.Checked {
			marker = "  \u2611 "
		} else {
			marker = "  \u2610 "
		}
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render(marker)
		rendered = prefix + taView

	case block.CodeBlock:
		label := ""
		if b.Language != "" {
			label = lipgloss.NewStyle().
				Faint(true).
				Render(" " + b.Language)
		}
		border := lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(th.Border)).
			PaddingLeft(1)
		rendered = label + "\n" + border.Render(taView)

	case block.Quote:
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render("\u2502 ")
		// Prepend the bar to each line of the textarea view.
		lines := strings.Split(taView, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		w := m.width
		if w <= 0 {
			w = 40
		}
		if w > 40 {
			w = 40
		}
		rendered = lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render(strings.Repeat("\u2500", w))

	default:
		rendered = taView
	}

	// Prepend the accent-colored active block indicator to each line.
	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		lines[i] = indicator + " " + l
	}
	return strings.Join(lines, "\n")
}

// renderInactiveBlock renders a block as styled static text (no cursor).
func renderInactiveBlock(b block.Block, content string, width int) string {
	switch b.Type {
	case block.Heading1:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Current().Accent))
		return "\n" + style.Render(content) + "\n"

	case block.Heading2:
		style := lipgloss.NewStyle().Bold(true)
		return style.Render(content)

	case block.Heading3:
		style := lipgloss.NewStyle().Bold(true).Faint(true)
		return style.Render(content)

	case block.BulletList:
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("  \u2022  ")
		return prefix + content

	case block.NumberedList:
		// Inactive numbered list items don't know their sequence position
		// from this function alone; the caller should handle sequencing.
		// We use a placeholder that will be replaced at render time.
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("  -  ")
		return prefix + content

	case block.Checklist:
		var marker string
		if b.Checked {
			marker = "  \u2611 "
		} else {
			marker = "  \u2610 "
		}
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render(marker)
		return prefix + content

	case block.CodeBlock:
		label := ""
		if b.Language != "" {
			label = lipgloss.NewStyle().
				Faint(true).
				Render(" " + b.Language)
		}
		border := lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Current().Border)).
			PaddingLeft(1)
		return label + "\n" + border.Render(content)

	case block.Quote:
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("\u2502 ")
		lines := strings.Split(content, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		return strings.Join(lines, "\n")

	case block.Divider:
		w := width
		if w <= 0 {
			w = 40
		}
		if w > 40 {
			w = 40
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render(strings.Repeat("\u2500", w))

	case block.Paragraph:
		if content == "" {
			return ""
		}
		return content

	default:
		return content
	}
}
