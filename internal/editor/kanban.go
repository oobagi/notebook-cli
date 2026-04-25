package editor

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// renderEditingCardText renders the textarea content with a manually
// drawn cursor, bypassing the textarea's internal viewport. The textarea's
// own View() can clip top lines once it has scrolled internally; rendering
// from Value() + cursor position avoids that and matches the rest of the
// block editor's pattern.
func renderEditingCardText(ta *textarea.Model, contentWidth int) string {
	li := ta.LineInfo()
	cursorRawRow := ta.Line()
	cursorColInWrap := li.ColumnOffset
	cursorWrapRow := li.RowOffset

	content := ta.Value()
	if content == "" {
		// Show a single cursor on an otherwise empty card.
		ta.CursorSetChar(" ")
		return ta.CursorView()
	}

	rawLines := strings.Split(content, "\n")
	var visualLines []string
	for rawIdx, raw := range rawLines {
		segs := textarea.Wrap([]rune(raw), contentWidth)
		if len(segs) == 0 {
			segs = [][]rune{{}}
		}
		for wIdx, seg := range segs {
			line := strings.TrimSuffix(string(seg), " ")
			if rawIdx == cursorRawRow && wIdx == cursorWrapRow {
				runes := []rune(line)
				col := cursorColInWrap
				if col > len(runes) {
					col = len(runes)
				}
				before := string(runes[:col])
				curChar := " "
				after := ""
				if col < len(runes) {
					curChar = string(runes[col : col+1])
					after = string(runes[col+1:])
				}
				ta.CursorSetChar(curChar)
				line = before + ta.CursorView() + after
			}
			visualLines = append(visualLines, line)
		}
	}
	return strings.Join(visualLines, "\n")
}

// kanbanState holds the in-memory representation of a focused Kanban block.
// Mirrors the role of tableState for tables — non-nil only while a Kanban
// block is the active block in the editor.
type kanbanState struct {
	cols      []block.KanbanColumn
	col       int  // selected column index
	card      int  // selected card index within selected column (-1 = column header)
	colOffset int  // index of the leftmost visible column (horizontal scroll)
	edit      bool // true when the inline text editor is active
	editTA    textarea.Model
	// addedCardIdx tracks a card index inserted by addCard; if the user
	// cancels editing it before typing anything, cancelEdit drops it and
	// restores the prior selection. -1 when not in an addCard flow.
	addedCardIdx int
}

// kanbanTargetColWidth is the preferred visible width of a single column
// (including borders/padding of its cards). The renderer uses this to
// decide how many columns fit on screen at once; the rest stay off-screen
// and are reached by scrolling horizontally as the focus moves.
const kanbanTargetColWidth = 32

// kanbanIndicatorWidth reserves space for the ◀ / ▶ off-screen-column
// indicators on the left/right edges of the board (one cell each).
const kanbanIndicatorWidth = 1

// newKanbanState parses a kanban body into editable state, defaulting the
// selection to the first card (or first column if there are no cards).
func newKanbanState(body string) *kanbanState {
	ks := &kanbanState{cols: block.ParseKanban(body), addedCardIdx: -1}
	if len(ks.cols) == 0 {
		ks.cols = block.ParseKanban(block.DefaultKanbanContent)
	}
	ks.col = 0
	if len(ks.cols[0].Cards) > 0 {
		ks.card = 0
	} else {
		ks.card = -1
	}
	return ks
}

// serialize emits the current board to kanban-fence body markdown.
func (ks *kanbanState) serialize() string {
	return block.SerializeKanban(ks.cols)
}

// enterFromAbove positions the selection for entering the kanban from a
// block above (e.g. via Down arrow). Lands on the leftmost column that
// has cards, first card. If all columns are empty, col=0 with no card.
func (ks *kanbanState) enterFromAbove() {
	for ci := range ks.cols {
		if len(ks.cols[ci].Cards) > 0 {
			ks.col = ci
			ks.card = 0
			return
		}
	}
	ks.col = 0
	ks.card = -1
}

// enterFromBelow positions the selection for entering the kanban from a
// block below (e.g. via Up arrow). Lands on the leftmost column that has
// cards, last card. Mirrors enterFromAbove for symmetric navigation.
func (ks *kanbanState) enterFromBelow() {
	for ci := range ks.cols {
		if len(ks.cols[ci].Cards) > 0 {
			ks.col = ci
			ks.card = len(ks.cols[ci].Cards) - 1
			return
		}
	}
	ks.col = 0
	ks.card = -1
}

