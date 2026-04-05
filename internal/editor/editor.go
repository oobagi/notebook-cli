// Package editor provides a Bubble Tea TUI for editing markdown notes
// using a block-based editing surface where each block has its own textarea.
package editor

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/config"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// Config holds the configuration for the editor.
type Config struct {
	// Title is displayed in the header bar (e.g. "book > note").
	Title string
	// FilePath is the full filesystem path to the file being edited.
	FilePath string
	// FileSize is the file size in bytes.
	FileSize int64
	// Content is the initial text content to edit.
	Content string
	// Save is called with the current content when the user presses Ctrl+S.
	Save func(string) error
	// DismissedHints tracks which onboarding hints have been dismissed.
	DismissedHints map[string]bool
}

// savedMsg is sent after a successful save.
type savedMsg struct{ content string }

// saveErrMsg is sent when a save fails.
type saveErrMsg struct{ err error }

// savedQuitMsg is sent after a successful save-then-quit.
type savedQuitMsg struct{ content string }

// statusTimeoutMsg is sent after the status auto-dismiss delay.
// The generation field is compared against Model.statusGen to ensure
// only the most recent status message is cleared.
type statusTimeoutMsg struct{ generation int }

// statusTimeout is the delay before auto-dismissing a transient status message.
const statusTimeout = 4 * time.Second

// Model is the Bubble Tea model for the block-based editor.
type Model struct {
	blocks    []block.Block    // the data model
	textareas []textarea.Model // one textarea per block
	active    int              // index of focused block
	viewport  viewport.Model   // scrollable container

	config      Config
	initial     string
	width       int
	height      int
	status      string
	statusStyle statusKind
	quitPrompt  bool
	quitting    bool
	showHelp    bool
	blockClip   *block.Block // block-level clipboard for Ctrl+K block cut
	statusGen   int    // generation counter for status auto-dismiss
	palette     palette // "/" command palette for block type insertion
	wordWrap         bool            // when true, text wraps at terminal width
	viewMode         bool            // when true, read-only rendering with no cursor
	hoverBlock       int             // view mode: block index under mouse cursor (-1 = none)
	cursorCmd        tea.Cmd         // pending cursor blink command from Focus()
	blockLineOffsets []int           // view mode: starting Y line of each block in rendered output
	blockLineCounts  []int           // per-block rendered line counts from renderAllBlocks
	dismissedHints   map[string]bool // tracks which onboarding hints the user has dismissed
	undo      undoStack // undo history
	redo      undoStack // redo history
	undoDirty bool      // true when textarea content changed since last snapshot
}

type statusKind int

const (
	statusNone statusKind = iota
	statusSuccess
	statusError
	statusWarning
)

// defaultWidth and defaultHeight are sensible initial dimensions so the
// cursor is visible even before the first WindowSizeMsg arrives.
const (
	defaultWidth  = 80
	defaultHeight = 24
)

// noWrapWidth is a very large width used when word-wrap is disabled so that
// text never wraps within the textarea.
const noWrapWidth = 1000

// gutterWidth is the fixed width of the left gutter that shows block type
// labels. The format is "h1 │ " = label(2) + space + sep(1) + space = 5.
//
// blockPrefixWidth returns the additional visual width consumed by the
// inline prefix rendered before the textarea for certain block types
// (e.g. bullet markers, numbered list prefixes, checkbox markers).
const gutterWidth = 5

func blockPrefixWidth(bt block.BlockType) int {
	th := theme.Current()
	switch bt {
	case block.BulletList:
		return lipgloss.Width(th.Blocks.Bullet.Marker)
	case block.NumberedList:
		return lipgloss.Width(fmt.Sprintf(th.Blocks.Numbered.Format, 1))
	case block.Checklist:
		return lipgloss.Width(th.Blocks.Checklist.Unchecked)
	case block.Quote:
		return lipgloss.Width(th.Blocks.Quote.Bar)
	case block.CodeBlock:
		return 4
	default:
		return 0
	}
}

// contentWidth returns the effective textarea width for a block, accounting
// for the gutter, block prefix, and word-wrap setting.
func (m Model) contentWidth(bt block.BlockType) int {
	mw := m.width
	if mw <= 0 {
		mw = defaultWidth
	}
	w := mw - gutterWidth - blockPrefixWidth(bt)
	if w < 1 {
		w = 1
	}
	if !m.wordWrap {
		w = noWrapWidth
	}
	return w
}

// New creates a new editor Model from the given config.
func New(cfg Config) Model {
	blocks := block.Parse(cfg.Content)
	textareas := make([]textarea.Model, len(blocks))

	for i, b := range blocks {
		textareas[i] = newTextareaForBlock(b, defaultWidth)
	}

	// Focus the first block.
	if len(textareas) > 0 {
		textareas[0].Focus()
	}

	vp := viewport.New(viewport.WithWidth(defaultWidth), viewport.WithHeight(defaultHeight-2)) // -1 header, -1 status bar

	dismissed := cfg.DismissedHints
	if dismissed == nil {
		dismissed = make(map[string]bool)
	}

	return Model{
		blocks:         blocks,
		textareas:      textareas,
		active:         0,
		viewport:       vp,
		config:         cfg,
		initial:        cfg.Content,
		width:          defaultWidth,
		height:         defaultHeight,
		palette:        newPalette(),
		wordWrap:       true,
		dismissedHints: dismissed,
		hoverBlock:     -1,
	}
}

// newTextareaForBlock creates a textarea configured for the given block type.
func newTextareaForBlock(b block.Block, width int) textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	// Set width BEFORE value so the textarea's internal state (viewport
	// scroll, wrap cache) is initialized with the correct dimensions.
	taWidth := width - gutterWidth - blockPrefixWidth(b.Type)
	if taWidth < 1 {
		taWidth = 1
	}
	ta.SetWidth(taWidth)
	ta.SetValue(b.Content)
	ta.MoveToBegin()

	// Set height from the textarea's own wrapping — it knows best.
	ta.SetHeight(ta.VisualLineCount())

	ta.Blur()
	return ta
}

// Init returns the initial command for the editor (start cursor blinking).
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// modified returns true if the content has been changed from the initial value.
func (m Model) modified() bool {
	return m.Content() != m.initial
}

// Content returns the current text content by syncing block data from
// textareas and serializing back to markdown.
func (m Model) Content() string {
	synced := m.syncBlocks()
	return block.Serialize(synced)
}

