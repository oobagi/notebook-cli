package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// palette is the "/" command palette for changing a block's type.
type palette struct {
	visible     bool
	items       []paletteItem
	filtered    []int          // indices into items matching current filter
	filter      string         // typed text after /
	cursor      int            // selection in filtered list
	blockIdx    int            // which block triggered the palette
	currentType block.BlockType // current block type (for visual indicator)
	hasContent  bool           // whether the block has content (hides Divider)
}

// paletteItem describes one entry in the palette.
type paletteItem struct {
	Label string
	Type  block.BlockType
	Icon  string
}

// defaultPaletteItems returns the full list of block-type entries.
func defaultPaletteItems() []paletteItem {
	return []paletteItem{
		{Icon: "\u00b6", Label: "Paragraph", Type: block.Paragraph},
		{Icon: "H1", Label: "Heading 1", Type: block.Heading1},
		{Icon: "H2", Label: "Heading 2", Type: block.Heading2},
		{Icon: "H3", Label: "Heading 3", Type: block.Heading3},
		{Icon: "\u2022", Label: "Bullet List", Type: block.BulletList},
		{Icon: "1.", Label: "Numbered List", Type: block.NumberedList},
		{Icon: "\u2610", Label: "Checklist", Type: block.Checklist},
		{Icon: "``", Label: "Code Block", Type: block.CodeBlock},
		{Icon: ">", Label: "Quote", Type: block.Quote},
		{Icon: ":", Label: "Definition", Type: block.DefinitionList},
		{Icon: "!", Label: "Callout", Type: block.Callout},
		{Icon: "\u2014", Label: "Divider", Type: block.Divider},
		{Icon: "\u2197", Label: "Embed", Type: block.Embed},
	}
}

// newPalette creates a palette with the default items and all indices visible.
func newPalette() palette {
	items := defaultPaletteItems()
	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}
	return palette{
		items:    items,
		filtered: filtered,
	}
}

// open makes the palette visible for the given block index and resets state.
func (p *palette) open(blockIdx int) {
	p.visible = true
	p.blockIdx = blockIdx
	p.filter = ""
	p.cursor = 0
	p.currentType = -1
	p.hasContent = false
	p.refilter()
}

// openForBlock makes the palette visible with awareness of the current block's
// type and whether it has content. Divider is hidden when the block has content
// to avoid silent data loss.
func (p *palette) openForBlock(blockIdx int, currentType block.BlockType, hasContent bool) {
	p.visible = true
	p.blockIdx = blockIdx
	p.filter = ""
	p.cursor = 0
	p.currentType = currentType
	p.hasContent = hasContent
	p.refilter()
}

// close hides the palette.
func (p *palette) close() {
	p.visible = false
	p.filter = ""
	p.cursor = 0
	p.currentType = -1
	p.hasContent = false
}

// refilter rebuilds the filtered index list based on the current filter text.
func (p *palette) refilter() {
	if p.filter == "" {
		var result []int
		for i, item := range p.items {
			// Hide Divider when the block has content (avoids silent data loss).
			if p.hasContent && item.Type == block.Divider {
				continue
			}
			result = append(result, i)
		}
		p.filtered = result
		p.cursor = 0
		return
	}

	lower := strings.ToLower(p.filter)
	var result []int
	for i, item := range p.items {
		// Hide Divider when the block has content (avoids silent data loss).
		if p.hasContent && item.Type == block.Divider {
			continue
		}
		if strings.Contains(strings.ToLower(item.Label), lower) {
			result = append(result, i)
		}
	}
	p.filtered = result
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// addFilterRune appends a rune to the filter and refilters.
func (p *palette) addFilterRune(r rune) {
	p.filter += string(r)
	p.refilter()
}

// deleteFilterRune removes the last rune from the filter. If the filter is
// already empty, it returns false to signal the palette should close.
func (p *palette) deleteFilterRune() bool {
	if p.filter == "" {
		return false
	}
	runes := []rune(p.filter)
	p.filter = string(runes[:len(runes)-1])
	p.refilter()
	return true
}

// moveUp moves the cursor up in the filtered list.
func (p *palette) moveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

// moveDown moves the cursor down in the filtered list.
func (p *palette) moveDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}
}

// selected returns the currently highlighted item, or nil if none.
func (p *palette) selected() *paletteItem {
	if len(p.filtered) == 0 {
		return nil
	}
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return nil
	}
	return &p.items[p.filtered[p.cursor]]
}

// render draws the palette as a floating box.
func (p *palette) render(width int) string {
	if !p.visible {
		return ""
	}

	th := theme.Current()

	// Filter display.
	filterLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Muted)).
		Render("/" + p.filter)

	var body string

	if len(p.filtered) == 0 {
		noMatch := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render("No matches")
		body = filterLine + "\n" + noMatch
	} else {
		// Build rows.
		var rows []string
		for i, idx := range p.filtered {
			item := p.items[idx]
			isCurrent := item.Type == p.currentType

			icon := lipgloss.NewStyle().
				Width(4).
				Align(lipgloss.Right).
				Foreground(lipgloss.Color(th.Muted)).
				Render(item.Icon)

			label := item.Label
			if isCurrent {
				label += " \u2713"
			}

			if i == p.cursor {
				color := th.Accent
				if isCurrent {
					color = th.Muted
				}
				label = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(color)).
					Render(label)
				icon = lipgloss.NewStyle().
					Width(4).
					Align(lipgloss.Right).
					Bold(true).
					Foreground(lipgloss.Color(color)).
					Render(item.Icon)
			} else if isCurrent {
				label = lipgloss.NewStyle().
					Foreground(lipgloss.Color(th.Muted)).
					Render(label)
			}

			rows = append(rows, icon+" "+label)
		}

		body = filterLine + "\n" + strings.Join(rows, "\n")
	}

	boxWidth := 28
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 20 {
		boxWidth = 20
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.Border)).
		Padding(0, 1).
		Width(boxWidth)

	return box.Render(body)
}
