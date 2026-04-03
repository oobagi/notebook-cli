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
		dth := theme.Current()
		bs := dth.Blocks.Divider
		divColor := bs.Color
		if divColor == "" {
			divColor = dth.Accent
		}
		maxW := bs.MaxWidth
		if maxW <= 0 {
			maxW = 40
		}
		w := m.width - gutterWidth
		if w <= 0 {
			w = maxW
		}
		if w > maxW {
			w = maxW
		}
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(divColor)).
			Render(strings.Repeat(bs.Char, w))
		label := fmt.Sprintf("%2s", b.Type.Short())
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(dth.Accent))
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

			// Pad to contentWidth (only in wrap mode — in no-wrap mode,
			// padding to 1000 causes false truncation indicators).
			if m.wordWrap {
				if pad := contentWidth - lipgloss.Width(line); pad > 0 {
					line += strings.Repeat(" ", pad)
				}
			}
			visualLines = append(visualLines, line)
		}
	}

	taView := strings.Join(visualLines, "\n")
	cursorANSI := ta.Cursor.View()

	th := theme.Current()

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := th.Blocks.Heading1.Text.ToLipgloss(th.Accent)
		rendered = styleAroundCursor(taView, style, cursorANSI)

	case block.Heading2:
		style := th.Blocks.Heading2.Text.ToLipgloss("")
		rendered = styleAroundCursor(taView, style, cursorANSI)

	case block.Heading3:
		style := th.Blocks.Heading3.Text.ToLipgloss("")
		rendered = styleAroundCursor(taView, style, cursorANSI)

	case block.BulletList:
		bs := th.Blocks.Bullet
		markerColor := bs.MarkerColor
		if markerColor == "" {
			markerColor = th.Muted
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, taView)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := 1
		for i := idx - 1; i >= 0; i-- {
			if m.blocks[i].Type == block.NumberedList {
				num++
			} else {
				break
			}
		}
		markerColor := bs.MarkerColor
		if markerColor == "" {
			markerColor = th.Muted
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, taView)

	case block.Checklist:
		bs := th.Blocks.Checklist
		if b.Checked {
			checkedColor := bs.CheckedColor
			if checkedColor == "" {
				checkedColor = th.Accent
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
			if bs.CheckedBold {
				style = style.Bold(true)
			}
			prefix := style.Render(bs.Checked)
			text := taView
			if bs.CheckedTextFaint {
				text = styleAroundCursor(taView, lipgloss.NewStyle().Faint(true), cursorANSI)
			}
			rendered = prefixFirstLine(prefix, text)
		} else {
			uncheckedColor := bs.UncheckedColor
			if uncheckedColor == "" {
				uncheckedColor = th.Muted
			}
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked)
			rendered = prefixFirstLine(prefix, taView)
		}

	case block.CodeBlock:
		bs := th.Blocks.Code
		rendered = renderCodeBox(taView, b.Language, th.Border, bs.LabelPosition, contentWidth)

	case block.Quote:
		bs := th.Blocks.Quote
		barColor := bs.BarColor
		if barColor == "" {
			barColor = th.Muted
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render(bs.Bar)
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

	// Truncate or horizontally scroll lines that exceed terminal width.
	if m.wordWrap {
		for i, l := range lines {
			if lipgloss.Width(l) > m.width {
				lines[i] = ansi.Truncate(l, m.width, "")
			}
		}
	} else {
		// Determine which output line has the cursor so we can scroll to it.
		cursorLine := cursorVisIdx
		if b.Type == block.CodeBlock {
			cursorLine++ // top border line
			if b.Language != "" && th.Blocks.Code.LabelPosition == "inside" {
				cursorLine++ // label line inside box
			}
		}

		for i, l := range lines {
			lineW := lipgloss.Width(l)
			if lineW <= m.width {
				continue
			}
			if i == cursorLine {
				// Scroll to keep cursor visible.
				cursorCol := gutterWidth + blockPrefixWidth(b.Type) + cursorColInWrap
				if cursorCol < m.width {
					// Cursor on screen — just truncate right.
					lines[i] = ansi.Truncate(l, m.width-1, "\u2192")
				} else {
					// Cursor off screen — shift window to show it.
					margin := m.width / 4
					if margin < 5 {
						margin = 5
					}
					scrollLeft := cursorCol - m.width + margin + 1
					scrollRight := scrollLeft + m.width
					lines[i] = "\u2190" + ansi.Cut(l, scrollLeft+1, scrollRight)
					if lineW > scrollRight {
						lines[i] = ansi.Truncate(lines[i], m.width-1, "\u2192")
					}
				}
			} else {
				lines[i] = ansi.Truncate(l, m.width, "\u2192")
			}
		}
	}

	return strings.Join(lines, "\n")
}

