package editor

import (
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/oobagi/notebook-cli/internal/ui"
)

// embedPicker is a picker for selecting a notebook/note target
// when inserting or editing an embed block.
type embedPicker struct {
	ui.Picker
}

// embedItem is a path string that implements ui.PickerItem.
type embedItem string

func (e embedItem) FilterValue() string { return string(e) }

func (e embedItem) RenderRow(selected bool, width int) string {
	th := theme.Current()
	label := string(e)
	if selected {
		label = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(th.Accent)).
			Render(label)
	} else {
		label = lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Muted)).
			Render(label)
	}
	return "  " + label
}

// newEmbedPicker creates an embed picker ready for use.
func newEmbedPicker() embedPicker {
	p := embedPicker{}
	p.Prompt = "\u2197 "
	p.Placeholder = "search notes..."
	p.NoMatchText = "No matching notes"
	p.MaxVisible = 10
	return p
}

// open populates the picker with paths and shows it.
func (p *embedPicker) open(paths []string) {
	items := make([]ui.PickerItem, len(paths))
	for i, path := range paths {
		items[i] = embedItem(path)
	}
	p.Open(items)
}

// selected returns the currently highlighted path, or "".
func (p *embedPicker) selected() string {
	sel := p.Selected()
	if sel == nil {
		return ""
	}
	return string(sel.(embedItem))
}