// selectedCard returns a pointer to the selected card, or nil.
func (ks *kanbanState) selectedCard() *block.KanbanCard {
	if ks.col < 0 || ks.col >= len(ks.cols) {
		return nil
	}
	if ks.card < 0 || ks.card >= len(ks.cols[ks.col].Cards) {
		return nil
	}
	return &ks.cols[ks.col].Cards[ks.card]
}

// clamp keeps col/card in valid bounds. Called after every mutation.
func (ks *kanbanState) clamp() {
	if len(ks.cols) == 0 {
		ks.col, ks.card = 0, -1
		return
	}
	if ks.col < 0 {
		ks.col = 0
	}
	if ks.col >= len(ks.cols) {
		ks.col = len(ks.cols) - 1
	}
	n := len(ks.cols[ks.col].Cards)
	if n == 0 {
		ks.card = -1
		return
	}
	if ks.card < 0 {
		ks.card = 0
	}
	if ks.card >= n {
		ks.card = n - 1
	}
}

// clampColOffset shifts the horizontal viewport so the selected column is
// visible within a window of `visible` columns. When the selected column
// is already in view, colOffset is unchanged (no jitter).
func (ks *kanbanState) clampColOffset(visible int) {
	if visible <= 0 || len(ks.cols) == 0 {
		ks.colOffset = 0
		return
	}
	if visible >= len(ks.cols) {
		ks.colOffset = 0
		return
	}
	if ks.col < ks.colOffset {
		ks.colOffset = ks.col
	} else if ks.col >= ks.colOffset+visible {
		ks.colOffset = ks.col - visible + 1
	}
	if ks.colOffset < 0 {
		ks.colOffset = 0
	}
	if ks.colOffset > len(ks.cols)-visible {
		ks.colOffset = len(ks.cols) - visible
	}
}

// moveSelection nudges the selection in a direction. Falls through to
// adjacent columns at the edges, so navigation feels continuous.
func (ks *kanbanState) moveSelection(dCol, dCard int) {
	if len(ks.cols) == 0 {
		return
	}
	if dCol != 0 {
		ks.col += dCol
		if ks.col < 0 {
			ks.col = 0
		}
		if ks.col >= len(ks.cols) {
			ks.col = len(ks.cols) - 1
		}
		// Try to keep the same card index when switching columns; clamp.
		ks.clamp()
		return
	}
	if dCard != 0 {
		n := len(ks.cols[ks.col].Cards)
		if n == 0 {
			return
		}
		ks.card += dCard
		if ks.card < 0 {
			ks.card = 0
		}
		if ks.card >= n {
			ks.card = n - 1
		}
	}
}

// moveCard moves the selected card within its column or across columns,
// preserving selection (selection follows the moved card).
func (ks *kanbanState) moveCard(dCol, dCard int) {
	c := ks.selectedCard()
	if c == nil {
		return
	}
	src := *c
	col := &ks.cols[ks.col]

	if dCard != 0 && dCol == 0 {
		// Reorder within column.
		newIdx := ks.card + dCard
		if newIdx < 0 || newIdx >= len(col.Cards) {
			return
		}
		col.Cards[ks.card], col.Cards[newIdx] = col.Cards[newIdx], col.Cards[ks.card]
		ks.card = newIdx
		return
	}
	if dCol != 0 {
		newCol := ks.col + dCol
		if newCol < 0 || newCol >= len(ks.cols) {
			return
		}
		// Remove from current.
		col.Cards = append(col.Cards[:ks.card], col.Cards[ks.card+1:]...)
		// Append to bottom of destination column (most natural for kanban).
		ks.cols[newCol].Cards = append(ks.cols[newCol].Cards, src)
		ks.col = newCol
		ks.card = len(ks.cols[newCol].Cards) - 1
	}
}

// addCard inserts a new empty card after the current selection (or at top
// when the column is empty), then enters edit mode on it.
func (ks *kanbanState) addCard(width int) {
	if len(ks.cols) == 0 {
		return
	}
	col := &ks.cols[ks.col]
	newIdx := ks.card + 1
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx > len(col.Cards) {
		newIdx = len(col.Cards)
	}
	card := block.KanbanCard{}
	col.Cards = append(col.Cards[:newIdx], append([]block.KanbanCard{card}, col.Cards[newIdx:]...)...)
	ks.card = newIdx
	ks.addedCardIdx = newIdx
	ks.startEdit(width)
}

