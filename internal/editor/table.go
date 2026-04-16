package editor

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// tableState tracks the cell grid and active cell position for a Table block.
type tableState struct {
	cells  [][]string
	aligns []string
	row    int
	col    int
}

// initTable parses table content into a tableState.
func initTable(content string) *tableState {
	cells, aligns := block.ParseTableCells(content)
	if len(cells) == 0 {
		cells = [][]string{{"", ""}, {"", ""}}
		aligns = []string{"left", "left"}
	}
	return &tableState{cells: cells, aligns: aligns}
}

// syncCell writes the textarea's current value back to cells[row][col].
func (ts *tableState) syncCell(ta textarea.Model) {
	if ts.row < 0 || ts.row >= len(ts.cells) {
		return
	}
	if ts.col < 0 || ts.col >= len(ts.cells[ts.row]) {
		return
	}
	ts.cells[ts.row][ts.col] = ta.Value()
}

// loadCell sets the textarea's value to cells[row][col] and configures width.
// If toEnd is true the cursor is placed at the end; otherwise at the beginning.
func (ts *tableState) loadCell(ta *textarea.Model, width int, toEnd bool) {
	val := ""
	if ts.row >= 0 && ts.row < len(ts.cells) && ts.col >= 0 && ts.col < len(ts.cells[ts.row]) {
		val = ts.cells[ts.row][ts.col]
	}
	ta.SetWidth(width)
	ta.SetValue(val)
	ta.SetHeight(ta.VisualLineCount())
	if toEnd {
		ta.MoveToEnd()
	} else {
		ta.MoveToBegin()
	}
}

// serialize converts cells back to pipe content.
func (ts *tableState) serialize() string {
	return block.SerializeTableCells(ts.cells, ts.aligns)
}

// numCols returns the column count.
func (ts *tableState) numCols() int {
	if len(ts.cells) == 0 {
		return 0
	}
	return len(ts.cells[0])
}

// isRowEmpty reports whether every cell in the given row is empty.
func (ts *tableState) isRowEmpty(row int) bool {
	if row < 0 || row >= len(ts.cells) {
		return false
	}
	for _, cell := range ts.cells[row] {
		if cell != "" {
			return false
		}
	}
	return true
}

// nextCell advances to the next cell. Returns true if a new row was added.
func (ts *tableState) nextCell() bool {
	nc := ts.numCols()
	if nc == 0 {
		return false
	}
	ts.col++
	if ts.col >= nc {
		ts.col = 0
		ts.row++
	}
	if ts.row >= len(ts.cells) {
		// Append new empty row.
		newRow := make([]string, nc)
		ts.cells = append(ts.cells, newRow)
		return true
	}
	return false
}

// prevCell retreats to the previous cell.
func (ts *tableState) prevCell() {
	nc := ts.numCols()
	if nc == 0 {
		return
	}
	ts.col--
	if ts.col < 0 {
		ts.col = nc - 1
		ts.row--
	}
	if ts.row < 0 {
		ts.row = 0
		ts.col = 0
	}
}

// addRow inserts an empty row after the current row and moves to it.
func (ts *tableState) addRow() {
	nc := ts.numCols()
	if nc == 0 {
		return
	}
	newRow := make([]string, nc)
	insertAt := ts.row + 1
	ts.cells = append(ts.cells[:insertAt], append([][]string{newRow}, ts.cells[insertAt:]...)...)
	ts.row = insertAt
	ts.col = 0
}

// addCol inserts an empty column after the current column and moves to it.
func (ts *tableState) addCol() {
	insertAt := ts.col + 1
	for i := range ts.cells {
		row := ts.cells[i]
		newRow := make([]string, len(row)+1)
		copy(newRow, row[:insertAt])
		newRow[insertAt] = ""
		copy(newRow[insertAt+1:], row[insertAt:])
		ts.cells[i] = newRow
	}
	ts.aligns = append(ts.aligns[:insertAt], append([]string{"left"}, ts.aligns[insertAt:]...)...)
	ts.col = insertAt
}

