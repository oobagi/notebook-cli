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


// calloutVariantColor returns the hex color for a given callout variant.
func calloutVariantColor(cs theme.CalloutStyle, v block.CalloutVariant) string {
	switch v {
	case block.CalloutNote:
		return cs.NoteColor
	case block.CalloutTip:
		return cs.TipColor
	case block.CalloutWarning:
		return cs.WarningColor
	case block.CalloutCaution:
		return cs.CautionColor
	case block.CalloutImportant:
		return cs.ImportantColor
	default:
		return cs.NoteColor
	}
}

// listIndent returns the whitespace prefix for a list item's indent level.
func listIndent(b block.Block) string {
	if b.Indent == 0 {
		return ""
	}
	return strings.Repeat("    ", b.Indent)
}

// resolveColor returns color if non-empty, otherwise returns fallback.
func resolveColor(color, fallback string) string {
	if color == "" {
		return fallback
	}
	return color
}

// priorityColor returns the theme color for a priority level.
// HIGH → Error (red), MED → Warning (yellow), LOW → Muted (gray).
func priorityColor(p block.Priority, th theme.Theme) string {
	switch p {
	case block.PriorityHigh:
		return th.Error
	case block.PriorityMed:
		return th.Warning
	case block.PriorityLow:
		return th.Muted
	default:
		return ""
	}
}

