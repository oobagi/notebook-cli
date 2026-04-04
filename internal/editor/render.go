package editor

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// countNumberedPosition returns the 1-based position of a numbered list block
// by counting consecutive NumberedList blocks preceding it.
func countNumberedPosition(blocks []block.Block, idx int) int {
	num := 1
	for i := idx - 1; i >= 0; i-- {
		if blocks[i].Type == block.NumberedList {
			num++
		} else {
			break
		}
	}
	return num
}

// resolveColor returns color if non-empty, otherwise returns fallback.
func resolveColor(color, fallback string) string {
	if color == "" {
		return fallback
	}
	return color
}

// renderHeader builds the header bar displayed at the top of the editor.
// It shows the title on the left and file path + size on the right.
func (m Model) renderHeader() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	titleStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(1).PaddingRight(1)
	metaStyle := lipgloss.NewStyle().Faint(true)

	left := " " + titleStyle.Render(m.config.Title)

	var rightParts []string
	if m.config.FilePath != "" {
		displayPath := format.ShortenHome(m.config.FilePath)
		rightParts = append(rightParts, displayPath)
	}
	if m.config.FileSize > 0 {
		rightParts = append(rightParts, format.HumanSize(m.config.FileSize))
	}
	right := metaStyle.Render(strings.Join(rightParts, " \u00B7 "))

	// If everything doesn't fit, truncate the right side.
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		// Drop the file path, keep just the size.
		if m.config.FileSize > 0 {
			right = metaStyle.Render(format.HumanSize(m.config.FileSize))
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

	th := theme.Current()

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
		bs := th.Blocks.Divider
		divColor := resolveColor(bs.Color, th.Accent)
		w := m.width - gutterWidth
		if w <= 0 {
			w = 40
		}
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(divColor)).
			Render(strings.Repeat(bs.Char, w))
		label := fmt.Sprintf("%2s", b.Type.Short())
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))
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
				ta.CursorSetChar(curChar)
				line = before + ta.CursorView() + after
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
	cursorANSI := ta.CursorView()

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
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, taView)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := countNumberedPosition(m.blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, taView)

	case block.Checklist:
		bs := th.Blocks.Checklist
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
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
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked)
			rendered = prefixFirstLine(prefix, taView)
		}

	case block.CodeBlock:
		bs := th.Blocks.Code
		boxWidth := contentWidth
		if !m.wordWrap {
			// In no-wrap mode contentWidth is 1000; size the box border to
			// the terminal instead so it doesn't overflow every line.
			boxWidth = m.width - gutterWidth - blockPrefixWidth(b.Type)
			if boxWidth < 10 {
				boxWidth = 10
			}
		}
		// First line is always the title/language field, rendered in the
		// top border. Use clean text with manual cursor (not raw textarea
		// ANSI) so the border doesn't break.
		rawLines := strings.Split(ta.Value(), "\n")
		titleText := ""
		if len(rawLines) > 0 {
			titleText = rawLines[0]
		}

		faintStyle := lipgloss.NewStyle().Faint(true)
		label := ""
		if ta.Line() == 0 {
			// Cursor is on the title — render with visible cursor.
			if titleText == "" {
				cursor := lipgloss.NewStyle().Reverse(true)
				label = cursor.Render(" ") + faintStyle.Render("language")
			} else {
				label = renderLabelCursor(titleText, ta.LineInfo().ColumnOffset, faintStyle)
			}
		} else if titleText != "" {
			label = faintStyle.Render(titleText)
		} else {
			label = faintStyle.Render("language")
		}

		// Strip the title line from the textarea view.
		cLines := strings.Split(taView, "\n")
		if len(cLines) > 0 {
			cLines = cLines[1:]
		}
		taView = strings.Join(cLines, "\n")
		if cursorVisIdx > 0 {
			cursorVisIdx--
		}
		rendered = renderCodeBox(taView, label, th.Border, bs.LabelAlign, boxWidth)

	case block.Quote:
		bs := th.Blocks.Quote
		barColor := resolveColor(bs.BarColor, th.Muted)
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
	} else if b.Type == block.CodeBlock {
		// Cursor column accounts for gutter + left box border (│ + space).
		cursorCol := gutterWidth + 2 + cursorColInWrap
		// cursorVisIdx is relative to content lines; +1 for top border.
		adjustedCursor := cursorVisIdx + 1
		for i, l := range lines {
			lines[i] = scrollOrTruncate(l, m.width, cursorCol, i == adjustedCursor)
		}
	} else {
		cursorCol := gutterWidth + blockPrefixWidth(b.Type) + cursorColInWrap
		for i, l := range lines {
			lines[i] = scrollOrTruncate(l, m.width, cursorCol, i == cursorVisIdx)
		}
	}

	return strings.Join(lines, "\n")
}

