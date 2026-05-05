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
	"github.com/oobagi/notebook-cli/internal/clipboard"
	"github.com/oobagi/notebook-cli/internal/config"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/oobagi/notebook-cli/internal/ui"
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
	// HideChecked enables sort-checked-to-bottom for checklists.
	HideChecked *bool
	// CascadeChecks: checking a parent also checks/unchecks children.
	CascadeChecks *bool
	// WordWrap from config; nil = default (true).
	WordWrap *bool
	// KanbanSortByPrio sorts kanban cards by priority desc within each column.
	KanbanSortByPrio *bool
	// ResolveEmbed resolves an embed path (e.g. "notebook/note") to its title
	// and content. Returns an error if the note doesn't exist.
	ResolveEmbed func(path string) (title, content string, err error)
	// SaveEmbed saves modified content back to an embedded note.
	SaveEmbed func(path, content string) error
	// ListEmbedTargets returns all available "notebook/note" paths for the
	// embed picker. Called when the picker opens.
	ListEmbedTargets func() []string
	// ListAllDefinitions returns definition list entries gathered from every
	// note across all notebooks, excluding the note currently being edited
	// (since its in-memory blocks may differ from disk). When nil, the ":"
	// lookup is restricted to the current note.
	ListAllDefinitions func() []DefinitionEntry
}

// DefinitionEntry is a single definition list block sourced from another
// note. The Notebook/Note pair locates the source for display in the lookup.
type DefinitionEntry struct {
	Term       string
	Definition string
	Notebook   string
	Note       string
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

type clipboardReadErrMsg struct{ err error }

func readClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.Read()
		if err != nil {
			return clipboardReadErrMsg{err: err}
		}
		return tea.PasteMsg{Content: text}
	}
}

// Model is the Bubble Tea model for the block-based editor.
type Model struct {
	blocks    []block.Block    // the data model
	textareas []textarea.Model // one textarea per block
	active    int              // index of focused block
	viewport  viewport.Model   // scrollable container

	config           Config
	initial          string
	width            int
	height           int
	status           string
	statusStyle      statusKind
	quitPrompt       bool
	quitting         bool
	showHelp         bool
	blockClip        *block.Block    // block-level clipboard for Ctrl+K block cut
	statusGen        int             // generation counter for status auto-dismiss
	palette          palette         // "/" command palette for block type insertion
	defLookup        defLookup       // ":" definition lookup palette
	wordWrap         bool            // when true, text wraps at terminal width
	viewMode         bool            // when true, read-only rendering with no cursor
	hoverBlock       int             // view mode: block index under mouse cursor (-1 = none)
	cursorCmd        tea.Cmd         // pending cursor blink command from Focus()
	blockLineOffsets []int           // view mode: starting Y line of each block in rendered output
	blockLineCounts  []int           // per-block rendered line counts from renderAllBlocks
	dismissedHints   map[string]bool // tracks which onboarding hints the user has dismissed
	undo             undoStack       // undo history
	redo             undoStack       // redo history
	undoDirty        bool            // true when textarea content changed since last snapshot
	sortChecked      bool            // sort checked checklist items to bottom of each group
	cascadeChecks    bool            // checking parent also checks/unchecks children
	kanbanSortByPrio bool            // sort kanban cards by priority desc within each column
	embedModal       embedModalState // overlay for viewing embedded note references
	linkPrompt       linkPromptState // bottom-of-screen input for editing a link's URL or title
	embedPicker      embedPicker     // note picker for embed block insertion
	table            *tableState     // active table cell state (non-nil when editing a Table block)
	kanban           *kanbanState    // active kanban board state (non-nil when editing a Kanban block)
	viewKanbanOffset int             // horizontal scroll for kanban blocks in view mode
	kanbanOffsets    map[int]int     // saved colOffset per block index, restored when refocusing
	kanbanAnchorTop  bool            // one-shot: next viewport update anchors to kanban title row
	defPreview       defPreviewState // focused preview for a single cross-note definition
	jumpTarget       string          // "notebook/note" to open after editor quits; read via JumpTarget()
	pendingJump      string          // jump requested but current note is modified — set alongside quitPrompt
}

// defPreviewState holds a focused preview of one cross-note definition.
// Renders as a compact auto-height footer pane above the status bar, sized
// to the wrapped content (not full-screen).
type defPreviewState struct {
	visible    bool
	term       string
	definition string
	path       string // "notebook/note" for the Enter-to-open keybind
}

func (d *defPreviewState) open(term, definition, path string) {
	d.visible = true
	d.term = term
	d.definition = definition
	d.path = path
}

func (d *defPreviewState) close() {
	d.visible = false
	d.term = ""
	d.definition = ""
	d.path = ""
}

// JumpTarget returns the "notebook/note" path the editor was asked to open
// before quitting, or "". Non-empty means the host should re-launch the
// editor on that path instead of exiting normally.
func (m Model) JumpTarget() string {
	return m.jumpTarget
}

type statusKind int

const (
	statusNone statusKind = iota
	statusSuccess
	statusError
	statusWarning
)

// embedModalState holds the state for the embedded note bottom sheet.
type embedModalState struct {
	visible          bool
	title            string        // display title for the sheet header
	path             string        // original embed path for save-back
	blocks           []block.Block // parsed blocks of the referenced note
	scroll           int           // scroll offset (line index)
	lines            []string      // pre-split rendered lines
	blockLineOffsets []int         // starting line of each block within lines
	hoverBlock       int           // block under mouse cursor (-1 = none)
	sheetStartY      int           // Y line where the sheet content begins
}

// open prepares the sheet with parsed blocks and resets state.
func (e *embedModalState) open(title, path string, blocks []block.Block) {
	e.visible = true
	e.title = title
	e.path = path
	e.blocks = blocks
	e.scroll = 0
	e.hoverBlock = -1
	// Layout: line 0=context, 1=blank, 2=sep → sheet starts at Y=2.
	e.sheetStartY = 2
	// lines and blockLineOffsets are populated by renderEmbedSheetContent.
}

// close hides the sheet.
func (e *embedModalState) close() {
	e.visible = false
	e.title = ""
	e.path = ""
	e.blocks = nil
	e.lines = nil
	e.blockLineOffsets = nil
	e.scroll = 0
	e.hoverBlock = -1
}

// blockAtLine returns the block index containing the given line, or -1.
func (e *embedModalState) blockAtLine(line int) int {
	if len(e.blockLineOffsets) == 0 {
		return -1
	}
	result := -1
	for i, offset := range e.blockLineOffsets {
		if offset <= line {
			result = i
		} else {
			break
		}
	}
	return result
}

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

func blockPrefixWidth(b block.Block) int {
	th := theme.Current()
	base := 0
	switch b.Type {
	case block.BulletList:
		base = lipgloss.Width(th.Blocks.Bullet.Marker)
	case block.NumberedList:
		base = lipgloss.Width(fmt.Sprintf(th.Blocks.Numbered.Format, 1))
	case block.Checklist:
		base = lipgloss.Width(th.Blocks.Checklist.Unchecked)
		if m := b.Priority.Marker(); m != "" {
			base += len(m) + 1 // marker + trailing space
		}
	case block.Quote, block.Callout:
		base = lipgloss.Width(th.Blocks.Quote.Bar)
	case block.CodeBlock:
		base = 4
	case block.DefinitionList:
		base = lipgloss.Width(th.Blocks.Definition.Marker)
	case block.Embed:
		icon := th.Blocks.Embed.Icon
		if icon == "" {
			icon = "\u2197 "
		}
		base = lipgloss.Width(icon)
	case block.Link:
		// Link card prefixes the title with the link icon; editing happens
		// in the bottom-sheet modal, not inline.
		base = lipgloss.Width(linkIcon)
	case block.Table:
		base = 0
	}
	return b.Indent*4 + base
}

// contentWidth returns the effective textarea width for a block, accounting
// for the gutter, block prefix, indent, and word-wrap setting.
func (m Model) contentWidth(b block.Block) int {
	mw := m.width
	if mw <= 0 {
		mw = defaultWidth
	}
	w := mw - gutterWidth - blockPrefixWidth(b)
	if w < 1 {
		w = 1
	}
	if !m.wordWrap {
		w = noWrapWidth
	}
	return w
}

// tableCellTAWidth returns the textarea width for a table cell.
// In word-wrap mode this is the visual cell width so text wraps within
// the cell. In no-wrap mode this is noWrapWidth so text stays on one
// line; the render handles per-cell scrolling with ← → indicators.
func (m Model) tableCellTAWidth() int {
	if !m.wordWrap {
		return noWrapWidth
	}
	if m.table == nil {
		return 5
	}
	mw := m.width
	if mw <= 0 {
		mw = defaultWidth
	}
	return tableCellWidth(mw-gutterWidth, m.table.numCols())
}

// tableCellDisplayWidth returns the visual column width for rendering,
// always based on the window width regardless of word-wrap.
func (m Model) tableCellDisplayWidth() int {
	if m.table == nil {
		return 5
	}
	mw := m.width
	if mw <= 0 {
		mw = defaultWidth
	}
	return tableCellWidth(mw-gutterWidth, m.table.numCols())
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

	m := Model{
		blocks:           blocks,
		textareas:        textareas,
		active:           0,
		viewport:         vp,
		config:           cfg,
		initial:          cfg.Content,
		width:            defaultWidth,
		height:           defaultHeight,
		palette:          newPalette(),
		defLookup:        newDefLookup(),
		embedPicker:      newEmbedPicker(),
		wordWrap:         config.BoolVal(cfg.WordWrap, true),
		dismissedHints:   dismissed,
		hoverBlock:       -1,
		sortChecked:      config.BoolVal(cfg.HideChecked, false),
		cascadeChecks:    config.BoolVal(cfg.CascadeChecks, true),
		kanbanSortByPrio: config.BoolVal(cfg.KanbanSortByPrio, false),
		kanbanOffsets:    map[int]int{},
	}

	// If first block is a Table, init table state.
	if len(m.blocks) > 0 && m.blocks[0].Type == block.Table {
		m.table = initTable(m.blocks[0].Content)
		cw := m.tableCellTAWidth()
		m.table.loadCell(&m.textareas[0], cw, false)
		m.cursorCmd = m.textareas[0].Focus()
	}
	// If first block is a Kanban, init kanban state. Offset starts at 0
	// for fresh editor sessions; the offset map persists across focus
	// changes within the same session.
	if len(m.blocks) > 0 && m.blocks[0].Type == block.Kanban {
		m.kanban = newKanbanState(m.blocks[0].Content)
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		m.textareas[0].Blur()
	}

	return m
}

// newTextareaForBlock creates a textarea configured for the given block type.
func newTextareaForBlock(b block.Block, width int) textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	// Set width BEFORE value so the textarea's internal state (viewport
	// scroll, wrap cache) is initialized with the correct dimensions.
	taWidth := width - gutterWidth - blockPrefixWidth(b)
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
			if m.table != nil && i == m.active && m.blocks[i].Type == block.Table {
				// For the active table, sync the current cell and serialize.
				ts := *m.table // copy to avoid mutating receiver
				ts.syncCell(m.textareas[i])
				result[i].Content = ts.serialize()
			} else if m.kanban != nil && i == m.active && m.blocks[i].Type == block.Kanban {
				// For the active kanban, serialize state. If mid-edit, fold
				// the textarea value into the selected card's text in a
				// deep copy so the live state isn't mutated by a save.
				cols := cloneKanbanCols(m.kanban.cols)
				if m.kanban.edit && m.kanban.col >= 0 && m.kanban.col < len(cols) {
					col := &cols[m.kanban.col]
					if m.kanban.card >= 0 && m.kanban.card < len(col.Cards) {
						col.Cards[m.kanban.card].Text = strings.TrimRight(m.kanban.editTA.Value(), "\n")
					}
				}
				result[i].Content = block.SerializeKanban(cols)
			} else {
				result[i].Content = m.textareas[i].Value()
			}
		}
	}
	return result
}

// BlockCount returns the number of blocks in the editor.
func (m Model) BlockCount() int {
	return len(m.blocks)
}

// sortCheckedToBottom reorders each contiguous group of checklist blocks
// so that unchecked bundles come first and checked bundles come last
// (stable partition). A bundle is a parent item plus all consecutive
// items with deeper indent. Children within each bundle are also sorted
// by checked state at their indent level. Updates m.active to follow
// the focused block.
func (m *Model) sortCheckedToBottom() {
	i := 0
	for i < len(m.blocks) {
		if m.blocks[i].Type != block.Checklist {
			i++
			continue
		}
		start := i
		// Include all consecutive list items (any type) in the group,
		// so mixed-type children stay bundled with checklist parents.
		for i < len(m.blocks) && m.blocks[i].Type.IsListItem() {
			i++
		}
		end := i
		if end-start < 2 {
			continue
		}
		m.sortChecklistGroup(start, end)
	}
}

