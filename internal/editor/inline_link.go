package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
)

func checklistPrefixVisualWidth(b block.Block) int {
	return blockPrefixWidth(b)
}

func checklistInlineLinkAt(b block.Block, line, x, contentColumnWidth int) string {
	return inlineLinkAt(b, b.Content, line, x, contentColumnWidth)
}

func inlineLinkAt(b block.Block, content string, line, x, contentColumnWidth int) string {
	if !supportsClickableInlineLinks(b.Type) {
		return ""
	}
	display, spans := format.InlineLinkDisplay(content)
	if len(spans) == 0 {
		return ""
	}
	wrapWidth := contentColumnWidth - blockPrefixWidth(b)
	if wrapWidth < 1 {
		wrapWidth = 1
	}

	col := x - inlineContentPrefixVisualWidth(b)
	if line < 0 || col < 0 {
		return ""
	}

	for _, span := range spans {
		startLine, startCol := wrappedTextPosition(display, span.Start, wrapWidth)
		endLine, endCol := wrappedTextPosition(display, span.End, wrapWidth)
		switch {
		case line < startLine || line > endLine:
			continue
		case startLine == endLine:
			if col >= startCol && col < endCol {
				return span.URL
			}
		case line == startLine:
			if col >= startCol {
				return span.URL
			}
		case line == endLine:
			if col < endCol {
				return span.URL
			}
		default:
			return span.URL
		}
	}
	return ""
}

func supportsClickableInlineLinks(bt block.BlockType) bool {
	switch bt {
	case block.Paragraph, block.Heading1, block.Heading2, block.Heading3,
		block.BulletList, block.NumberedList, block.Checklist, block.Quote:
		return true
	default:
		return false
	}
}

func inlineContentPrefixVisualWidth(b block.Block) int {
	return blockPrefixWidth(b)
}

func renderInlineMarkdownText(content string, width int, wordWrap bool, hoverURL string) string {
	return renderInlineMarkdownBlockText(block.Paragraph, content, width, wordWrap, hoverURL)
}

func renderInlineMarkdownBlockText(bt block.BlockType, content string, width int, wordWrap bool, hoverURL string) string {
	display, spans := format.InlineLinkDisplay(content)
	if wordWrap {
		wrapped := wrapText(display, width)
		spans = remapInlineLinkSpans(display, spans, width)
		display = wrapped
	}
	return format.RenderInlineMarkdownWithLinksStyled(display, spans, hoverURL, inlineLinkRenderStyle(bt))
}

func inlineLinkRenderStyle(bt block.BlockType) format.InlineLinkRenderStyle {
	th := theme.Current()
	bs := th.Blocks.Link
	labelColor := resolveColor(bs.TitleColor, th.Accent)
	urlColor := resolveColor(bs.URLColor, th.Muted)
	return format.InlineLinkRenderStyle{
		LabelColor: labelColor,
		URLColor:   urlColor,
		HoverColor: labelColor,
		BaseColor:  inlineMarkdownBaseColor(bt, th),
	}
}

func inlineMarkdownBaseColor(bt block.BlockType, th theme.Theme) string {
	switch bt {
	case block.Heading1:
		return textStyleColor(th.Blocks.Heading1.Text, th.Accent)
	case block.Heading2:
		return textStyleColor(th.Blocks.Heading2.Text, "")
	case block.Heading3:
		return textStyleColor(th.Blocks.Heading3.Text, "")
	default:
		return ""
	}
}

func textStyleColor(ts theme.TextStyle, fallback string) string {
	if ts.Color == "-" {
		return ""
	}
	if ts.Color != "" {
		return ts.Color
	}
	return fallback
}

func remapInlineLinkSpans(display string, spans []format.InlineLinkSpan, width int) []format.InlineLinkSpan {
	if len(spans) == 0 {
		return nil
	}
	remapped := make([]format.InlineLinkSpan, len(spans))
	for i, span := range spans {
		remapped[i] = format.InlineLinkSpan{
			Start:    wrappedRuneOffset(display, span.Start, width),
			End:      wrappedRuneOffset(display, span.End, width),
			URLStart: wrappedRuneOffset(display, span.URLStart, width),
			URLEnd:   wrappedRuneOffset(display, span.URLEnd, width),
			URL:      span.URL,
		}
	}
	return remapped
}

func wrappedRuneOffset(text string, offset, width int) int {
	if width <= 0 {
		width = 1
	}
	runes := []rune(text)
	if offset < 0 {
		offset = 0
	}
	if offset > len(runes) {
		offset = len(runes)
	}
	if offset == 0 {
		return 0
	}
	return len([]rune(wrapText(string(runes[:offset]), width)))
}

func wrappedTextPosition(text string, offset, width int) (line, col int) {
	if width <= 0 {
		width = 1
	}
	runes := []rune(text)
	if offset < 0 {
		offset = 0
	}
	if offset > len(runes) {
		offset = len(runes)
	}
	if offset == 0 {
		return 0, 0
	}

	prefix := string(runes[:offset])
	lines := strings.Split(wrapText(prefix, width), "\n")
	if len(lines) == 0 {
		return 0, 0
	}
	return len(lines) - 1, lipgloss.Width(lines[len(lines)-1])
}

func checklistClickShouldToggle(b block.Block, hasInlineLinks bool, line, x int) bool {
	if !hasInlineLinks {
		return true
	}
	return line == 0 && x >= lipgloss.Width(listIndent(b)) && x < checklistPrefixVisualWidth(b)
}