// renderCodeBox renders code in a bordered box with the label always in the
// top border. labelAlign controls horizontal placement: "left", "center", or
// "right" (default "left"). The label is pre-styled by the caller.
func renderCodeBox(code, label, borderColor, labelAlign string, padWidth int) string {
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

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

	topBorder := bc.Render("\u256D" + strings.Repeat("\u2500", innerW+2) + "\u256E")
	bottomBorder := bc.Render("\u2570" + strings.Repeat("\u2500", innerW+2) + "\u256F")

	if label != "" {
		labelW := lipgloss.Width(label)
		switch labelAlign {
		case "center":
			total := innerW - labelW
			if total < 2 {
				total = 2
			}
			left := total / 2
			right := total - left
			topBorder = bc.Render("\u256D"+strings.Repeat("\u2500", left)+" ") + label + bc.Render(" "+strings.Repeat("\u2500", right)+"\u256E")
		case "right":
			dashes := innerW - 1 - labelW
			if dashes < 1 {
				dashes = 1
			}
			topBorder = bc.Render("\u256D"+strings.Repeat("\u2500", dashes)+" ") + label + bc.Render(" \u2500\u256E")
		default: // "left"
			dashes := innerW - 1 - labelW
			if dashes < 1 {
				dashes = 1
			}
			topBorder = bc.Render("\u256D\u2500 ") + label + bc.Render(" "+strings.Repeat("\u2500", dashes)+"\u256E")
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

// renderLabelCursor renders a label string with a reverse-video cursor at
// the given column position. Text outside the cursor is styled with base.
func renderLabelCursor(text string, col int, base lipgloss.Style) string {
	runes := []rune(text)
	cursor := lipgloss.NewStyle().Reverse(true)
	if col >= len(runes) {
		return base.Render(text) + cursor.Render(" ")
	}
	before := string(runes[:col])
	ch := string(runes[col : col+1])
	after := string(runes[col+1:])
	return base.Render(before) + cursor.Render(ch) + base.Render(after)
}

// scrollOrTruncate fits a line within availWidth, adding ← → scroll
// indicators as needed. For the cursor line, it scrolls to keep cursorCol
// visible. For other lines, it simply truncates with a → indicator.
func scrollOrTruncate(line string, availWidth, cursorCol int, isCursorLine bool) string {
	lineW := lipgloss.Width(line)
	if lineW <= availWidth {
		return line
	}
	if !isCursorLine {
		return ansi.Truncate(line, availWidth, "\u2192")
	}
	if cursorCol < availWidth-2 {
		return ansi.Truncate(line, availWidth, "\u2192")
	}
	// Scroll to keep cursor visible. The cursor line doesn't show a →
	// indicator — the cursor itself signals the position, and ← on the
	// left shows there's hidden content.
	scrollLeft := cursorCol - availWidth + 2
	scrollRight := scrollLeft + availWidth
	result := "\u2190" + ansi.Cut(line, scrollLeft+1, scrollRight)
	if lipgloss.Width(result) > availWidth {
		result = ansi.Truncate(result, availWidth, "")
	}
	return result
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

	th := theme.Current()

	// Wrap content to fit within the available width.
	wrapped := content
	if wordWrap {
		wrapped = wrapText(content, contentWidth)
	}

	// Apply inline markdown formatting (bold, italic, etc.) to all blocks
	// except code blocks, which display raw content.
	if b.Type != block.CodeBlock {
		wrapped = format.RenderInlineMarkdown(wrapped)
	}

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := th.Blocks.Heading1.Text.ToLipgloss(th.Accent)
		rendered = style.Render(wrapped)

	case block.Heading2:
		style := th.Blocks.Heading2.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.Heading3:
		style := th.Blocks.Heading3.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.BulletList:
		bs := th.Blocks.Bullet
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := countNumberedPosition(blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		bs := th.Blocks.Checklist
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
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
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked)
			rendered = prefixFirstLine(prefix, wrapped)
		}

	case block.CodeBlock:
		bs := th.Blocks.Code
		title, body := block.ExtractCodeLanguage(content)
		label := ""
		if title != "" {
			label = lipgloss.NewStyle().Faint(true).Render(title)
		}
		// Display only the body (after the title line).
		displayContent := body
		if wordWrap {
			displayContent = wrapText(body, contentWidth)
		}
		// Syntax highlighting when the title is a recognized language.
		if title != "" && lexers.Get(title) != nil {
			displayContent = highlightCode(displayContent, title)
		}
		rendered = renderCodeBox(displayContent, label, th.Border, bs.LabelAlign, contentWidth)

	case block.Quote:
		bs := th.Blocks.Quote
		barColor := resolveColor(bs.BarColor, th.Muted)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render(bs.Bar)
		lines := strings.Split(wrapped, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		bs := th.Blocks.Divider
		divColor := resolveColor(bs.Color, th.Muted)
		w := width - gutterWidth
		if w <= 0 {
			w = 40
		}
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color(divColor)).Render(strings.Repeat(bs.Char, w))

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
			lines[i] = scrollOrTruncate(l, width, 0, false)
		}
	}

	return strings.Join(lines, "\n")
}