// sortChecklistGroup sorts a contiguous checklist group [start, end).
// Bundles at the base indent are sorted by parent checked state.
// Children within each bundle are also sorted by checked state.
func (m *Model) sortChecklistGroup(start, end int) {
	origActive := m.active

	// A bundle is a sequence of original indices: parent + children.
	type bundle struct {
		indices []int
		checked bool
	}

	var bundles []bundle
	j := start
	for j < end {
		parentIdx := j
		idxs := []int{j}
		j++
		for j < end && m.blocks[j].Indent > m.blocks[parentIdx].Indent {
			idxs = append(idxs, j)
			j++
		}
		bundles = append(bundles, bundle{
			indices: idxs,
			checked: m.blocks[parentIdx].Checked,
		})
	}

	// Sort children within each bundle (indices[1:]) by checked state.
	for bi := range bundles {
		children := bundles[bi].indices[1:]
		if len(children) < 2 {
			continue
		}
		sorted := make([]int, 0, len(children))
		for _, idx := range children {
			if !m.blocks[idx].Checked {
				sorted = append(sorted, idx)
			}
		}
		for _, idx := range children {
			if m.blocks[idx].Checked {
				sorted = append(sorted, idx)
			}
		}
		copy(bundles[bi].indices[1:], sorted)
	}

	var unchecked, checked []bundle
	for _, b := range bundles {
		if b.checked {
			checked = append(checked, b)
		} else {
			unchecked = append(unchecked, b)
		}
	}
	order := append(unchecked, checked...)

	// Apply reordering.
	n := end - start
	tmpB := make([]block.Block, n)
	tmpT := make([]textarea.Model, n)
	pos := 0
	for _, b := range order {
		for _, idx := range b.indices {
			tmpB[pos] = m.blocks[idx]
			tmpT[pos] = m.textareas[idx]
			if idx == origActive {
				m.active = start + pos
			}
			pos++
		}
	}
	if pos != n {
		return // bail out rather than corrupt data
	}
	copy(m.blocks[start:end], tmpB)
	copy(m.textareas[start:end], tmpT)
}

// persistSortChecked saves the current sort-checked setting to the global config.
func (m *Model) persistSortChecked() {
	if globalCfg, err := config.Load(); err == nil {
		globalCfg.HideChecked = config.BoolPtr(m.sortChecked)
		_ = config.Save(globalCfg)
	}
}

// persistKanbanSort saves the kanban sort-by-priority toggle to global config.
func (m *Model) persistKanbanSort() {
	if globalCfg, err := config.Load(); err == nil {
		globalCfg.KanbanSortByPrio = config.BoolPtr(m.kanbanSortByPrio)
		_ = config.Save(globalCfg)
	}
}

// toggleChecklist toggles the checked state of the block at idx. When
// cascadeChecks is enabled, indented checklist children are also toggled.
func (m *Model) toggleChecklist(idx int) {
	if idx < 0 || idx >= len(m.blocks) || m.blocks[idx].Type != block.Checklist {
		return
	}
	newState := !m.blocks[idx].Checked
	m.blocks[idx].Checked = newState
	if m.cascadeChecks {
		parentIndent := m.blocks[idx].Indent
		for j := idx + 1; j < len(m.blocks) && m.blocks[j].Type.IsListItem() && m.blocks[j].Indent > parentIndent; j++ {
			if m.blocks[j].Type == block.Checklist {
				m.blocks[j].Checked = newState
			}
		}
	}
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
		if m.table != nil && i == m.active && b.Type == block.Table {
			// Active table textarea shows a cell; size to cell width.
			cw := m.tableCellTAWidth()
			m.textareas[i].SetWidth(cw)
		} else {
			m.textareas[i].SetWidth(m.contentWidth(b))
		}
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

	// Leaving a table: sync cell and serialize back to block content.
	if m.table != nil && m.active >= 0 && m.active < len(m.textareas) {
		m.table.syncCell(m.textareas[m.active])
		serialized := m.table.serialize()
		m.blocks[m.active].Content = serialized
		m.textareas[m.active].SetValue(serialized)
		m.table = nil
	}
	// Leaving a kanban: serialize state back to block content and remember
	// the horizontal scroll offset for when we focus this block again.
	if m.kanban != nil && m.active >= 0 && m.active < len(m.textareas) {
		if m.kanban.edit {
			m.kanban.commitEdit()
		}
		if m.kanbanOffsets == nil {
			m.kanbanOffsets = map[int]int{}
		}
		m.kanbanOffsets[m.active] = m.kanban.colOffset
		serialized := m.kanban.serialize()
		m.blocks[m.active].Content = serialized
		m.textareas[m.active].SetValue(serialized)
		m.kanban = nil
	}

	if m.active >= 0 && m.active < len(m.textareas) {
		m.blocks[m.active].Content = m.textareas[m.active].Value()
		m.textareas[m.active].Blur()
	}
	m.active = idx
	m.cursorCmd = m.textareas[idx].Focus()

	// Entering a table: init table state and load first cell.
	if m.blocks[idx].Type == block.Table {
		m.table = initTable(m.blocks[idx].Content)
		cw := m.tableCellTAWidth()
		m.table.loadCell(&m.textareas[idx], cw, false)
		m.cursorCmd = m.textareas[idx].Focus()
	}
	// Entering a kanban: init kanban state from block content and restore
	// the saved horizontal scroll offset (if any).
	if m.blocks[idx].Type == block.Kanban {
		m.kanban = newKanbanState(m.blocks[idx].Content)
		if m.kanbanOffsets != nil {
			m.kanban.colOffset = m.kanbanOffsets[idx]
		}
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		m.textareas[idx].Blur()
	}
}

// navigateUp moves focus to the previous block, preserving horizontal position.
func (m *Model) navigateUp() {
	if m.active <= 0 {
		return
	}
	charOffset := m.textareas[m.active].LineInfo().CharOffset
	m.focusBlock(m.active - 1)

	// Entering a table from below: bottom-left cell, apply charOffset.
	if m.table != nil {
		lastRow := len(m.table.cells) - 1
		if lastRow < 0 {
			lastRow = 0
		}
		m.table.row = lastRow
		cw := m.tableCellTAWidth()
		m.table.loadCell(&m.textareas[m.active], cw, true)
		ta := &m.textareas[m.active]
		li := ta.LineInfo()
		ta.SetCursorColumn(li.StartColumn + charOffset)
		m.cursorCmd = ta.Focus()
		return
	}
	// Entering a kanban from below: land on the last card of the
	// leftmost non-empty column. Mirrors table behavior, which always
	// enters at column 0 regardless of direction.
	if m.kanban != nil {
		m.kanban.enterFromBelow()
		m.kanbanAnchorTop = true
		return
	}

	ta := &m.textareas[m.active]
	if m.blocks[m.active].Type == block.Link {
		ta.MoveToBegin()
		ta.CursorEnd()
		return
	}
	ta.MoveToEnd()
	li := ta.LineInfo()
	ta.SetCursorColumn(li.StartColumn + charOffset)
}

// navigateDown moves focus to the next block, preserving horizontal position.
func (m *Model) navigateDown() {
	if m.active >= len(m.textareas)-1 {
		return
	}
	charOffset := m.textareas[m.active].LineInfo().CharOffset
	m.focusBlock(m.active + 1)

	// Entering a table from above: top-left cell, apply charOffset.
	if m.table != nil {
		ta := &m.textareas[m.active]
		ta.SetCursorColumn(charOffset)
		m.cursorCmd = ta.Focus()
		return
	}
	// Entering a kanban from above: land on the first card of the
	// leftmost non-empty column. Symmetric with navigateUp's entry.
	if m.kanban != nil {
		m.kanban.enterFromAbove()
		return
	}

	ta := &m.textareas[m.active]
	if m.blocks[m.active].Type == block.Link {
		ta.MoveToBegin()
		ta.CursorEnd()
		return
	}
	ta.MoveToBegin()
	ta.SetCursorColumn(charOffset)
}

// isMultiLine returns true if the block type allows multi-line content.
func isMultiLine(bt block.BlockType) bool {
	return bt == block.Paragraph || bt == block.CodeBlock || bt == block.Quote || bt == block.DefinitionList || bt == block.Callout || bt == block.Table || bt == block.Link
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

	// Shift m.active forward so focusBlock's "leaving the previously-active
	// block" logic (e.g. serializing kanban/table state back into the block
	// it came from) writes to the correct, post-shift index. Without this,
	// the leaving kanban's serialized content gets written into the new
	// paragraph at idx.
	if m.active >= idx {
		m.active++
	}

	// Shift kanbanOffsets keys >= idx forward by one so saved horizontal
	// scroll positions stay attached to the right Kanban block.
	m.shiftKanbanOffsets(idx, +1)

	m.focusBlock(idx)
}

// insertBlockAfter inserts a new block after the given index, creates a
// textarea for it, and focuses the new block.
func (m *Model) insertBlockAfter(idx int, b block.Block) {
	if idx < 0 || idx >= len(m.blocks) {
		return
	}
	insertAt := idx

	ta := newTextareaForBlock(b, m.width)

	newBlocks := make([]block.Block, 0, len(m.blocks)+1)
	newBlocks = append(newBlocks, m.blocks[:insertAt+1]...)
	newBlocks = append(newBlocks, b)
	newBlocks = append(newBlocks, m.blocks[insertAt+1:]...)
	m.blocks = newBlocks

	newTAs := make([]textarea.Model, 0, len(m.textareas)+1)
	newTAs = append(newTAs, m.textareas[:insertAt+1]...)
	newTAs = append(newTAs, ta)
	newTAs = append(newTAs, m.textareas[insertAt+1:]...)
	m.textareas = newTAs

	// Shift kanbanOffsets keys > insertAt forward by one so saved
	// horizontal scroll positions stay attached to the right Kanban.
	m.shiftKanbanOffsets(insertAt+1, +1)

	m.focusBlock(insertAt + 1)
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

	// Drop the deleted block's saved kanban offset (if any) and shift
	// later keys back by one to keep them aligned.
	if m.kanbanOffsets != nil {
		delete(m.kanbanOffsets, idx)
	}
	m.shiftKanbanOffsets(idx+1, -1)

	// Adjust active index.
	if idx >= len(m.blocks) {
		m.active = len(m.blocks) - 1
	} else if idx > 0 {
		m.active = idx - 1
	} else {
		m.active = 0
	}

	m.cursorCmd = m.textareas[m.active].Focus()

	// If the new active block is a Table or Kanban, init its state so the
	// renderer doesn't fall back to a collapsed single-line view.
	m.initActiveContainerState()
}

// shiftKanbanOffsets remaps kanbanOffsets keys at or above `from` by `delta`.
// Used by insertBlockBefore/After (delta=+1) and deleteBlock (delta=-1) so
// that per-block horizontal scroll positions stay attached to the right
// Kanban as block indexes shift around them.
func (m *Model) shiftKanbanOffsets(from, delta int) {
	if len(m.kanbanOffsets) == 0 || delta == 0 {
		return
	}
	next := make(map[int]int, len(m.kanbanOffsets))
	for k, v := range m.kanbanOffsets {
		if k >= from {
			next[k+delta] = v
		} else {
			next[k] = v
		}
	}
	m.kanbanOffsets = next
}

// initActiveContainerState initializes table/kanban state when the active
// block is one of those types and the corresponding state is nil. Used by
// deleteBlock and other paths that change m.active without going through
// focusBlock (which has its own init logic).
func (m *Model) initActiveContainerState() {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}
	bt := m.blocks[m.active].Type
	if bt == block.Table && m.table == nil {
		m.table = initTable(m.blocks[m.active].Content)
		cw := m.tableCellTAWidth()
		m.table.loadCell(&m.textareas[m.active], cw, false)
		m.cursorCmd = m.textareas[m.active].Focus()
	}
	if bt == block.Kanban && m.kanban == nil {
		m.kanban = newKanbanState(m.blocks[m.active].Content)
		if m.kanbanOffsets != nil {
			m.kanban.colOffset = m.kanbanOffsets[m.active]
		}
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		m.textareas[m.active].Blur()
	}
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
	m.textareas[idx-1].SetWidth(m.contentWidth(m.blocks[idx-1]))
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

// blockBundle returns the range [start, end) of the bundle rooted at idx.
// For list items, this includes all consecutive deeper-indented children.
// For non-list items, returns just [idx, idx+1).
func (m *Model) blockBundle(idx int) (int, int) {
	end := idx + 1
	if m.blocks[idx].Type.IsListItem() {
		parentIndent := m.blocks[idx].Indent
		for end < len(m.blocks) && m.blocks[end].Type.IsListItem() && m.blocks[end].Indent > parentIndent {
			end++
		}
	}
	return idx, end
}

