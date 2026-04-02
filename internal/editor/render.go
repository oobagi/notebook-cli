package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/block"
	"github.com/oobagi/notebook/internal/theme"
)

// renderDebugInfo builds the debug panel that shows block metadata.
// It is displayed between the viewport and the status bar when debug mode
// is active (Ctrl+D).
func (m Model) renderDebugInfo() string {
	th := theme.Current()
	muted := lipgloss.NewStyle().Faint(true)

	// Summary line.
	summary := muted.Render(fmt.Sprintf(
		"DEBUG  blocks:%d  active:%d  viewport:%dx%d  scroll:%d",
		len(m.blocks), m.active,
		m.viewport.Width, m.viewport.Height,
		m.viewport.YOffset,
	))

	// One line per block.
	var lines []string
	for i, b := range m.blocks {
		content := b.Content
		if i < len(m.textareas) {
			content = m.textareas[i].Value()
		}

		// Content preview: first 30 chars, truncated with "...".
		preview := content
		if preview == "" {
			preview = "(empty)"
		} else {
			// Replace newlines with spaces for the preview.
			preview = strings.ReplaceAll(preview, "\n", " ")
			previewRunes := []rune(preview)
			if len(previewRunes) > 30 {
				preview = string(previewRunes[:30]) + "..."
			}
		}

		// Textarea height.
		h := 0
		if i < len(m.textareas) {
			h = m.textareas[i].Height()
		}

		// Active marker.
		marker := ""
		if i == m.active {
			marker = lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Accent)).
				Render("  <- active")
		}

		line := muted.Render(fmt.Sprintf(
			"[%d] %-12s \"%s\"  h:%d",
			i, b.Type.String(), preview, h,
		)) + marker

		lines = append(lines, line)
	}

	// Top border line.
	w := m.width
	if w <= 0 {
		w = 80
	}
	border := muted.Render(strings.Repeat("\u2500", w))

	return border + "\n" + summary + "\n" + strings.Join(lines, "\n")
}

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
	var rendered string

	switch b.Type {
	case block.Heading1:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Current().Accent))
		rendered = "\n" + style.Render(content) + "\n"

	case block.Heading2:
		style := lipgloss.NewStyle().Bold(true)
		rendered = style.Render(content)

	case block.Heading3:
		style := lipgloss.NewStyle().Bold(true).Faint(true)
		rendered = style.Render(content)

	case block.BulletList:
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("  \u2022  ")
		rendered = prefix + content

	case block.NumberedList:
		// Inactive numbered list items don't know their sequence position
		// from this function alone; the caller should handle sequencing.
		// We use a placeholder that will be replaced at render time.
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("  -  ")
		rendered = prefix + content

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
		rendered = prefix + content

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
		rendered = label + "\n" + border.Render(content)

	case block.Quote:
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("\u2502 ")
		lines := strings.Split(content, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		w := width
		if w <= 0 {
			w = 40
		}
		if w > 40 {
			w = 40
		}
		rendered = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render(strings.Repeat("\u2500", w))

	case block.Paragraph:
		if content == "" {
			rendered = ""
		} else {
			rendered = content
		}

	default:
		rendered = content
	}

	// Prepend 2-space left padding to each line to match the active block's
	// accent indicator width (▎ + space = 2 characters), so text stays at
	// the same column regardless of whether the block is selected.
	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
