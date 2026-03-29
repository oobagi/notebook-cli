// Package editor provides a Bubble Tea TUI for editing markdown notes
// using a block-based editing surface where each block has its own textarea.
package editor

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/block"
	"github.com/oobagi/notebook/internal/theme"
)

// Config holds the configuration for the editor.
type Config struct {
	// Title is displayed in the status bar (e.g. "book > note").
	Title string
	// Content is the initial text content to edit.
	Content string
	// Save is called with the current content when the user presses Ctrl+S.
	Save func(string) error
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

	vp := viewport.New(defaultWidth, defaultHeight-1)

	return Model{
		blocks:    blocks,
		textareas: textareas,
		active:    0,
		viewport:  vp,
		config:    cfg,
		initial:   cfg.Content,
		width:     defaultWidth,
		height:    defaultHeight,
		palette:   newPalette(),
	}
}

// newTextareaForBlock creates a textarea configured for the given block type.
func newTextareaForBlock(b block.Block, width int) textarea.Model {
	ta := textarea.New()
	ta.SetValue(b.Content)
	ta.ShowLineNumbers = false
	ta.SetWidth(width)

	// Code blocks and paragraphs are multi-line; others are single-line.
	switch b.Type {
	case block.CodeBlock, block.Paragraph, block.Quote:
		// Multi-line: allow enough height for content.
		lines := strings.Count(b.Content, "\n") + 1
		if lines < 3 {
			lines = 3
		}
		ta.SetHeight(lines)
	default:
		// Single-line blocks.
		ta.SetHeight(1)
	}

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

// resizeTextareas updates all textarea widths for the current layout.
func (m *Model) resizeTextareas() {
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	for i := range m.textareas {
		m.textareas[i].SetWidth(w)
	}
	// Update viewport dimensions (reserve 1 line for status bar).
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	m.viewport.Width = w
	m.viewport.Height = h
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
func (m *Model) focusBlock(idx int) {
	if idx < 0 || idx >= len(m.textareas) {
		return
	}
	if m.active >= 0 && m.active < len(m.textareas) {
		// Sync content from old active textarea back to block.
		m.blocks[m.active].Content = m.textareas[m.active].Value()
		m.textareas[m.active].Blur()
	}
	m.active = idx
	m.textareas[idx].Focus()
}

// navigateUp moves focus to the previous block, placing cursor at end.
func (m *Model) navigateUp() {
	if m.active <= 0 {
		return
	}
	m.focusBlock(m.active - 1)
	// Place cursor at the end of the previous block's textarea.
	ta := &m.textareas[m.active]
	ta.CursorEnd()
}

// navigateDown moves focus to the next block, placing cursor at start.
func (m *Model) navigateDown() {
	if m.active >= len(m.textareas)-1 {
		return
	}
	m.focusBlock(m.active + 1)
	// Place cursor at the start of the next block's textarea.
	ta := &m.textareas[m.active]
	ta.CursorStart()
}

// isMultiLine returns true if the block type allows multi-line content.
func isMultiLine(bt block.BlockType) bool {
	return bt == block.Paragraph || bt == block.CodeBlock || bt == block.Quote
}

// insertBlockAfter inserts a new block after the given index, creates a
// textarea for it, and focuses the new block.
func (m *Model) insertBlockAfter(idx int, b block.Block) {
	if idx < 0 || idx >= len(m.blocks) {
		return
	}
	ta := newTextareaForBlock(b, m.width)

	// Insert into blocks slice.
	newBlocks := make([]block.Block, 0, len(m.blocks)+1)
	newBlocks = append(newBlocks, m.blocks[:idx+1]...)
	newBlocks = append(newBlocks, b)
	newBlocks = append(newBlocks, m.blocks[idx+1:]...)
	m.blocks = newBlocks

	// Insert into textareas slice.
	newTAs := make([]textarea.Model, 0, len(m.textareas)+1)
	newTAs = append(newTAs, m.textareas[:idx+1]...)
	newTAs = append(newTAs, ta)
	newTAs = append(newTAs, m.textareas[idx+1:]...)
	m.textareas = newTAs

	// Focus the new block.
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
		m.textareas[0].Focus()
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
	m.textareas[m.active].Focus()
}

// mergeBlockUp merges block at idx into block at idx-1. The merged block
// keeps the type of the previous block. Cursor is placed at the merge point.
func (m *Model) mergeBlockUp(idx int) {
	if idx <= 0 || idx >= len(m.blocks) {
		return
	}

	// Sync content from textarea.
	currentContent := m.textareas[idx].Value()
	prevContent := m.textareas[idx-1].Value()

	// Remember the merge point (end of previous content).
	mergeCol := len([]rune(prevContent))

	// Merge content into previous block.
	var merged string
	if prevContent == "" {
		merged = currentContent
	} else if currentContent == "" {
		merged = prevContent
	} else {
		merged = prevContent + "\n" + currentContent
	}

	// Update previous block content.
	m.blocks[idx-1].Content = merged
	m.textareas[idx-1].SetValue(merged)

	// For multi-line blocks, grow the textarea.
	if isMultiLine(m.blocks[idx-1].Type) {
		lines := strings.Count(merged, "\n") + 1
		if lines < 3 {
			lines = 3
		}
		m.textareas[idx-1].SetHeight(lines)
	}

	// Remove the current block.
	m.blocks = append(m.blocks[:idx], m.blocks[idx+1:]...)
	m.textareas = append(m.textareas[:idx], m.textareas[idx+1:]...)

	// Focus previous block and position cursor at merge point.
	m.active = idx - 1
	m.textareas[m.active].Focus()

	// Position cursor at the merge point.
	m.textareas[m.active].SetCursor(mergeCol)
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

	// Swap blocks.
	m.blocks[m.active], m.blocks[target] = m.blocks[target], m.blocks[m.active]
	m.textareas[m.active], m.textareas[target] = m.textareas[target], m.textareas[m.active]

	// Move focus to the new position.
	m.textareas[m.active].Blur()
	m.active = target
	m.textareas[m.active].Focus()
}

// handleEnter processes the Enter key for block creation/splitting.
func (m *Model) handleEnter() {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}

	bt := m.blocks[m.active].Type
	content := m.textareas[m.active].Value()

	if isMultiLine(bt) {
		// Multi-line block: only create new block if cursor is at the very end.
		ta := &m.textareas[m.active]
		cursorLine := ta.Line()
		lines := strings.Split(content, "\n")
		totalLines := len(lines)
		isLastLine := cursorLine >= totalLines-1

		if isLastLine {
			col := ta.LineInfo().ColumnOffset
			lastLineRunes := []rune(lines[totalLines-1])
			if col >= len(lastLineRunes) {
				// Cursor is at the very end of the last line.
				// Create new empty paragraph below.
				m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph, Content: ""})
				return
			}
		}
		// Otherwise, let the textarea handle Enter normally (insert newline).
		return
	}

	// Single-line blocks: create a new block below.
	var newBlock block.Block
	switch bt {
	case block.Checklist:
		newBlock = block.Block{Type: block.Checklist, Content: "", Checked: false}
	case block.BulletList:
		newBlock = block.Block{Type: block.BulletList, Content: ""}
	case block.NumberedList:
		newBlock = block.Block{Type: block.NumberedList, Content: ""}
	default:
		// Headings, dividers, etc. create a paragraph.
		newBlock = block.Block{Type: block.Paragraph, Content: ""}
	}

	m.insertBlockAfter(m.active, newBlock)
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
	if ta.LineInfo().ColumnOffset != 0 {
		return false
	}

	content := ta.Value()

	if content == "" {
		// Empty block: delete it, focus previous.
		if m.active == 0 {
			if len(m.blocks) <= 1 {
				return true // Already empty paragraph, nothing to do.
			}
		}
		m.deleteBlock(m.active)
		m.textareas[m.active].CursorEnd()
		return true
	}

	// Non-empty block at position 0: merge with previous block.
	if m.active == 0 {
		return false // No previous block to merge into.
	}

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
	m.textareas[m.active].Focus()
}