// renderPriorityBadge returns the colored priority badge, or "" for None.
// Format: "[!!!] " (HIGH bold red), "[!!] " (MED yellow), "[!] " (LOW dim).
// Trailing space included so it can be appended to a prefix directly.
func renderPriorityBadge(p block.Priority, th theme.Theme) string {
	if p == block.PriorityNone {
		return ""
	}
	color := priorityColor(p, th)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	if p == block.PriorityHigh {
		style = style.Bold(true)
	}
	return style.Render(p.Marker()) + " "
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
	if m.modified() {
		left += " " + metaStyle.Render("[modified]")
	}

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
	contentWidth := m.width - gutterWidth - blockPrefixWidth(b)
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
		charWidth := len([]rune(bs.Char))
		if charWidth < 1 {
			charWidth = 1
		}
		divLine := strings.Repeat(bs.Char, w/charWidth)
		if runes := []rune(divLine); len(runes) > w {
			divLine = string(runes[:w])
		}
		rendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color(divColor)).
			Render(divLine)
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
	cursorByteOffset := -1

	for rawIdx, raw := range rawLines {
		segs := textarea.Wrap([]rune(raw), contentWidth)
		if len(segs) == 0 {
			segs = [][]rune{{}}
		}
		for wIdx, seg := range segs {
			line := strings.TrimSuffix(string(seg), " ")
			visIdx := len(visualLines)

			if rawIdx == cursorRawRow && wIdx == cursorWrapRow {
				cursorVisIdx = visIdx
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
				cursorByteOffset = len(before)
				line = before + ta.CursorView() + after
			}

			// Pad to contentWidth so all lines share the same width.
			// Skip headings — padding spaces would get underlined/styled.
			if m.wordWrap && !b.Type.IsHeading() {
				if pad := contentWidth - lipgloss.Width(line); pad > 0 {
					line += strings.Repeat(" ", pad)
				}
			}
			visualLines = append(visualLines, line)
		}
	}

	taView := strings.Join(visualLines, "\n")
	cursorLen := len(ta.CursorView())

	var rendered string

	switch b.Type {
	case block.Heading1:
		style := th.Blocks.Heading1.Text.ToLipgloss(th.Accent)
		rendered = styleAroundCursor(taView, style, cursorByteOffset, cursorLen, cursorVisIdx)

	case block.Heading2:
		style := th.Blocks.Heading2.Text.ToLipgloss("")
		rendered = styleAroundCursor(taView, style, cursorByteOffset, cursorLen, cursorVisIdx)

	case block.Heading3:
		style := th.Blocks.Heading3.Text.ToLipgloss("")
		rendered = styleAroundCursor(taView, style, cursorByteOffset, cursorLen, cursorVisIdx)

	case block.BulletList:
		bs := th.Blocks.Bullet
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, taView)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := block.CountNumberedPosition(m.blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, taView)

	case block.Checklist:
		bs := th.Blocks.Checklist
		indent := listIndent(b)
		badge := renderPriorityBadge(b.Priority, th)
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
			if bs.CheckedBold {
				style = style.Bold(true)
			}
			prefix := indent + style.Render(bs.Checked) + badge
			text := taView
			if bs.CheckedTextFaint {
				text = styleAroundCursor(taView, lipgloss.NewStyle().Faint(true), cursorByteOffset, cursorLen, cursorVisIdx)
			}
			rendered = prefixFirstLine(prefix, text)
		} else {
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked) + badge
			rendered = prefixFirstLine(prefix, taView)
		}

	case block.CodeBlock:
		bs := th.Blocks.Code
		boxWidth := contentWidth
		if !m.wordWrap {
			// In no-wrap mode contentWidth is 1000; size the box border to
			// the terminal instead so it doesn't overflow every line.
			boxWidth = m.width - gutterWidth - blockPrefixWidth(b)
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
		cursorOnTitle := ta.Line() == 0
		if cursorOnTitle {
			if titleText == "" {
				label = renderPlaceholder("language", true)
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

	case block.DefinitionList:
		bs := th.Blocks.Definition
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		marker := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		markerIndent := strings.Repeat(" ", lipgloss.Width(marker))
		rawLines := strings.Split(ta.Value(), "\n")
		lines := strings.Split(taView, "\n")

		// Build a map: for each raw line, how many visual lines it occupies.
		// Visual line index → (rawLineIdx, isFirstVisualLine).
		type visInfo struct {
			rawIdx  int
			isFirst bool
		}
		visMap := make([]visInfo, len(lines))
		visIdx := 0
		for ri, raw := range rawLines {
			nVis := 1
			if m.wordWrap {
				segs := textarea.Wrap([]rune(raw), contentWidth)
				if len(segs) > 0 {
					nVis = len(segs)
				}
			}
			for v := 0; v < nVis && visIdx < len(lines); v++ {
				visMap[visIdx] = visInfo{rawIdx: ri, isFirst: v == 0}
				visIdx++
			}
		}

		for i, l := range lines {
			vi := visMap[i]
			if vi.rawIdx == 0 {
				// Term line: no prefix, just bold + placeholder.
				if rawLines[0] == "" && vi.isFirst {
					lines[i] = renderPlaceholder("Term", cursorVisIdx == i)
				} else if bs.TermBold {
					lines[i] = styleAroundCursor(l, lipgloss.NewStyle().Bold(true), cursorByteOffset, cursorLen, cursorVisIdx-i)
				}
			} else if vi.isFirst {
				// First visual line of a definition: marker prefix.
				if rawLines[vi.rawIdx] == "" {
					lines[i] = marker + renderPlaceholder("Definition", cursorVisIdx == i)
				} else {
					lines[i] = marker + l
				}
			} else {
				// Wrapped continuation: indent to match marker width.
				lines[i] = markerIndent + l
			}
		}
		rendered = strings.Join(lines, "\n")

	case block.Embed:
		bs := th.Blocks.Embed
		embedColor := resolveColor(bs.Color, th.Accent)
		icon := bs.Icon
		if icon == "" {
			icon = "\u2197 "
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(embedColor)).Render(icon)
		if content == "" {
			rendered = prefix + renderPlaceholder("Link or note path", true)
		} else {
			rendered = prefixFirstLine(prefix, taView)
		}

	case block.Bookmark:
		bs := th.Blocks.Bookmark
		titleColor := resolveColor(bs.TitleColor, th.Accent)
		urlColor := resolveColor(bs.URLColor, th.Muted)
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(titleColor))
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(urlColor)).Underline(true)
		rawLines := strings.Split(ta.Value(), "\n")
		title := ""
		url := ""
		if len(rawLines) > 0 {
			title = rawLines[0]
		}
		if len(rawLines) > 1 {
			url = rawLines[1]
		}
		cursorOnTitle := ta.Line() == 0
		var titleLine, urlLine string
		if title == "" {
			titleLine = renderPlaceholder("Bookmark title", cursorOnTitle)
		} else if cursorOnTitle {
			titleLine = renderLabelCursor(title, ta.LineInfo().ColumnOffset, titleStyle)
		} else {
			titleLine = titleStyle.Render(title)
		}
		if url == "" {
			urlLine = renderPlaceholder("https://", !cursorOnTitle)
		} else if !cursorOnTitle {
			urlLine = renderLabelCursor(url, ta.LineInfo().ColumnOffset, urlStyle)
		} else {
			urlLine = urlStyle.Render(url)
		}
		rendered = titleLine + "\n" + urlLine

	case block.Callout:
		cs := th.Blocks.Callout
		variantColor := calloutVariantColor(cs, b.Variant)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(variantColor)).Render(th.Blocks.Quote.Bar)
		variantLabel := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(variantColor)).
			Render(b.Variant.String())
		var result []string
		result = append(result, bar+variantLabel)
		if content == "" {
			result = append(result, bar+renderPlaceholder("Empty callout", true))
		} else {
			for _, l := range strings.Split(taView, "\n") {
				result = append(result, bar+l)
			}
		}
		rendered = strings.Join(result, "\n")

	case block.Table:
		// Table rendering is handled entirely by renderActiveTableFull.
		return m.renderActiveTableFull(idx, b)

	case block.Kanban:
		return m.renderActiveKanban(idx, b)

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
		// Exception: when the cursor is on the title (raw row 0), the
		// label lives in the top border line itself (line 0), so the
		// adjusted index should be 0, not 1.
		adjustedCursor := cursorVisIdx + 1
		if cursorRawRow == 0 {
			adjustedCursor = 0
		}
		for i, l := range lines {
			lines[i] = scrollOrTruncate(l, m.width, cursorCol, i == adjustedCursor)
		}
	} else {
		cursorCol := gutterWidth + blockPrefixWidth(b) + cursorColInWrap
		for i, l := range lines {
			lines[i] = scrollOrTruncate(l, m.width, cursorCol, i == cursorVisIdx)
		}
	}

	return strings.Join(lines, "\n")
}

