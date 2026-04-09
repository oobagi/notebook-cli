package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/block"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// maxDefLookupItems is the maximum number of items shown at once.
const maxDefLookupItems = 6

// defLookup is a searchable panel that lists all definition list blocks
// in the current document, allowing the user to filter by term and jump
// to a selected definition. Renders as a footer modal above the status bar.
type defLookup struct {
	visible  bool
	items    []defItem
	filtered []int  // indices into items matching filter
	filter   string // typed search text
	cursor   int    // selection in filtered list
}

// defItem represents one definition list block in the document.
type defItem struct {
	Term       string
	Definition string
	BlockIdx   int // index in Model.blocks
}

// open scans blocks for definition lists and shows the lookup.
func (d *defLookup) open(blocks []block.Block) {
	d.visible = true
	d.filter = ""
	d.cursor = 0
	d.items = nil
	for i, b := range blocks {
		if b.Type == block.DefinitionList {
			term, def := block.ExtractDefinition(b.Content)
			d.items = append(d.items, defItem{
				Term:       term,
				Definition: def,
				BlockIdx:   i,
			})
		}
	}
	d.refilter()
}

// close hides the lookup.
func (d *defLookup) close() {
	d.visible = false
	d.filter = ""
	d.cursor = 0
	d.items = nil
}

// refilter rebuilds the filtered list based on current filter text.
func (d *defLookup) refilter() {
	if d.filter == "" {
		d.filtered = make([]int, len(d.items))
		for i := range d.items {
			d.filtered[i] = i
		}
		d.cursor = 0
		return
	}
	lower := strings.ToLower(d.filter)
	var result []int
	for i, item := range d.items {
		if strings.Contains(strings.ToLower(item.Term), lower) {
			result = append(result, i)
		}
	}
	d.filtered = result
	if d.cursor >= len(d.filtered) {
		d.cursor = len(d.filtered) - 1
	}
	if d.cursor < 0 {
		d.cursor = 0
	}
}

func (d *defLookup) addFilterRune(r rune) {
	d.filter += string(r)
	d.refilter()
}

func (d *defLookup) deleteFilterRune() bool {
	if d.filter == "" {
		return false
	}
	runes := []rune(d.filter)
	d.filter = string(runes[:len(runes)-1])
	d.refilter()
	return true
}

func (d *defLookup) moveUp() {
	if d.cursor > 0 {
		d.cursor--
	}
}

func (d *defLookup) moveDown() {
	if d.cursor < len(d.filtered)-1 {
		d.cursor++
	}
}

func (d *defLookup) selected() *defItem {
	if len(d.filtered) == 0 || d.cursor < 0 || d.cursor >= len(d.filtered) {
		return nil
	}
	return &d.items[d.filtered[d.cursor]]
}

// height returns the number of terminal lines the modal occupies.
func (d *defLookup) height() int {
	if !d.visible {
		return 0
	}
	if d.filter == "" {
		return 3 // border + filter + gap
	}
	n := len(d.filtered)
	if n > maxDefLookupItems {
		n = maxDefLookupItems
	}
	if n == 0 {
		n = 1 // "No definitions" / "No matches" line
	}
	return n + 3 // border + filter + items + gap
}

// visibleWindow returns the slice of filtered indices to display,
// keeping the cursor visible.
func (d *defLookup) visibleWindow() (start, end int) {
	n := len(d.filtered)
	if n <= maxDefLookupItems {
		return 0, n
	}
	// Keep cursor centered when possible.
	half := maxDefLookupItems / 2
	start = d.cursor - half
	if start < 0 {
		start = 0
	}
	end = start + maxDefLookupItems
	if end > n {
		end = n
		start = end - maxDefLookupItems
	}
	return start, end
}

// render draws the definition lookup as a full-width footer modal.
func (d *defLookup) render(width int) string {
	if !d.visible {
		return ""
	}

	th := theme.Current()
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted))
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true)

	// Top border — full width horizontal line.
	border := dim.Render(strings.Repeat("─", width))

	// Filter line.
	prompt := dim.Render(": ")
	filterText := d.filter
	if filterText == "" {
		filterText = dim.Render("search definitions...")
	}
	filterLine := " " + prompt + filterText

	// Only show results once the user starts typing.
	if d.filter == "" {
		return border + "\n" + filterLine + "\n"
	}

	// Items.
	var rows []string
	if len(d.items) == 0 {
		rows = append(rows, " "+dim.Render("No definitions in this note"))
	} else if len(d.filtered) == 0 {
		rows = append(rows, " "+dim.Render("No matches"))
	} else {
		start, end := d.visibleWindow()
		for vi := start; vi < end; vi++ {
			item := d.items[d.filtered[vi]]

			term := item.Term
			if term == "" {
				term = "(empty)"
			}
			def := item.Definition
			maxDef := width - len(term) - 8 // term + spacing + margins
			if maxDef < 10 {
				maxDef = 10
			}
			if len(def) > maxDef {
				def = def[:maxDef-3] + "..."
			}

			if vi == d.cursor {
				marker := accent.Render("›")
				termStr := accent.Render(term)
				defStr := dim.Render(def)
				rows = append(rows, " "+marker+" "+termStr+"  "+defStr)
			} else {
				termStr := lipgloss.NewStyle().Bold(true).Render(term)
				defStr := dim.Render(def)
				rows = append(rows, "   "+termStr+"  "+defStr)
			}
		}
	}

	return border + "\n" + filterLine + "\n" + strings.Join(rows, "\n") + "\n"
}
