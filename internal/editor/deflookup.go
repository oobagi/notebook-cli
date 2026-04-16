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

// defItem represents one definition list block. Local items (from the
// current document) have BlockIdx >= 0 and Notebook/Note empty; remote
// items (gathered across notebooks) have BlockIdx == -1 and a populated
// Notebook/Note pair shown alongside the term.
type defItem struct {
	Term       string
	Definition string
	BlockIdx   int
	Notebook   string
	Note       string
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

	var location string
	if d.Notebook != "" || d.Note != "" {
		location = d.Notebook + "/" + d.Note
	}

	def := d.Definition
	overhead := len(term) + 8
	if location != "" {
		overhead += len(location) + 3
	}
	maxDef := width - overhead
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
		out := " " + marker + " " + termStr + "  " + defStr
		if location != "" {
			out += "  " + dim.Render("["+location+"]")
		}
		return out
	}
	termStr := lipgloss.NewStyle().Bold(true).Render(term)
	defStr := dim.Render(def)
	out := "   " + termStr + "  " + defStr
	if location != "" {
		out += "  " + dim.Render("["+location+"]")
	}
	return out
}

// newDefLookup creates a definition lookup ready for use.
func newDefLookup() defLookup {
	d := defLookup{}
	d.Prompt = ": "
	d.Placeholder = "search definitions across all files..."
	d.NoMatchText = "No matches"
	d.MaxVisible = 6
	return d
}

// open scans the current document for definition lists, optionally
// merging in cross-note definitions when remote != nil. Local items
// always come first so jumping within the current note stays cheap.
func (d *defLookup) open(blocks []block.Block, remote []DefinitionEntry) {
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
	for _, e := range remote {
		d.defItems = append(d.defItems, defItem{
			Term:       e.Term,
			Definition: e.Definition,
			BlockIdx:   -1,
			Notebook:   e.Notebook,
			Note:       e.Note,
		})
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
