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

// Block holds a single parsed content block.
type Block struct {
	Type     BlockType // kind of block
	Content  string    // text content without markdown prefix
	Language string    // code block language hint (CodeBlock only)
	Checked  bool      // whether checklist item is checked (Checklist only)
}