// blockBundleUp returns the range [start, end) of the bundle that sits
// immediately above idx. For list items, it walks backwards past any
// deeper-indented children to find the sibling root at the same indent
// level as idx, then returns that root's full bundle.
func (m *Model) blockBundleUp(idx int) (int, int) {
	if idx <= 0 {
		return 0, 0
	}
	above := idx - 1
	if !m.blocks[above].Type.IsListItem() {
		return above, above + 1
	}
	// Walk backwards past items deeper than our indent level to find
	// the sibling root.
	activeIndent := m.blocks[idx].Indent
	root := above
	for root > 0 && m.blocks[root].Indent > activeIndent && m.blocks[root-1].Type.IsListItem() {
		root--
	}
	if m.blocks[root].Indent == activeIndent {
		return m.blockBundle(root)
	}
	// No sibling at our level — swap with the single block above.
	return above, above + 1
}

// rotateBlocks swaps two adjacent segments [aStart, aEnd) and [aEnd, bEnd)
// in both m.blocks and m.textareas so that segment B comes first.
func (m *Model) rotateBlocks(aStart, aEnd, bEnd int) {
	aLen := aEnd - aStart
	bLen := bEnd - aEnd
	total := aLen + bLen
	secB := make([]block.Block, total)
	secT := make([]textarea.Model, total)
	copy(secB, m.blocks[aStart:bEnd])
	copy(secT, m.textareas[aStart:bEnd])
	for i := 0; i < bLen; i++ {
		m.blocks[aStart+i] = secB[aLen+i]
		m.textareas[aStart+i] = secT[aLen+i]
	}
	for i := 0; i < aLen; i++ {
		m.blocks[aStart+bLen+i] = secB[i]
		m.textareas[aStart+bLen+i] = secT[i]
	}
}