// renderCodeBox renders code in a bordered box with configurable label placement.
func renderCodeBox(code, language, borderColor, labelPos string, padWidth int) string {
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))
	faint := lipgloss.NewStyle().Faint(true)

	lines := strings.Split(code, "\n")

	// Determine inner width from the widest line.
	innerW := padWidth
	if innerW < 10 {
		innerW = 10
	}

	// Pad each line to innerW.
	for i, l := range lines {
		if pad := innerW - lipgloss.Width(l); pad > 0 {
			lines[i] = l + strings.Repeat(" ", pad)
		}
	}

	// Default top/bottom borders.
	topBorder := bc.Render("\u256D" + strings.Repeat("\u2500", innerW+2) + "\u256E")
	bottomBorder := bc.Render("\u2570" + strings.Repeat("\u2500", innerW+2) + "\u256F")

	if language != "" {
		label := faint.Render(language)
		labelW := lipgloss.Width(label)
		switch labelPos {
		case "top":
			dashes := innerW + 2 - labelW - 3
			if dashes < 1 {
				dashes = 1
			}
			topBorder = bc.Render("\u256D\u2500 ") + label + bc.Render(" "+strings.Repeat("\u2500", dashes)+"\u256E")
		case "bottom":
			dashes := innerW + 2 - labelW - 3
			if dashes < 1 {
				dashes = 1
			}
			bottomBorder = bc.Render("\u2570\u2500 ") + label + bc.Render(" "+strings.Repeat("\u2500", dashes)+"\u256F")
		default: // "inside"
			lines = append([]string{faint.Render(language) + strings.Repeat(" ", innerW-labelW)}, lines...)
		}
	}

	var out []string
	out = append(out, topBorder)
	for _, l := range lines {
		out = append(out, bc.Render("\u2502")+" "+l+" "+bc.Render("\u2502"))
	}
	out = append(out, bottomBorder)
	return strings.Join(out, "\n")
}

// styleAroundCursor applies a lipgloss style to text while preserving cursor
// ANSI codes. The cursor's reset escape would otherwise leak and break the
// surrounding style on every blink cycle. This splits each line around the
// cursor view and styles the segments independently.
func styleAroundCursor(text string, style lipgloss.Style, cursorView string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if idx := strings.Index(l, cursorView); idx >= 0 {
			before := l[:idx]
			after := l[idx+len(cursorView):]
			lines[i] = style.Render(before) + style.Render(cursorView) + style.Render(after)
		} else {
			lines[i] = style.Render(l)
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
		style := theme.Current().Blocks.Heading1.Text.ToLipgloss(theme.Current().Accent)
		rendered = style.Render(wrapped)

	case block.Heading2:
		style := theme.Current().Blocks.Heading2.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.Heading3:
		style := theme.Current().Blocks.Heading3.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.BulletList:
		bs := theme.Current().Blocks.Bullet
		markerColor := bs.MarkerColor
		if markerColor == "" {
			markerColor = theme.Current().Muted
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		bs := theme.Current().Blocks.Numbered
		num := 1
		for i := idx - 1; i >= 0; i-- {
			if blocks[i].Type == block.NumberedList {
				num++
			} else {
				break
			}
		}
		markerColor := bs.MarkerColor
		if markerColor == "" {
			markerColor = theme.Current().Muted
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		bs := theme.Current().Blocks.Checklist
		if b.Checked {
			checkedColor := bs.CheckedColor
			if checkedColor == "" {
				checkedColor = theme.Current().Accent
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
			if bs.CheckedBold {
				style = style.Bold(true)
			}
			prefix := style.Render(bs.Checked)
			text := wrapped
			if bs.CheckedTextFaint {
				text = lipgloss.NewStyle().Faint(true).Render(wrapped)
			}
			rendered = prefixFirstLine(prefix, text)
		} else {
			uncheckedColor := bs.UncheckedColor
			if uncheckedColor == "" {
				uncheckedColor = theme.Current().Muted
			}
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked)
			rendered = prefixFirstLine(prefix, wrapped)
		}

	case block.CodeBlock:
		bs := theme.Current().Blocks.Code
		highlighted := wrapped
		if b.Language != "" {
			highlighted = highlightCode(wrapped, b.Language)
		}
		rendered = renderCodeBox(highlighted, b.Language, theme.Current().Border, bs.LabelPosition, contentWidth)

	case block.Quote:
		bs := theme.Current().Blocks.Quote
		barColor := bs.BarColor
		if barColor == "" {
			barColor = theme.Current().Muted
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render(bs.Bar)
		lines := strings.Split(wrapped, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		bs := theme.Current().Blocks.Divider
		divColor := bs.Color
		if divColor == "" {
			divColor = theme.Current().Muted
		}
		maxW := bs.MaxWidth
		if maxW <= 0 {
			maxW = 40
		}
		w := width - gutterWidth
		if w <= 0 {
			w = maxW
		}
		if w > maxW {
			w = maxW
		}
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color(divColor)).Render(strings.Repeat(bs.Char, w))

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