// deleteCard removes the selected card.
func (ks *kanbanState) deleteCard() {
	c := ks.selectedCard()
	if c == nil {
		return
	}
	col := &ks.cols[ks.col]
	col.Cards = append(col.Cards[:ks.card], col.Cards[ks.card+1:]...)
	ks.clamp()
}

// isEmpty reports whether the board has zero cards across all columns.
func (ks *kanbanState) isEmpty() bool {
	for _, c := range ks.cols {
		if len(c.Cards) > 0 {
			return false
		}
	}
	return true
}

// togglePriority cycles the priority of the selected card.
func (ks *kanbanState) togglePriority() {
	c := ks.selectedCard()
	if c == nil {
		return
	}
	c.Priority = c.Priority.Next()
}

// sortByPriority sorts cards within each column by priority descending
// (HIGH → MED → LOW → none) using a stable sort, so cards of the same
// priority keep their relative order. Selection follows the focused card
// to its new position.
func (ks *kanbanState) sortByPriority() {
	// Tag the selected card so we can find it after sort.
	type marker struct{}
	var sel *marker
	if c := ks.selectedCard(); c != nil {
		sel = &marker{}
	}
	for ci, col := range ks.cols {
		// Pair (originalIdx, card) for stable sort.
		idxs := make([]int, len(col.Cards))
		for i := range idxs {
			idxs[i] = i
		}
		// Stable sort: higher priority first; ties preserve original order.
		for i := 1; i < len(idxs); i++ {
			for j := i; j > 0; j-- {
				a := col.Cards[idxs[j-1]].Priority
				b := col.Cards[idxs[j]].Priority
				if a >= b {
					break
				}
				idxs[j-1], idxs[j] = idxs[j], idxs[j-1]
			}
		}
		// Apply.
		newCards := make([]block.KanbanCard, len(col.Cards))
		for i, oi := range idxs {
			newCards[i] = col.Cards[oi]
		}
		ks.cols[ci].Cards = newCards
		// Update selection if this is the selected column.
		if ci == ks.col && sel != nil {
			for newIdx, oi := range idxs {
				if oi == ks.card {
					ks.card = newIdx
					break
				}
			}
		}
	}
}

// startEdit opens an inline textarea seeded with the selected card's text.
// The textarea wraps at the card's inner width and grows in height as the
// user types or inserts newlines (Shift+Enter).
func (ks *kanbanState) startEdit(width int) {
	c := ks.selectedCard()
	if c == nil {
		return
	}
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetWidth(width)
	ta.SetValue(c.Text)
	ta.MoveToEnd()
	ta.SetHeight(ta.VisualLineCount())
	ta.Focus()
	ks.editTA = ta
	ks.edit = true
}

// commitEdit writes the textarea contents back to the selected card and
// exits edit mode. Empty card text is left empty (the card stays).
func (ks *kanbanState) commitEdit() {
	c := ks.selectedCard()
	if c == nil {
		ks.edit = false
		ks.addedCardIdx = -1
		return
	}
	c.Text = strings.TrimRight(ks.editTA.Value(), "\n")
	ks.edit = false
	ks.addedCardIdx = -1
}

// cancelEdit discards textarea changes and exits edit mode. If the card
// was just created via addCard and never typed into, drop it AND restore
// the selection to the card the user was on before pressing `n` so Esc
// doesn't shift their focus to a different card.
func (ks *kanbanState) cancelEdit() {
	if c := ks.selectedCard(); c != nil && c.Text == "" && ks.editTA.Value() == "" {
		justAdded := ks.addedCardIdx == ks.card
		ks.deleteCard()
		if justAdded {
			// addCard set ks.card = prev+1; restore by stepping back.
			if ks.card > 0 {
				ks.card--
			}
			ks.clamp()
		}
	}
	ks.edit = false
	ks.addedCardIdx = -1
}

