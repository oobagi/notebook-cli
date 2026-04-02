// Package block defines the Block type and BlockType constants used to
// represent parsed markdown content as a sequence of typed blocks.
//
// This package is pure data — it has no TUI or rendering dependencies and
// is safe to use from tests, CLI tools, or any other context.
package block

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
	default:
		return "Unknown"
	}
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
	default:
		return "?"
	}
}

// Block holds a single parsed content block.
type Block struct {
	Type     BlockType // kind of block
	Content  string    // text content without markdown prefix
	Language string    // code block language hint (CodeBlock only)
	Checked  bool      // whether checklist item is checked (Checklist only)
}