// syncBlocks copies current textarea values back into block data and returns
// the updated slice. It does not mutate the receiver.
func (m Model) syncBlocks() []block.Block {
	result := make([]block.Block, len(m.blocks))
	copy(result, m.blocks)
	for i := range result {
		if i < len(m.textareas) {
			result[i].Content = m.textareas[i].Value()
		}
	}
	return result
}

// BlockCount returns the number of blocks in the editor.
func (m Model) BlockCount() int {
	return len(m.blocks)
}

// statusBarHeight returns the number of terminal lines the rendered status
// bar will occupy. When the bar text is wider than the terminal, lipgloss
// wraps it onto multiple lines; this helper accounts for that.
func (m Model) statusBarHeight() int {
	rendered := m.renderStatusBar()
	return strings.Count(rendered, "\n") + 1
}

// headerHeight returns the number of terminal lines the header bar occupies.
func (m Model) headerHeight() int {
	if m.viewMode {
		return 0
	}
	return 1
}

// viewMaxWidth is the maximum content width in view mode for centered reading.
const viewMaxWidth = 72

// resizeTextareas updates all textarea widths for the current layout.
func (m *Model) resizeTextareas() {
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	for i, b := range m.blocks {
		m.textareas[i].SetWidth(m.contentWidth(b.Type))
		m.textareas[i].SetHeight(m.textareas[i].VisualLineCount())
	}
	// Update viewport dimensions, reserving space for the header and status bar
	// (which may wrap to multiple lines on narrow terminals).
	h := m.height - m.headerHeight() - m.statusBarHeight()
	if h < 1 {
		h = 1
	}
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

// scheduleStatusDismiss increments the generation counter and returns a tick
// command that will clear the status after statusTimeout.
func (m *Model) scheduleStatusDismiss() tea.Cmd {
	m.statusGen++
	gen := m.statusGen
	return tea.Tick(statusTimeout, func(t time.Time) tea.Msg {
		return statusTimeoutMsg{generation: gen}
	})
}

// focusBlock switches focus from the current block to the block at index idx.
// If undoDirty is true (content changed via typing since the last snapshot),
// a snapshot is pushed so that character-level edits are batched per focus change.
func (m *Model) focusBlock(idx int) {
	if idx < 0 || idx >= len(m.textareas) {
		return
	}
	if m.undoDirty {
		m.pushUndo()
	}
	if m.active >= 0 && m.active < len(m.textareas) {
		m.blocks[m.active].Content = m.textareas[m.active].Value()
		m.textareas[m.active].Blur()
	}
	m.active = idx
	m.cursorCmd = m.textareas[idx].Focus()
}

// navigateUp moves focus to the previous block, preserving horizontal position.
func (m *Model) navigateUp() {
	if m.active <= 0 {
		return
	}
	// Capture horizontal position before leaving.
	charOffset := m.textareas[m.active].LineInfo().CharOffset
	m.focusBlock(m.active - 1)
	// Move to the last visual line and restore horizontal offset.
	ta := &m.textareas[m.active]
	ta.MoveToEnd()
	li := ta.LineInfo()
	ta.SetCursorColumn(li.StartColumn + charOffset)
}

// navigateDown moves focus to the next block, preserving horizontal position.
func (m *Model) navigateDown() {
	if m.active >= len(m.textareas)-1 {
		return
	}
	// Capture horizontal position before leaving.
	charOffset := m.textareas[m.active].LineInfo().CharOffset
	m.focusBlock(m.active + 1)
	// Move to the first visual line and restore horizontal offset.
	ta := &m.textareas[m.active]
	ta.MoveToBegin()
	ta.SetCursorColumn(charOffset)
}

// isMultiLine returns true if the block type allows multi-line content.
func isMultiLine(bt block.BlockType) bool {
	return bt == block.Paragraph || bt == block.CodeBlock || bt == block.Quote
}

// insertBlockBefore inserts a new block before the given index, creates a
// textarea for it, and focuses it. The previously-active block shifts down.
func (m *Model) insertBlockBefore(idx int, b block.Block) {
	if idx < 0 || idx >= len(m.blocks) {
		return
	}
	ta := newTextareaForBlock(b, m.width)

	newBlocks := make([]block.Block, 0, len(m.blocks)+1)
	newBlocks = append(newBlocks, m.blocks[:idx]...)
	newBlocks = append(newBlocks, b)
	newBlocks = append(newBlocks, m.blocks[idx:]...)
	m.blocks = newBlocks

	newTAs := make([]textarea.Model, 0, len(m.textareas)+1)
	newTAs = append(newTAs, m.textareas[:idx]...)
	newTAs = append(newTAs, ta)
	newTAs = append(newTAs, m.textareas[idx:]...)
	m.textareas = newTAs

	m.focusBlock(idx)
}

// insertBlockAfter inserts a new block after the given index, creates a
// textarea for it, and focuses the new block.
func (m *Model) insertBlockAfter(idx int, b block.Block) {
	if idx < 0 || idx >= len(m.blocks) {
		return
	}
	ta := newTextareaForBlock(b, m.width)

	newBlocks := make([]block.Block, 0, len(m.blocks)+1)
	newBlocks = append(newBlocks, m.blocks[:idx+1]...)
	newBlocks = append(newBlocks, b)
	newBlocks = append(newBlocks, m.blocks[idx+1:]...)
	m.blocks = newBlocks

	newTAs := make([]textarea.Model, 0, len(m.textareas)+1)
	newTAs = append(newTAs, m.textareas[:idx+1]...)
	newTAs = append(newTAs, ta)
	newTAs = append(newTAs, m.textareas[idx+1:]...)
	m.textareas = newTAs

	m.focusBlock(idx + 1)
}

// deleteBlock removes the block at the given index. If it is the last block,
// it converts it to an empty paragraph instead of deleting.
func (m *Model) deleteBlock(idx int) {
	if len(m.blocks) <= 1 {
		// Convert last block to empty paragraph.
		m.blocks[0] = block.Block{Type: block.Paragraph, Content: ""}
		m.textareas[0] = newTextareaForBlock(m.blocks[0], m.width)
		m.active = 0
		m.cursorCmd = m.textareas[0].Focus()
		return
	}

	m.blocks = append(m.blocks[:idx], m.blocks[idx+1:]...)
	m.textareas = append(m.textareas[:idx], m.textareas[idx+1:]...)

	// Adjust active index.
	if idx >= len(m.blocks) {
		m.active = len(m.blocks) - 1
	} else if idx > 0 {
		m.active = idx - 1
	} else {
		m.active = 0
	}

	m.cursorCmd = m.textareas[m.active].Focus()
}

// mergeBlockUp merges block at idx into block at idx-1. The merged block
// keeps the type of the previous block. Cursor is placed at the merge point.
func (m *Model) mergeBlockUp(idx int) {
	if idx <= 0 || idx >= len(m.blocks) {
		return
	}

	currentContent := m.textareas[idx].Value()
	prevContent := m.textareas[idx-1].Value()
	// Merge content: concatenate directly (no added newline), matching
	// Notion/Google Docs behavior where backspace joins text on the same line.
	merged := prevContent + currentContent

	m.blocks[idx-1].Content = merged
	m.textareas[idx-1].SetValue(merged)

	// If the target block was an empty paragraph, adopt the source block's
	// type so that merging a heading into an empty paragraph preserves the
	// heading type. Non-paragraph targets (e.g. list items) keep their type.
	if prevContent == "" && m.blocks[idx-1].Type == block.Paragraph {
		m.blocks[idx-1].Type = m.blocks[idx].Type
		m.blocks[idx-1].Checked = m.blocks[idx].Checked
	}

	// Reconfigure textarea for the (possibly changed) block type.
	m.textareas[idx-1].SetWidth(m.contentWidth(m.blocks[idx-1].Type))
	m.textareas[idx-1].SetHeight(m.textareas[idx-1].VisualLineCount())

	m.blocks = append(m.blocks[:idx], m.blocks[idx+1:]...)
	m.textareas = append(m.textareas[:idx], m.textareas[idx+1:]...)

	// Focus previous block and position cursor at the merge point.
	m.active = idx - 1
	m.cursorCmd = m.textareas[m.active].Focus()

	// Navigate to the merge point: SetValue leaves the cursor at the end
	// of the content. Walk up from the end to reach the target row (last
	// line of prevContent), then set the column within that row.
	mergedLines := strings.Split(merged, "\n")
	prevLines := strings.Split(prevContent, "\n")
	mergeRow := len(prevLines) - 1
	rowsFromEnd := len(mergedLines) - 1 - mergeRow
	for i := 0; i < rowsFromEnd; i++ {
		m.textareas[m.active].CursorUp()
	}
	m.textareas[m.active].SetCursorColumn(len([]rune(prevLines[mergeRow])))
}

// swapBlocks swaps the block at idx with the block at idx+delta (delta is
// -1 for up, +1 for down). No-op at boundaries.
func (m *Model) swapBlocks(delta int) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}
	target := m.active + delta
	if target < 0 || target >= len(m.blocks) {
		return
	}

	// Sync current textarea content before swap.
	m.blocks[m.active].Content = m.textareas[m.active].Value()
	m.blocks[target].Content = m.textareas[target].Value()

	m.blocks[m.active], m.blocks[target] = m.blocks[target], m.blocks[m.active]
	m.textareas[m.active], m.textareas[target] = m.textareas[target], m.textareas[m.active]

	m.textareas[m.active].Blur()
	m.active = target
	m.cursorCmd = m.textareas[m.active].Focus()
}