// selectedCardLineRange returns the inclusive [top, bottom] line offsets
// of the selected card within the rendered kanban board, relative to the
// board's first line. Returns (0, 0) when there is no selection (e.g.
// empty column). Used to drive viewport auto-scroll so the selected card
// stays visible when the board is taller than the viewport.
func (m Model) selectedCardLineRange() (top, bottom int) {
	if m.kanban == nil || len(m.kanban.cols) == 0 {
		return 0, 0
	}
	col := m.kanban.col
	if col < 0 || col >= len(m.kanban.cols) {
		return 0, 0
	}
	cards := m.kanban.cols[col].Cards
	if m.kanban.card < 0 || m.kanban.card >= len(cards) {
		// Empty / no card selected: return header line range.
		return 0, 1 // title + underline
	}

	// Header (title + underline) is 2 lines.
	line := 2
	cardOuterWidth := m.kanbanCardOuterWidth()
	contentWidth := cardOuterWidth - kanbanCardChromeWidth
	if contentWidth < 1 {
		contentWidth = 1
	}
	for i := 0; i < m.kanban.card; i++ {
		line += cardRenderHeight(cards[i], contentWidth)
	}
	height := cardRenderHeight(cards[m.kanban.card], contentWidth)
	if m.kanban.edit {
		// Editing: textarea visual line count drives the height.
		taLines := m.kanban.editTA.VisualLineCount()
		if taLines < 1 {
			taLines = 1
		}
		// 2 border + priority badge (1 if present).
		extra := 2
		if cards[m.kanban.card].Priority != block.PriorityNone {
			extra++
		}
		height = taLines + extra
	}
	return line, line + height - 1
}

// maxViewKanbanOffset returns the largest valid offset for the view-mode
// horizontal scroll, based on the widest kanban block in the document at
// the current terminal width. Zero when no kanban exists or the board
// already fits.
func (m Model) maxViewKanbanOffset() int {
	maxOff := 0
	width := m.width - 4
	if width < 30 {
		width = 30
	}
	for _, b := range m.blocks {
		if b.Type != block.Kanban {
			continue
		}
		cols := block.ParseKanban(b.Content)
		visible, _ := kanbanVisibleCols(len(cols), width)
		off := len(cols) - visible
		if off > maxOff {
			maxOff = off
		}
	}
	return maxOff
}

// kanbanCardOuterWidth returns the visible width of a card box, computed
// the same way as renderKanbanBoard so callers stay in sync.
func (m Model) kanbanCardOuterWidth() int {
	if m.kanban == nil || len(m.kanban.cols) == 0 {
		return 20
	}
	width := m.width - gutterWidth
	if width < 30 {
		width = 30
	}
	_, colWidth := kanbanVisibleCols(len(m.kanban.cols), width)
	return colWidth
}

// cardRenderHeight predicts the rendered line count of a single card box
// given its content width. Mirrors the math in renderKanbanCard.
func cardRenderHeight(card block.KanbanCard, contentWidth int) int {
	textLines := 1
	if card.Text != "" {
		// Account for explicit newlines in card text and word-wrapping at
		// contentWidth (matches wrapPlain).
		wrapped := wrapPlain(card.Text, contentWidth)
		textLines = strings.Count(wrapped, "\n") + 1
	}
	extra := 2 // top + bottom border
	if card.Priority != block.PriorityNone {
		extra++ // priority header line
	}
	return textLines + extra
}

// kanbanCardChromeWidth is the visual width consumed by a card's border
// (left + right) plus its horizontal padding (1 + 1).
const kanbanCardChromeWidth = 4

// renderKanbanCard renders one card box. outerWidth is the total visible
// width of the card box (including border and padding); the actual text
// content area is outerWidth - kanbanCardChromeWidth wide.
func renderKanbanCard(card block.KanbanCard, outerWidth int, selected, editing bool, editView string, th theme.Theme) string {
	border := lipgloss.RoundedBorder()
	borderColor := th.Border
	if selected {
		borderColor = th.Accent
	}

	style := lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(outerWidth)

	contentWidth := outerWidth - kanbanCardChromeWidth
	if contentWidth < 1 {
		contentWidth = 1
	}

	var text string
	switch {
	case editing:
		text = editView
		if text == "" {
			text = " "
		}
	case card.Text == "":
		text = lipgloss.NewStyle().Faint(true).Render("(empty)")
	default:
		// Wrap the plain text first (wrap uses visual width and would
		// miscount ANSI escapes), then apply inline markdown formatting.
		text = format.RenderInlineMarkdown(wrapPlain(card.Text, contentWidth))
	}

	// Priority badge on its own short header line.
	header := ""
	if m := card.Priority.Marker(); m != "" {
		color := priorityColor(card.Priority, th)
		bs := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		if card.Priority == block.PriorityHigh {
			bs = bs.Bold(true)
		}
		header = bs.Render(m)
	}

	body := text
	if header != "" {
		body = header + "\n" + text
	}
	return style.Render(body)
}