// deleteRow removes the current row. Returns false if only one data row remains.
func (ts *tableState) deleteRow() bool {
	if len(ts.cells) <= 1 {
		return false
	}
	ts.cells = append(ts.cells[:ts.row], ts.cells[ts.row+1:]...)
	if ts.row >= len(ts.cells) {
		ts.row = len(ts.cells) - 1
	}
	return true
}

// deleteCol removes the current column. Returns false if only one column remains.
func (ts *tableState) deleteCol() bool {
	nc := ts.numCols()
	if nc <= 1 {
		return false
	}
	for i := range ts.cells {
		ts.cells[i] = append(ts.cells[i][:ts.col], ts.cells[i][ts.col+1:]...)
	}
	if ts.col < len(ts.aligns) {
		ts.aligns = append(ts.aligns[:ts.col], ts.aligns[ts.col+1:]...)
	}
	if ts.col >= ts.numCols() {
		ts.col = ts.numCols() - 1
	}
	return true
}

// tableCellWidth computes the base content width for each cell column.
// This is the minimum width; some columns may be 1 wider to fill the
// available space (see tableCellWidths).
func tableCellWidth(totalWidth, numCols int) int {
	if numCols <= 0 {
		return 5
	}
	avail := totalWidth - numCols - 1 - 2*numCols
	cw := avail / numCols
	if cw < 5 {
		cw = 5
	}
	return cw
}

// tableCellWidths returns per-column widths that exactly fill totalWidth.
// The first `remainder` columns are 1 wider than the rest so the table
// grid spans the full available width without gaps.
func tableCellWidths(totalWidth, numCols int) []int {
	if numCols <= 0 {
		return nil
	}
	avail := totalWidth - numCols - 1 - 2*numCols
	base := avail / numCols
	if base < 5 {
		base = 5
	}
	rem := avail - base*numCols
	if rem < 0 {
		rem = 0
	}
	widths := make([]int, numCols)
	for i := range widths {
		widths[i] = base
		if i < rem {
			widths[i]++
		}
	}
	return widths
}

// renderActiveTable renders the table grid with one cell showing the textarea cursor.
func renderActiveTable(m Model, idx int, ta textarea.Model, width int, wordWrap bool) string {
	ts := m.table
	if ts == nil {
		return ""
	}
	th := theme.Current()
	numCols := ts.numCols()
	if numCols == 0 {
		return ""
	}

	// Content width for the table area (no prefix for tables).
	tableWidth := width - gutterWidth
	if tableWidth < 10 {
		tableWidth = 10
	}
	colWidths := tableCellWidths(tableWidth, numCols)
	cw := tableCellWidth(tableWidth, numCols) // base width for textarea

	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Border))

	// Get cursor info from the textarea for the active cell.
	cursorRawRow := ta.Line()
	li := ta.LineInfo()
	cursorColInWrap := li.ColumnOffset
	cursorWrapRow := li.RowOffset

	var gridRows [][]cellRender
	for r, row := range ts.cells {
		var cellRenders []cellRender
		for c, cell := range row {
			cellCW := cw
			if c < len(colWidths) {
				cellCW = colWidths[c]
			}
			if r == ts.row && c == ts.col {
				cr := renderCellWithCursor(ta, cell, cellCW, cursorRawRow, cursorColInWrap, cursorWrapRow, r == 0 && th.Blocks.Table.HeaderBold, wordWrap)
				cellRenders = append(cellRenders, cr)
			} else {
				cr := renderStaticCell(cell, cellCW, r == 0 && th.Blocks.Table.HeaderBold, wordWrap)
				cellRenders = append(cellRenders, cr)
			}
		}
		gridRows = append(gridRows, cellRenders)
	}

	return buildGrid(gridRows, colWidths, numCols, tableWidth, bc, ts.row, ts.col)
}

// cellRender holds pre-rendered cell lines for grid assembly.
type cellRender struct {
	lines []string
}