// handleEnter processes the Enter key for block splitting/creation.
func (m *Model) handleEnter() {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}

	bt := m.blocks[m.active].Type
	ta := &m.textareas[m.active]
	content := ta.Value()

	// Divider: selected as a unit, Enter inserts paragraph below.
	if bt == block.Divider {
		m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
		return
	}

	// Code block / Quote: Enter inserts a newline within the block.
	// On an empty last line, exit the block by trimming the empty line
	// and inserting a new paragraph below.
	if bt == block.CodeBlock || bt == block.Quote {
		lines := strings.Split(content, "\n")
		cursorLine := ta.Line()
		isLastLine := cursorLine >= len(lines)-1
		currentLineEmpty := cursorLine < len(lines) && lines[cursorLine] == ""

		if isLastLine && currentLineEmpty && len(lines) > 1 {
			// Remove trailing empty line and insert paragraph.
			trimmed := strings.Join(lines[:len(lines)-1], "\n")
			ta.SetValue(trimmed)
			m.blocks[m.active].Content = trimmed
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
			m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
			return
		}

		// Otherwise insert a newline within the block.
		m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount() + 1)
		m.textareas[m.active], _ = m.textareas[m.active].Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
		return
	}

	// Empty list item: exit list by converting to paragraph.
	if content == "" && (bt == block.BulletList || bt == block.NumberedList || bt == block.Checklist) {
		m.blocks[m.active].Type = block.Paragraph
		m.blocks[m.active].Checked = false
		newTA := newTextareaForBlock(m.blocks[m.active], m.width)
		m.cursorCmd = newTA.Focus()
		m.textareas[m.active] = newTA
		return
	}

	// Split content at cursor position.
	var before, after string
	if isMultiLine(bt) {
		cursorLine := ta.Line()
		info := ta.LineInfo()
		charOffset := info.StartColumn + info.ColumnOffset
		lines := strings.Split(content, "\n")
		if cursorLine >= len(lines) {
			cursorLine = len(lines) - 1
		}
		lineRunes := []rune(lines[cursorLine])
		if charOffset > len(lineRunes) {
			charOffset = len(lineRunes)
		}

		// Before: all lines up to cursor line + text before cursor on cursor line.
		// After: text after cursor on cursor line + all remaining lines.
		if charOffset == 0 {
			// Cursor at start of a line: everything before this line is "before",
			// this line and everything after is "after".
			before = strings.Join(lines[:cursorLine], "\n")
			after = strings.Join(lines[cursorLine:], "\n")
		} else {
			beforeLines := append([]string{}, lines[:cursorLine]...)
			beforeLines = append(beforeLines, string(lineRunes[:charOffset]))
			before = strings.Join(beforeLines, "\n")

			afterLines := []string{string(lineRunes[charOffset:])}
			if cursorLine+1 < len(lines) {
				afterLines = append(afterLines, lines[cursorLine+1:]...)
			}
			after = strings.Join(afterLines, "\n")
		}
	} else {
		info := ta.LineInfo()
		charOffset := info.StartColumn + info.ColumnOffset
		contentRunes := []rune(content)
		if charOffset > len(contentRunes) {
			charOffset = len(contentRunes)
		}
		before = string(contentRunes[:charOffset])
		after = string(contentRunes[charOffset:])
	}

	// Determine new block type.
	var newType block.BlockType
	if before == "" {
		// Cursor at beginning: content moves to new block, keeping original type.
		newType = bt
	} else {
		// Cursor at middle/end: new block gets continuation type.
		switch bt {
		case block.BulletList:
			newType = block.BulletList
		case block.NumberedList:
			newType = block.NumberedList
		case block.Checklist:
			newType = block.Checklist
		default:
			newType = block.Paragraph
		}
	}

	// Update current block with text before cursor.
	m.blocks[m.active].Content = before
	ta.SetValue(before)

	// If current block is now empty and was a heading, convert to paragraph.
	if before == "" && (bt == block.Heading1 || bt == block.Heading2 || bt == block.Heading3) {
		m.blocks[m.active].Type = block.Paragraph
	}

	// Reconfigure current block textarea height.
	m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())

	// Create new block with text after cursor.
	newBlock := block.Block{Type: newType, Content: after}
	if newType == block.Checklist {
		newBlock.Checked = false
	}
	m.insertBlockAfter(m.active, newBlock)
	// Place cursor at the start of the new block (not at the end, which is
	// where SetValue leaves it).
	m.textareas[m.active].MoveToBegin()
}