// wrapPlain word-wraps text to the given visual width. Empty preserves
// existing line breaks so multi-line card bodies render naturally.
func wrapPlain(text string, width int) string {
	if width <= 0 {
		return text
	}
	out := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, line)
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if lipgloss.Width(cur)+1+lipgloss.Width(w) <= width {
				cur += " " + w
			} else {
				out = append(out, cur)
				cur = w
			}
		}
		out = append(out, cur)
	}
	return strings.Join(out, "\n")
}

// kanbanVisibleCols returns the number of columns that fit in width and
// the chosen per-column width. Always at least 1; capped at the total
// column count.
func kanbanVisibleCols(totalCols, width int) (visible, colWidth int) {
	if totalCols <= 0 {
		return 0, 0
	}
	gap := 1
	// Reserve indicator strips on both edges when not all columns fit.
	usable := width
	// Account for inter-column gaps when fitting N columns: gap*(N-1).
	// Greedy: try N from totalCols down to 1.
	for n := totalCols; n >= 1; n-- {
		gaps := gap * (n - 1)
		// If we'd be hiding columns, reserve 2*indicator for ◀ / ▶.
		reserved := 0
		if n < totalCols {
			reserved = kanbanIndicatorWidth * 2
		}
		w := (usable - gaps - reserved) / n
		if w >= kanbanTargetColWidth || n == 1 {
			if w < 14 {
				w = 14
			}
			return n, w
		}
	}
	return 1, max(usable, 14)
}

// renderKanbanBoard renders the full board. Width is the available width
// for the kanban (including borders). Columns slide horizontally so the
// selected column is always in view.
func (m Model) renderKanbanBoard(blockIdx, width int) string {
	if m.kanban == nil {
		return ""
	}
	th := theme.Current()
	cols := m.kanban.cols
	if len(cols) == 0 {
		return ""
	}

	visible, colWidth := kanbanVisibleCols(len(cols), width)
	m.kanban.clampColOffset(visible)
	startCol := m.kanban.colOffset
	endCol := startCol + visible
	if endCol > len(cols) {
		endCol = len(cols)
	}
	// Cards span the full column width. Title/underline use the same width
	// so everything aligns visually.
	cardOuterWidth := colWidth
	gap := 1

	colStyle := lipgloss.NewStyle().Width(colWidth)

	rendered := make([]string, 0, endCol-startCol)
	for ci := startCol; ci < endCol; ci++ {
		col := cols[ci]
		colSelected := m.kanban.col == ci
		titleColor := th.Accent
		if !colSelected {
			titleColor = th.Muted
		}
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(titleColor)).
			Width(colWidth).
			Padding(0, 1)
		count := lipgloss.NewStyle().
			Faint(true).
			Render(formatCount(len(col.Cards)))
		title := titleStyle.Render(format.StripControlChars(col.Title) + "  " + count)

		// Underline: solid accent rule when this column is selected,
		// otherwise a faint border-color rule.
		underColor := th.Border
		if colSelected {
			underColor = th.Accent
		}
		under := lipgloss.NewStyle().
			Foreground(lipgloss.Color(underColor)).
			Render(strings.Repeat("─", colWidth))

		var cardLines []string
		cardLines = append(cardLines, title)
		cardLines = append(cardLines, under)

		if len(col.Cards) == 0 {
			// Empty placeholder. When this column is the focus, draw it
			// in accent so the user can see where they are.
			placeholderColor := th.Muted
			placeholderText := "no cards · n to add"
			if colSelected {
				placeholderColor = th.Accent
			}
			placeholder := lipgloss.NewStyle().
				Foreground(lipgloss.Color(placeholderColor)).
				Italic(true).
				Padding(0, 2).
				Render(placeholderText)
			cardLines = append(cardLines, placeholder)
		}

		for cardI, card := range col.Cards {
			isSel := m.kanban.col == ci && m.kanban.card == cardI
			editing := isSel && m.kanban.edit
			editView := ""
			if editing {
				editView = renderEditingCardText(&m.kanban.editTA, cardOuterWidth-kanbanCardChromeWidth)
			}
			cardLines = append(cardLines, renderKanbanCard(card, cardOuterWidth, isSel, editing, editView, th))
		}

		rendered = append(rendered, colStyle.Render(strings.Join(cardLines, "\n")))
	}

	gapStr := strings.Repeat(" ", gap)
	board := lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(rendered, gapStr)...)

	// Sandwich with off-screen-column indicators when more cols exist.
	leftHidden := startCol
	rightHidden := len(cols) - endCol
	if leftHidden > 0 || rightHidden > 0 {
		left := renderColIndicator(leftHidden, true, th)
		right := renderColIndicator(rightHidden, false, th)
		board = lipgloss.JoinHorizontal(lipgloss.Top, left, board, right)
	}
	return board
}

