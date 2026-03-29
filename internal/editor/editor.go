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
	clipboard   string // line-level clipboard for Ctrl+Y
	blockClip   *block.Block // block-level clipboard for Ctrl+K block cut
	statusGen   int    // generation counter for status auto-dismiss
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

// toggleCheckbox toggles the Checked field on the active block if it is a
// Checklist block. For non-checklist blocks, it is a no-op.
func (m *Model) toggleCheckbox() {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}
	if m.blocks[m.active].Type != block.Checklist {
		return
	}
	m.blocks[m.active].Checked = !m.blocks[m.active].Checked
}

// deleteToLineStart removes all text from the cursor to the start of the
// current line in the active textarea.
func (m *Model) deleteToLineStart() {
	if m.active < 0 || m.active >= len(m.textareas) {
		return
	}
	ta := &m.textareas[m.active]
	value := ta.Value()
	lines := strings.Split(value, "\n")

	line := ta.Line()
	if line < 0 || line >= len(lines) {
		return
	}

	col := ta.LineInfo().ColumnOffset
	if col <= 0 {
		return
	}

	runes := []rune(lines[line])
	if col > len(runes) {
		col = len(runes)
	}
	lines[line] = string(runes[col:])

	ta.SetValue(strings.Join(lines, "\n"))

	// Reposition cursor.
	ta.SetCursor(0)
	for ta.Line() > line {
		ta.CursorUp()
	}
	for ta.Line() < line {
		ta.CursorDown()
	}
}

// cutLine removes the current line from the active textarea and stores it
// in the line clipboard.
func (m *Model) cutLine() string {
	if m.active < 0 || m.active >= len(m.textareas) {
		return ""
	}
	ta := &m.textareas[m.active]
	value := ta.Value()
	lines := strings.Split(value, "\n")

	line := ta.Line()
	if line < 0 || line >= len(lines) {
		return ""
	}

	cut := lines[line]
	newLines := make([]string, 0, len(lines)-1)
	newLines = append(newLines, lines[:line]...)
	newLines = append(newLines, lines[line+1:]...)

	ta.SetValue(strings.Join(newLines, "\n"))

	newLineCount := len(newLines)
	targetLine := line
	if targetLine >= newLineCount {
		targetLine = newLineCount - 1
	}
	if targetLine < 0 {
		targetLine = 0
	}
	ta.SetCursor(0)
	for ta.Line() > targetLine {
		ta.CursorUp()
	}
	for ta.Line() < targetLine {
		ta.CursorDown()
	}

	return cut
}

// pasteLine inserts the line clipboard content at the current cursor line
// in the active textarea.
func (m *Model) pasteLine() {
	if m.clipboard == "" {
		return
	}
	if m.active < 0 || m.active >= len(m.textareas) {
		return
	}
	ta := &m.textareas[m.active]
	value := ta.Value()
	lines := strings.Split(value, "\n")

	line := ta.Line()
	if line < 0 {
		line = 0
	}
	if line > len(lines) {
		line = len(lines)
	}

	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:line]...)
	newLines = append(newLines, m.clipboard)
	newLines = append(newLines, lines[line:]...)

	ta.SetValue(strings.Join(newLines, "\n"))

	ta.SetCursor(0)
	for ta.Line() < line {
		ta.CursorDown()
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

		case "ctrl+u":
			m.deleteToLineStart()
			m.updateViewport()
			return m, nil

		case "ctrl+d":
			m.toggleCheckbox()
			m.updateViewport()
			return m, nil

		case "ctrl+y":
			m.pasteLine()
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
		// Block Enter key in single-line blocks to prevent newlines
		// from corrupting the block structure.
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEnter {
			bt := m.blocks[m.active].Type
			if bt != block.Paragraph && bt != block.CodeBlock && bt != block.Quote {
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.textareas[m.active], cmd = m.textareas[m.active].Update(msg)

		// Grow multi-line textareas dynamically as the user types.
		bt := m.blocks[m.active].Type
		if bt == block.Paragraph || bt == block.CodeBlock || bt == block.Quote {
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
// auto-scrolls to keep the active block visible.
func (m *Model) updateViewport() {
	content := m.renderAllBlocks()
	m.viewport.SetContent(content)

	// Auto-scroll: calculate where the active block starts in the rendered
	// output and ensure it is visible.
	if m.active >= 0 && m.active < len(m.blocks) {
		lineOffset := 0
		for i := 0; i < m.active; i++ {
			rendered := m.renderBlock(i)
			lineOffset += strings.Count(rendered, "\n") + 1
		}
		// Try to center the active block in the viewport.
		target := lineOffset - m.viewport.Height/3
		if target < 0 {
			target = 0
		}
		m.viewport.SetYOffset(target)
	}
}

// renderAllBlocks renders each block and joins them vertically.
func (m Model) renderAllBlocks() string {
	var parts []string
	for i := range m.blocks {
		parts = append(parts, m.renderBlock(i))
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
  Ctrl+Y    Paste line
  Ctrl+U    Delete to line start
  Ctrl+D    Toggle checkbox

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
		Width(36).
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
		right = "Ctrl+S save \u00B7 Ctrl+G help \u00B7 Ctrl+Q quit"
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
