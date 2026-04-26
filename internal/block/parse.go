package block

import (
	"regexp"
	"strings"
)

var (
	linkLinkRe = regexp.MustCompile(`^\[([^\]]*)\]\((https?://[^\s)]+)\)\s*$`)
	linkBareRe = regexp.MustCompile(`^(https?://\S+)\s*$`)
)

// ParseLink reports whether a single line is a link — either a
// titled markdown link `[title](url)` or a bare http(s) URL — and returns
// the extracted title (may be empty) and URL.
func ParseLink(line string) (title, url string, ok bool) {
	if m := linkLinkRe.FindStringSubmatch(line); m != nil {
		return m[1], m[2], true
	}
	if m := linkBareRe.FindStringSubmatch(line); m != nil {
		return "", m[1], true
	}
	return "", "", false
}

// stripListIndent counts leading 4-space groups on a line.
func stripListIndent(line string) (indent int, stripped string) {
	i := 0
	for i+3 < len(line) && line[i] == ' ' && line[i+1] == ' ' && line[i+2] == ' ' && line[i+3] == ' ' {
		indent++
		i += 4
	}
	return indent, line[indent*4:]
}

// Parse splits a markdown document into a sequence of Blocks. It walks the
// input line by line, recognising headings, lists, checklists, code fences,
// block quotes, dividers, and paragraphs. Consecutive paragraph lines and
// consecutive quote lines are merged into single blocks. Blank lines produce
// empty Paragraph blocks to preserve vertical spacing.
//
// An unclosed code fence (no closing ```) treats all remaining lines as code
// block content.
func Parse(markdown string) []Block {
	// Special case: completely empty input.
	if markdown == "" {
		return []Block{{Type: Paragraph, Content: ""}}
	}

	lines := strings.Split(markdown, "\n")
	var blocks []Block

	i := 0
	for i < len(lines) {
		line := lines[i]

		// --- Code fence ---
		if strings.HasPrefix(line, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(line, "```"))
			var contentLines []string
			i++
			for i < len(lines) {
				if strings.HasPrefix(lines[i], "```") {
					i++ // skip closing fence
					break
				}
				contentLines = append(contentLines, lines[i])
				i++
			}
			content := strings.Join(contentLines, "\n")

			// Special-case kanban fence: own block type, content stored verbatim.
			if strings.EqualFold(lang, "kanban") {
				blocks = append(blocks, Block{
					Type:    Kanban,
					Content: content,
				})
				continue
			}

			if lang != "" {
				if content == "" {
					content = lang
				} else {
					content = lang + "\n" + content
				}
			} else if content != "" {
				// No language: prepend empty title line so the first line
				// of content is always the title/language field.
				content = "\n" + content
			}
			blocks = append(blocks, Block{
				Type:    CodeBlock,
				Content: content,
			})
			continue
		}

		// --- Table (pipe-delimited) ---
		if strings.HasPrefix(line, "|") {
			var tableLines []string
			for i < len(lines) && strings.HasPrefix(lines[i], "|") {
				tableLines = append(tableLines, lines[i])
				i++
			}
			blocks = append(blocks, Block{
				Type:    Table,
				Content: strings.Join(tableLines, "\n"),
			})
			continue
		}

		// --- Blank line ---
		if line == "" {
			blocks = append(blocks, Block{Type: Paragraph, Content: ""})
			i++
			continue
		}

		// --- Divider (---, ***, ___) ---
		if isDivider(line) {
			blocks = append(blocks, Block{Type: Divider})
			i++
			continue
		}

		// --- Headings ---
		if strings.HasPrefix(line, "### ") {
			blocks = append(blocks, Block{
				Type:    Heading3,
				Content: strings.TrimPrefix(line, "### "),
			})
			i++
			continue
		}
		if strings.HasPrefix(line, "## ") {
			blocks = append(blocks, Block{
				Type:    Heading2,
				Content: strings.TrimPrefix(line, "## "),
			})
			i++
			continue
		}
		if strings.HasPrefix(line, "# ") {
			blocks = append(blocks, Block{
				Type:    Heading1,
				Content: strings.TrimPrefix(line, "# "),
			})
			i++
			continue
		}

		// --- List items (with optional indent) ---
		indent, stripped := stripListIndent(line)

		if strings.HasPrefix(stripped, "- [x] ") || strings.HasPrefix(stripped, "- [X] ") {
			prio, body := ParsePriorityMarker(stripped[6:])
			blocks = append(blocks, Block{Type: Checklist, Content: body, Checked: true, Indent: indent, Priority: prio})
			i++
			continue
		}
		if strings.HasPrefix(stripped, "- [ ] ") {
			prio, body := ParsePriorityMarker(stripped[6:])
			blocks = append(blocks, Block{Type: Checklist, Content: body, Checked: false, Indent: indent, Priority: prio})
			i++
			continue
		}
		if strings.HasPrefix(stripped, "- ") || strings.HasPrefix(stripped, "* ") {
			blocks = append(blocks, Block{Type: BulletList, Content: stripped[2:], Indent: indent})
			i++
			continue
		}
		if _, content, ok := parseNumberedItem(stripped); ok {
			blocks = append(blocks, Block{Type: NumberedList, Content: content, Indent: indent})
			i++
			continue
		}

		// --- Block quotes / Callouts ---
		if strings.HasPrefix(line, "> ") || line == ">" {
			var quoteLines []string
			for i < len(lines) {
				if strings.HasPrefix(lines[i], "> ") {
					quoteLines = append(quoteLines, strings.TrimPrefix(lines[i], "> "))
				} else if lines[i] == ">" {
					quoteLines = append(quoteLines, "")
				} else {
					break
				}
				i++
			}

			// Check if this is a callout/admonition: first line matches [!TYPE]
			if len(quoteLines) > 0 {
				if variant, ok := ParseCalloutMarker(quoteLines[0]); ok {
					content := ""
					if len(quoteLines) > 1 {
						content = strings.Join(quoteLines[1:], "\n")
					}
					blocks = append(blocks, Block{
						Type:    Callout,
						Content: content,
						Variant: variant,
					})
					continue
				}
			}

			blocks = append(blocks, Block{
				Type:    Quote,
				Content: strings.Join(quoteLines, "\n"),
			})
			continue
		}

		// --- Embed (![[path]]) ---
		if strings.HasPrefix(line, "![[") && strings.HasSuffix(line, "]]") {
			path := line[3 : len(line)-2]
			blocks = append(blocks, Block{Type: Embed, Content: path})
			i++
			continue
		}

		// --- Link ([title](url) or bare URL on its own line) ---
		if title, url, ok := ParseLink(line); ok {
			content := url
			if title != "" {
				content = title + "\n" + url
			}
			blocks = append(blocks, Block{Type: Link, Content: content})
			i++
			continue
		}

		// --- Definition (term followed by one or more ": definition" lines) ---
		if i+1 < len(lines) && !isBlockStart(line) && isDefinitionLine(lines[i+1]) {
			term := line
			i++
			var defs []string
			for i < len(lines) && isDefinitionLine(lines[i]) {
				defs = append(defs, strings.TrimPrefix(lines[i], ": "))
				i++
			}
			blocks = append(blocks, Block{
				Type:    DefinitionList,
				Content: term + "\n" + strings.Join(defs, "\n"),
			})
			continue
		}

		// --- Paragraph (merge consecutive non-special lines) ---
		var paraLines []string
		for i < len(lines) {
			if isBlockStart(lines[i]) {
				break
			}
			paraLines = append(paraLines, lines[i])
			i++
		}
		// Safety: if no lines were collected (current line is a block start
		// that wasn't matched by any preceding handler), consume it as a
		// single-line paragraph to avoid an infinite loop.
		if len(paraLines) == 0 {
			paraLines = append(paraLines, lines[i])
			i++
		}
		blocks = append(blocks, Block{
			Type:    Paragraph,
			Content: strings.Join(paraLines, "\n"),
		})
	}

	return blocks
}