// renderColIndicator renders a single-cell arrow showing whether columns
// are hidden on the left or right edge of the board. Returns a single
// space when count is 0 so the layout stays stable.
func renderColIndicator(count int, left bool, th theme.Theme) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted))
	if count <= 0 {
		return " "
	}
	if left {
		return style.Render("◀")
	}
	return style.Render("▶")
}

// joinWithGap interleaves a gap string between rendered columns.
func joinWithGap(parts []string, gap string) []string {
	if len(parts) == 0 {
		return parts
	}
	out := make([]string, 0, len(parts)*2-1)
	for i, p := range parts {
		if i > 0 {
			out = append(out, gap)
		}
		out = append(out, p)
	}
	return out
}

// cloneKanbanCols deep-copies a column slice (and its card slices) so the
// caller can mutate the result without aliasing live editor state.
func cloneKanbanCols(in []block.KanbanColumn) []block.KanbanColumn {
	out := make([]block.KanbanColumn, len(in))
	for i, col := range in {
		copyCards := make([]block.KanbanCard, len(col.Cards))
		copy(copyCards, col.Cards)
		out[i] = block.KanbanColumn{Title: col.Title, Cards: copyCards}
	}
	return out
}

// kanbanCardEditWidth returns the available textarea width for the inline
// edit textarea: matches the card's inner content width so the textarea
// wraps at exactly the same column the rendered box does.
func (m Model) kanbanCardEditWidth() int {
	return m.kanbanCardOuterWidth() - kanbanCardChromeWidth
}