// handleBackspace processes Backspace at position 0 for block deletion/merging.
// Returns true if the backspace was handled (caller should not forward to textarea).
func (m *Model) handleBackspace() bool {
	if m.active < 0 || m.active >= len(m.blocks) {
		return false
	}

	ta := &m.textareas[m.active]

	// Check if cursor is at position 0 (line 0, column 0).
	if ta.Line() != 0 {
		return false
	}
	info := ta.LineInfo()
	if info.StartColumn+info.ColumnOffset != 0 {
		return false
	}

	content := ta.Value()
	bt := m.blocks[m.active].Type

	// Divider: backspace always deletes the divider (selected as a unit).
	if bt == block.Divider {
		m.pushUndo()
		m.deleteBlock(m.active)
		m.textareas[m.active].MoveToEnd()
		return true
	}

	if content == "" {
		// Empty block: delete it, focus previous.
		if m.active == 0 {
			if len(m.blocks) <= 1 {
				if m.blocks[0].Type == block.Paragraph {
					return true // Already empty paragraph, nothing to do.
				}
			}
		}
		m.pushUndo()
		m.deleteBlock(m.active)
		m.textareas[m.active].MoveToEnd()
		return true
	}

	// List item at position 0: convert to paragraph, keeping text.
	if bt == block.BulletList || bt == block.NumberedList || bt == block.Checklist {
		m.pushUndo()
		m.blocks[m.active].Type = block.Paragraph
		m.blocks[m.active].Checked = false
		newTA := newTextareaForBlock(m.blocks[m.active], m.width)
		newTA.SetValue(content)
		m.cursorCmd = newTA.Focus()
		newTA.SetCursorColumn(0)
		m.textareas[m.active] = newTA
		return true
	}

	// Non-empty block at position 0: merge with previous block.
	if m.active == 0 {
		return false // No previous block to merge into.
	}

	// If previous block is a divider, just delete the divider.
	if m.blocks[m.active-1].Type == block.Divider {
		m.pushUndo()
		m.deleteBlock(m.active - 1)
		return true
	}

	// Don't merge content into code blocks or quote blocks.
	if m.blocks[m.active-1].Type == block.CodeBlock || m.blocks[m.active-1].Type == block.Quote {
		return false
	}

	m.pushUndo()
	m.mergeBlockUp(m.active)
	return true
}

// cutBlock removes the active block and stores it in the block clipboard.
func (m *Model) cutBlock() {
	if len(m.blocks) <= 1 {
		// Don't remove the last block; store it and clear it instead.
		b := m.blocks[m.active]
		b.Content = m.textareas[m.active].Value()
		m.blockClip = &b
		m.blocks[m.active] = block.Block{Type: block.Paragraph, Content: ""}
		m.textareas[m.active].SetValue("")
		return
	}

	b := m.blocks[m.active]
	b.Content = m.textareas[m.active].Value()
	m.blockClip = &b

	idx := m.active
	m.blocks = append(m.blocks[:idx], m.blocks[idx+1:]...)
	m.textareas = append(m.textareas[:idx], m.textareas[idx+1:]...)

	// Adjust active index.
	if m.active >= len(m.blocks) {
		m.active = len(m.blocks) - 1
	}
	if m.active < 0 {
		m.active = 0
	}

	m.cursorCmd = m.textareas[m.active].Focus()
}

// applyPaletteSelection changes the active block's type to the selected
// palette item type. Special handling is applied for dividers and code blocks.
func (m *Model) applyPaletteSelection(bt block.BlockType) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}
	m.blocks[m.active].Type = bt
	m.textareas[m.active].SetWidth(m.contentWidth(bt))

	if bt == block.Divider {
		m.blocks[m.active].Content = ""
		m.textareas[m.active].SetValue("")
	}
	m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
}

// isAtFirstLine returns true if the cursor is on the first visual line
// of the active textarea, accounting for soft-wrapped lines.
func (m Model) isAtFirstLine() bool {
	if m.active < 0 || m.active >= len(m.textareas) {
		return false
	}
	ta := m.textareas[m.active]
	// First raw line AND first wrapped sub-line within it.
	return ta.Line() == 0 && ta.LineInfo().RowOffset == 0
}