// renderCellWithCursor renders the active cell with the textarea cursor.
func renderCellWithCursor(ta textarea.Model, _ string, cw int, cursorRawRow, cursorColInWrap, cursorWrapRow int, bold bool, wordWrap bool) cellRender {
	content := ta.Value()
	rawLines := strings.Split(content, "\n")
	boldStyle := lipgloss.NewStyle()
	if bold {
		boldStyle = boldStyle.Bold(true)
	}

	if !wordWrap {
		// No-wrap: one visual line per raw line, scrolled to cursor.
		var visualLines []string
		for rawIdx, raw := range rawLines {
			line := raw
			isCursorLine := rawIdx == cursorRawRow

			if isCursorLine {
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
				if bold {
					line = boldStyle.Render(before) + ta.CursorView() + boldStyle.Render(after)
				} else {
					line = before + ta.CursorView() + after
				}
				line = scrollOrTruncate(line, cw, col, true)
			} else {
				if bold {
					line = boldStyle.Render(line)
				}
				line = scrollOrTruncate(line, cw, 0, false)
			}

			// Pad to cw after truncation.
			if pad := cw - lipgloss.Width(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			visualLines = append(visualLines, line)
		}
		return cellRender{lines: visualLines}
	}

	// Word-wrap: wrap text within the cell.
	var visualLines []string
	for rawIdx, raw := range rawLines {
		segs := textarea.Wrap([]rune(raw), cw)
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
				if bold {
					line = boldStyle.Render(before) + ta.CursorView() + boldStyle.Render(after)
				} else {
					line = before + ta.CursorView() + after
				}
			} else if bold {
				line = boldStyle.Render(line)
			}

			// Pad to cw.
			if pad := cw - lipgloss.Width(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			visualLines = append(visualLines, line)
		}
	}

	return cellRender{lines: visualLines}
}

// renderStaticCell renders an inactive cell with wrapping and inline markdown.
func renderStaticCell(content string, cw int, bold bool, wordWrap bool) cellRender {
	var rendered string
	if wordWrap {
		rendered = wrapText(content, cw)
	} else {
		rendered = content
	}
	rendered = format.RenderInlineMarkdown(rendered)

	boldStyle := lipgloss.NewStyle()
	if bold {
		boldStyle = boldStyle.Bold(true)
	}

	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		if bold {
			l = boldStyle.Render(l)
		}
		if !wordWrap {
			l = scrollOrTruncate(l, cw, 0, false)
		}
		if pad := cw - lipgloss.Width(l); pad > 0 {
			l += strings.Repeat(" ", pad)
		}
		lines[i] = l
	}

	return cellRender{lines: lines}
}

// buildGrid assembles the visual grid from cell renders.
func buildGrid(gridRows [][]cellRender, colWidths []int, numCols, tableWidth int, bc lipgloss.Style, activeRow, activeCol int) string {
	pipe := bc.Render("\u2502")

	// Build border lines with per-column widths.
	var dashes []string
	for _, w := range colWidths {
		dashes = append(dashes, strings.Repeat("\u2500", w+2))
	}
	topBorder := bc.Render("\u256D" + strings.Join(dashes, "\u252C") + "\u256E")
	midBorder := bc.Render("\u251C" + strings.Join(dashes, "\u253C") + "\u2524")
	botBorder := bc.Render("\u2570" + strings.Join(dashes, "\u2534") + "\u256F")

	var out []string
	out = append(out, topBorder)

	for r, rowCells := range gridRows {
		if r > 0 {
			out = append(out, midBorder)
		}

		// Determine max visual lines in this row.
		maxLines := 1
		for _, cr := range rowCells {
			if len(cr.lines) > maxLines {
				maxLines = len(cr.lines)
			}
		}

		// Pad cells with fewer lines using per-column widths.
		for c := range rowCells {
			w := colWidths[0]
			if c < len(colWidths) {
				w = colWidths[c]
			}
			for len(rowCells[c].lines) < maxLines {
				rowCells[c].lines = append(rowCells[c].lines, strings.Repeat(" ", w))
			}
		}

		// Build each visual line of this row.
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			var parts []string
			for _, cr := range rowCells {
				parts = append(parts, " "+cr.lines[lineIdx]+" ")
			}
			out = append(out, pipe+strings.Join(parts, pipe)+pipe)
		}
	}

	out = append(out, botBorder)
	return strings.Join(out, "\n")
}

