package editor

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/oobagi/notebook/internal/block"
	"github.com/oobagi/notebook/internal/theme"
)

// renderHeader builds the header bar displayed at the top of the editor.
// It shows the title on the left and file path + size on the right.
func (m Model) renderHeader() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	titleStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Faint(true)

	left := titleStyle.Render(m.config.Title)

	var rightParts []string
	if m.config.FilePath != "" {
		displayPath := m.config.FilePath
		if home, err := os.UserHomeDir(); err == nil {
			displayPath = strings.Replace(displayPath, home, "~", 1)
		}
		rightParts = append(rightParts, displayPath)
	}
	if m.config.FileSize > 0 {
		rightParts = append(rightParts, humanSize(m.config.FileSize))
	}
	right := metaStyle.Render(strings.Join(rightParts, " \u00B7 "))

	// If everything doesn't fit, truncate the right side.
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		// Drop the file path, keep just the size.
		if m.config.FileSize > 0 {
			right = metaStyle.Render(humanSize(m.config.FileSize))
		} else {
			right = ""
		}
		gap = width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
	}

	bar := left + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().Width(width).Render(bar)
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

		// Measure actual view width for diagnostics.
		viewW := 0
		if i < len(m.textareas) {
			for _, vl := range strings.Split(m.textareas[i].View(), "\n") {
				if lw := lipgloss.Width(vl); lw > viewW {
					viewW = lw
				}
			}
		}

		taW := 0
		if i < len(m.textareas) {
			taW = m.textareas[i].Width()
		}

		line := muted.Render(fmt.Sprintf(
			"[%d] %-12s \"%s\"  h:%d  taW:%d  viewW:%d  pfx:%d",
			i, b.Type.String(), preview, h, taW, viewW, blockPrefixWidth(b.Type),
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
	return renderInactiveBlock(b, content, m.width, m.wordWrap, m.blocks, idx)
}

// renderActiveBlock renders the block that currently has focus.
// We render the content ourselves (same wrapping as inactive blocks)
// and draw the cursor at the correct position using the textarea's
// cursor model. This gives us full control over line width.
func (m Model) renderActiveBlock(idx int, b block.Block, _ string) string {
	if idx < 0 || idx >= len(m.textareas) {
		return ""
	}

	ta := m.textareas[idx]
	content := ta.Value()

	// Content width for this block type.
	contentWidth := m.width - gutterWidth - blockPrefixWidth(b.Type)
	if contentWidth < 1 {
		contentWidth = 1
	}
	if !m.wordWrap {
		contentWidth = 1000
	}

	// Divider: selected as a unit — render highlighted hr, no cursor.
	if b.Type == block.Divider {
		w := m.width - gutterWidth
		if w <= 0 {
			w = 40
		}
		if w > 40 {
			w = 40
		}
		accent := theme.Current().Accent
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(accent)).
			Render(strings.Repeat("\u2500", w))
		label := fmt.Sprintf("%2s", b.Type.Short())
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(accent))
		return accentStyle.Render(label) + " " + accentStyle.Render("│") + " " + rendered
	}

	// Get cursor position from the textarea.
	li := ta.LineInfo()
	cursorRawRow := ta.Line()
	cursorColInWrap := li.ColumnOffset
	cursorWrapRow := li.RowOffset

	// Map (rawRow, wrapRow) to a visual line index and insert cursor.
	rawLines := strings.Split(content, "\n")
	var visualLines []string
	cursorVisIdx := 0

	for rawIdx, raw := range rawLines {
		segs := textarea.Wrap([]rune(raw), contentWidth)
		if len(segs) == 0 {
			segs = [][]rune{{}}
		}
		for wIdx, seg := range segs {
			line := string(seg)
			visIdx := len(visualLines)

			if rawIdx == cursorRawRow && wIdx == cursorWrapRow {
				cursorVisIdx = visIdx
				// Insert cursor character.
				runes := []rune(line)
				col := cursorColInWrap
				if col > len(runes) {
					col = len(runes)
				}
				before := string(runes[:col])
				curChar := " "
				after := ""
				if col < len(runes) {
					curChar = string(runes[col : col+1])
					after = string(runes[col+1:])
				}
				ta.Cursor.SetChar(curChar)
				line = before + ta.Cursor.View() + after
			}

			// Pad to contentWidth.
			if pad := contentWidth - lipgloss.Width(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			visualLines = append(visualLines, line)
		}
	}

	_ = cursorVisIdx
	taView := strings.Join(visualLines, "\n")

	th := theme.Current()

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(th.Accent))
		rendered = style.Render(taView)

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
		rendered = prefixFirstLine(prefix, taView)

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
		rendered = prefixFirstLine(prefix, taView)

	case block.Checklist:
		if b.Checked {
			prefix := lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Accent)).
				Bold(true).
				Render(" [x] ")
			faintText := lipgloss.NewStyle().Faint(true).Render(taView)
			rendered = prefixFirstLine(prefix, faintText)
		} else {
			prefix := lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Muted)).
				Render(" [ ] ")
			rendered = prefixFirstLine(prefix, taView)
		}

	case block.CodeBlock:
		codeContent := taView
		if b.Language != "" {
			langLabel := lipgloss.NewStyle().
				Faint(true).
				Render(b.Language)
			codeContent = langLabel + "\n" + taView
		}
		codeStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(th.Border)).
			PaddingLeft(1).
			PaddingRight(1)
		rendered = codeStyle.Render(codeContent)

	case block.Quote:
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render("\u2502 ")
		lines := strings.Split(taView, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	default:
		rendered = taView
	}

	// Build gutter with block type label.
	label := fmt.Sprintf("%2s", b.Type.Short())
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))
	sepStr := accentStyle.Render("│")
	labelStr := accentStyle.Render(label)
	blankLabel := accentStyle.Render("  ")

	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		if i == 0 {
			lines[i] = labelStr + " " + sepStr + " " + l
		} else {
			lines[i] = blankLabel + " " + sepStr + " " + l
		}
	}

	// Truncate any line that exceeds terminal width.
	for i, l := range lines {
		if lipgloss.Width(l) > m.width {
			if m.wordWrap {
				lines[i] = ansi.Truncate(l, m.width, "")
			} else {
				lines[i] = ansi.Truncate(l, m.width, "\u2192")
			}
		}
	}

	return strings.Join(lines, "\n")
}