// renderBookmarkCard renders a bookmark as a bordered card with title on
// top and URL below. width is the content column width.
func renderBookmarkCard(content string, width int, hovered bool) string {
	th := theme.Current()
	bs := th.Blocks.Bookmark
	borderColor := resolveColor(bs.Border, th.Border)
	titleColor := resolveColor(bs.TitleColor, th.Accent)
	urlColor := resolveColor(bs.URLColor, th.Muted)

	title, url := block.ExtractBookmark(content)
	if title == "" && url == "" {
		title = "Bookmark"
	}
	if title == "" {
		title = url
	}

	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(titleColor))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(urlColor))
	if hovered {
		urlStyle = urlStyle.Underline(true)
		titleStyle = titleStyle.Underline(true)
	}
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

	titleR := []rune(title)
	if len(titleR) > innerW {
		titleR = append(titleR[:innerW-1], '…')
	}
	urlR := []rune(url)
	if len(urlR) > innerW {
		urlR = append(urlR[:innerW-1], '…')
	}
	titleLine := titleStyle.Render(string(titleR))
	urlLine := urlStyle.Render(string(urlR))

	titlePad := innerW - len([]rune(string(titleR)))
	if titlePad < 0 {
		titlePad = 0
	}
	urlPad := innerW - len([]rune(string(urlR)))
	if urlPad < 0 {
		urlPad = 0
	}

	top := bc.Render("╭" + strings.Repeat("─", innerW+2) + "╮")
	bottom := bc.Render("╰" + strings.Repeat("─", innerW+2) + "╯")
	bar := bc.Render("│")
	titleRow := bar + " " + titleLine + strings.Repeat(" ", titlePad) + " " + bar
	urlRow := bar + " " + urlLine + strings.Repeat(" ", urlPad) + " " + bar
	return strings.Join([]string{top, titleRow, urlRow, bottom}, "\n")
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