// repeatStr returns a slice of n copies of s.
func repeatStr(s string, n int) []string {
	result := make([]string, n)
	for i := range result {
		result[i] = s
	}
	return result
}

// renderTableGrid renders a static table grid (for inactive blocks and view mode).
func renderTableGrid(content string, width int, borderColor string, headerBold bool, wordWrap bool) string {
	cells, _ := block.ParseTableCells(content)
	if len(cells) == 0 {
		return ""
	}

	numCols := 0
	for _, row := range cells {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return ""
	}

	colWidths := tableCellWidths(width, numCols)

	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

	var gridRows [][]cellRender
	for r, row := range cells {
		var cellRenders []cellRender
		for c, cell := range row {
			cellCW := colWidths[0]
			if c < len(colWidths) {
				cellCW = colWidths[c]
			}
			cr := renderStaticCell(cell, cellCW, r == 0 && headerBold, wordWrap)
			cellRenders = append(cellRenders, cr)
		}
		// Normalize column count.
		for c := len(cellRenders); c < numCols; c++ {
			cellCW := colWidths[0]
			if c < len(colWidths) {
				cellCW = colWidths[c]
			}
			cellRenders = append(cellRenders, renderStaticCell("", cellCW, false, wordWrap))
		}
		gridRows = append(gridRows, cellRenders)
	}

	return buildGrid(gridRows, colWidths, numCols, width, bc, -1, -1)
}

// renderActiveTableFull renders the active table block with gutter and scroll handling.
func (m Model) renderActiveTableFull(idx int, b block.Block) string {
	if idx < 0 || idx >= len(m.textareas) || m.table == nil {
		return ""
	}

	ta := m.textareas[idx]
	th := theme.Current()

	tableWidth := m.width - gutterWidth
	if tableWidth < 10 {
		tableWidth = 10
	}

	rendered := renderActiveTable(m, idx, ta, m.width, m.wordWrap)

	// Build gutter with block type label.
	label := fmt.Sprintf("%2s", b.Type.Short())
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))
	sepStr := accentStyle.Render("\u2502")
	labelStr := accentStyle.Render(label)
	blankLabel := accentStyle.Render("  ")

	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		if i == 0 {
			lines[i] = labelStr + " " + sepStr + " " + l
		} else {
			lines[i] = blankLabel + " " + sepStr + " " + l
		}
	}

	// Scroll/truncate for no-wrap mode.
	if !m.wordWrap {
		// Compute cursor column: gutter + border + padding + cell cursor position.
		cursorCol := gutterWidth + 1 + 1 + ta.LineInfo().ColumnOffset
		// Add offset for columns before the active one.
		colWidths := tableCellWidths(tableWidth, m.table.numCols())
		for c := 0; c < m.table.col && c < len(colWidths); c++ {
			cursorCol += colWidths[c] + 3 // content + pad + pad + border
		}

		// Find which line has the cursor.
		// The cursor is in the active row. Account for top border (1 line)
		// + rows before active * (max_lines_per_row + 1 separator).
		// For simplicity, don't try to compute exact cursor line for scroll;
		// just truncate all lines.
		for i, l := range lines {
			lines[i] = scrollOrTruncate(l, m.width, cursorCol, false)
		}
	} else {
		for i, l := range lines {
			if lipgloss.Width(l) > m.width {
				lines[i] = ansi.Truncate(l, m.width, "")
			}
		}
	}

	return strings.Join(lines, "\n")
}