// prefixFirstLine prepends a prefix to the first line of a multiline string,
// indenting continuation lines with spaces of matching width so wrapped
// content aligns with the first line's text.
func prefixFirstLine(prefix, content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return prefix
	}
	indent := strings.Repeat(" ", lipgloss.Width(prefix))
	lines[0] = prefix + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

// wrapText wraps content using the same word-wrap + hard-break algorithm
// as the forked textarea, so active and inactive blocks render identically.
func wrapText(content string, width int) string {
	if width <= 0 {
		width = 1
	}
	rawLines := strings.Split(content, "\n")
	var out []string
	for _, raw := range rawLines {
		wrapped := textarea.Wrap([]rune(raw), width)
		if len(wrapped) == 0 {
			out = append(out, "")
			continue
		}
		for _, seg := range wrapped {
			out = append(out, string(seg))
		}
	}
	return strings.Join(out, "\n")
}

// highlightCode applies syntax highlighting to code using chroma.
// Returns the original content unchanged if the language is unknown or highlighting fails.
func highlightCode(code, language string) string {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		return code
	}
	lexer = chroma.Coalesce(lexer)

	styleName := "monokai"
	if theme.Current().Background == "light" {
		styleName = "github"
	}
	style := styles.Get(styleName)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf bytes.Buffer
	err = formatters.TTY16m.Format(&buf, style, iterator)
	if err != nil {
		return code
	}

	// Trim trailing newline that chroma may add.
	result := buf.String()
	result = strings.TrimRight(result, "\n")
	return result
}

// renderInactiveBlock renders a block as styled static text (no cursor).
func renderInactiveBlock(b block.Block, content string, width int, wordWrap bool, blocks []block.Block, idx int) string {
	// Compute the available content width, matching the active block's calculation.
	contentWidth := width - gutterWidth - blockPrefixWidth(b.Type)
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Wrap content to fit within the available width.
	wrapped := content
	if wordWrap {
		wrapped = wrapText(content, contentWidth)
	}

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Current().Accent))
		rendered = style.Render(wrapped)

	case block.Heading2:
		style := lipgloss.NewStyle().Bold(true)
		rendered = style.Render(wrapped)

	case block.Heading3:
		style := lipgloss.NewStyle().Bold(true).Faint(true)
		rendered = style.Render(wrapped)

	case block.BulletList:
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("  \u2022  ")
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		num := 1
		for i := idx - 1; i >= 0; i-- {
			if blocks[i].Type == block.NumberedList {
				num++
			} else {
				break
			}
		}
		prefix := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render(fmt.Sprintf("  %d. ", num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		if b.Checked {
			prefix := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Current().Accent)).
				Bold(true).
				Render(" [x] ")
			faintText := lipgloss.NewStyle().Faint(true).Render(wrapped)
			rendered = prefixFirstLine(prefix, faintText)
		} else {
			prefix := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Current().Muted)).
				Render(" [ ] ")
			rendered = prefixFirstLine(prefix, wrapped)
		}

	case block.CodeBlock:
		highlighted := wrapped
		if b.Language != "" {
			highlighted = highlightCode(wrapped, b.Language)
		}
		codeContent := highlighted
		if b.Language != "" {
			langLabel := lipgloss.NewStyle().
				Faint(true).
				Render(b.Language)
			codeContent = langLabel + "\n" + highlighted
		}
		codeStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Current().Border)).
			PaddingLeft(1).
			PaddingRight(1)
		rendered = codeStyle.Render(codeContent)

	case block.Quote:
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Current().Muted)).
			Render("\u2502 ")
		lines := strings.Split(wrapped, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		w := width - gutterWidth
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
		if wrapped == "" {
			rendered = ""
		} else {
			rendered = wrapped
		}

	default:
		rendered = wrapped
	}

	// Build gutter with block type label (muted for inactive blocks).
	label := fmt.Sprintf("%2s", b.Type.Short())
	mutedStyle := lipgloss.NewStyle().Faint(true)
	sepStr := mutedStyle.Render("│")
	labelStr := mutedStyle.Render(label)
	blankLabel := mutedStyle.Render("  ")

	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		if i == 0 {
			lines[i] = labelStr + " " + sepStr + " " + l
		} else {
			lines[i] = blankLabel + " " + sepStr + " " + l
		}
	}

	// In no-wrap mode, truncate lines to terminal width.
	if !wordWrap {
		for i, l := range lines {
			if lipgloss.Width(l) > width {
				lines[i] = ansi.Truncate(l, width, "\u2192")
			}
		}
	}

	return strings.Join(lines, "\n")
}