// renderPlaceholder renders placeholder text with consistent behavior: the
// cursor sits on the first letter (plain reverse) and the rest is faint. When
// the cursor is not on this field, the entire text is faint.
func renderPlaceholder(text string, hasCursor bool) string {
	faint := lipgloss.NewStyle().Faint(true)
	if !hasCursor {
		return faint.Render(text)
	}
	runes := []rune(text)
	cursor := lipgloss.NewStyle().Reverse(true)
	return cursor.Render(string(runes[:1])) + faint.Render(string(runes[1:]))
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
// ANSI codes. Uses explicit byte offsets to locate the cursor, avoiding false
// matches when the cursor's blink-off view is a plain space.
func styleAroundCursor(text string, style lipgloss.Style, cursorOffset, cursorLen, cursorLine int) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if i == cursorLine && cursorOffset >= 0 && cursorOffset+cursorLen <= len(l) {
			before := l[:cursorOffset]
			cursor := l[cursorOffset : cursorOffset+cursorLen]
			after := l[cursorOffset+cursorLen:]
			lines[i] = style.Render(before) + cursor + style.Render(after)
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
			out = append(out, strings.TrimSuffix(string(seg), " "))
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
	contentWidth := width - gutterWidth - blockPrefixWidth(b)
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
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := block.CountNumberedPosition(blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		bs := th.Blocks.Checklist
		indent := listIndent(b)
		badge := renderPriorityBadge(b.Priority, th)
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
			if bs.CheckedBold {
				style = style.Bold(true)
			}
			prefix := indent + style.Render(bs.Checked) + badge
			text := wrapped
			if bs.CheckedTextFaint {
				text = lipgloss.NewStyle().Faint(true).Render(wrapped)
			}
			rendered = prefixFirstLine(prefix, text)
		} else {
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked) + badge
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

	case block.Callout:
		cs := th.Blocks.Callout
		variantColor := calloutVariantColor(cs, b.Variant)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(variantColor)).Render(th.Blocks.Quote.Bar)
		variantLabel := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(variantColor)).
			Render(b.Variant.String())
		var result []string
		result = append(result, bar+variantLabel)
		if wrapped == "" {
			placeholder := lipgloss.NewStyle().Faint(true).Render("Empty callout")
			result = append(result, bar+placeholder)
		} else {
			for _, l := range strings.Split(wrapped, "\n") {
				result = append(result, bar+l)
			}
		}
		rendered = strings.Join(result, "\n")

	case block.Divider:
		bs := th.Blocks.Divider
		divColor := resolveColor(bs.Color, th.Muted)
		w := width - gutterWidth
		if w <= 0 {
			w = 40
		}
		charW := len([]rune(bs.Char))
		if charW < 1 {
			charW = 1
		}
		divLine := strings.Repeat(bs.Char, w/charW)
		if runes := []rune(divLine); len(runes) > w {
			divLine = string(runes[:w])
		}
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color(divColor)).Render(divLine)

	case block.DefinitionList:
		bs := th.Blocks.Definition
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		marker := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		markerIndent := strings.Repeat(" ", lipgloss.Width(marker))
		term, def := block.ExtractDefinition(content)
		termWrapped := term
		if wordWrap {
			// Term has no marker prefix; contentWidth already subtracts
			// blockPrefixWidth (marker), so term wraps at the same width
			// as the textarea in active mode.
			termWrapped = wrapText(term, contentWidth)
		}
		termWrapped = format.RenderInlineMarkdown(termWrapped)
		if bs.TermBold {
			termWrapped = lipgloss.NewStyle().Bold(true).Render(termWrapped)
		}
		// Definition lines wrap at contentWidth — the marker is rendered
		// within the prefix space already reserved by blockPrefixWidth.
		// Do NOT subtract marker width again.
		var defOutput []string
		for _, rawDef := range strings.Split(def, "\n") {
			dw := rawDef
			if wordWrap {
				dw = wrapText(rawDef, contentWidth)
			}
			dw = format.RenderInlineMarkdown(dw)
			for j, vl := range strings.Split(dw, "\n") {
				if j == 0 {
					defOutput = append(defOutput, marker+vl)
				} else {
					defOutput = append(defOutput, markerIndent+vl)
				}
			}
		}
		rendered = termWrapped + "\n" + strings.Join(defOutput, "\n")

	case block.Embed:
		bs := th.Blocks.Embed
		embedColor := resolveColor(bs.Color, th.Accent)
		icon := bs.Icon
		if icon == "" {
			icon = "\u2197 "
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(embedColor)).Render(icon)
		if wrapped == "" {
			rendered = prefix + renderPlaceholder("Link or note path", false)
		} else {
			rendered = lipgloss.NewStyle().
				Foreground(lipgloss.Color(embedColor)).
				Render(icon + wrapped)
		}

	case block.Bookmark:
		rendered = renderBookmarkCard(content, contentWidth, false)

	case block.Table:
		tableWidth := width - gutterWidth
		if tableWidth < 10 {
			tableWidth = 10
		}
		rendered = renderTableGrid(content, tableWidth, th.Border, th.Blocks.Table.HeaderBold, wordWrap)

	case block.Kanban:
		boardWidth := width - gutterWidth
		if boardWidth < 30 {
			boardWidth = 30
		}
		rendered = renderInactiveKanbanBoard(content, boardWidth, 0, th)

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
func renderViewBlock(b block.Block, content string, width int, wordWrap bool, blocks []block.Block, idx int, hovered bool, kanbanOffset int) string {
	// Width here is the content column width (already constrained to viewMaxWidth).
	contentWidth := width - blockPrefixWidth(b)
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
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		rendered = prefixFirstLine(prefix, wrapped)

	case block.NumberedList:
		bs := th.Blocks.Numbered
		num := block.CountNumberedPosition(blocks, idx)
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		indent := listIndent(b)
		prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(fmt.Sprintf(bs.Format, num))
		rendered = prefixFirstLine(prefix, wrapped)

	case block.Checklist:
		bs := th.Blocks.Checklist
		indent := listIndent(b)
		badge := renderPriorityBadge(b.Priority, th)
		if b.Checked {
			checkedColor := resolveColor(bs.CheckedColor, th.Accent)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
			if bs.CheckedBold {
				style = style.Bold(true)
			}
			prefix := indent + style.Render(bs.Checked) + badge
			text := wrapped
			if bs.CheckedTextFaint {
				text = lipgloss.NewStyle().Faint(true).Render(wrapped)
			}
			rendered = prefixFirstLine(prefix, text)
		} else {
			uncheckedColor := resolveColor(bs.UncheckedColor, th.Muted)
			if hovered {
				uncheckedColor = th.Accent
			}
			prefix := indent + lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor)).Render(bs.Unchecked) + badge
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

	case block.Callout:
		cs := th.Blocks.Callout
		variantColor := calloutVariantColor(cs, b.Variant)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(variantColor)).Render(th.Blocks.Quote.Bar)
		variantLabel := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(variantColor)).
			Render(b.Variant.String())
		var result []string
		result = append(result, bar+variantLabel)
		if wrapped == "" {
			placeholder := lipgloss.NewStyle().Faint(true).Render("Empty callout")
			result = append(result, bar+placeholder)
		} else {
			for _, l := range strings.Split(wrapped, "\n") {
				result = append(result, bar+l)
			}
		}
		rendered = strings.Join(result, "\n")

	case block.Divider:
		bs := th.Blocks.Divider
		divColor := resolveColor(bs.Color, th.Muted)
		charWidth := len([]rune(bs.Char))
		if charWidth < 1 {
			charWidth = 1
		}
		repeatCount := width / charWidth
		divLine := strings.Repeat(bs.Char, repeatCount)
		// Trim to exact width in case of rounding.
		if runes := []rune(divLine); len(runes) > width {
			divLine = string(runes[:width])
		}
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color(divColor)).Render(divLine)

	case block.DefinitionList:
		bs := th.Blocks.Definition
		markerColor := resolveColor(bs.MarkerColor, th.Muted)
		marker := lipgloss.NewStyle().Foreground(lipgloss.Color(markerColor)).Render(bs.Marker)
		markerIndent := strings.Repeat(" ", lipgloss.Width(marker))
		term, def := block.ExtractDefinition(content)
		termWrapped := wrapText(term, contentWidth)
		termWrapped = format.RenderInlineMarkdown(termWrapped)
		if bs.TermBold {
			termWrapped = lipgloss.NewStyle().Bold(true).Render(termWrapped)
		}
		// contentWidth already has marker space reserved via blockPrefixWidth.
		var defOutput []string
		for _, rawDef := range strings.Split(def, "\n") {
			dw := wrapText(rawDef, contentWidth)
			dw = format.RenderInlineMarkdown(dw)
			for j, vl := range strings.Split(dw, "\n") {
				if j == 0 {
					defOutput = append(defOutput, marker+vl)
				} else {
					defOutput = append(defOutput, markerIndent+vl)
				}
			}
		}
		rendered = termWrapped + "\n" + strings.Join(defOutput, "\n")

	case block.Embed:
		bs := th.Blocks.Embed
		embedColor := resolveColor(bs.Color, th.Accent)
		icon := bs.Icon
		if icon == "" {
			icon = "\u2197 "
		}
		if wrapped == "" {
			prefix := lipgloss.NewStyle().Foreground(lipgloss.Color(embedColor)).Render(icon)
			rendered = prefix + renderPlaceholder("Link or note path", false)
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(embedColor))
			if hovered {
				style = style.Underline(true)
			}
			rendered = style.Render(icon + wrapped)
		}

	case block.Bookmark:
		rendered = renderBookmarkCard(content, contentWidth, hovered)

	case block.Table:
		rendered = renderTableGrid(content, contentWidth, th.Border, th.Blocks.Table.HeaderBold, true)

	case block.Kanban:
		rendered = renderInactiveKanbanBoard(content, contentWidth, kanbanOffset, th)

	default:
		rendered = wrapped
	}

	return rendered
}
