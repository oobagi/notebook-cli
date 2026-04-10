package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// PickerItem is the interface that picker entries must satisfy.
type PickerItem interface {
	FilterValue() string                      // text matched against the filter
	RenderRow(selected bool, width int) string // render one row
}

// Picker is a generic filterable list with cursor navigation.
// It backs the command palette, embed picker, and definition lookup.
type Picker struct {
	Visible     bool
	items       []PickerItem
	filtered    []int
	filter      string
	cursor      int
	Prompt      string // displayed before the filter text, e.g. "/", "↗ ", ": "
	Placeholder string // shown when filter is empty, e.g. "type to filter..."
	NoMatchText string // shown when filter yields no results
	MaxVisible  int    // max items shown at once; 0 = unlimited

	// FilterFunc is an optional predicate evaluated during refilter.
	// When non-nil it is called for every item; items for which it
	// returns false are excluded even if they match the text filter.
	// This supports use-cases like hiding the Divider option when
	// the current block has content.
	FilterFunc func(PickerItem) bool
}

// Open populates the picker with items, resets state, and makes it visible.
func (p *Picker) Open(items []PickerItem) {
	p.Visible = true
	p.items = items
	p.filter = ""
	p.cursor = 0
	p.Refilter()
}

// Close hides the picker and resets state.
func (p *Picker) Close() {
	p.Visible = false
	p.filter = ""
	p.cursor = 0
	p.items = nil
	p.FilterFunc = nil
}

// Filter returns the current filter text.
func (p *Picker) Filter() string { return p.filter }

// Refilter rebuilds the filtered index list from the current filter text.
func (p *Picker) Refilter() {
	lower := strings.ToLower(p.filter)
	var result []int
	for i, item := range p.items {
		if p.FilterFunc != nil && !p.FilterFunc(item) {
			continue
		}
		if lower == "" || strings.Contains(strings.ToLower(item.FilterValue()), lower) {
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

// AddFilterRune appends a rune to the filter and refilters.
func (p *Picker) AddFilterRune(r rune) {
	p.filter += string(r)
	p.Refilter()
}

// DeleteFilterRune removes the last rune. Returns false if the filter was
// already empty, signalling the caller should close the picker.
func (p *Picker) DeleteFilterRune() bool {
	if p.filter == "" {
		return false
	}
	runes := []rune(p.filter)
	p.filter = string(runes[:len(runes)-1])
	p.Refilter()
	return true
}

// DeleteFilterWord removes the last word from the filter. Returns false if the
// filter was already empty, signalling the caller should close the picker.
func (p *Picker) DeleteFilterWord() bool {
	if p.filter == "" {
		return false
	}
	// Trim trailing spaces, then remove back to the next space.
	s := strings.TrimRight(p.filter, " ")
	if s == "" {
		p.filter = ""
		p.Refilter()
		return true
	}
	if i := strings.LastIndex(s, " "); i >= 0 {
		p.filter = s[:i+1]
	} else {
		p.filter = ""
	}
	p.Refilter()
	return true
}

// ClearFilter resets the filter text and refilters. Returns false if the
// filter was already empty.
func (p *Picker) ClearFilter() bool {
	if p.filter == "" {
		return false
	}
	p.filter = ""
	p.Refilter()
	return true
}

// MoveUp moves the cursor up in the filtered list.
func (p *Picker) MoveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

// MoveDown moves the cursor down in the filtered list.
func (p *Picker) MoveDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}
}

// Selected returns the currently highlighted item, or nil if none.
func (p *Picker) Selected() PickerItem {
	if len(p.filtered) == 0 || p.cursor < 0 || p.cursor >= len(p.filtered) {
		return nil
	}
	return p.items[p.filtered[p.cursor]]
}

// Cursor returns the current cursor index within the filtered list.
func (p *Picker) Cursor() int { return p.cursor }

// FilteredCount returns the number of items matching the current filter.
func (p *Picker) FilteredCount() int { return len(p.filtered) }

// FilteredIndices returns a copy of the filtered index list. Useful for
// testing and inspection.
func (p *Picker) FilteredIndices() []int {
	out := make([]int, len(p.filtered))
	copy(out, p.filtered)
	return out
}

// Items returns the full item list. Useful for testing and inspection.
func (p *Picker) Items() []PickerItem { return p.items }

// Height returns the number of terminal lines the footer panel occupies.
func (p *Picker) Height() int {
	if !p.Visible {
		return 0
	}
	// border + filter line = 2
	n := p.visibleCount()
	if n == 0 && p.filter != "" {
		n = 1 // "No matches" line
	}
	return n + 2
}

func (p *Picker) visibleCount() int {
	n := len(p.filtered)
	if p.MaxVisible > 0 && n > p.MaxVisible {
		n = p.MaxVisible
	}
	return n
}

// visibleWindow returns the start/end slice of filtered indices to show.
func (p *Picker) visibleWindow() (start, end int) {
	n := len(p.filtered)
	max := p.MaxVisible
	if max <= 0 || n <= max {
		return 0, n
	}
	half := max / 2
	start = p.cursor - half
	if start < 0 {
		start = 0
	}
	end = start + max
	if end > n {
		end = n
		start = end - max
	}
	return start, end
}

// RenderFooter draws the picker as a footer panel with a top border.
func (p *Picker) RenderFooter(width int) string {
	if !p.Visible {
		return ""
	}

	th := theme.Current()
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted))

	// Top border.
	border := dim.Render(strings.Repeat("\u2500", width))

	// Filter line.
	prompt := dim.Render(p.Prompt)
	filterText := p.filter
	if filterText == "" && p.Placeholder != "" {
		filterText = dim.Render(p.Placeholder)
	}
	filterLine := " " + prompt + filterText

	// No items to show if filter is empty and MaxVisible allows.
	if p.filter == "" && len(p.filtered) > 0 {
		// Show the list even when filter is empty.
	}

	if len(p.filtered) == 0 && p.filter != "" {
		noMatch := dim.Render(p.NoMatchText)
		return border + "\n" + filterLine + "\n" + " " + noMatch + "\n"
	}

	if len(p.filtered) == 0 {
		return border + "\n" + filterLine + "\n"
	}

	// Build rows.
	start, end := p.visibleWindow()
	var rows []string

	// Scroll-up indicator.
	if start > 0 {
		rows = append(rows, dim.Render("  \u2191 more"))
	}

	rowWidth := width - 4 // margin for borders/padding
	if rowWidth < 20 {
		rowWidth = 20
	}

	for i := start; i < end; i++ {
		item := p.items[p.filtered[i]]
		rows = append(rows, item.RenderRow(i == p.cursor, rowWidth))
	}

	// Scroll-down indicator.
	if end < len(p.filtered) {
		rows = append(rows, dim.Render("  \u2193 more"))
	}

	return border + "\n" + filterLine + "\n" + strings.Join(rows, "\n") + "\n"
}
