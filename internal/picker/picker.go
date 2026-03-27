// Package picker provides a lightweight Bubble Tea TUI for selecting an item
// from a list using arrow keys and Enter.
package picker

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Config holds the configuration for the picker.
type Config struct {
	// Title is displayed above the item list (e.g. "Pick a notebook to delete").
	Title string
	// Items is the list of options the user can choose from.
	Items []string
}

// Model is the Bubble Tea model for the picker.
type Model struct {
	config   Config
	cursor   int
	selected string
	quitting bool
	width    int
	height   int
}

// New creates a new picker model from the given config.
func New(cfg Config) Model {
	return Model{
		config: cfg,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.config.Items)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.config.Items) > 0 {
				m.selected = m.config.Items[m.cursor]
			}
			m.quitting = true
			return m, tea.Quit
		case "esc", "q":
			m.selected = ""
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title.
	titleStyle := lipgloss.NewStyle().Bold(true).Padding(1, 0, 1, 2)
	b.WriteString(titleStyle.Render(m.config.Title))
	b.WriteString("\n")

	// Items.
	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	for i, item := range m.config.Items {
		if i == m.cursor {
			b.WriteString(fmt.Sprintf("  %s %s\n", focusedStyle.Render("\u25cf"), focusedStyle.Render(item)))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", item))
		}
	}

	// Hint.
	hintStyle := lipgloss.NewStyle().Faint(true).Padding(1, 0, 0, 2)
	b.WriteString(hintStyle.Render("\u2191/\u2193 navigate \u00b7 Enter select \u00b7 Esc cancel"))
	b.WriteString("\n")

	return b.String()
}

// Selected returns the picked item, or "" if the user cancelled.
func (m Model) Selected() string {
	return m.selected
}

// Run is a convenience function that creates and runs the picker, returning
// the selected item. It returns "" if the user cancelled or the item list
// is empty.
func Run(cfg Config) (string, error) {
	if len(cfg.Items) == 0 {
		return "", nil
	}

	m := New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("run picker: %w", err)
	}

	result, ok := final.(Model)
	if !ok {
		return "", fmt.Errorf("unexpected model type from picker")
	}
	return result.Selected(), nil
}