// isAtLastLine returns true if the cursor is on the last visual line
// of the active textarea, accounting for soft-wrapped lines.
func (m Model) isAtLastLine() bool {
	if m.active < 0 || m.active >= len(m.textareas) {
		return false
	}
	ta := m.textareas[m.active]
	value := ta.Value()
	rawLineCount := strings.Count(value, "\n") + 1
	li := ta.LineInfo()
	// Must be on the last raw line AND the last wrapped sub-line within it.
	return ta.Line() >= rawLineCount-1 && li.RowOffset >= li.Height-1
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.update(msg)
	// Drain any pending cursor blink command (set by focusBlock and friends).
	if em, ok := model.(Model); ok && em.cursorCmd != nil {
		blinkCmd := em.cursorCmd
		em.cursorCmd = nil
		return em, tea.Batch(cmd, blinkCmd)
	}
	return model, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTextareas()
		m.updateViewport()
		return m, nil

	case statusTimeoutMsg:
		if msg.generation == m.statusGen {
			m.status = ""
			m.statusStyle = statusNone
		}
		return m, nil

	case tea.MouseMotionMsg:
		if m.viewMode {
			// Track hover state for checklist visual feedback.
			hoverY := m.viewport.YOffset() + msg.Y
			idx := m.blockIndexAtLine(hoverY)
			oldHover := m.hoverBlock
			m.hoverBlock = idx

			// Re-render if hovering over a different checklist block.
			if idx != oldHover {
				isChecklistHover := idx >= 0 && idx < len(m.blocks) && m.blocks[idx].Type == block.Checklist
				wasChecklistHover := oldHover >= 0 && oldHover < len(m.blocks) && m.blocks[oldHover].Type == block.Checklist
				if isChecklistHover || wasChecklistHover {
					m.updateViewport()
				}
			}
		} else {
			m.hoverBlock = -1
		}
		return m, nil

	case tea.MouseClickMsg:
		if m.viewMode {
			// Track hover state for checklist visual feedback.
			hoverY := m.viewport.YOffset() + msg.Y
			idx := m.blockIndexAtLine(hoverY)
			m.hoverBlock = idx

			// Left-click on a checklist block toggles its checked state.
			if msg.Button == tea.MouseLeft {
				if idx >= 0 && idx < len(m.blocks) && m.blocks[idx].Type == block.Checklist {
					m.pushUndo()
					m.blocks[idx].Checked = !m.blocks[idx].Checked
					if idx < len(m.textareas) {
						m.blocks[idx].Content = m.textareas[idx].Value()
					}
					m.updateViewport()
					return m, nil
				}
			}
		} else {
			m.hoverBlock = -1
		}
		return m, nil

	case tea.MouseWheelMsg:
		// Let wheel scroll pass through to the viewport.
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// Any keypress clears the manual scroll flag so auto-scroll
		// to cursor resumes.
		m.viewport.UserScrolled = false

		// When help overlay is showing, Ctrl+G/Esc close it, Ctrl+C quits.
		if m.showHelp {
			switch msg.String() {
			case "ctrl+g", "esc":
				m.showHelp = false
			case "ctrl+c":
				m.showHelp = false
				if m.modified() {
					m.quitPrompt = true
					m.status = "Save before quitting? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// When quit prompt is showing, handle the 3-option response.
		if m.quitPrompt {
			switch msg.String() {
			case "y", "Y", "enter":
				m.quitPrompt = false
				if m.config.Save != nil {
					content := m.Content()
					return m, func() tea.Msg {
						if err := m.config.Save(content); err != nil {
							return saveErrMsg{err: err}
						}
						return savedQuitMsg{content: content}
					}
				}
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.quitPrompt = false
				m.status = ""
				m.statusStyle = statusNone
				return m, nil
			}
			return m, nil
		}

		// When palette is visible, intercept all keys.
		if m.palette.visible {
			switch msg.String() {
			case "up":
				m.palette.moveUp()
				m.updateViewport()
				return m, nil
			case "down":
				m.palette.moveDown()
				m.updateViewport()
				return m, nil
			case "enter":
				if sel := m.palette.selected(); sel != nil {
					m.pushUndo()
					m.applyPaletteSelection(sel.Type)
				}
				m.palette.close()
				m.updateViewport()
				return m, nil
			case "esc":
				m.palette.close()
				m.updateViewport()
				return m, nil
			case "backspace":
				if !m.palette.deleteFilterRune() {
					m.palette.close()
				}
				m.updateViewport()
				return m, nil
			default:
				if msg.Code == '/' {
					m.palette.close()
					m.updateViewport()
					return m, nil
				}
				if len(msg.Text) > 0 {
					for _, r := range msg.Text {
						m.palette.addFilterRune(r)
					}
					m.updateViewport()
					return m, nil
				}
			}
			// Ignore other keys while palette is open.
			return m, nil
		}

		// Clear transient status on any keypress.
		if m.statusStyle == statusSuccess {
			m.status = ""
			m.statusStyle = statusNone
		}

		// View mode: intercept keys early.
		if m.viewMode {
			switch msg.String() {
			case "ctrl+r":
				m.viewMode = false
				m.hoverBlock = -1
				// Re-focus the previously active block to restore cursor.
				if m.active >= 0 && m.active < len(m.textareas) {
					m.cursorCmd = m.textareas[m.active].Focus()
				}
				m.updateViewport()
				return m, nil
			case "esc", "ctrl+c":
				if m.modified() {
					m.quitPrompt = true
					m.status = "Save before quitting? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit
			case "ctrl+s":
				if m.config.Save != nil {
					content := m.Content()
					return m, func() tea.Msg {
						if err := m.config.Save(content); err != nil {
							return saveErrMsg{err: err}
						}
						return savedMsg{content: content}
					}
				}
				return m, nil
			case "h":
				if !m.dismissedHints["editor.checkbox"] {
					m.dismissedHints["editor.checkbox"] = true
					config.DismissHint("editor.checkbox")
				}
				return m, nil
			case "ctrl+g":
				m.showHelp = true
				return m, nil
			case "up":
				m.viewport.ScrollUp(1)
				return m, nil
			case "pgup":
				m.viewport.HalfPageUp()
				return m, nil
			case "down":
				m.viewport.ScrollDown(1)
				return m, nil
			case "pgdown":
				m.viewport.HalfPageDown()
				return m, nil
			}
			// Swallow everything else in view mode.
			return m, nil
		}

		switch msg.String() {
		case "ctrl+r":
			m.viewMode = true
			m.hoverBlock = -1
			// Blur the active textarea but preserve m.active.
			if m.active >= 0 && m.active < len(m.textareas) {
				m.blocks[m.active].Content = m.textareas[m.active].Value()
				m.textareas[m.active].Blur()
			}
			m.updateViewport()
			return m, nil

		case "ctrl+g":
			m.showHelp = true
			return m, nil

		case "ctrl+z":
			if m.performUndo() {
				m.updateViewport()
			} else {
				m.status = "Nothing to undo"
				m.statusStyle = statusWarning
			}
			return m, m.scheduleStatusDismiss()

		case "ctrl+y":
			if m.performRedo() {
				m.updateViewport()
			} else {
				m.status = "Nothing to redo"
				m.statusStyle = statusWarning
			}
			return m, m.scheduleStatusDismiss()

		case "ctrl+k":
			m.pushUndo()
			m.cutBlock()
			m.updateViewport()
			return m, nil

		case "ctrl+s":
			if m.config.Save != nil {
				content := m.Content()
				return m, func() tea.Msg {
					if err := m.config.Save(content); err != nil {
						return saveErrMsg{err: err}
					}
					return savedMsg{content: content}
				}
			}
			return m, nil

		case "esc", "ctrl+c":
			if m.modified() {
				m.quitPrompt = true
				m.status = "Save before quitting? [Y/n/Esc]"
				m.statusStyle = statusWarning
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "up":
			if m.isAtFirstLine() && m.active == 0 {
				bt := m.blocks[0].Type
				if bt == block.CodeBlock || bt == block.Quote || bt == block.Divider {
					// These types don't split on Enter, so there's no other
					// way to insert content above them. Create a paragraph.
					m.pushUndo()
					m.insertBlockBefore(0, block.Block{Type: block.Paragraph})
					m.updateViewport()
					return m, nil
				}
			}
			if m.isAtFirstLine() && m.active > 0 {
				m.navigateUp()
				m.updateViewport()
				return m, nil
			}

		case "down":
			if m.isAtLastLine() && m.active < len(m.textareas)-1 {
				m.navigateDown()
				m.updateViewport()
				return m, nil
			}

		case "alt+up":
			m.pushUndo()
			m.swapBlocks(-1)
			m.updateViewport()
			return m, nil

		case "alt+down":
			m.pushUndo()
			m.swapBlocks(1)
			m.updateViewport()
			return m, nil

		case "ctrl+w", "alt+z":
			m.wordWrap = !m.wordWrap
			m.resizeTextareas()
			m.updateViewport()
			return m, nil

		case "ctrl+u":
			ta := &m.textareas[m.active]
			info := ta.LineInfo()
			logicalCol := info.StartColumn + info.ColumnOffset
			if logicalCol > 0 {
				// Let the textarea's built-in DeleteBeforeCursor handle it.
				// It operates on internal state without resetting cursor position.
				m.textareas[m.active], _ = m.textareas[m.active].Update(msg)
				m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
				m.updateViewport()
				return m, nil
			}
			// Cursor at column 0: merge/delete block or join lines.
			if m.handleBackspace() {
				m.updateViewport()
				return m, nil
			}
			m.textareas[m.active], _ = m.textareas[m.active].Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
			m.updateViewport()
			return m, nil

		case "ctrl+x":
			// Toggle checklist checked state.
			if m.active >= 0 && m.active < len(m.blocks) && m.blocks[m.active].Type == block.Checklist {
				m.pushUndo()
				m.blocks[m.active].Checked = !m.blocks[m.active].Checked
				m.updateViewport()
				return m, nil
			}

		case "ctrl+j", "shift+enter":
			// Ctrl+J / Shift+Enter inserts a newline within the current block,
			// bypassing the Enter handler that creates new blocks.
			// Only for multi-line blocks (paragraph, code, quote).
			// Headings and list items are single-line by definition.
			if m.active >= 0 && m.active < len(m.blocks) && isMultiLine(m.blocks[m.active].Type) {
				// Pre-grow textarea so the internal viewport doesn't clip
				// the first line when the newline is inserted.
				m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount() + 1)
				var cmd tea.Cmd
				m.textareas[m.active], cmd = m.textareas[m.active].Update(tea.KeyPressMsg{Code: tea.KeyEnter})
				m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
				m.updateViewport()
				return m, cmd
			}
			return m, nil
		}

	case savedMsg:
		m.initial = msg.content
		m.status = "Saved"
		m.statusStyle = statusSuccess
		return m, m.scheduleStatusDismiss()

	case savedQuitMsg:
		m.initial = msg.content
		m.quitting = true
		return m, tea.Quit

	case saveErrMsg:
		m.status = fmt.Sprintf("Save failed: %s", msg.err)
		m.statusStyle = statusError
		return m, m.scheduleStatusDismiss()
	}

	// Forward remaining messages to the active textarea.
	if m.active >= 0 && m.active < len(m.textareas) {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			// Handle "/" at position 0 to open the command palette.
			if keyMsg.Code == '/' && len(keyMsg.Text) > 0 {
				ta := &m.textareas[m.active]
				if ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0 && ta.Value() == "" {
					m.palette.open(m.active)
					m.updateViewport()
					return m, nil
				}
			}

			// Handle Enter key: always split/create via handleEnter.
			if keyMsg.Code == tea.KeyEnter {
				m.pushUndo()
				m.handleEnter()
				m.updateViewport()
				return m, nil
			}

			// Handle Backspace at position 0 for delete/merge.
			if keyMsg.Code == tea.KeyBackspace {
				if m.handleBackspace() {
					m.updateViewport()
					return m, nil
				}
			}

			// Divider: selected as a unit — no text input forwarded.
			// Enter and Backspace are handled above; everything else is ignored.
			if m.blocks[m.active].Type == block.Divider {
				return m, nil
			}

			// Tab inserts 4 spaces.
			if keyMsg.Code == tea.KeyTab {
				ta := &m.textareas[m.active]
				for _, r := range "    " {
					*ta, _ = ta.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
				}
				m.updateViewport()
				return m, nil
			}
		}

		// Capture state before the textarea processes the keystroke so we
		// can detect whether content actually changed (vs. cursor movement).
		var preState editorState
		if !m.undoDirty {
			preState = m.captureState()
		}

		var cmd tea.Cmd
		m.textareas[m.active], cmd = m.textareas[m.active].Update(msg)

		// Only push an undo entry when content actually changed, not on
		// cursor-only movements (left, right, home, end, etc.).
		if !m.undoDirty && m.textareas[m.active].Value() != preState.blocks[preState.active].Content {
			m.undo.push(preState)
			m.redo.clear()
			m.undoDirty = true
		}

		// Re-enforce width and recalculate height after every keystroke.
		// This ensures the textarea re-wraps correctly after content changes
		// (e.g. deleting a newline inserted via Ctrl+J).
		m.textareas[m.active].SetWidth(m.contentWidth(m.blocks[m.active].Type))
		m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())

		m.updateViewport()
		return m, cmd
	}

	return m, nil
}

// updateViewport renders all blocks and sets the viewport content, then
// auto-scrolls to keep the active block's cursor line visible.
func (m *Model) updateViewport() {
	// Recalculate viewport height to account for the header and status bar
	// wrapping which may change as content is modified (e.g. "[modified]"
	// indicator).
	h := m.height - m.headerHeight() - m.statusBarHeight()
	if h < 1 {
		h = 1
	}
	m.viewport.SetHeight(h)

	content := m.renderAllBlocks()
	m.viewport.SetContent(content)

	// In view mode, compute block line offsets for mouse click handling
	// and skip auto-scroll to cursor — the user scrolls manually.
	if m.viewMode {
		m.computeBlockLineOffsets()
		return
	}

	// Skip auto-scroll if the user manually scrolled with the mouse.
	// The flag is cleared on the next cursor movement (see below).
	if m.viewport.UserScrolled {
		return
	}

	// Auto-scroll: calculate the cursor's actual line in the rendered output
	// and ensure the viewport shows it. We compute the offset from scratch
	// because SetContent may reset the viewport's internal scroll state.
	if m.active >= 0 && m.active < len(m.blocks) && m.active < len(m.textareas) {
		// Count rendered lines for all blocks before the active one,
		// using cached line counts from renderAllBlocks to avoid re-rendering.
		lineOffset := 0
		for i := 0; i < m.active; i++ {
			if i < len(m.blockLineCounts) {
				lineOffset += m.blockLineCounts[i]
			}
		}

		// Account for chrome lines the active block's renderActiveBlock
		// prepends before the textarea content.
		chromeLines := 0
		if m.active > 0 && m.blocks[m.active].Type == block.Heading1 {
			chromeLines = 1 // "\n" prefix for non-first H1
		}
		if m.blocks[m.active].Type == block.CodeBlock {
			chromeLines = 1 // top border
		}

		ta := m.textareas[m.active]

		// Count visual cursor line, accounting for word wrapping.
		// ta.Line() is the raw line number, but wrapped lines before the
		// cursor occupy more visual lines than raw lines.
		cursorRawLine := ta.Line()
		visualLine := cursorRawLine
		if m.wordWrap {
			contentWidth := m.width - gutterWidth - blockPrefixWidth(m.blocks[m.active].Type)
			if contentWidth < 1 {
				contentWidth = 1
			}
			rawLines := strings.Split(ta.Value(), "\n")
			visualLine = 0
			for i := 0; i < cursorRawLine && i < len(rawLines); i++ {
				segs := textarea.Wrap([]rune(rawLines[i]), contentWidth)
				if len(segs) == 0 {
					visualLine++
				} else {
					visualLine += len(segs)
				}
			}
			visualLine += ta.LineInfo().RowOffset
		}

		// Code blocks always strip the title line (raw line 0) from
		// rendered content into the border. Adjust visual line count.
		if m.blocks[m.active].Type == block.CodeBlock && cursorRawLine > 0 {
			visualLine--
		}

		cursorLine := lineOffset + chromeLines + visualLine

		// Always ensure the cursor line is visible. Prefer keeping the
		// current scroll position when the cursor is already on screen.
		// When cursor is on the first content line, also show the block's
		// chrome (borders, labels) above it.
		yOffset := m.viewport.YOffset()
		scrollTarget := cursorLine
		if cursorRawLine == 0 && chromeLines > 0 {
			scrollTarget = lineOffset // show from block start
		}
		if scrollTarget < yOffset {
			yOffset = scrollTarget
		}
		if cursorLine >= yOffset+m.viewport.Height() {
			yOffset = cursorLine - m.viewport.Height() + 1
		}
		if yOffset < 0 {
			yOffset = 0
		}

		m.viewport.SetYOffset(yOffset)
	}
}

// computeBlockLineOffsets mirrors the spacing logic of renderViewContent to
// determine the starting line (Y coordinate) of each block in the rendered
// viewport content. This is used for mouse click → block mapping in view mode.
//
// It builds the same parts slice as renderViewContent and tracks the starting
// line of each block. When parts are joined with "\n", part[p] starts at line
// equal to the sum of (strings.Count(parts[j], "\n") + 1) for j in 0..p-1.
func (m *Model) computeBlockLineOffsets() {
	contentWidth := viewMaxWidth
	if m.width < contentWidth {
		contentWidth = m.width - 4
		if contentWidth < 20 {
			contentWidth = 20
		}
	}

	// nextLine tracks the line number where the next appended part will start.
	nextLine := 0

	// advance moves nextLine past a part that was just appended.
	advance := func(part string) {
		nextLine += strings.Count(part, "\n") + 1
	}

	// Top padding.
	advance("")

	// Title + blank line after.
	if m.config.Title != "" {
		advance("title") // single-line; actual content irrelevant
		advance("")
	}

	offsets := make([]int, len(m.blocks))
	for i, b := range m.blocks {
		content := b.Content
		if i == m.active && i < len(m.textareas) {
			content = m.textareas[i].Value()
		}

		// Spacing lines before this block (mirrors renderViewContent exactly).
		if i > 0 {
			prev := m.blocks[i-1]
			switch {
			case b.Type == block.Heading1:
				advance("") // 2 blank lines before H1
				advance("")
			case b.Type == block.Heading2 || b.Type == block.Heading3:
				advance("") // 1 blank line before H2/H3
			case b.Type == block.CodeBlock || b.Type == block.Quote:
				if prev.Type != b.Type {
					advance("")
				}
			case prev.Type == block.CodeBlock || prev.Type == block.Quote:
				advance("")
			case b.Type == block.Divider:
				advance("")
			case prev.Type == block.Divider:
				advance("")
			}
		}

		// Record where this block starts.
		offsets[i] = nextLine

		// Render the block and advance past its lines.
		rendered := renderViewBlock(b, content, contentWidth, m.wordWrap, m.blocks, i, false)
		advance(rendered)
	}

	m.blockLineOffsets = offsets
}

// blockIndexAtLine returns the block index that contains the given absolute
// line number in view-mode rendered content, or -1 if no block matches.
func (m Model) blockIndexAtLine(line int) int {
	if len(m.blockLineOffsets) == 0 {
		return -1
	}
	// Find the last block whose starting offset is <= line.
	result := -1
	for i, offset := range m.blockLineOffsets {
		if offset <= line {
			result = i
		} else {
			break
		}
	}
	return result
}

// renderAllBlocks renders each block and joins them vertically.
// When the command palette is visible, it is rendered below the active block.
// As a side effect, it populates m.blockLineCounts with the number of rendered
// lines for each block (including any palette lines appended after it).
func (m *Model) renderAllBlocks() string {
	if m.viewMode {
		return m.renderViewContent()
	}
	m.blockLineCounts = make([]int, len(m.blocks))
	var parts []string
	for i := range m.blocks {
		rendered := m.renderBlock(i)
		lineCount := strings.Count(rendered, "\n") + 1
		parts = append(parts, rendered)
		if m.palette.visible && i == m.active {
			if pv := m.palette.render(m.width); pv != "" {
				lineCount += strings.Count(pv, "\n") + 1
				parts = append(parts, pv)
			}
		}
		m.blockLineCounts[i] = lineCount
	}
	return strings.Join(parts, "\n")
}

// renderViewContent builds the full view-mode content: centered, max-width,
// with generous spacing for a clean reading experience.
func (m Model) renderViewContent() string {
	contentWidth := viewMaxWidth
	if m.width < contentWidth {
		contentWidth = m.width - 4 // leave some margin even on small terms
		if contentWidth < 20 {
			contentWidth = 20
		}
	}

	// Horizontal padding to center the content column.
	leftPad := (m.width - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	padStr := strings.Repeat(" ", leftPad)

	var parts []string

	// Top padding — breathing room before content starts.
	parts = append(parts, "")

	// Title — centered, bold, faint.
	if m.config.Title != "" {
		titleStyle := lipgloss.NewStyle().Bold(true)
		title := titleStyle.Render(m.config.Title)
		titlePad := (m.width - lipgloss.Width(title)) / 2
		if titlePad < 0 {
			titlePad = 0
		}
		parts = append(parts, strings.Repeat(" ", titlePad)+title)
		parts = append(parts, "")
	}

	for i, b := range m.blocks {
		content := b.Content
		if i == m.active && i < len(m.textareas) {
			content = m.textareas[i].Value()
		}

		hovered := i == m.hoverBlock
		rendered := renderViewBlock(b, content, contentWidth, m.wordWrap, m.blocks, i, hovered)

		// Add vertical spacing before certain block types.
		if i > 0 {
			prev := m.blocks[i-1]
			switch {
			case b.Type == block.Heading1:
				parts = append(parts, "", "") // extra blank before H1
			case b.Type == block.Heading2 || b.Type == block.Heading3:
				parts = append(parts, "") // blank before H2/H3
			case b.Type == block.CodeBlock || b.Type == block.Quote:
				if prev.Type != b.Type {
					parts = append(parts, "") // blank before code/quote blocks
				}
			case prev.Type == block.CodeBlock || prev.Type == block.Quote:
				parts = append(parts, "") // blank after code/quote blocks
			case b.Type == block.Divider:
				parts = append(parts, "") // blank before dividers
			case prev.Type == block.Divider:
				parts = append(parts, "") // blank after dividers
			}
		}

		// Pad each line of the rendered block to center it.
		lines := strings.Split(rendered, "\n")
		for j, l := range lines {
			lines[j] = padStr + l
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}

	// Bottom padding so content doesn't end right at the status bar.
	parts = append(parts, "", "")

	return strings.Join(parts, "\n")
}

// View renders the editor UI.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	var content string
	if m.showHelp {
		content = m.renderHelpOverlay()
	} else if m.viewMode {
		statusBar := m.renderStatusBar()
		content = m.viewport.View() + "\n" + statusBar
	} else {
		header := m.renderHeader()
		statusBar := m.renderStatusBar()
		content = header + "\n" + m.viewport.View() + "\n" + statusBar
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// renderHelpOverlay builds the full-screen help panel.
func (m Model) renderHelpOverlay() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent)).Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	sep := dim.Render("─────────────────────────")
	s := dim.Render("/") // dimmed slash separator

	var help strings.Builder
	help.WriteString("  " + accent.Render("Keybindings") + "\n")
	help.WriteString("  " + sep + "\n")
	help.WriteString("\n")
	help.WriteString("  Enter        New block\n")
	help.WriteString("  ⇧Enter       Newline\n")
	help.WriteString("  Backspace    Merge " + s + " delete\n")
	help.WriteString("  ⌃K           Cut block\n")
	help.WriteString("  ⌃Z" + s + "⌃Y        Undo " + s + " redo\n")
	help.WriteString("  ⌥↑" + s + "⌥↓        Move block\n")
	help.WriteString("  /            Block type\n")
	help.WriteString("\n")
	help.WriteString("  ⌃R           View mode\n")
	help.WriteString("  ⌃X           Checkbox\n")
	help.WriteString("  ⌃W           Word wrap\n")
	help.WriteString("\n")
	help.WriteString("  ⌃S           Save\n")
	help.WriteString("  Esc" + s + "⌃C       Quit")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Current().Border)).
		Padding(1, 2).
		Width(36).
		Align(lipgloss.Left)

	rendered := box.Render(help.String())

	statusHint := dim.Render("Esc/⌃G to close")

	full := rendered + "\n" + statusHint

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, full)
}

// renderStatusBar builds the bottom status bar.
func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	left := " "
	if !m.wordWrap {
		left += " [no-wrap]"
	}
	if m.modified() {
		left += " [modified]"
	}

	var hint string
	var right string
	if m.status != "" {
		right = m.status
	} else if m.viewMode {
		if !m.dismissedHints["editor.checkbox"] {
			hint = "click checkboxes to toggle!  [h]ide"
		}
		right = "\u2303R edit \u00B7 Esc quit"
	} else {
		right = "/ blocks \u00B7 \u2303G help \u00B7 Esc quit"
	}

	bar := format.StatusBar(left, hint, right, width)

	style := lipgloss.NewStyle().Width(width)
	switch m.statusStyle {
	case statusSuccess:
		style = style.Foreground(lipgloss.Color(theme.Current().Success))
	case statusError:
		style = style.Foreground(lipgloss.Color(theme.Current().Error))
	case statusWarning:
		style = style.Foreground(lipgloss.Color(theme.Current().Warning))
	default:
		style = style.Faint(true)
	}

	return style.Render(bar)
}