// handleKanbanKey routes a key press to the kanban state machine while a
// Kanban block is active. Returns (handled, cmd). When handled is true,
// the caller must not process the key further; when false, the global
// editor handler runs (e.g. for Ctrl+S, Ctrl+R, Esc).
func (m *Model) handleKanbanKey(msg tea.KeyPressMsg) (handled bool, cmd tea.Cmd) {
	if m.kanban == nil {
		return false, nil
	}

	// Edit mode: most keys go to the textarea. Enter commits, Shift+Enter
	// inserts a newline (matches the rest of the editor's "soft newline"
	// convention). Esc cancels.
	if m.kanban.edit {
		switch msg.String() {
		case "esc":
			m.kanban.cancelEdit()
			return true, nil
		case "enter":
			m.kanban.commitEdit()
			return true, nil
		case "shift+enter", "ctrl+j":
			// Insert a literal newline by forwarding a plain Enter to the
			// textarea (which interprets it as a newline since CharLimit=0
			// and we don't intercept it).
			m.kanban.editTA.SetHeight(m.kanban.editTA.VisualLineCount() + 1)
			var c tea.Cmd
			m.kanban.editTA, c = m.kanban.editTA.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			m.kanban.editTA.SetHeight(m.kanban.editTA.VisualLineCount())
			return true, c
		case "ctrl+s":
			// Let main handler save; first commit edit so latest text is in state.
			m.kanban.commitEdit()
			return false, nil
		}
		var c tea.Cmd
		m.kanban.editTA, c = m.kanban.editTA.Update(msg)
		// Reflow height so the box grows with wrapped/multi-line content.
		m.kanban.editTA.SetHeight(m.kanban.editTA.VisualLineCount())
		return true, c
	}

	// Selection mode keys.
	switch msg.String() {
	case "left":
		m.kanban.moveSelection(-1, 0)
		m.kanbanAnchorTop = true
		return true, nil
	case "right":
		m.kanban.moveSelection(1, 0)
		m.kanbanAnchorTop = true
		return true, nil
	case "up":
		// At top of column (or empty column), navigate out to the previous
		// block so the page can scroll past the board.
		col := m.kanban.cols[m.kanban.col]
		if m.kanban.card <= 0 || len(col.Cards) == 0 {
			if m.active > 0 {
				m.navigateUp()
			} else if m.blocks[0].Type != block.Paragraph {
				m.pushUndo()
				m.insertBlockBefore(0, block.Block{Type: block.Paragraph})
			}
			return true, nil
		}
		m.kanban.moveSelection(0, -1)
		m.kanbanAnchorTop = true
		return true, nil
	case "down":
		col := m.kanban.cols[m.kanban.col]
		if len(col.Cards) == 0 || m.kanban.card >= len(col.Cards)-1 {
			if m.active < len(m.textareas)-1 {
				m.navigateDown()
			} else {
				// At last card of last block — insert a paragraph
				// below so the user can escape downward.
				m.pushUndo()
				m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
			}
			return true, nil
		}
		m.kanban.moveSelection(0, 1)
		m.kanbanAnchorTop = true
		return true, nil
	case "pgup":
		// Always page-scroll past the board.
		m.viewport.HalfPageUp()
		return true, nil
	case "pgdown":
		m.viewport.HalfPageDown()
		return true, nil
	case "shift+left":
		m.pushUndo()
		m.kanban.moveCard(-1, 0)
		// Cross-column move: the destination column may need re-sorting
		// when auto-sort is on. Within-column moves don't reorder via
		// this branch (dCol != 0), but we sort defensively for parity
		// with shift+up/down so the post-state is always consistent.
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		m.kanbanAnchorTop = true
		return true, nil
	case "shift+right":
		m.pushUndo()
		m.kanban.moveCard(1, 0)
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		m.kanbanAnchorTop = true
		return true, nil
	case "shift+up":
		// Within-column reorder. When auto-sort is on, manual reordering
		// is meaningless — the sort will run after and undo the move.
		// Block the move with a status hint so the list doesn't appear
		// to "snap back" mysteriously.
		if m.kanbanSortByPrio {
			m.status = "Manual reorder disabled while sort by priority is on (s to toggle)"
			m.statusStyle = statusWarning
			return true, m.scheduleStatusDismiss()
		}
		m.pushUndo()
		m.kanban.moveCard(0, -1)
		m.kanbanAnchorTop = true
		return true, nil
	case "shift+down":
		if m.kanbanSortByPrio {
			m.status = "Manual reorder disabled while sort by priority is on (s to toggle)"
			m.statusStyle = statusWarning
			return true, m.scheduleStatusDismiss()
		}
		m.pushUndo()
		m.kanban.moveCard(0, 1)
		m.kanbanAnchorTop = true
		return true, nil
	case "enter":
		if m.kanban.selectedCard() != nil {
			m.pushUndo()
			m.kanban.startEdit(m.kanbanCardEditWidth())
			return true, nil
		}
		// No card to edit — escape downward by inserting a paragraph
		// below the kanban so the user can keep writing.
		m.pushUndo()
		m.insertBlockAfter(m.active, block.Block{Type: block.Paragraph})
		return true, nil
	case "n", "ctrl+n":
		m.pushUndo()
		m.kanban.addCard(m.kanbanCardEditWidth())
		// New card has no priority, so sorting puts it after prioritized
		// cards. Apply sort so it lands in the right slot immediately.
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
		}
		return true, nil
	case "backspace", "delete":
		if m.kanban.selectedCard() != nil {
			m.pushUndo()
			m.kanban.deleteCard()
			return true, nil
		}
		// On an empty column (no card selected) → delete the whole
		// kanban block. Undo restores it.
		m.pushUndo()
		m.kanban = nil
		m.deleteBlock(m.active)
		m.textareas[m.active].MoveToEnd()
		return true, nil
	case "p":
		if m.kanban.selectedCard() != nil {
			m.pushUndo()
			m.kanban.togglePriority()
			if m.kanbanSortByPrio {
				m.kanban.sortByPriority()
			}
		}
		return true, nil
	case "s":
		// Toggle priority sort for the whole document and persist.
		m.pushUndo()
		m.kanbanSortByPrio = !m.kanbanSortByPrio
		if m.kanbanSortByPrio {
			m.kanban.sortByPriority()
			m.status = "Sort by priority: on"
		} else {
			m.status = "Sort by priority: off"
		}
		m.statusStyle = statusSuccess
		m.persistKanbanSort()
		return true, m.scheduleStatusDismiss()
	case "tab":
		// Tab cycles to the next column.
		m.kanban.moveSelection(1, 0)
		m.kanbanAnchorTop = true
		return true, nil
	case "shift+tab":
		m.kanban.moveSelection(-1, 0)
		m.kanbanAnchorTop = true
		return true, nil
	}

	// Swallow plain printable keystrokes so the underlying textarea
	// (which holds serialized kanban markdown) isn't mutated. A direct
	// mutation would push a phantom undo entry whose state is the
	// serialized board, and the next focusBlock would silently overwrite
	// the typed characters via m.kanban.serialize(). Modifier-bearing
	// keys (Ctrl+S, Ctrl+R, Ctrl+G, Ctrl+Z, etc.) and special keys
	// (Esc, function keys) still fall through to the global handler.
	if msg.Text != "" && msg.Mod == 0 {
		return true, nil
	}
	return false, nil
}

