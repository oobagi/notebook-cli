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
	Embed                         // ![[path]] embedded note reference
	Callout                       // > [!NOTE] callout/admonition
	Table                         // GFM pipe table
	Kanban                        // ```kanban``` fenced kanban board
	Link                      // [title](url) link card
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
	case Embed:
		return "Embed"
	case Callout:
		return "Callout"
	case Table:
		return "Table"
	case Kanban:
		return "Kanban"
	case Link:
		return "Link"
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
	case Embed:
		return "em"
	case Callout:
		return "co"
	case Table:
		return "tb"
	case Kanban:
		return "kb"
	case Link:
		return "ln"
	default:
		return "?"
	}
}

// CalloutVariant enumerates the supported callout/admonition types.
type CalloutVariant int

const (
	CalloutNote      CalloutVariant = iota // [!NOTE]
	CalloutTip                             // [!TIP]
	CalloutWarning                         // [!WARNING]
	CalloutCaution                         // [!CAUTION]
	CalloutImportant                       // [!IMPORTANT]
)

// String returns the uppercase name of a CalloutVariant.
func (cv CalloutVariant) String() string {
	switch cv {
	case CalloutNote:
		return "NOTE"
	case CalloutTip:
		return "TIP"
	case CalloutWarning:
		return "WARNING"
	case CalloutCaution:
		return "CAUTION"
	case CalloutImportant:
		return "IMPORTANT"
	default:
		return "NOTE"
	}
}

// Next returns the next CalloutVariant in the cycle (Note → Tip → ... → Important → Note).
func (cv CalloutVariant) Next() CalloutVariant {
	return (cv + 1) % (CalloutImportant + 1)
}

// ParseCalloutVariant returns the CalloutVariant for a string (case-insensitive).
// The second return value is false if the string is not a recognized variant.
func ParseCalloutVariant(s string) (CalloutVariant, bool) {
	switch strings.ToUpper(s) {
	case "NOTE":
		return CalloutNote, true
	case "TIP":
		return CalloutTip, true
	case "WARNING":
		return CalloutWarning, true
	case "CAUTION":
		return CalloutCaution, true
	case "IMPORTANT":
		return CalloutImportant, true
	default:
		return CalloutNote, false
	}
}

// Block holds a single parsed content block.
type Block struct {
	Type     BlockType      // kind of block
	Content  string         // text content without markdown prefix
	Checked  bool           // whether checklist item is checked (Checklist only)
	Indent   int            // nesting level for list items (0 = top level)
	Variant  CalloutVariant // admonition variant (Callout only)
	Priority Priority       // priority label (Checklist only)
}

// Priority enumerates the priority labels available to checklist items.
// These let checklists serve as kanban-style cards: items can be tagged
// with a priority and moved between heading-defined "sections".
type Priority int

const (
	PriorityNone Priority = iota // no priority label
	PriorityLow                  // ! marker
	PriorityMed                  // !! marker
	PriorityHigh                 // !!! marker
)

// String returns the uppercase short label of a Priority.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "LOW"
	case PriorityMed:
		return "MED"
	case PriorityHigh:
		return "HIGH"
	default:
		return ""
	}
}

// Marker returns the inline markdown marker for a Priority (e.g. "!!").
// Returns "" for PriorityNone.
func (p Priority) Marker() string {
	switch p {
	case PriorityLow:
		return "!"
	case PriorityMed:
		return "!!"
	case PriorityHigh:
		return "!!!"
	default:
		return ""
	}
}

// Next cycles to the next priority: None → Low → Med → High → None.
func (p Priority) Next() Priority {
	return (p + 1) % (PriorityHigh + 1)
}

// ParsePriorityMarker reads an optional leading "!", "!!", or "!!!" marker
// (followed by a space) from content and returns the priority and the
// remaining text. If no marker is present, returns PriorityNone and the
// original string. The marker is recognized only when followed by a space
// so plain "!" exclamations don't accidentally tag items.
func ParsePriorityMarker(content string) (Priority, string) {
	if strings.HasPrefix(content, "!!! ") {
		return PriorityHigh, content[4:]
	}
	if strings.HasPrefix(content, "!! ") {
		return PriorityMed, content[3:]
	}
	if strings.HasPrefix(content, "! ") {
		return PriorityLow, content[2:]
	}
	return PriorityNone, content
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

// ExtractLink splits a link block's content into its URL line (first line)
// and title (second line). The URL is stored first so the active edit form
// presents it before the title. When only a URL is stored, title is empty.
func ExtractLink(content string) (title, url string) {
	first, rest, found := strings.Cut(content, "\n")
	if found {
		return rest, first
	}
	return "", first
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
