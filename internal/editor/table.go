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
func (ts *tableState) loadCell(ta *textarea.Model, width int) {
	val := ""
	if ts.row >= 0 && ts.row < len(ts.cells) && ts.col >= 0 && ts.col < len(ts.cells[ts.row]) {
		val = ts.cells[ts.row][ts.col]
	}
	ta.SetWidth(width)
	ta.SetValue(val)
	ta.SetHeight(ta.VisualLineCount())
	ta.MoveToEnd()
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

// tableCellWidth computes the width for each cell column given available terminal width.
// Layout: | pad content pad | pad content pad | ...
// = (numCols+1) border chars + numCols * (2 padding + cellWidth)
func tableCellWidth(totalWidth, numCols int) int {
	if numCols <= 0 {
		return 5
	}
	// totalWidth = (numCols+1)*1 + numCols*(2+cw)
	// cw = (totalWidth - numCols - 1 - 2*numCols) / numCols
	cw := (totalWidth - numCols - 1 - 2*numCols) / numCols
	if cw < 5 {
		cw = 5
	}
	return cw
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
	cw := tableCellWidth(tableWidth, numCols)

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
			if r == ts.row && c == ts.col {
				// Active cell: render with cursor.
				cr := renderCellWithCursor(ta, cell, cw, cursorRawRow, cursorColInWrap, cursorWrapRow, r == 0 && th.Blocks.Table.HeaderBold)
				cellRenders = append(cellRenders, cr)
			} else {
				// Inactive cell: wrap and apply inline markdown.
				cr := renderStaticCell(cell, cw, r == 0 && th.Blocks.Table.HeaderBold)
				cellRenders = append(cellRenders, cr)
			}
		}
		gridRows = append(gridRows, cellRenders)
	}

	return buildGrid(gridRows, cw, numCols, tableWidth, bc, ts.row, ts.col)
}

// cellRender holds pre-rendered cell lines for grid assembly.
type cellRender struct {
	lines []string
}

// renderCellWithCursor renders the active cell with the textarea cursor.
func renderCellWithCursor(ta textarea.Model, _ string, cw int, cursorRawRow, cursorColInWrap, cursorWrapRow int, bold bool) cellRender {
	content := ta.Value()
	rawLines := strings.Split(content, "\n")
	boldStyle := lipgloss.NewStyle()
	if bold {
		boldStyle = boldStyle.Bold(true)
	}

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
func renderStaticCell(content string, cw int, bold bool) cellRender {
	wrapped := wrapText(content, cw)
	wrapped = format.RenderInlineMarkdown(wrapped)

	boldStyle := lipgloss.NewStyle()
	if bold {
		boldStyle = boldStyle.Bold(true)
	}

	lines := strings.Split(wrapped, "\n")
	for i, l := range lines {
		if bold {
			l = boldStyle.Render(l)
		}
		if pad := cw - lipgloss.Width(l); pad > 0 {
			l += strings.Repeat(" ", pad)
		}
		lines[i] = l
	}

	return cellRender{lines: lines}
}

// buildGrid assembles the visual grid from cell renders.
func buildGrid(gridRows [][]cellRender, cw, numCols, tableWidth int, bc lipgloss.Style, activeRow, activeCol int) string {
	pipe := bc.Render("\u2502")

	// Build border lines.
	cellDash := strings.Repeat("\u2500", cw+2) // +2 for padding spaces
	topBorder := bc.Render("\u256D" + strings.Join(repeatStr(cellDash, numCols), "\u252C") + "\u256E")
	midBorder := bc.Render("\u251C" + strings.Join(repeatStr(cellDash, numCols), "\u253C") + "\u2524")
	botBorder := bc.Render("\u2570" + strings.Join(repeatStr(cellDash, numCols), "\u2534") + "\u256F")

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

		// Pad cells with fewer lines.
		for c := range rowCells {
			for len(rowCells[c].lines) < maxLines {
				rowCells[c].lines = append(rowCells[c].lines, strings.Repeat(" ", cw))
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

	var cw int
	if wordWrap {
		cw = tableCellWidth(width, numCols)
	} else {
		// Natural content-width columns.
		cw = 3
		for _, row := range cells {
			for _, cell := range row {
				if len(cell) > cw {
					cw = len(cell)
				}
			}
		}
	}

	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

	var gridRows [][]cellRender
	for r, row := range cells {
		var cellRenders []cellRender
		for _, cell := range row {
			cr := renderStaticCell(cell, cw, r == 0 && headerBold)
			cellRenders = append(cellRenders, cr)
		}
		// Normalize column count.
		for len(cellRenders) < numCols {
			cellRenders = append(cellRenders, renderStaticCell("", cw, false))
		}
		gridRows = append(gridRows, cellRenders)
	}

	return buildGrid(gridRows, cw, numCols, width, bc, -1, -1)
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
		cw := tableCellWidth(tableWidth, m.table.numCols())
		cursorCol += m.table.col * (cw + 3) // each col: pad + content + pad + border

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