// renderActiveKanban renders the focused Kanban block including the gutter
// label, mirroring the structure used by other active block render branches.
func (m Model) renderActiveKanban(idx int, b block.Block) string {
	width := m.width - gutterWidth
	if width < 30 {
		width = 30
	}
	board := m.renderKanbanBoard(idx, width)
	return prefixGutter(board, b, true)
}

// renderInactiveKanbanBoard renders a kanban board read-only (used for
// inactive editor blocks and view mode). Uses the same horizontal
// sliding-window layout as the active board so the look matches; the
// only difference is no card is highlighted as selected. The window
// offset is taken from `colOffset` — callers can drive scroll in view
// mode by adjusting it.
func renderInactiveKanbanBoard(content string, width, colOffset int, th theme.Theme) string {
	cols := block.ParseKanban(content)
	if len(cols) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("(empty kanban — focus and press n to add a card)")
	}

	visible, colWidth := kanbanVisibleCols(len(cols), width)
	startCol := colOffset
	if startCol < 0 {
		startCol = 0
	}
	if startCol > len(cols)-visible {
		startCol = len(cols) - visible
	}
	endCol := startCol + visible
	if endCol > len(cols) {
		endCol = len(cols)
	}

	colStyle := lipgloss.NewStyle().Width(colWidth)

	rendered := make([]string, 0, endCol-startCol)
	for ci := startCol; ci < endCol; ci++ {
		col := cols[ci]
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(th.Accent)).
			Width(colWidth).
			Padding(0, 1)
		count := lipgloss.NewStyle().
			Faint(true).
			Render(formatCount(len(col.Cards)))
		title := titleStyle.Render(format.StripControlChars(col.Title) + "  " + count)
		under := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Border)).
			Render(strings.Repeat("─", colWidth))

		var cardLines []string
		cardLines = append(cardLines, title)
		cardLines = append(cardLines, under)

		if len(col.Cards) == 0 {
			placeholder := lipgloss.NewStyle().Faint(true).Italic(true).Padding(0, 2).Render("no cards")
			cardLines = append(cardLines, placeholder)
		}

		for _, card := range col.Cards {
			cardLines = append(cardLines, renderKanbanCard(card, colWidth, false, false, "", th))
		}

		rendered = append(rendered, colStyle.Render(strings.Join(cardLines, "\n")))
	}

	gapStr := " "
	board := lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(rendered, gapStr)...)

	leftHidden := startCol
	rightHidden := len(cols) - endCol
	if leftHidden > 0 || rightHidden > 0 {
		left := renderColIndicator(leftHidden, true, th)
		right := renderColIndicator(rightHidden, false, th)
		board = lipgloss.JoinHorizontal(lipgloss.Top, left, board, right)
	}
	return board
}

// prefixGutter mirrors the gutter+separator decoration applied to all
// other active block renders so kanban aligns with surrounding blocks.
func prefixGutter(body string, b block.Block, active bool) string {
	th := theme.Current()
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))
	if !active {
		style = lipgloss.NewStyle().Faint(true)
	}
	label := style.Render(formatBlockLabel(b.Type))
	sep := style.Render("│")
	blank := style.Render("  ")
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if i == 0 {
			lines[i] = label + " " + sep + " " + l
		} else {
			lines[i] = blank + " " + sep + " " + l
		}
	}
	return strings.Join(lines, "\n")
}

// formatBlockLabel returns the 2-char gutter label for a block type.
func formatBlockLabel(bt block.BlockType) string {
	s := bt.Short()
	if len(s) >= 2 {
		return s[:2]
	}
	return s + " "
}

// formatCount returns "(n)" for a card count.
func formatCount(n int) string {
	if n == 0 {
		return "(0)"
	}
	if n == 1 {
		return "(1)"
	}
	// Avoid pulling fmt for one tiny use.
	digits := ""
	for n > 0 {
		digits = string(rune('0'+(n%10))) + digits
		n /= 10
	}
	return "(" + digits + ")"
}