// isBlockStart reports whether a line begins a new block (i.e. is not plain
// paragraph text). Used by the paragraph merger to know when to stop.
func isBlockStart(line string) bool {
	if line == "" || line == ">" {
		return true
	}
	if strings.HasPrefix(line, "```") ||
		strings.HasPrefix(line, "# ") ||
		strings.HasPrefix(line, "## ") ||
		strings.HasPrefix(line, "### ") ||
		strings.HasPrefix(line, "> ") ||
		strings.HasPrefix(line, ": ") ||
		strings.HasPrefix(line, "|") {
		return true
	}
	_, stripped := stripListIndent(line)
	if strings.HasPrefix(stripped, "- ") ||
		strings.HasPrefix(stripped, "* ") ||
		strings.HasPrefix(stripped, "- [") {
		return true
	}
	if _, _, ok := parseNumberedItem(stripped); ok {
		return true
	}
	if strings.HasPrefix(line, "![[") && strings.HasSuffix(line, "]]") {
		return true
	}
	if _, _, ok := ParseLink(line); ok {
		return true
	}
	return isDivider(line)
}

// isDefinitionLine reports whether a line starts a definition (": ").
func isDefinitionLine(line string) bool {
	return strings.HasPrefix(line, ": ")
}

// isDivider reports whether a line is a thematic break (---, ***, or ___).
func isDivider(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	return allSameChar(trimmed, '-') || allSameChar(trimmed, '*') || allSameChar(trimmed, '_')
}

// allSameChar reports whether s consists entirely of character c.
func allSameChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != c {
			return false
		}
	}
	return true
}

// ParseCalloutMarker checks if a line matches the [!TYPE] callout marker
// pattern (case-insensitive). Returns the variant and true if matched.
func ParseCalloutMarker(line string) (CalloutVariant, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[!") || !strings.HasSuffix(trimmed, "]") {
		return CalloutNote, false
	}
	inner := trimmed[2 : len(trimmed)-1]
	return ParseCalloutVariant(inner)
}

// parseNumberedItem splits "123. text" into the number prefix, text, and
// whether the line matched. Combines detection and extraction in one pass.
func parseNumberedItem(line string) (prefix, content string, ok bool) {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i+2 > len(line) || line[i] != '.' || line[i+1] != ' ' {
		return "", "", false
	}
	return line[:i], line[i+2:], true
}
