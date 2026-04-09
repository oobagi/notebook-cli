package editor

import (
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/oobagi/notebook-cli/internal/ui"
)

// defLookup is a searchable panel that lists all definition list blocks
// in the current document, allowing the user to filter by term and jump
// to a selected definition. Renders as a footer modal above the status bar.
type defLookup struct {
	ui.Picker
	defItems []defItem // raw items with block indices
}

// defItem represents one definition list block in the document.
type defItem struct {
	Term       string
	Definition string
	BlockIdx   int
}

func (d defItem) FilterValue() string { return d.Term }

func (d defItem) RenderRow(selected bool, width int) string {
	th := theme.Current()
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted))
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true)

	term := d.Term
	if term == "" {
		term = "(empty)"
	}
	def := d.Definition
	maxDef := width - len(term) - 8
	if maxDef < 10 {
		maxDef = 10
	}
	if len(def) > maxDef {
		def = def[:maxDef-3] + "..."
	}

	if selected {
		marker := accent.Render("\u203a")
		termStr := accent.Render(term)
		defStr := dim.Render(def)
		return " " + marker + " " + termStr + "  " + defStr
	}
	termStr := lipgloss.NewStyle().Bold(true).Render(term)
	defStr := dim.Render(def)
	return "   " + termStr + "  " + defStr
}

// newDefLookup creates a definition lookup ready for use.
func newDefLookup() defLookup {
	d := defLookup{}
	d.Prompt = ": "
	d.Placeholder = "search definitions..."
	d.NoMatchText = "No matches"
	d.MaxVisible = 6
	return d
}

// open scans blocks for definition lists and shows the lookup.
func (d *defLookup) open(blocks []block.Block) {
	d.defItems = nil
	for i, b := range blocks {
		if b.Type == block.DefinitionList {
			term, def := block.ExtractDefinition(b.Content)
			d.defItems = append(d.defItems, defItem{
				Term:       term,
				Definition: def,
				BlockIdx:   i,
			})
		}
	}
	items := make([]ui.PickerItem, len(d.defItems))
	for i, di := range d.defItems {
		items[i] = di
	}
	d.Open(items)
}

// close hides the lookup.
func (d *defLookup) close() {
	d.Close()
	d.defItems = nil
}

// selected returns the currently highlighted definition item, or nil.
func (d *defLookup) selected() *defItem {
	sel := d.Selected()
	if sel == nil {
		return nil
	}
	di := sel.(defItem)
	return &di
}

// height returns the number of terminal lines the modal occupies.
func (d *defLookup) height() int {
	return d.Picker.Height()
}