// applyPaletteSelection changes the active block's type to the selected
// palette item type. Special handling is applied for dividers and code blocks.
func (m *Model) applyPaletteSelection(bt block.BlockType) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}

	m.blocks[m.active].Type = bt

	switch bt {
	case block.Divider:
		// Dividers have no editable content.
		m.blocks[m.active].Content = ""
		m.textareas[m.active].SetValue("")
	case block.CodeBlock:
		// Reconfigure textarea for multi-line editing.
		m.textareas[m.active].SetHeight(3)
	default:
		if isMultiLine(bt) {
			lines := strings.Count(m.textareas[m.active].Value(), "\n") + 1
			if lines < 3 {
				lines = 3
			}
			m.textareas[m.active].SetHeight(lines)
		} else {
			m.textareas[m.active].SetHeight(1)
		}
	}
}

// isAtFirstLine returns true if the cursor is on the first line of the
// active textarea.
func (m Model) isAtFirstLine() bool {
	if m.active < 0 || m.active >= len(m.textareas) {
		return false
	}
	return m.textareas[m.active].Line() == 0
}

// isAtLastLine returns true if the cursor is on the last line of the
// active textarea.
func (m Model) isAtLastLine() bool {
	if m.active < 0 || m.active >= len(m.textareas) {
		return false
	}
	ta := m.textareas[m.active]
	value := ta.Value()
	lineCount := strings.Count(value, "\n") + 1
	return ta.Line() >= lineCount-1
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyMsg:
		// When help overlay is showing, only Ctrl+G and Esc dismiss it.
		if m.showHelp {
			switch msg.String() {
			case "ctrl+g", "esc":
				m.showHelp = false
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
					m.updateViewport()
				}
				return m, nil
			default:
				if msg.Type == tea.KeyRunes {
					for _, r := range msg.Runes {
						m.palette.addFilterRune(r)
					}
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

		switch msg.String() {
		case "ctrl+g":
			m.showHelp = true
			return m, nil

		case "ctrl+k":
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

		case "ctrl+q":
			if m.modified() {
				m.quitPrompt = true
				m.status = "Save before quitting? [Y/n/Esc]"
				m.statusStyle = statusWarning
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up":
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
			m.swapBlocks(-1)
			m.updateViewport()
			return m, nil

		case "alt+down":
			m.swapBlocks(1)
			m.updateViewport()
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
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			bt := m.blocks[m.active].Type

			// Handle "/" at position 0 to open the command palette.
			if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) == 1 && keyMsg.Runes[0] == '/' {
				ta := &m.textareas[m.active]
				if ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0 && ta.Value() == "" {
					m.palette.open(m.active)
					m.updateViewport()
					return m, nil
				}
			}

			// Handle Enter key for block creation.
			if keyMsg.Type == tea.KeyEnter {
				if !isMultiLine(bt) {
					// Single-line blocks: create new block below.
					m.handleEnter()
					m.updateViewport()
					return m, nil
				}
				// Multi-line blocks: check if cursor is at the very end.
				ta := &m.textareas[m.active]
				content := ta.Value()
				lines := strings.Split(content, "\n")
				totalLines := len(lines)
				cursorLine := ta.Line()
				isLastLine := cursorLine >= totalLines-1

				if isLastLine {
					col := ta.LineInfo().ColumnOffset
					lastLineRunes := []rune(lines[totalLines-1])
					if col >= len(lastLineRunes) {
						m.handleEnter()
						m.updateViewport()
						return m, nil
					}
				}
				// Otherwise fall through to let textarea handle it.
			}

			// Handle Backspace at position 0 for delete/merge.
			if keyMsg.Type == tea.KeyBackspace {
				if m.handleBackspace() {
					m.updateViewport()
					return m, nil
				}
			}
		}

		var cmd tea.Cmd
		m.textareas[m.active], cmd = m.textareas[m.active].Update(msg)

		// Grow multi-line textareas dynamically as the user types.
		bt := m.blocks[m.active].Type
		if isMultiLine(bt) {
			lines := strings.Count(m.textareas[m.active].Value(), "\n") + 1
			if lines < 3 {
				lines = 3
			}
			m.textareas[m.active].SetHeight(lines)
		}

		m.updateViewport()
		return m, cmd
	}

	return m, nil
}

// updateViewport renders all blocks and sets the viewport content, then
// auto-scrolls to keep the active block's cursor line visible.
func (m *Model) updateViewport() {
	content := m.renderAllBlocks()
	m.viewport.SetContent(content)

	// Auto-scroll: calculate the cursor's actual line in the rendered output
	// and ensure the viewport shows it. We compute the offset from scratch
	// because SetContent may reset the viewport's internal scroll state.
	if m.active >= 0 && m.active < len(m.blocks) && m.active < len(m.textareas) {
		// Count rendered lines for all blocks before the active one,
		// including the "\n" join separators between blocks.
		lineOffset := 0
		for i := 0; i < m.active; i++ {
			rendered := m.renderBlock(i)
			lineOffset += strings.Count(rendered, "\n") + 1
			// Account for the "\n" join separator in renderAllBlocks.
			lineOffset++
			// Account for palette rendered after a preceding block.
			if m.palette.visible && i == m.palette.blockIdx {
				if pv := m.palette.render(m.width); pv != "" {
					lineOffset += strings.Count(pv, "\n") + 1
					lineOffset++ // join separator after palette
				}
			}
		}

		// Account for chrome lines the active block's renderActiveBlock
		// prepends before the textarea content.
		chromeLines := 0
		if m.active > 0 && m.blocks[m.active].Type == block.Heading1 {
			chromeLines = 1 // "\n" prefix for non-first H1
		}
		if m.blocks[m.active].Type == block.CodeBlock {
			chromeLines = 1 // label + "\n" before bordered textarea
		}

		cursorLine := lineOffset + chromeLines + m.textareas[m.active].Line()

		// Always ensure the cursor line is visible. Prefer keeping the
		// current scroll position when the cursor is already on screen.
		yOffset := m.viewport.YOffset
		if cursorLine < yOffset {
			yOffset = cursorLine
		}
		if cursorLine >= yOffset+m.viewport.Height {
			yOffset = cursorLine - m.viewport.Height + 1
		}
		if yOffset < 0 {
			yOffset = 0
		}

		m.viewport.SetYOffset(yOffset)
	}
}

// renderAllBlocks renders each block and joins them vertically.
// When the command palette is visible, it is rendered below the active block.
func (m Model) renderAllBlocks() string {
	var parts []string
	for i := range m.blocks {
		parts = append(parts, m.renderBlock(i))
		if m.palette.visible && i == m.active {
			if pv := m.palette.render(m.width); pv != "" {
				parts = append(parts, pv)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// View renders the editor UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.renderHelpOverlay()
	}

	statusBar := m.renderStatusBar()

	return m.viewport.View() + "\n" + statusBar
}

// renderHelpOverlay builds the full-screen help panel.
func (m Model) renderHelpOverlay() string {
	help := `  Keybindings
  ───────────────────────────

  Ctrl+S    Save
  Ctrl+Q    Quit
  Ctrl+C    Force quit (no save)
  Ctrl+G    Toggle this help
  Ctrl+K    Cut block
  Enter     New block below
  Backspace Merge/delete block
  Alt+Up    Move block up
  Alt+Down  Move block down
  /         Block type palette

  Press Ctrl+G or Esc to close`

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Current().Border)).
		Padding(1, 2).
		Width(38).
		Align(lipgloss.Left)

	rendered := box.Render(help)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, rendered)
}

// renderStatusBar builds the bottom status bar.
func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	left := m.config.Title
	if m.modified() {
		left += " [modified]"
	}

	var right string
	if m.status != "" {
		right = m.status
	} else {
		right = "/ for commands \u00B7 Ctrl+S save \u00B7 Ctrl+G help \u00B7 Ctrl+Q quit"
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

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