// swapBlocks moves the active block's bundle up or down past the adjacent
// bundle. delta is -1 for up, +1 for down.
func (m *Model) swapBlocks(delta int) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}

	bStart, bEnd := m.blockBundle(m.active)

	// Sync the active cell into the block's serialized content so the
	// swap persists user edits. The textarea itself must stay as the
	// cell view — clobbering it with the full serialized table would
	// cause the next render to draw the whole table inside one cell.
	if m.table != nil && m.blocks[m.active].Type == block.Table {
		m.table.syncCell(m.textareas[m.active])
		m.blocks[m.active].Content = m.table.serialize()
	}

	// Sync textarea content in the active bundle.
	for i := bStart; i < bEnd; i++ {
		if m.table != nil && i == m.active && m.blocks[i].Type == block.Table {
			continue // Already synced above.
		}
		m.blocks[i].Content = m.textareas[i].Value()
	}

	if delta < 0 {
		if bStart <= 0 {
			return
		}
		aboveStart, aboveEnd := m.blockBundleUp(bStart)
		for i := aboveStart; i < aboveEnd; i++ {
			m.blocks[i].Content = m.textareas[i].Value()
		}
		m.textareas[m.active].Blur()
		m.rotateBlocks(aboveStart, aboveEnd, bEnd)
		m.active -= aboveEnd - aboveStart
		m.cursorCmd = m.textareas[m.active].Focus()
	} else {
		if bEnd >= len(m.blocks) {
			return
		}
		_, belowEnd := m.blockBundle(bEnd)
		for i := bEnd; i < belowEnd; i++ {
			m.blocks[i].Content = m.textareas[i].Value()
		}
		m.textareas[m.active].Blur()
		m.rotateBlocks(bStart, bEnd, belowEnd)
		m.active += belowEnd - bEnd
		m.cursorCmd = m.textareas[m.active].Focus()
	}
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

	// Definition block: same pattern as code/quote — Enter inserts a
	// newline via the textarea natively. Empty last line exits to paragraph.
	// Empty block converts to paragraph.
	if bt == block.DefinitionList {
		if content == "" || content == "\n" {
			m.blocks[m.active].Type = block.Paragraph
			m.blocks[m.active].Content = ""
			newTA := newTextareaForBlock(m.blocks[m.active], m.width)
			m.cursorCmd = newTA.Focus()
			m.textareas[m.active] = newTA
			return
		}

		lines := strings.Split(content, "\n")
		cursorLine := ta.Line()
		isLastLine := cursorLine >= len(lines)-1
		currentLineEmpty := cursorLine < len(lines) && lines[cursorLine] == ""

		if isLastLine && currentLineEmpty && len(lines) > 1 {
			// Empty definition line: trim it and exit to paragraph.
			trimmed := strings.Join(lines[:len(lines)-1], "\n")
			ta.SetValue(trimmed)
			m.blocks[m.active].Content = trimmed
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
			m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
			return
		}

		// Term line: if a definition line already exists below, just move
		// the cursor down instead of inserting a duplicate empty line.
		if cursorLine == 0 && len(lines) > 1 {
			ta.CursorDown()
			ta.CursorEnd()
			m.cursorCmd = ta.Focus()
			return
		}

		// Otherwise insert a newline within the block (new def line).
		m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount() + 1)
		m.textareas[m.active], _ = m.textareas[m.active].Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
		return
	}

	// Empty multi-line or embed block: convert to paragraph.
	// For code blocks, content is "\n" when empty (language separator).
	if bt == block.CodeBlock || bt == block.Quote || bt == block.Callout || bt == block.Embed {
		if strings.TrimRight(content, "\n") == "" {
			m.blocks[m.active].Type = block.Paragraph
			m.blocks[m.active].Content = ""
			newTA := newTextareaForBlock(m.blocks[m.active], m.width)
			m.cursorCmd = newTA.Focus()
			m.textareas[m.active] = newTA
			return
		}
	}

	// Link: Enter opens the URL prompt. The textarea is just storage; edits
	// happen in the bottom prompt, not inline.
	if bt == block.Link {
		ta.Blur()
		_, url := block.ExtractLink(content)
		m.linkPrompt.open(m.active, linkPromptURL, url, false)
		return
	}

	// Code block / Quote / Callout: Enter inserts a newline within the block.
	// On an empty last line, exit the block by trimming the empty line
	// and inserting a new paragraph below.
	if bt == block.CodeBlock || bt == block.Quote || bt == block.Callout {
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

	// Empty list item: outdent if indented, otherwise convert to paragraph.
	if content == "" && bt.IsListItem() {
		if m.blocks[m.active].Indent > 0 {
			m.blocks[m.active].Indent--
			m.textareas[m.active].SetWidth(m.contentWidth(m.blocks[m.active]))
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
			return
		}
		m.blocks[m.active].Type = block.Paragraph
		m.blocks[m.active].Checked = false
		m.blocks[m.active].Indent = 0
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

	// Create new block with text after cursor, inheriting indent for lists.
	newBlock := block.Block{Type: newType, Content: after}
	if newType.IsListItem() {
		newBlock.Indent = m.blocks[m.active].Indent
	}
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
//
// Unified behavior:
//   - Table: no-op (cells handle their own editing).
//   - Divider: delete block, focus previous.
//   - List items: outdent if indented, else convert to paragraph.
//   - Any other non-paragraph type: convert to paragraph (unwrap formatting).
//     Code blocks keep body only (drop language line). Definition keeps term only.
//   - Empty paragraph: delete block, focus previous.
//   - Non-empty paragraph: merge with previous block.
func (m *Model) handleBackspace() bool {
	if m.active < 0 || m.active >= len(m.blocks) {
		return false
	}

	bt := m.blocks[m.active].Type

	// Divider and Link cards are selected as a unit. Their textarea cursor is
	// hidden storage, so deletion must not depend on its current position.
	if bt == block.Divider || bt == block.Link {
		m.pushUndo()
		m.deleteBlock(m.active)
		m.textareas[m.active].MoveToEnd()
		return true
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

	// Table: at cell (0,0) convert to paragraph; otherwise no-op.
	if bt == block.Table {
		if m.table == nil || m.table.row > 0 || m.table.col > 0 {
			return true
		}
		m.pushUndo()
		m.table.syncCell(*ta)
		keepContent := m.table.cells[0][0]
		m.table = nil
		m.convertToParagraph(keepContent)
		return true
	}

	// List items: outdent if indented, else convert to paragraph.
	if bt.IsListItem() {
		m.pushUndo()
		if m.blocks[m.active].Indent > 0 {
			m.blocks[m.active].Indent--
			m.textareas[m.active].SetWidth(m.contentWidth(m.blocks[m.active]))
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
			return true
		}
		m.convertToParagraph(content)
		return true
	}

	// Any non-paragraph type: convert to paragraph (unwrap).
	if bt != block.Paragraph {
		m.pushUndo()
		// Determine what content to keep.
		keepContent := content
		switch bt {
		case block.CodeBlock:
			// Drop the language line, keep body only.
			if idx := strings.Index(content, "\n"); idx >= 0 {
				keepContent = content[idx+1:]
			}
		case block.DefinitionList:
			// Keep only the term line.
			keepContent, _ = block.ExtractDefinition(content)
		}
		m.convertToParagraph(keepContent)
		return true
	}

	// --- From here, block is a Paragraph. ---

	// Empty paragraph: delete it, focus previous.
	if content == "" {
		if m.active == 0 && len(m.blocks) <= 1 {
			return true // Last block, nothing to do.
		}
		m.pushUndo()
		m.deleteBlock(m.active)
		m.textareas[m.active].MoveToEnd()
		return true
	}

	// Non-empty paragraph at position 0: merge with previous block.
	if m.active == 0 {
		return false
	}

	// If previous block is a divider, just delete the divider.
	if m.blocks[m.active-1].Type == block.Divider {
		m.pushUndo()
		m.deleteBlock(m.active - 1)
		return true
	}

	// Don't merge into multi-line container blocks.
	prev := m.blocks[m.active-1].Type
	if prev == block.CodeBlock || prev == block.Quote || prev == block.DefinitionList || prev == block.Callout || prev == block.Table || prev == block.Kanban {
		return false
	}

	m.pushUndo()
	m.mergeBlockUp(m.active)
	return true
}

// convertToParagraph changes the active block to a paragraph with the given
// content, rebuilds its textarea, and places the cursor at position 0.
func (m *Model) convertToParagraph(content string) {
	m.blocks[m.active].Type = block.Paragraph
	m.blocks[m.active].Content = content
	m.blocks[m.active].Checked = false
	m.blocks[m.active].Indent = 0
	m.blocks[m.active].Variant = 0
	newTA := newTextareaForBlock(m.blocks[m.active], m.width)
	newTA.SetValue(content)
	m.cursorCmd = newTA.Focus()
	newTA.SetCursorColumn(0)
	m.textareas[m.active] = newTA
}

// cutBlock removes the active block and stores it in the block clipboard.
func (m *Model) cutBlock() {
	// Sync table cell before cutting.
	if m.table != nil && m.blocks[m.active].Type == block.Table {
		m.table.syncCell(m.textareas[m.active])
		m.blocks[m.active].Content = m.table.serialize()
		m.table = nil
	}

	if len(m.blocks) <= 1 {
		// Don't remove the last block; store it and clear it instead.
		b := m.blocks[m.active]
		if b.Type != block.Table {
			b.Content = m.textareas[m.active].Value()
		}
		m.blockClip = &b
		m.blocks[m.active] = block.Block{Type: block.Paragraph, Content: ""}
		m.textareas[m.active].SetValue("")
		return
	}

	b := m.blocks[m.active]
	if b.Type != block.Table {
		b.Content = m.textareas[m.active].Value()
	}
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

// copyActiveBlockContents copies the current block's text to the system
// clipboard. Table blocks return the serialized markdown grid; all others
// return the live textarea value so partial edits copy too. Returns a
// status message describing the result.
func (m *Model) copyActiveBlockContents() (string, statusKind) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return "Nothing to copy", statusWarning
	}
	var content string
	if m.blocks[m.active].Type == block.Table && m.table != nil {
		m.table.syncCell(m.textareas[m.active])
		content = m.table.serialize()
	} else {
		content = m.textareas[m.active].Value()
	}
	if content == "" {
		return "Block is empty", statusWarning
	}
	if err := clipboard.Copy(content); err != nil {
		return fmt.Sprintf("Could not copy: %s", err), statusError
	}
	return "Block copied to clipboard", statusSuccess
}

// applyPaletteSelection changes the active block's type to the selected
// palette item type. Special handling is applied for dividers and code blocks.
func (m *Model) applyPaletteSelection(bt block.BlockType) {
	if m.active < 0 || m.active >= len(m.blocks) {
		return
	}
	m.blocks[m.active].Type = bt
	if !bt.IsListItem() {
		m.blocks[m.active].Indent = 0
	}
	m.textareas[m.active].SetWidth(m.contentWidth(m.blocks[m.active]))

	if bt == block.Divider {
		m.blocks[m.active].Content = ""
		m.textareas[m.active].SetValue("")
	}
	// Ensure new code blocks have a body line so the user can arrow down
	// from the title into the code area immediately.
	if bt == block.CodeBlock && !strings.Contains(m.textareas[m.active].Value(), "\n") {
		m.textareas[m.active].SetValue(m.textareas[m.active].Value() + "\n")
		m.textareas[m.active].MoveToBegin()
		m.cursorCmd = m.textareas[m.active].Focus()
	}
	// Ensure new definition list blocks have term\ndefinition structure.
	// Place cursor on the term line so the user types the term first.
	if bt == block.DefinitionList && !strings.Contains(m.textareas[m.active].Value(), "\n") {
		m.textareas[m.active].SetValue(m.textareas[m.active].Value() + "\n")
		m.textareas[m.active].MoveToBegin()
		m.cursorCmd = m.textareas[m.active].Focus()
	}
	if bt == block.Link {
		if !strings.Contains(m.textareas[m.active].Value(), "\n") {
			m.textareas[m.active].SetValue("\n")
			m.textareas[m.active].MoveToBegin()
		}
		m.textareas[m.active].Blur()
		// Sequential wizard: prompt for URL first; on commit, chain to title.
		title, url := block.ExtractLink(m.textareas[m.active].Value())
		_ = title
		m.linkPrompt.open(m.active, linkPromptURL, url, true)
	}
	// Open the note picker for embed blocks so the user can browse targets.
	if bt == block.Embed && m.config.ListEmbedTargets != nil {
		targets := m.config.ListEmbedTargets()
		if len(targets) > 0 {
			m.embedPicker.open(targets)
		}
	}
	// Table: set default template and init table state.
	if bt == block.Table {
		content := m.textareas[m.active].Value()
		if content == "" || !strings.Contains(content, "|") {
			// Set a default 2x2 table template.
			content = "|  |  |\n| --- | --- |\n|  |  |"
		}
		m.blocks[m.active].Content = content
		m.textareas[m.active].SetValue(content)
		m.table = initTable(content)
		cw := m.tableCellTAWidth()
		m.table.loadCell(&m.textareas[m.active], cw, false)
		m.cursorCmd = m.textareas[m.active].Focus()
	}
	// Kanban: insert default board and enter card-selection mode.
	if bt == block.Kanban {
		m.blocks[m.active].Content = block.DefaultKanbanContent
		m.textareas[m.active].SetValue(block.DefaultKanbanContent)
		m.textareas[m.active].Blur()
		m.kanban = newKanbanState(m.blocks[m.active].Content)
	}
	m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
}

// activePicker returns whichever picker is currently visible, or nil.
func (m *Model) activePicker() *ui.Picker {
	if m.palette.Visible {
		return &m.palette.Picker
	}
	if m.embedPicker.Visible {
		return &m.embedPicker.Picker
	}
	if m.defLookup.Visible {
		return &m.defLookup.Picker
	}
	return nil
}

// handlePickerSelect dispatches the "enter" action to whichever picker is active.
func (m *Model) handlePickerSelect() {
	switch {
	case m.palette.Visible:
		if sel := m.palette.selected(); sel != nil {
			m.pushUndo()
			m.applyPaletteSelection(sel.Type)
		}
		m.palette.close()

	case m.embedPicker.Visible:
		if sel := m.embedPicker.selected(); sel != "" {
			m.textareas[m.active].SetValue(sel)
			m.cursorCmd = m.textareas[m.active].Focus()
		}
		m.embedPicker.Close()

	case m.defLookup.Visible:
		sel := m.defLookup.selected()
		m.defLookup.close()
		if sel == nil {
			return
		}
		if sel.BlockIdx >= 0 {
			// Local definition: focus the block directly.
			if m.viewMode {
				m.active = sel.BlockIdx
				if sel.BlockIdx < len(m.blockLineOffsets) {
					m.viewport.SetYOffset(m.blockLineOffsets[sel.BlockIdx])
				}
			} else {
				m.focusBlock(sel.BlockIdx)
			}
			return
		}
		// Remote definition: show a focused preview of just this
		// definition. Enter inside the preview jumps to the source note.
		m.openDefPreview(sel.Notebook, sel.Note, sel.Term, sel.Definition)
	}
}

// remoteDefinitions returns definition entries from notes other than the
// current one, or nil when the host hasn't wired up the lookup.
func (m Model) remoteDefinitions() []DefinitionEntry {
	if m.config.ListAllDefinitions == nil {
		return nil
	}
	return m.config.ListAllDefinitions()
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

	case clipboardReadErrMsg:
		m.status = "Paste failed: " + msg.err.Error()
		m.statusStyle = statusError
		return m, m.scheduleStatusDismiss()

	case tea.MouseMotionMsg:
		if m.embedModal.visible {
			contentStartY := m.embedModal.sheetStartY + 2
			if msg.Y >= contentStartY {
				contentLine := m.embedModal.scroll + (msg.Y - contentStartY)
				idx := m.embedModal.blockAtLine(contentLine)
				old := m.embedModal.hoverBlock
				m.embedModal.hoverBlock = idx
				if idx != old {
					// Re-render for hover feedback on interactive blocks.
					needsUpdate := false
					for _, checkIdx := range []int{idx, old} {
						if checkIdx >= 0 && checkIdx < len(m.embedModal.blocks) {
							bt := m.embedModal.blocks[checkIdx].Type
							if bt == block.Checklist || bt == block.Embed {
								needsUpdate = true
								break
							}
						}
					}
					if needsUpdate {
						m.renderEmbedSheetContent()
					}
				}
			} else {
				m.embedModal.hoverBlock = -1
			}
			return m, nil
		}
		if m.viewMode {
			// Track hover state for checklist/embed visual feedback. Drop
			// hover when the cursor is in the centered-column margin so
			// the highlight matches click activation bounds.
			leftPad, contentWidth := m.viewContentXRange()
			oldHover := m.hoverBlock
			idx := -1
			if msg.X >= leftPad && msg.X < leftPad+contentWidth {
				hoverY := m.viewport.YOffset() + msg.Y
				idx = m.blockIndexAtLine(hoverY)
			}
			m.hoverBlock = idx

			// Re-render if hovering over a different interactive block.
			if idx != oldHover {
				needsUpdate := false
				for _, checkIdx := range []int{idx, oldHover} {
					if checkIdx >= 0 && checkIdx < len(m.blocks) {
						bt := m.blocks[checkIdx].Type
						if bt == block.Checklist || bt == block.Embed {
							needsUpdate = true
							break
						}
					}
				}
				if needsUpdate {
					m.updateViewport()
				}
			}
		} else {
			m.hoverBlock = -1
		}
		return m, nil

	case tea.MouseClickMsg:
		if m.embedModal.visible {
			// Click above the sheet (context area) dismisses it.
			if msg.Y < m.embedModal.sheetStartY {
				m.embedModal.close()
				m.updateViewport()
				return m, nil
			}
			// Map click Y to a content line, accounting for scroll and sheet header.
			// Sheet layout: sheetStartY=sep, +1=title, +2..=content
			contentStartY := m.embedModal.sheetStartY + 2
			if msg.Button == tea.MouseLeft && msg.Y >= contentStartY {
				contentLine := m.embedModal.scroll + (msg.Y - contentStartY)
				idx := m.embedModal.blockAtLine(contentLine)
				if idx >= 0 && idx < len(m.embedModal.blocks) {
					if m.embedModal.blocks[idx].Type == block.Checklist {
						m.embedModal.blocks[idx].Checked = !m.embedModal.blocks[idx].Checked
						// Save changes back to the referenced note.
						if m.config.SaveEmbed != nil {
							content := block.Serialize(m.embedModal.blocks)
							_ = m.config.SaveEmbed(m.embedModal.path, content)
						}
						m.renderEmbedSheetContent()
					}
				}
			}
			return m, nil
		}
		if m.viewMode {
			// Reject clicks that land in the centered-column margin so
			// stray clicks outside the rendered block content don't fire
			// checklist toggles or embed opens.
			leftPad, contentWidth := m.viewContentXRange()
			if msg.X < leftPad || msg.X >= leftPad+contentWidth {
				m.hoverBlock = -1
				return m, nil
			}
			// Track hover state for checklist visual feedback.
			hoverY := m.viewport.YOffset() + msg.Y
			idx := m.blockIndexAtLine(hoverY)
			m.hoverBlock = idx

			if msg.Button == tea.MouseLeft && idx >= 0 && idx < len(m.blocks) {
				// Left-click on a checklist block toggles its checked state.
				if m.blocks[idx].Type == block.Checklist {
					m.pushUndo()
					if idx < len(m.textareas) {
						m.blocks[idx].Content = m.textareas[idx].Value()
					}
					m.toggleChecklist(idx)
					if m.sortChecked {
						m.sortCheckedToBottom()
					}
					m.updateViewport()
					return m, nil
				}
				// Left-click on an embed block opens the referenced note.
				if m.blocks[idx].Type == block.Embed {
					m.openEmbedModal(idx)
					return m, nil
				}
				if m.blocks[idx].Type == block.Link {
					_, u := block.ExtractLink(m.blocks[idx].Content)
					if u == "" {
						// No URL yet — open the URL prompt so the user can fill it in.
						if idx < len(m.textareas) {
							m.textareas[idx].Blur()
						}
						m.linkPrompt.open(idx, linkPromptURL, u, false)
						m.updateViewport()
						return m, nil
					}
					if err := openURL(u); err == nil {
						m.status = "Opened: " + u
						m.statusStyle = statusSuccess
					} else {
						m.status = "Open failed: " + err.Error()
						m.statusStyle = statusError
					}
					return m, m.scheduleStatusDismiss()
				}
			}
		} else {
			m.hoverBlock = -1
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.embedModal.visible {
			// Scroll the embed modal content.
			if msg.Button == tea.MouseWheelDown {
				m.embedModal.scroll++
				m.clampEmbedScroll()
			} else if msg.Button == tea.MouseWheelUp {
				if m.embedModal.scroll > 0 {
					m.embedModal.scroll--
				}
			}
			return m, nil
		}
		// View mode captures mouse — forward wheel to viewport for scrolling.
		if m.viewMode {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.PasteMsg:
		// Bracketed-paste content. Route to whatever text widget is
		// currently accepting input so multi-line clipboards land in
		// one shot instead of being swallowed.
		if m.linkPrompt.visible {
			m.linkPrompt.input.InsertText(msg.Content)
			m.updateViewport()
			return m, nil
		}
		if picker := m.activePicker(); picker != nil {
			picker.AddFilterText(msg.Content)
			m.updateViewport()
			return m, nil
		}
		if m.viewMode || m.showHelp || m.quitPrompt || m.embedModal.visible ||
			m.defPreview.visible {
			return m, nil
		}
		if m.kanban != nil && m.kanban.edit {
			var c tea.Cmd
			m.kanban.editTA, c = m.kanban.editTA.Update(msg)
			m.kanban.editTA.SetHeight(m.kanban.editTA.VisualLineCount())
			m.updateViewport()
			return m, c
		}
		if m.active >= 0 && m.active < len(m.textareas) {
			m.pushUndo()
			var c tea.Cmd
			m.textareas[m.active], c = m.textareas[m.active].Update(msg)
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
			m.updateViewport()
			return m, c
		}
		return m, nil

	case tea.KeyPressMsg:
		// When help overlay is showing, Ctrl+G/Esc close it, Ctrl+C quits.
		if m.showHelp {
			switch msg.String() {
			case "ctrl+g", "esc":
				m.showHelp = false
				m.updateViewport()
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
			case "ctrl+s":
				m.quitPrompt = false
				m.status = ""
				m.statusStyle = statusNone
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
			case "y", "Y", "enter":
				m.quitPrompt = false
				if m.pendingJump != "" {
					m.jumpTarget = m.pendingJump
					m.pendingJump = ""
				}
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
				if m.pendingJump != "" {
					m.jumpTarget = m.pendingJump
					m.pendingJump = ""
				}
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.quitPrompt = false
				m.pendingJump = ""
				m.status = ""
				m.statusStyle = statusNone
				return m, nil
			}
			return m, nil
		}

		// Unified picker key handling: palette, embed picker, or def lookup.
		if picker := m.activePicker(); picker != nil {
			if msg.String() == "ctrl+v" {
				return m, readClipboardCmd()
			}
			// Check trigger-key-closes before generic handling:
			// "/" re-typed closes palette, ":" re-typed closes deflookup.
			if m.palette.Visible && msg.Code == '/' {
				m.palette.close()
				m.updateViewport()
				return m, nil
			}
			if m.defLookup.Visible && msg.Code == ':' {
				m.defLookup.close()
				m.updateViewport()
				return m, nil
			}

			handled, closed := ui.HandlePickerKey(picker, msg.String(), msg.Text, msg.Code)
			if handled {
				if !closed && msg.String() == "enter" {
					m.handlePickerSelect()
				}
				m.updateViewport()
				return m, nil
			}
			return m, nil
		}

		// Clear transient status on any keypress.
		if m.statusStyle == statusSuccess {
			m.status = ""
			m.statusStyle = statusNone
		}

		// Link prompt: single-field input footer (URL or title).
		if m.linkPrompt.visible {
			key := msg.String()
			switch {
			case key == "ctrl+v":
				return m, readClipboardCmd()
			case key == "esc":
				m.linkPrompt.close()
				m.updateViewport()
				return m, nil
			case key == "ctrl+c":
				m.linkPrompt.close()
				if m.modified() {
					m.quitPrompt = true
					m.status = "Save before quitting? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit
			case key == "enter":
				idx := m.linkPrompt.blockIdx
				value := m.linkPrompt.input.Value()
				field := m.linkPrompt.field
				chain := m.linkPrompt.chain
				m.linkPrompt.close()
				if idx >= 0 && idx < len(m.blocks) {
					title, url := block.ExtractLink(m.blocks[idx].Content)
					if field == linkPromptURL {
						url = value
					} else {
						title = value
					}
					m.pushUndo()
					content := url
					if title != "" {
						content = url + "\n" + title
					}
					m.blocks[idx].Content = content
					if idx < len(m.textareas) {
						m.textareas[idx].SetValue(content)
						m.textareas[idx].SetHeight(m.textareas[idx].VisualLineCount())
					}
					if field == linkPromptURL && chain {
						m.linkPrompt.open(idx, linkPromptTitle, title, false)
					}
				}
				m.updateViewport()
				return m, nil
			}
			if m.linkPrompt.input.HandleKey(msg) {
				return m, nil
			}
			return m, nil
		}

		// Embed modal: intercept keys when overlay is showing.
		if m.embedModal.visible {
			switch msg.String() {
			case "esc", "q":
				m.embedModal.close()
				m.updateViewport()
			case "enter":
				// Jump to the previewed note. If current note has unsaved
				// changes, trigger the save-prompt flow with pendingJump set
				// so the "Y" path routes to save-then-jump instead of quit.
				target := m.embedModal.path
				if target == "" {
					return m, nil
				}
				m.embedModal.close()
				if m.modified() {
					m.pendingJump = target
					m.quitPrompt = true
					m.status = "Save before opening? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.jumpTarget = target
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.embedModal.scroll > 0 {
					m.embedModal.scroll--
				}
			case "down", "j":
				m.embedModal.scroll++
				m.clampEmbedScroll()
			case "pgup":
				m.embedModal.scroll -= m.height / 2
				if m.embedModal.scroll < 0 {
					m.embedModal.scroll = 0
				}
			case "pgdown":
				m.embedModal.scroll += m.height / 2
				m.clampEmbedScroll()
			case "ctrl+c":
				m.embedModal.close()
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

		// Definition preview: focused pane for a single remote definition.
		if m.defPreview.visible {
			switch msg.String() {
			case "esc", "q":
				m.defPreview.close()
				m.updateViewport()
				return m, nil
			case "enter":
				target := m.defPreview.path
				m.defPreview.close()
				if target == "" {
					m.updateViewport()
					return m, nil
				}
				if m.modified() {
					m.pendingJump = target
					m.quitPrompt = true
					m.status = "Save before opening? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.jumpTarget = target
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// View mode: intercept keys early.
		if m.viewMode {
			switch msg.String() {
			case "ctrl+h":
				m.sortChecked = !m.sortChecked
				if m.sortChecked {
					m.sortCheckedToBottom()
					m.status = "Sort checked: on"
				} else {
					m.status = "Sort checked: off"
				}
				m.statusStyle = statusSuccess
				m.persistSortChecked()
				m.updateViewport()
				return m, m.scheduleStatusDismiss()
			case "ctrl+r":
				m.viewMode = false
				m.hoverBlock = -1
				// Re-focus the previously active block to restore cursor.
				if m.active >= 0 && m.active < len(m.textareas) {
					m.cursorCmd = m.textareas[m.active].Focus()
					// Re-init table state if returning to a table block.
					if m.blocks[m.active].Type == block.Table {
						m.table = initTable(m.blocks[m.active].Content)
						cw := m.tableCellTAWidth()
						m.table.loadCell(&m.textareas[m.active], cw, false)
						m.cursorCmd = m.textareas[m.active].Focus()
					}
				}
				m.updateViewport()
				return m, nil
			case "esc":
				m.viewMode = false
				m.hoverBlock = -1
				if m.active >= 0 && m.active < len(m.textareas) {
					m.cursorCmd = m.textareas[m.active].Focus()
					if m.blocks[m.active].Type == block.Table {
						m.table = initTable(m.blocks[m.active].Content)
						cw := m.tableCellTAWidth()
						m.table.loadCell(&m.textareas[m.active], cw, false)
						m.cursorCmd = m.textareas[m.active].Focus()
					}
				}
				m.updateViewport()
				return m, nil
			case "ctrl+c":
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
				m.updateViewport()
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
			case "left":
				if m.viewKanbanOffset > 0 {
					m.viewKanbanOffset--
					m.updateViewport()
				}
				return m, nil
			case "right":
				maxOff := m.maxViewKanbanOffset()
				if m.viewKanbanOffset < maxOff {
					m.viewKanbanOffset++
					m.updateViewport()
				}
				return m, nil
			default:
				if msg.Code == ':' {
					m.defLookup.open(m.blocks, m.remoteDefinitions())
					m.updateViewport()
					return m, nil
				}
			}
			// Swallow everything else in view mode.
			return m, nil
		}

		// Kanban-block intercept: consumes most keys when a kanban is active.
		// Save/quit/help/view-mode still work via the main switch below.
		if m.kanban != nil && m.active >= 0 && m.active < len(m.blocks) && m.blocks[m.active].Type == block.Kanban {
			handled, kanbanCmd := m.handleKanbanKey(msg)
			if handled {
				m.updateViewport()
				return m, kanbanCmd
			}
		}

		switch msg.String() {
		case "ctrl+r":
			m.viewMode = true
			m.hoverBlock = -1
			// Blur the active textarea but preserve m.active.
			if m.active >= 0 && m.active < len(m.textareas) {
				if m.table != nil && m.blocks[m.active].Type == block.Table {
					m.table.syncCell(m.textareas[m.active])
					m.blocks[m.active].Content = m.table.serialize()
					m.textareas[m.active].SetValue(m.blocks[m.active].Content)
				} else if m.kanban != nil && m.blocks[m.active].Type == block.Kanban {
					if m.kanban.edit {
						m.kanban.commitEdit()
					}
					m.blocks[m.active].Content = m.kanban.serialize()
					m.textareas[m.active].SetValue(m.blocks[m.active].Content)
				} else {
					m.blocks[m.active].Content = m.textareas[m.active].Value()
				}
				m.textareas[m.active].Blur()
			}
			m.updateViewport()
			return m, nil

		case "ctrl+g":
			m.showHelp = true
			m.updateViewport()
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

		case "alt+k":
			m.status, m.statusStyle = m.copyActiveBlockContents()
			return m, m.scheduleStatusDismiss()

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
			// Link blocks: up always navigates to the previous block,
			// never swaps between URL and title slots.
			if m.blocks[m.active].Type == block.Link {
				if m.active == 0 {
					m.pushUndo()
					m.insertBlockBefore(0, block.Block{Type: block.Paragraph})
					m.updateViewport()
					return m, nil
				}
				m.navigateUp()
				m.updateViewport()
				return m, nil
			}
			// Table: move to cell above, preserving horizontal position.
			if m.isAtFirstLine() && m.table != nil && m.table.row > 0 {
				ta := &m.textareas[m.active]
				charOffset := ta.LineInfo().CharOffset
				cw := m.tableCellTAWidth()
				m.table.syncCell(*ta)
				m.table.row--
				m.table.loadCell(ta, cw, true)
				li := ta.LineInfo()
				ta.SetCursorColumn(li.StartColumn + charOffset)
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}
			if m.isAtFirstLine() && m.active == 0 && m.blocks[0].Type != block.Paragraph {
				m.pushUndo()
				m.insertBlockBefore(0, block.Block{Type: block.Paragraph})
				m.updateViewport()
				return m, nil
			}
			if m.isAtFirstLine() && m.active > 0 {
				m.navigateUp()
				m.updateViewport()
				return m, nil
			}

		case "down":
			// Link blocks: down always navigates to the next block,
			// never swaps between URL and title slots.
			if m.blocks[m.active].Type == block.Link {
				if m.active < len(m.textareas)-1 {
					m.navigateDown()
					m.updateViewport()
				}
				return m, nil
			}
			// Table: move to cell below, preserving horizontal position.
			if m.isAtLastLine() && m.table != nil && m.table.row < len(m.table.cells)-1 {
				ta := &m.textareas[m.active]
				charOffset := ta.LineInfo().CharOffset
				cw := m.tableCellTAWidth()
				m.table.syncCell(*ta)
				m.table.row++
				m.table.loadCell(ta, cw, false)
				ta.SetCursorColumn(charOffset)
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}
			if m.isAtLastLine() && m.active < len(m.textareas)-1 {
				m.navigateDown()
				m.updateViewport()
				return m, nil
			}

		case "alt+up":
			// Table: at top line of cell, move to cell above
			// (preserving horizontal position). Falls through to block
			// swap when already on the top row.
			if m.isAtFirstLine() && m.table != nil && m.table.row > 0 {
				ta := &m.textareas[m.active]
				charOffset := ta.LineInfo().CharOffset
				cw := m.tableCellTAWidth()
				m.table.syncCell(*ta)
				m.table.row--
				m.table.loadCell(ta, cw, true)
				li := ta.LineInfo()
				ta.SetCursorColumn(li.StartColumn + charOffset)
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}
			m.pushUndo()
			m.swapBlocks(-1)
			m.updateViewport()
			return m, nil

		case "alt+down":
			// Table: at bottom line of cell, move to cell below
			// (preserving horizontal position). Falls through to block
			// swap when already on the last row.
			if m.isAtLastLine() && m.table != nil && m.table.row < len(m.table.cells)-1 {
				ta := &m.textareas[m.active]
				charOffset := ta.LineInfo().CharOffset
				cw := m.tableCellTAWidth()
				m.table.syncCell(*ta)
				m.table.row++
				m.table.loadCell(ta, cw, false)
				ta.SetCursorColumn(charOffset)
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}
			m.pushUndo()
			m.swapBlocks(1)
			m.updateViewport()
			return m, nil

		case "ctrl+t":
			if m.active >= 0 && m.blocks[m.active].Type == block.Callout {
				m.pushUndo()
				m.blocks[m.active].Variant = m.blocks[m.active].Variant.Next()
				m.updateViewport()
				return m, nil
			}
			return m, nil

		case "ctrl+w", "alt+z":
			m.wordWrap = !m.wordWrap
			if m.wordWrap {
				m.status = "Word wrap on"
			} else {
				m.status = "Word wrap off"
			}
			m.statusStyle = statusSuccess
			if globalCfg, err := config.Load(); err == nil {
				globalCfg.WordWrap = config.BoolPtr(m.wordWrap)
				_ = config.Save(globalCfg)
			}
			m.resizeTextareas()
			m.updateViewport()
			return m, m.scheduleStatusDismiss()

		case "ctrl+h":
			m.sortChecked = !m.sortChecked
			if m.sortChecked {
				m.sortCheckedToBottom()
				m.status = "Sort checked: on"
			} else {
				m.status = "Sort checked: off"
			}
			m.statusStyle = statusSuccess
			m.persistSortChecked()
			m.updateViewport()
			return m, m.scheduleStatusDismiss()

		case "ctrl+u":
			ta := &m.textareas[m.active]
			info := ta.LineInfo()
			logicalCol := info.StartColumn + info.ColumnOffset
			if logicalCol > 0 {
				m.pushUndo()
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
			// Universal interact key — action depends on active block type.
			if m.active >= 0 && m.active < len(m.blocks) {
				switch m.blocks[m.active].Type {
				case block.Checklist:
					m.pushUndo()
					m.toggleChecklist(m.active)
					if m.sortChecked {
						m.sortCheckedToBottom()
					}
					m.updateViewport()
					return m, nil
				case block.Embed:
					m.openEmbedModal(m.active)
					return m, nil
				case block.DefinitionList:
					m.defLookup.open(m.blocks, m.remoteDefinitions())
					m.updateViewport()
					return m, nil
				case block.Link:
					content := m.blocks[m.active].Content
					if m.active < len(m.textareas) {
						content = m.textareas[m.active].Value()
					}
					_, u := block.ExtractLink(content)
					if u == "" {
						m.status = "No URL to open"
						m.statusStyle = statusWarning
						return m, m.scheduleStatusDismiss()
					}
					if err := openURL(u); err != nil {
						m.status = "Open failed: " + err.Error()
						m.statusStyle = statusError
					} else {
						m.status = "Opened: " + u
						m.statusStyle = statusSuccess
					}
					return m, m.scheduleStatusDismiss()
				}
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
			// Shift+Enter on single-line blocks (headings, lists) falls
			// through to regular Enter so users don't have to release Shift.
			// Ctrl+J stays a no-op here to preserve the "soft newline only"
			// contract it carries in other text editors.
			if msg.String() == "shift+enter" && m.active >= 0 && m.active < len(m.blocks) {
				m.pushUndo()
				m.handleEnter()
				m.updateViewport()
				return m, nil
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
				if ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0 {
					m.palette.openForBlock(m.active, m.blocks[m.active].Type, ta.Value() != "")
					m.updateViewport()
					return m, nil
				}
			}

			// Handle ":" at position 0 of an empty block to open definition lookup.
			if keyMsg.Code == ':' && len(keyMsg.Text) > 0 {
				ta := &m.textareas[m.active]
				if ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0 && ta.Value() == "" {
					m.defLookup.open(m.blocks, m.remoteDefinitions())
					m.updateViewport()
					return m, nil
				}
			}

			// On Link blocks, Tab opens the bottom-sheet editor for URL + title.
			// Tab → edit URL; Shift+Tab → edit title.
			if keyMsg.Code == tea.KeyTab && m.blocks[m.active].Type == block.Link {
				ta := &m.textareas[m.active]
				ta.Blur()
				title, url := block.ExtractLink(ta.Value())
				if keyMsg.Mod.Contains(tea.ModShift) {
					m.linkPrompt.open(m.active, linkPromptTitle, title, false)
				} else {
					m.linkPrompt.open(m.active, linkPromptURL, url, false)
				}
				m.updateViewport()
				return m, nil
			}

			// On Embed blocks, Tab opens the note picker.
			if keyMsg.Code == tea.KeyTab && m.blocks[m.active].Type == block.Embed {
				if m.config.ListEmbedTargets != nil {
					targets := m.config.ListEmbedTargets()
					if len(targets) > 0 {
						m.embedPicker.open(targets)
						m.updateViewport()
						return m, nil
					}
				}
			}

			// Table-specific key handling: only intercept Tab, Enter, and
			// boundary navigation. Everything else goes to the textarea.
			if m.table != nil && m.blocks[m.active].Type == block.Table {
				ta := &m.textareas[m.active]
				cw := m.tableCellTAWidth()

				if keyMsg.Code == tea.KeyTab {
					m.table.syncCell(*ta)
					if keyMsg.Mod.Contains(tea.ModShift) {
						m.table.prevCell()
						m.table.loadCell(ta, cw, true)
					} else {
						m.table.nextCell()
						m.table.loadCell(ta, cw, false)
					}
					m.cursorCmd = ta.Focus()
					m.updateViewport()
					return m, nil
				}

				if keyMsg.Code == tea.KeyEnter {
					m.table.syncCell(*ta)

					// Empty row → exit table and drop the row (matches the
					// "Enter on empty block exits" pattern used elsewhere).
					if m.table.isRowEmpty(m.table.row) {
						m.pushUndo()
						if len(m.table.cells) > 1 {
							m.table.deleteRow()
						}
						m.blocks[m.active].Content = m.table.serialize()
						m.textareas[m.active].SetValue(m.blocks[m.active].Content)
						m.table = nil
						m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
						m.updateViewport()
						return m, nil
					}

					m.table.row++
					if m.table.row >= len(m.table.cells) {
						newRow := make([]string, m.table.numCols())
						m.table.cells = append(m.table.cells, newRow)
					}
					m.table.loadCell(ta, cw, false)
					m.cursorCmd = ta.Focus()
					m.updateViewport()
					return m, nil
				}

				// Left at position 0: move to previous cell.
				if keyMsg.Code == tea.KeyLeft && !keyMsg.Mod.Contains(tea.ModAlt) {
					if ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0 && ta.LineInfo().RowOffset == 0 {
						if m.table.row > 0 || m.table.col > 0 {
							m.table.syncCell(*ta)
							m.table.prevCell()
							m.table.loadCell(ta, cw, true)
							m.cursorCmd = ta.Focus()
							m.updateViewport()
							return m, nil
						}
					}
				}

				// Right at end of text: move to next cell.
				if keyMsg.Code == tea.KeyRight && !keyMsg.Mod.Contains(tea.ModAlt) {
					if m.isAtLastLine() {
						li := ta.LineInfo()
						lineLen := len([]rune(strings.Split(ta.Value(), "\n")[ta.Line()]))
						atEnd := li.StartColumn+li.ColumnOffset >= lineLen
						if atEnd && (m.table.row < len(m.table.cells)-1 || m.table.col < m.table.numCols()-1) {
							m.table.syncCell(*ta)
							m.table.nextCell()
							m.table.loadCell(ta, cw, false)
							m.cursorCmd = ta.Focus()
							m.updateViewport()
							return m, nil
						}
					}
				}

				// Home (or Cmd+Left on macOS terminals): if cursor is
				// already at the cell's line-start, jump to prev cell.
				if keyMsg.Code == tea.KeyHome {
					li := ta.LineInfo()
					atLineStart := li.ColumnOffset == 0 && li.RowOffset == 0 && ta.Line() == 0
					if atLineStart && (m.table.row > 0 || m.table.col > 0) {
						m.table.syncCell(*ta)
						m.table.prevCell()
						m.table.loadCell(ta, cw, true)
						m.cursorCmd = ta.Focus()
						m.updateViewport()
						return m, nil
					}
				}

				// End (or Cmd+Right on macOS terminals): if already at
				// line-end of the last line, jump to next cell.
				if keyMsg.Code == tea.KeyEnd {
					li := ta.LineInfo()
					lines := strings.Split(ta.Value(), "\n")
					lineRunes := []rune{}
					if ta.Line() < len(lines) {
						lineRunes = []rune(lines[ta.Line()])
					}
					atLineEnd := m.isAtLastLine() && li.StartColumn+li.ColumnOffset >= len(lineRunes)
					if atLineEnd && (m.table.row < len(m.table.cells)-1 || m.table.col < m.table.numCols()-1) {
						m.table.syncCell(*ta)
						m.table.nextCell()
						m.table.loadCell(ta, cw, false)
						m.cursorCmd = ta.Focus()
						m.updateViewport()
						return m, nil
					}
				}

				// Alt+Left / Ctrl+A at cell start: jump to previous cell.
				// macOS terminals remap Alt+Left → Ctrl+A, so we accept
				// all three forms. Same edge-jump as plain Left but
				// reachable in a single keypress.
				keyStr := keyMsg.String()
				if keyStr == "alt+left" || keyStr == "alt+b" || keyStr == "ctrl+a" {
					li := ta.LineInfo()
					atStart := ta.Line() == 0 && li.ColumnOffset == 0 && li.RowOffset == 0
					if atStart && (m.table.row > 0 || m.table.col > 0) {
						m.table.syncCell(*ta)
						m.table.prevCell()
						m.table.loadCell(ta, cw, true)
						m.cursorCmd = ta.Focus()
						m.updateViewport()
						return m, nil
					}
				}

				// Alt+Right / Ctrl+E at cell end: jump to next cell.
				if keyStr == "alt+right" || keyStr == "alt+f" || keyStr == "ctrl+e" {
					li := ta.LineInfo()
					lines := strings.Split(ta.Value(), "\n")
					lineRunes := []rune{}
					if ta.Line() < len(lines) {
						lineRunes = []rune(lines[ta.Line()])
					}
					atEnd := m.isAtLastLine() && li.StartColumn+li.ColumnOffset >= len(lineRunes)
					if atEnd && (m.table.row < len(m.table.cells)-1 || m.table.col < m.table.numCols()-1) {
						m.table.syncCell(*ta)
						m.table.nextCell()
						m.table.loadCell(ta, cw, false)
						m.cursorCmd = ta.Focus()
						m.updateViewport()
						return m, nil
					}
				}

				// Alt+R: insert row below.
				if keyMsg.Code == 'r' && keyMsg.Mod.Contains(tea.ModAlt) {
					m.pushUndo()
					m.table.syncCell(*ta)
					m.table.addRow()
					m.table.loadCell(ta, cw, false)
					m.cursorCmd = ta.Focus()
					m.updateViewport()
					return m, nil
				}

				// Alt+C: insert column after.
				if keyMsg.Code == 'c' && keyMsg.Mod.Contains(tea.ModAlt) {
					m.pushUndo()
					m.table.syncCell(*ta)
					m.table.addCol()
					cw = m.tableCellTAWidth() // recompute — column count changed
					m.table.loadCell(ta, cw, false)
					m.cursorCmd = ta.Focus()
					m.resizeTextareas()
					m.updateViewport()
					return m, nil
				}

				// Alt+Shift+Backspace: delete current row. Using a shift
				// variant keeps plain Alt+Backspace available for
				// word-delete, matching behavior in other blocks.
				if keyMsg.Code == tea.KeyBackspace && keyMsg.Mod.Contains(tea.ModAlt) && keyMsg.Mod.Contains(tea.ModShift) {
					m.pushUndo()
					m.table.syncCell(*ta)
					if m.table.deleteRow() {
						m.table.loadCell(ta, cw, false)
						m.cursorCmd = ta.Focus()
						m.updateViewport()
					}
					return m, nil
				}

				// Alt+Shift+D: delete current column. Plain Alt+D remains
				// word-delete-forward to match other blocks.
				if (keyMsg.Code == 'd' || keyMsg.Code == 'D') && keyMsg.Mod.Contains(tea.ModAlt) && keyMsg.Mod.Contains(tea.ModShift) {
					m.pushUndo()
					m.table.syncCell(*ta)
					if m.table.deleteCol() {
						cw = m.tableCellTAWidth()
						m.table.loadCell(ta, cw, false)
						m.cursorCmd = ta.Focus()
						m.resizeTextareas()
						m.updateViewport()
					}
					return m, nil
				}

				// Fall through — let normal textarea forwarding handle it.
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
				// Definition block: prevent deleting the term/definition
				// separator newline. Backspace at line 1 col 0 → jump to term.
				if m.blocks[m.active].Type == block.DefinitionList {
					ta := &m.textareas[m.active]
					info := ta.LineInfo()
					// Only intercept at the true start of raw line 1
					// (not a wrapped continuation where StartColumn > 0).
					if ta.Line() == 1 && info.ColumnOffset == 0 && info.StartColumn == 0 {
						ta.MoveToBegin()
						ta.CursorEnd()
						m.cursorCmd = ta.Focus()
						m.updateViewport()
						return m, nil
					}
				}
				// The textarea's backspace binding is an exact-match on
				// "backspace"; "shift+backspace" gets dropped. Rewrite so a
				// held Shift doesn't swallow the delete.
				if keyMsg.Mod.Contains(tea.ModShift) {
					msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
				}
			}
			if keyMsg.Code == tea.KeyDelete && (m.blocks[m.active].Type == block.Divider || m.blocks[m.active].Type == block.Link) {
				m.pushUndo()
				m.deleteBlock(m.active)
				m.textareas[m.active].MoveToEnd()
				m.updateViewport()
				return m, nil
			}

			// Divider: selected as a unit — no text input forwarded.
			// Enter and Backspace are handled above; everything else is ignored.
			if m.blocks[m.active].Type == block.Divider {
				return m, nil
			}

			// Shift+Tab: outdent list items, or remove leading spaces.
			if keyMsg.Code == tea.KeyTab && keyMsg.Mod.Contains(tea.ModShift) {
				ta := &m.textareas[m.active]
				bt := m.blocks[m.active].Type
				col := ta.LineInfo().CharOffset
				if bt.IsListItem() && m.blocks[m.active].Indent > 0 {
					m.pushUndo()
					m.blocks[m.active].Indent--
					ta.SetWidth(m.contentWidth(m.blocks[m.active]))
					ta.SetHeight(ta.VisualLineCount())
					ta.SetCursorColumn(col)
				} else {
					// Snap back to previous tab stop, removing only spaces.
					li := ta.LineInfo()
					actualCol := li.StartColumn + li.ColumnOffset
					target := li.ColumnOffset % 4
					if target == 0 {
						target = 4
					}
					val := ta.Value()
					runes := []rune(val)
					spaces := 0
					for spaces < target && actualCol-spaces-1 >= 0 && runes[actualCol-spaces-1] == ' ' {
						spaces++
					}
					if spaces > 0 {
						m.pushUndo()
						for i := 0; i < spaces; i++ {
							*ta, _ = ta.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
						}
						ta.SetHeight(ta.VisualLineCount())
					}
				}
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}

			// Tab: indent list items, or snap to next tab stop.
			if keyMsg.Code == tea.KeyTab {
				ta := &m.textareas[m.active]
				bt := m.blocks[m.active].Type
				if bt.IsListItem() {
					m.pushUndo()
					col := ta.LineInfo().CharOffset
					m.blocks[m.active].Indent++
					ta.SetWidth(m.contentWidth(m.blocks[m.active]))
					ta.SetHeight(ta.VisualLineCount())
					ta.SetCursorColumn(col)
				} else {
					m.pushUndo()
					col := ta.LineInfo().ColumnOffset
					spaces := 4 - (col % 4)
					ta.InsertString(strings.Repeat(" ", spaces))
				}
				m.cursorCmd = ta.Focus()
				m.updateViewport()
				return m, nil
			}
		}

		// Word/line deletes should get their own undo entry rather than
		// being silently batched with prior character-level typing.
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && m.undoDirty {
			isWordOrLineDelete := false
			switch {
			case keyMsg.Code == tea.KeyBackspace && keyMsg.Mod.Contains(tea.ModAlt):
				isWordOrLineDelete = true
			case keyMsg.Code == tea.KeyDelete && keyMsg.Mod.Contains(tea.ModAlt):
				isWordOrLineDelete = true
			case keyMsg.Code == 'd' && keyMsg.Mod.Contains(tea.ModAlt):
				isWordOrLineDelete = true
			}
			if isWordOrLineDelete {
				m.pushUndo()
				// Keep undoDirty true so the generic handler below
				// doesn't capture a second undo entry for this same action.
				m.undoDirty = true
			}
		}

		// Capture state before the textarea processes the keystroke so we
		// can detect whether content actually changed (vs. cursor movement).
		// Also snapshot the raw textarea value — for tables, captureState
		// stores the serialized table in blocks[active].Content, which
		// never equals the per-cell textarea value, so we can't compare
		// against it to detect real edits.
		var preState editorState
		var preValue string
		preStateCaptured := false
		if !m.undoDirty && m.active >= 0 && m.active < len(m.textareas) {
			preState = m.captureState()
			preValue = m.textareas[m.active].Value()
			preStateCaptured = true
		}

		// Enforce correct width BEFORE the textarea processes the key,
		// so cursor movement uses the right wrapping.
		if m.table != nil && m.blocks[m.active].Type == block.Table {
			cw := m.tableCellTAWidth()
			m.textareas[m.active].SetWidth(cw)
			m.textareas[m.active].SetHeight(m.textareas[m.active].VisualLineCount())
		}

		// Link blocks are edited via the bottom-sheet modal, so the textarea
		// is just storage — don't forward keys to it (would otherwise let the
		// user mutate the URL/title inline by typing while focused).
		if m.active >= 0 && m.active < len(m.blocks) && m.blocks[m.active].Type == block.Link {
			return m, nil
		}

		var cmd tea.Cmd
		m.textareas[m.active], cmd = m.textareas[m.active].Update(msg)

		// Only push an undo entry when content actually changed, not on
		// cursor-only movements (left, right, home, end, etc.).
		if preStateCaptured && m.textareas[m.active].Value() != preValue {
			m.undo.push(preState)
			m.redo.clear()
			m.undoDirty = true
		}

		// Definition block guard: if a delete operation destroyed the
		// term/definition newline separator, restore it.
		if m.blocks[m.active].Type == block.DefinitionList {
			val := m.textareas[m.active].Value()
			if !strings.Contains(val, "\n") {
				m.textareas[m.active].SetValue(val + "\n")
				m.textareas[m.active].MoveToBegin()
				m.textareas[m.active].CursorEnd()
				m.cursorCmd = m.textareas[m.active].Focus()
			}
		}

		// Auto-convert Quote to Callout when user types [!TYPE] as first line.
		if m.blocks[m.active].Type == block.Quote {
			val := m.textareas[m.active].Value()
			firstLine := val
			if idx := strings.Index(val, "\n"); idx >= 0 {
				firstLine = val[:idx]
			}
			if variant, ok := block.ParseCalloutMarker(firstLine); ok {
				m.blocks[m.active].Type = block.Callout
				m.blocks[m.active].Variant = variant
				// Strip the marker line from content.
				rest := ""
				if idx := strings.Index(val, "\n"); idx >= 0 {
					rest = val[idx+1:]
				}
				m.textareas[m.active].SetValue(rest)
				m.cursorCmd = m.textareas[m.active].Focus()
			}
		}

		// Re-enforce width and recalculate height after every keystroke.
		// This ensures the textarea re-wraps correctly after content changes
		// (e.g. deleting a newline inserted via Ctrl+J).
		if m.table != nil && m.blocks[m.active].Type == block.Table {
			cw := m.tableCellTAWidth()
			m.textareas[m.active].SetWidth(cw)
		} else {
			m.textareas[m.active].SetWidth(m.contentWidth(m.blocks[m.active]))
		}
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
	h := m.height - m.headerHeight() - m.statusBarHeight() - m.footerHeight()
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
		if m.blocks[m.active].Type == block.Table {
			chromeLines = 1 // top border
		}

		ta := m.textareas[m.active]

		// Count visual cursor line, accounting for word wrapping.
		// ta.Line() is the raw line number, but wrapped lines before the
		// cursor occupy more visual lines than raw lines.
		cursorRawLine := ta.Line()
		visualLine := cursorRawLine

		if m.kanban != nil && m.blocks[m.active].Type == block.Kanban {
			// For kanban, the textarea cursor doesn't move — track the
			// selected card's row range instead so vertical navigation
			// keeps the focused card in view.
			top, _ := m.selectedCardLineRange()
			visualLine = top
		} else if m.table != nil && m.blocks[m.active].Type == block.Table {
			// For tables, compute cursor line within the grid.
			// Each row before the active row contributes: 1 content line + 1 separator.
			// (Simplified: assume 1 visual line per cell for scroll purposes.)
			visualLine = m.table.row*2 + cursorRawLine
		} else if m.wordWrap {
			contentWidth := m.width - gutterWidth - blockPrefixWidth(m.blocks[m.active])
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

		// Resolve cursorLine / bottomTarget overrides before computing the
		// scroll target so block-specific anchoring (kanban, table) wins.
		bottomTarget := cursorLine
		if m.table != nil && m.blocks[m.active].Type == block.Table {
			rowsBelow := len(m.table.cells) - 1 - m.table.row
			if rowsBelow < 0 {
				rowsBelow = 0
			}
			bottomTarget = cursorLine + rowsBelow*2 + 1
		}
		// Pre-compute scrollTarget after any block-specific cursorLine override.
		yOffset := m.viewport.YOffset()
		scrollTarget := cursorLine
		if cursorRawLine == 0 && chromeLines > 0 {
			scrollTarget = lineOffset // show from block start
		}

		if m.kanban != nil && m.blocks[m.active].Type == block.Kanban {
			top, bot := m.selectedCardLineRange()
			bottomTarget = lineOffset + chromeLines + bot
			cursorLine = lineOffset + chromeLines + top
			kanbanTop := lineOffset + chromeLines
			// The active board clips tall columns internally, so anchoring
			// to the board top keeps headers visible without hiding the
			// selected card.
			if m.kanbanAnchorTop {
				contextLines := 0
				if m.active >= 0 && m.active < len(m.blockLineCounts) {
					contextLines = m.viewport.Height() - m.blockLineCounts[m.active]
					if contextLines > kanbanDocumentContextLines {
						contextLines = kanbanDocumentContextLines
					}
					if contextLines < 0 {
						contextLines = 0
					}
				}
				scrollTarget = kanbanTop - contextLines/2
			} else {
				scrollTarget = cursorLine
			}
			m.kanbanAnchorTop = false
		}

		// Always ensure the cursor line is visible. Prefer keeping the
		// current scroll position when the cursor is already on screen.
		if scrollTarget < yOffset {
			yOffset = scrollTarget
		}
		if bottomTarget >= yOffset+m.viewport.Height() {
			candidate := bottomTarget - m.viewport.Height() + 1
			// Never push the cursor itself off the top of the viewport.
			if candidate > cursorLine {
				candidate = cursorLine
			}
			yOffset = candidate
		}
		if yOffset < 0 {
			yOffset = 0
		}

		m.viewport.SetYOffset(yOffset)
	}
}

// viewRange returns the [start, end) slice of m.blocks that view-mode
// renders — leading and trailing empty paragraph blocks are skipped so
// accidental blank space at the edges of the document doesn't bloat
// the rendered view. Internal blanks (paragraph separators) are kept.
func (m Model) viewRange() (int, int) {
	start, end := 0, len(m.blocks)
	for start < end && block.IsEmptyParagraph(m.blocks[start]) {
		start++
	}
	for end > start && block.IsEmptyParagraph(m.blocks[end-1]) {
		end--
	}
	return start, end
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

	// Skipped blocks keep offset -1 so blockIndexAtLine ignores them.
	offsets := make([]int, len(m.blocks))
	for i := range offsets {
		offsets[i] = -1
	}
	vStart, vEnd := m.viewRange()
	prevIdx := -1
	for i := vStart; i < vEnd; i++ {
		b := m.blocks[i]
		content := b.Content
		if i == m.active && i < len(m.textareas) {
			content = m.textareas[i].Value()
		}

		// Spacing lines before this block (mirrors renderViewContent exactly).
		if prevIdx >= 0 {
			prev := m.blocks[prevIdx]
			switch {
			case b.Type == block.Heading1:
				advance("")
				advance("")
			case b.Type == block.Heading2 || b.Type == block.Heading3:
				advance("")
			case b.Type == block.CodeBlock || b.Type == block.Quote || b.Type == block.Callout || b.Type == block.Table:
				if prev.Type != b.Type {
					advance("")
				}
			case prev.Type == block.CodeBlock || prev.Type == block.Quote || prev.Type == block.Callout || prev.Type == block.Table:
				advance("")
			case b.Type == block.Embed:
				advance("")
			case prev.Type == block.Embed:
				advance("")
			case b.Type == block.Divider:
				advance("")
			case prev.Type == block.Divider:
				advance("")
			}
		}

		offsets[i] = nextLine
		rendered := renderViewBlock(b, content, contentWidth, m.wordWrap, m.blocks, i, false, m.viewKanbanOffset)
		advance(rendered)
		prevIdx = i
	}

	m.blockLineOffsets = offsets
}

// blockIndexAtLine returns the block index that contains the given absolute
// line number in view-mode rendered content, or -1 if no block matches.
// Blocks skipped by viewRange have offset -1 and are ignored.
func (m Model) blockIndexAtLine(line int) int {
	if len(m.blockLineOffsets) == 0 {
		return -1
	}
	result := -1
	for i, offset := range m.blockLineOffsets {
		if offset < 0 {
			continue
		}
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
// lines for each block.
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
		m.blockLineCounts[i] = lineCount
	}
	return strings.Join(parts, "\n")
}

// viewContentXRange returns the [leftPad, leftPad+contentWidth) X range of
// the centered view-mode content column. Click and hover handlers use this
// to reject coordinates that fall in the surrounding margin.
func (m Model) viewContentXRange() (leftPad, contentWidth int) {
	contentWidth = viewMaxWidth
	if m.width < contentWidth {
		contentWidth = m.width - 4
		if contentWidth < 20 {
			contentWidth = 20
		}
	}
	leftPad = (m.width - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	return leftPad, contentWidth
}

// renderViewContent builds the full view-mode content: centered, max-width,
// with generous spacing for a clean reading experience.
func (m Model) renderViewContent() string {
	leftPad, contentWidth := m.viewContentXRange()
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

	vStart, vEnd := m.viewRange()
	prevIdx := -1
	for i := vStart; i < vEnd; i++ {
		b := m.blocks[i]
		content := b.Content
		if i == m.active && i < len(m.textareas) {
			content = m.textareas[i].Value()
		}

		hovered := i == m.hoverBlock
		rendered := renderViewBlock(b, content, contentWidth, m.wordWrap, m.blocks, i, hovered, m.viewKanbanOffset)

		// Add vertical spacing before certain block types.
		if prevIdx >= 0 {
			prev := m.blocks[prevIdx]
			switch {
			case b.Type == block.Heading1:
				parts = append(parts, "", "") // extra blank before H1
			case b.Type == block.Heading2 || b.Type == block.Heading3:
				parts = append(parts, "") // blank before H2/H3
			case b.Type == block.CodeBlock || b.Type == block.Quote || b.Type == block.Callout || b.Type == block.Table:
				if prev.Type != b.Type {
					parts = append(parts, "")
				}
			case prev.Type == block.CodeBlock || prev.Type == block.Quote || prev.Type == block.Callout || prev.Type == block.Table:
				parts = append(parts, "")
			case b.Type == block.Embed:
				parts = append(parts, "") // blank before embeds
			case prev.Type == block.Embed:
				parts = append(parts, "") // blank after embeds
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
		prevIdx = i
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
	if m.embedModal.visible {
		content = m.renderEmbedSheet()
	} else {
		statusBar := m.renderStatusBar()
		var footer string
		if m.showHelp {
			footer = m.renderHelpFooter()
		} else {
			footer = m.renderPickerFooter()
		}
		if m.viewMode {
			if footer != "" {
				content = m.viewport.View() + "\n" + footer + statusBar
			} else {
				content = m.viewport.View() + "\n" + statusBar
			}
		} else {
			header := m.renderHeader()
			if footer != "" {
				content = header + "\n" + m.viewport.View() + "\n" + footer + statusBar
			} else {
				content = header + "\n" + m.viewport.View() + "\n" + statusBar
			}
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	if m.viewMode {
		// AllMotion enables hover tracking for checklist highlight.
		v.MouseMode = tea.MouseModeAllMotion
	}
	// Edit mode: no mouse capture — terminal-native text selection works.
	return v
}

// renderHelpFooter builds the help footer panel with keybind reference.
func (m Model) renderHelpFooter() string {
	return ui.RenderHelpFooter([]ui.HelpBinding{
		{Key: "Enter", Desc: "New block"},
		{Key: "⇧Enter", Desc: "Newline"},
		{Key: "Bksp", Desc: "Delete / merge"},
		{Key: "⌃K", Desc: "Cut block"},
		{Key: "⌥K", Desc: "Copy block"},
		{Key: "⌃Z/⌃Y", Desc: "Undo / redo"},
		{Key: "⌥↑/⌥↓", Desc: "Move block"},
		{Key: "/", Desc: "Block type"},
		{Key: ":", Desc: "Definitions"},
		{Key: "Tab/⇧Tab", Desc: "Indent"},
		{Key: "⌃X", Desc: "Interact"},
		{Key: "⌃R", Desc: "View mode"},
		{Key: "⌃W", Desc: "Word wrap"},
		{Key: "⌃H", Desc: "Sort checked"},
		{Key: "⌃S", Desc: "Save"},
		{Key: "Esc/⌃C", Desc: "Quit"},
		{Key: "⌃G", Desc: "Close"},
	}, m.width)
}

// renderPickerFooter returns the footer panel for whichever picker is active,
// or "" if none. The result includes a trailing newline when non-empty.
func (m Model) renderPickerFooter() string {
	if m.palette.Visible {
		return m.palette.RenderFooter(m.width)
	}
	if m.embedPicker.Visible {
		return m.embedPicker.RenderFooter(m.width)
	}
	if m.defLookup.Visible {
		return m.defLookup.RenderFooter(m.width)
	}
	if m.defPreview.visible {
		return m.defPreview.RenderFooter(m.width)
	}
	if m.linkPrompt.visible {
		return m.renderLinkPrompt()
	}
	return ""
}

// footerHeight returns the number of lines occupied by the active picker footer.
func (m Model) footerHeight() int {
	if m.showHelp {
		return strings.Count(m.renderHelpFooter(), "\n")
	}
	if m.palette.Visible {
		return m.palette.Height()
	}
	if m.embedPicker.Visible {
		return m.embedPicker.Height()
	}
	if m.defLookup.Visible {
		return m.defLookup.Height()
	}
	if m.defPreview.visible {
		return m.defPreview.Height(m.width)
	}
	if m.linkPrompt.visible {
		return linkPromptHeight
	}
	return 0
}

// editModeHints builds the right-side status bar shortcuts for edit mode.
// The "/" and ":" shortcuts only show when the cursor is at position 0
// (i.e., when they would actually trigger the palette/deflookup).
func (m Model) editModeHints() string {
	var parts []string
	atPos0 := false
	if m.active >= 0 && m.active < len(m.textareas) {
		ta := m.textareas[m.active]
		atPos0 = ta.Line() == 0 && ta.LineInfo().ColumnOffset == 0
	}
	if atPos0 {
		parts = append(parts, "/ blocks")
		// ":" only works on empty blocks.
		if m.active >= 0 && m.active < len(m.textareas) && m.textareas[m.active].Value() == "" {
			parts = append(parts, ": defs")
		}
	}
	parts = append(parts, "\u2303G help", "Esc quit")
	return strings.Join(parts, " \u00B7 ")
}

// blockHint returns a context-sensitive hint for the active block type,
// shown in the status bar hint area.
func (m Model) blockHint() string {
	if m.active < 0 || m.active >= len(m.blocks) {
		return ""
	}
	switch m.blocks[m.active].Type {
	case block.Table:
		return "\u2325R +row \u00B7 \u2325C +col \u00B7 \u2325\u21E7\u232B del row \u00B7 \u2325\u21E7D del col"
	case block.Callout:
		return "\u2303T variant"
	case block.CodeBlock:
		return "line 1 sets language"
	case block.Checklist:
		return "\u2303X toggle"
	case block.Kanban:
		if m.kanban != nil && m.kanban.edit {
			return "\u23ce save \u00b7 \u21e7\u23ce newline \u00b7 esc cancel"
		}
		return "\u2190\u2192 col \u00b7 \u2191\u2193 card \u00b7 \u21e7+arrows move \u00b7 n new \u00b7 \u23ce edit \u00b7 p prio \u00b7 s sort \u00b7 bksp del (board if col empty)"
	case block.Embed:
		return "\u2303X open \u00B7 Tab pick"
	case block.Link:
		return "\u2303X open \u00B7 Tab url \u00B7 \u21E7Tab title"
	case block.DefinitionList:
		return "\u2303X search"
	default:
		return ""
	}
}

// renderStatusBar builds the bottom status bar.
func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	left := " "
	if m.viewport.TotalLineCount() > m.viewport.Height() {
		pct := int(m.viewport.ScrollPercent() * 100)
		left += fmt.Sprintf(" %d%%", pct)
	}

	var hint string
	var right string
	if m.status != "" {
		right = m.status
	} else if m.showHelp {
		right = "\u2303G close"
	} else if m.viewMode {
		if !m.dismissedHints["editor.checkbox"] {
			hint = "click checkboxes to toggle!  [h]ide"
		}
		right = ": defs \u00B7 \u2303R edit \u00B7 Esc quit"
	} else if m.embedModal.visible {
		right = "\u21B5 open \u00B7 \u2191\u2193 scroll \u00B7 Esc close"
	} else if m.defPreview.visible {
		right = "\u21B5 open note \u00B7 Esc close"
	} else if m.palette.Visible {
		right = "\u21B5 select \u00B7 \u2191\u2193 move \u00B7 Esc close"
	} else if m.embedPicker.Visible {
		right = "\u21B5 insert \u00B7 \u2191\u2193 move \u00B7 Esc close"
	} else if m.defLookup.Visible {
		right = "\u21B5 preview \u00B7 \u2191\u2193 move \u00B7 Esc close"
	} else {
		// Build right-side hints: block-specific hints + contextual shortcuts.
		bh := m.blockHint()
		right = m.editModeHints()
		if bh != "" {
			right = bh + " \u00B7 " + right
		}
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

// openEmbedModal resolves an embed block's reference and opens the sheet.
// clampEmbedScroll ensures the embed modal scroll doesn't exceed the content.
func (m *Model) clampEmbedScroll() {
	sheetH := m.height - 2
	if sheetH < 6 {
		sheetH = 6
	}
	contentH := sheetH - 3
	if contentH < 1 {
		contentH = 1
	}
	maxScroll := len(m.embedModal.lines) - contentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.embedModal.scroll > maxScroll {
		m.embedModal.scroll = maxScroll
	}
}

func (m *Model) openEmbedModal(idx int) {
	if idx < 0 || idx >= len(m.blocks) || m.blocks[idx].Type != block.Embed {
		return
	}
	path := m.blocks[idx].Content
	if idx < len(m.textareas) {
		path = m.textareas[idx].Value()
	}
	if path == "" {
		return
	}

	if m.config.ResolveEmbed == nil {
		errBlocks := []block.Block{{Type: block.Paragraph, Content: "Embed links not available."}}
		m.embedModal.open(path, path, errBlocks)
		m.renderEmbedSheetContent()
		return
	}

	title, content, err := m.config.ResolveEmbed(path)
	if err != nil {
		errBlocks := []block.Block{{Type: block.Paragraph, Content: "Note not found: " + path}}
		m.embedModal.open(path, path, errBlocks)
		m.renderEmbedSheetContent()
		return
	}

	refBlocks := block.Parse(content)
	m.embedModal.open(title, path, refBlocks)
	m.renderEmbedSheetContent()
}

// openDefPreview shows the focused definition preview for a remote def.
// The preview is an auto-sized footer, not a fullscreen sheet. Enter inside
// it jumps to the source note via jumpTarget.
func (m *Model) openDefPreview(notebook, note, term, definition string) {
	path := notebook + "/" + note
	m.defPreview.open(term, definition, path)
}

// renderEmbedSheetContent pre-renders the embed sheet blocks into lines
// and computes block line offsets for click hit-testing.
func (m *Model) renderEmbedSheetContent() {
	em := &m.embedModal
	contentWidth := viewMaxWidth
	if m.width-4 < contentWidth {
		contentWidth = m.width - 4
		if contentWidth < 20 {
			contentWidth = 20
		}
	}

	leftPad := (m.width - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	padStr := strings.Repeat(" ", leftPad)

	var parts []string
	offsets := make([]int, len(em.blocks))
	for i := range offsets {
		offsets[i] = -1
	}
	parts = append(parts, "") // top breathing room

	// Trim leading/trailing empty paragraphs so embedded previews don't
	// lead with a wall of whitespace.
	vStart, vEnd := 0, len(em.blocks)
	for vStart < vEnd && block.IsEmptyParagraph(em.blocks[vStart]) {
		vStart++
	}
	for vEnd > vStart && block.IsEmptyParagraph(em.blocks[vEnd-1]) {
		vEnd--
	}

	prevIdx := -1
	for i := vStart; i < vEnd; i++ {
		b := em.blocks[i]
		if prevIdx >= 0 {
			prev := em.blocks[prevIdx]
			switch {
			case b.Type == block.Heading1:
				parts = append(parts, "", "")
			case b.Type == block.Heading2 || b.Type == block.Heading3:
				parts = append(parts, "")
			case b.Type == block.CodeBlock || b.Type == block.Quote:
				if prev.Type != b.Type {
					parts = append(parts, "")
				}
			case prev.Type == block.CodeBlock || prev.Type == block.Quote:
				parts = append(parts, "")
			case b.Type == block.Embed:
				parts = append(parts, "")
			case prev.Type == block.Embed:
				parts = append(parts, "")
			case b.Type == block.Divider:
				parts = append(parts, "")
			case prev.Type == block.Divider:
				parts = append(parts, "")
			}
		}

		offsets[i] = len(parts)
		hovered := i == em.hoverBlock
		rendered := renderViewBlock(b, b.Content, contentWidth, true, em.blocks, i, hovered, 0)
		for _, l := range strings.Split(rendered, "\n") {
			parts = append(parts, padStr+l)
		}
		prevIdx = i
	}
	parts = append(parts, "", "")

	em.lines = parts
	em.blockLineOffsets = offsets
}

// renderEmbedSheet renders the full-width bottom sheet overlay for embedded notes.
func (m Model) renderEmbedSheet() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	th := theme.Current()
	dim := lipgloss.NewStyle().Faint(true)
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))

	// Sheet takes all but the top 2 lines (context peek).
	sheetH := h - 2
	if sheetH < 6 {
		sheetH = 6
	}

	// Title bar: centered, with separator line.
	titleText := accent.Bold(true).Render(m.embedModal.title)
	titleW := lipgloss.Width(titleText)
	titlePad := (w - titleW) / 2
	if titlePad < 0 {
		titlePad = 0
	}
	titleLine := strings.Repeat(" ", titlePad) + titleText

	sepChar := "\u2500"
	sep := dim.Render(strings.Repeat(sepChar, w))

	// Content area: sheetH minus title(1) + sep(1) + status(1) = sheetH-3
	contentH := sheetH - 3
	if contentH < 1 {
		contentH = 1
	}

	// Apply scroll.
	lines := m.embedModal.lines
	maxScroll := len(lines) - contentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.embedModal.scroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + contentH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[scroll:end]

	// Pad to fill the content area.
	body := strings.Join(visible, "\n")
	if len(visible) < contentH {
		body += strings.Repeat("\n", contentH-len(visible))
	}

	// Status bar.
	statusRight := "\u21B5 open \u00B7 \u2191\u2193 scroll \u00B7 Esc close"
	statusLeft := ""
	if len(lines) > contentH {
		pct := 0
		if maxScroll > 0 {
			pct = scroll * 100 / maxScroll
		}
		statusLeft = fmt.Sprintf(" %d%%", pct)
	}
	statusBar := format.StatusBar(statusLeft, "", statusRight, w)
	statusBar = dim.Render(statusBar)

	// Top context: dimmed current note title.
	contextLine := dim.Render("  " + m.config.Title)
	blank := ""

	return contextLine + "\n" + blank + "\n" + sep + "\n" + titleLine + "\n" + body + "\n" + statusBar
}
