package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/theme"
)

// embedPicker is a floating popup for selecting a notebook/note target
// when inserting or editing an embed block. Works like the command palette:
// type to filter, arrow keys to navigate, Enter to select.
type embedPicker struct {
	visible  bool
	items    []string // "notebook/note" paths
	filtered []int    // indices into items matching filter
	filter   string
	cursor   int
}

// open populates the picker with items and shows it.
func (p *embedPicker) open(items []string) {
	p.visible = true
	p.items = items
	p.filter = ""
	p.cursor = 0
	p.refilter()
}

// close hides the picker.
func (p *embedPicker) close() {
	p.visible = false
	p.filter = ""
	p.cursor = 0
}

// refilter rebuilds the filtered index list from the current filter text.
func (p *embedPicker) refilter() {
	if p.filter == "" {
		p.filtered = make([]int, len(p.items))
		for i := range p.items {
			p.filtered[i] = i
		}
		p.cursor = 0
		return
	}

	lower := strings.ToLower(p.filter)
	var result []int
	for i, item := range p.items {
		if strings.Contains(strings.ToLower(item), lower) {
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

// addFilterRune appends a rune and refilters.
func (p *embedPicker) addFilterRune(r rune) {
	p.filter += string(r)
	p.refilter()
}

// deleteFilterRune removes the last rune. Returns false if filter was empty.
func (p *embedPicker) deleteFilterRune() bool {
	if p.filter == "" {
		return false
	}
	runes := []rune(p.filter)
	p.filter = string(runes[:len(runes)-1])
	p.refilter()
	return true
}

func (p *embedPicker) moveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *embedPicker) moveDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}
}

// selected returns the currently highlighted path, or "" if none.
func (p *embedPicker) selected() string {
	if len(p.filtered) == 0 || p.cursor < 0 || p.cursor >= len(p.filtered) {
		return ""
	}
	return p.items[p.filtered[p.cursor]]
}

// render draws the picker as a floating box.
func (p *embedPicker) render(width int) string {
	if !p.visible {
		return ""
	}

	th := theme.Current()

	filterLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Muted)).
		Render("\u2197 " + p.filter)

	var body string

	if len(p.filtered) == 0 {
		noMatch := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render("No matching notes")
		body = filterLine + "\n" + noMatch
	} else {
		// Show at most 10 items to keep the popup compact.
		maxVisible := 10
		start := 0
		if p.cursor >= maxVisible {
			start = p.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(p.filtered) {
			end = len(p.filtered)
		}

		var rows []string
		for i := start; i < end; i++ {
			idx := p.filtered[i]
			path := p.items[idx]

			// Split into notebook and note parts for display.
			label := path
			if i == p.cursor {
				label = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(th.Accent)).
					Render(label)
			} else {
				label = lipgloss.NewStyle().
					Foreground(lipgloss.Color(th.Muted)).
					Render(label)
			}
			rows = append(rows, "  "+label)
		}

		// Scroll indicators.
		if start > 0 {
			rows = append([]string{lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("  \u2191 more")}, rows...)
		}
		if end < len(p.filtered) {
			rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("  \u2193 more"))
		}

		body = filterLine + "\n" + strings.Join(rows, "\n")
	}

	boxWidth := 40
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 24 {
		boxWidth = 24
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.Border)).
		Padding(0, 1).
		Width(boxWidth)

	return box.Render(body)
}
