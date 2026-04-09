// Package block defines the Block type and BlockType constants used to
// represent parsed markdown content as a sequence of typed blocks.
//
// This package is pure data — it has no TUI or rendering dependencies and
// is safe to use from tests, CLI tools, or any other context.
package block

import "strings"

// BlockType enumerates the kinds of content blocks that a markdown document
// can contain.
type BlockType int

const (
	Paragraph    BlockType = iota // plain text paragraph
	Heading1                      // # heading
	Heading2                      // ## heading
	Heading3                      // ### heading
	BulletList                    // - or * list item
	NumberedList                  // 1. numbered list item
	Checklist                     // - [ ] or - [x] checklist item
	CodeBlock                     // fenced code block (``` ... ```)
	Quote                         // > block quote
	Divider                       // ---, ***, or ___
	DefinitionList                // term\n: definition
)

// String returns the human-readable name of a BlockType.
func (bt BlockType) String() string {
	switch bt {
	case Paragraph:
		return "Paragraph"
	case Heading1:
		return "Heading1"
	case Heading2:
		return "Heading2"
	case Heading3:
		return "Heading3"
	case BulletList:
		return "BulletList"
	case NumberedList:
		return "NumberedList"
	case Checklist:
		return "Checklist"
	case CodeBlock:
		return "CodeBlock"
	case Quote:
		return "Quote"
	case Divider:
		return "Divider"
	case DefinitionList:
		return "DefinitionList"
	default:
		return "Unknown"
	}
}

// IsHeading returns true for Heading1, Heading2, and Heading3.
func (bt BlockType) IsHeading() bool {
	return bt == Heading1 || bt == Heading2 || bt == Heading3
}

// IsListItem returns true for BulletList, NumberedList, and Checklist.
func (bt BlockType) IsListItem() bool {
	return bt == BulletList || bt == NumberedList || bt == Checklist
}

// Short returns a compact abbreviation of a BlockType for gutter labels.
func (bt BlockType) Short() string {
	switch bt {
	case Paragraph:
		return "p"
	case Heading1:
		return "h1"
	case Heading2:
		return "h2"
	case Heading3:
		return "h3"
	case BulletList:
		return "ul"
	case NumberedList:
		return "ol"
	case Checklist:
		return "cx"
	case CodeBlock:
		return "cd"
	case Quote:
		return "qt"
	case Divider:
		return "hr"
	case DefinitionList:
		return "df"
	default:
		return "?"
	}
}

// Block holds a single parsed content block.
type Block struct {
	Type    BlockType // kind of block
	Content string    // text content without markdown prefix
	Checked bool      // whether checklist item is checked (Checklist only)
	Indent  int       // nesting level for list items (0 = top level)
}

// CountNumberedPosition returns the 1-based position of a numbered list block
// among consecutive NumberedList blocks at the same indent level.
func CountNumberedPosition(blocks []Block, idx int) int {
	if idx < 0 || idx >= len(blocks) || blocks[idx].Type != NumberedList {
		return 1
	}
	targetIndent := blocks[idx].Indent
	num := 1
	for i := idx - 1; i >= 0; i-- {
		if blocks[i].Type == NumberedList && blocks[i].Indent == targetIndent {
			num++
		} else if blocks[i].Indent > targetIndent {
			continue
		} else {
			break
		}
	}
	return num
}

// ExtractDefinition splits a definition list block's content into its term
// (first line) and definition (remaining lines). The content stores both
// parts separated by a newline.
func ExtractDefinition(content string) (term, definition string) {
	first, rest, found := strings.Cut(content, "\n")
	if found {
		return first, rest
	}
	return first, ""
}

// ExtractCodeLanguage splits a code block's content into its title line
// (typically a language identifier) and the remaining body. The first line
// is always treated as the title regardless of whether it is a recognized
// programming language.
func ExtractCodeLanguage(content string) (title, body string) {
	first, rest, found := strings.Cut(content, "\n")
	t := strings.TrimSpace(first)
	if found {
		return t, rest
	}
	return t, ""
}
