package editor

import (
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/oobagi/notebook-cli/internal/ui"
)

// palette is the "/" command palette for changing a block's type.
type palette struct {
	ui.Picker
	blockIdx    int            // which block triggered the palette
	currentType block.BlockType // current block type (for visual indicator)
	hasContent  bool           // whether the block has content (hides Divider)
}

// paletteItem describes one entry in the palette.
type paletteItem struct {
	Label       string
	Type        block.BlockType
	Icon        string
	CurrentType block.BlockType // set at open time so RenderRow can show ✓
}

func (p paletteItem) FilterValue() string { return p.Label }

func (p paletteItem) RenderRow(selected bool, width int) string {
	th := theme.Current()
	isCurrent := p.Type == p.CurrentType

	icon := lipgloss.NewStyle().
		Width(4).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color(th.Muted)).
		Render(p.Icon)

	label := p.Label
	if isCurrent {
		label += " \u2713"
	}

	if selected {
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
			Render(p.Icon)
	} else if isCurrent {
		label = lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render(label)
	}

	return " " + icon + " " + label
}

// paletteItemDefs returns the raw palette item definitions (without currentType set).
var paletteItemDefs = []struct {
	Icon  string
	Label string
	Type  block.BlockType
}{
	{"\u00b6", "Paragraph", block.Paragraph},
	{"H1", "Heading 1", block.Heading1},
	{"H2", "Heading 2", block.Heading2},
	{"H3", "Heading 3", block.Heading3},
	{"\u2022", "Bullet List", block.BulletList},
	{"1.", "Numbered List", block.NumberedList},
	{"\u2610", "Checklist", block.Checklist},
	{"``", "Code Block", block.CodeBlock},
	{"\u229e", "Table", block.Table},
	{">", "Quote", block.Quote},
	{":", "Definition", block.DefinitionList},
	{"!", "Callout", block.Callout},
	{"\u2014", "Divider", block.Divider},
	{"\u2197", "Embed", block.Embed},
}

// defaultPaletteItems returns the full list of block-type entries.
func defaultPaletteItems(currentType block.BlockType) []ui.PickerItem {
	items := make([]ui.PickerItem, len(paletteItemDefs))
	for i, d := range paletteItemDefs {
		items[i] = paletteItem{
			Icon:        d.Icon,
			Label:       d.Label,
			Type:        d.Type,
			CurrentType: currentType,
		}
	}
	return items
}

// newPalette creates a palette ready for use.
func newPalette() palette {
	p := palette{}
	p.Prompt = "/ "
	p.Placeholder = "type to filter..."
	p.NoMatchText = "No matches"
	return p
}

// openForBlock makes the palette visible with awareness of the current block's
// type and whether it has content. Divider is hidden when the block has content
// to avoid silent data loss.
func (p *palette) openForBlock(blockIdx int, currentType block.BlockType, hasContent bool) {
	p.blockIdx = blockIdx
	p.currentType = currentType
	p.hasContent = hasContent

	// Set filter func to hide Divider when block has content.
	p.FilterFunc = func(item ui.PickerItem) bool {
		if pi, ok := item.(paletteItem); ok {
			if hasContent && pi.Type == block.Divider {
				return false
			}
		}
		return true
	}

	p.Open(defaultPaletteItems(currentType))
}

// open makes the palette visible for the given block index (simple open without type context).
func (p *palette) open(blockIdx int) {
	p.openForBlock(blockIdx, -1, false)
}

// close hides the palette and clears context.
func (p *palette) close() {
	p.Close()
	p.currentType = -1
	p.hasContent = false
}

// selected returns the currently highlighted palette item, or nil.
func (p *palette) selected() *paletteItem {
	sel := p.Selected()
	if sel == nil {
		return nil
	}
	pi := sel.(paletteItem)
	return &pi
}