// renderViewBlock renders a block for view mode: styled static text without
// the gutter, centered in a constrained content column for clean reading.
func renderViewBlock(b block.Block, content string, width int, wordWrap bool, blocks []block.Block, idx int, hovered bool) string {
	// Width here is the content column width (already constrained to viewMaxWidth).
	contentWidth := width - blockPrefixWidth(b.Type)
	if contentWidth < 1 {
		contentWidth = 1
	}

	th := theme.Current()

	// Always wrap in view mode regardless of the no-wrap setting.
	wrapped := wrapText(content, contentWidth)

	// Apply inline markdown formatting (bold, italic, etc.) to all blocks
	// except code blocks, which display raw content.
	if b.Type != block.CodeBlock {
		wrapped = format.RenderInlineMarkdown(wrapped)
	}

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := th.Blocks.Heading1.Text.ToLipgloss(th.Accent)
		rendered = style.Render(wrapped)

	case block.Heading2:
		style := th.Blocks.Heading2.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.Heading3:
		style := th.Blocks.Heading3.Text.ToLipgloss("")
		rendered = style.Render(wrapped)

	case block.BulletList:
		bs := th.Blocks.Bullet
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := countNumberedPosition(blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		bs := th.Blocks.Checklist
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
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
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			// On hover, light up the checkbox marker with the accent color.
			if hovered {
				uncheckedColor = th.Accent
			}
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked)
			rendered = prefixFirstLine(prefix, wrapped)
		}

	case block.CodeBlock:
		bs := th.Blocks.Code
		title, body := block.ExtractCodeLanguage(content)
		label := ""
		if title != "" {
			label = lipgloss.NewStyle().Faint(true).Render(title)
		}
		displayContent := body
		if wordWrap {
			displayContent = wrapText(body, contentWidth)
		}
		if title != "" && lexers.Get(title) != nil {
			displayContent = highlightCode(displayContent, title)
		}
		rendered = renderCodeBox(displayContent, label, th.Border, bs.LabelAlign, contentWidth)

	case block.Quote:
		bs := th.Blocks.Quote
		barColor := resolveColor(bs.BarColor, th.Muted)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render(bs.Bar)
		lines := strings.Split(wrapped, "\n")
		for i, l := range lines {
			lines[i] = bar + l
		}
		rendered = strings.Join(lines, "\n")

	case block.Divider:
		bs := th.Blocks.Divider
		divColor := resolveColor(bs.Color, th.Muted)
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color(divColor)).Render(strings.Repeat(bs.Char, width))

	default:
		rendered = wrapped
	}

	return rendered
}
