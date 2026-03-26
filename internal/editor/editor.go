// Package editor provides a Bubble Tea TUI for editing markdown notes.
package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Config holds the configuration for the editor.
type Config struct {
	// Title is displayed in the status bar (e.g. "book > note").
	Title string
	// Content is the initial text content to edit.
	Content string
	// Save is called with the current content when the user presses Ctrl+S.
	Save func(string) error
}

// savedMsg is sent after a successful save.
type savedMsg struct{ content string }

// saveErrMsg is sent when a save fails.
type saveErrMsg struct{ err error }

// Model is the Bubble Tea model for the editor.
type Model struct {
	textarea    textarea.Model
	config      Config
	initial     string
	width       int
	height      int
	status      string
	statusStyle statusKind
	quitWarning bool
	quitting    bool
}

type statusKind int

const (
	statusNone statusKind = iota
	statusSuccess
	statusError
	statusWarning
)

// New creates a new editor Model from the given config.
func New(cfg Config) Model {
	ta := textarea.New()
	ta.SetValue(cfg.Content)
	ta.ShowLineNumbers = false
	ta.Focus()

	return Model{
		textarea: ta,
		config:   cfg,
		initial:  cfg.Content,
	}
}

// Init returns the initial command for the editor (start cursor blinking).
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// modified returns true if the content has been changed from the initial value.
func (m Model) modified() bool {
	return m.textarea.Value() != m.initial
}

// Content returns the current text content.
func (m Model) Content() string {
	return m.textarea.Value()
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)
		// Reserve 1 line for the status bar.
		m.textarea.SetHeight(msg.Height - 1)
		return m, nil

	case tea.KeyMsg:
		// Clear transient status on any keypress, unless quitting.
		if m.statusStyle == statusSuccess {
			m.status = ""
			m.statusStyle = statusNone
		}

		switch msg.String() {
		case "ctrl+s":
			m.quitWarning = false
			if m.config.Save != nil {
				content := m.textarea.Value()
				return m, func() tea.Msg {
					if err := m.config.Save(content); err != nil {
						return saveErrMsg{err: err}
					}
					return savedMsg{content: content}
				}
			}
			return m, nil

		case "ctrl+q":
			if m.modified() && !m.quitWarning {
				m.quitWarning = true
				m.status = "Unsaved changes! Press Ctrl+Q again to quit without saving"
				m.statusStyle = statusWarning
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

		// Any key other than ctrl+q resets the quit warning.
		m.quitWarning = false

	case savedMsg:
		m.initial = msg.content
		m.status = "Saved"
		m.statusStyle = statusSuccess
		return m, nil

	case saveErrMsg:
		m.status = fmt.Sprintf("Save failed: %s", msg.err)
		m.statusStyle = statusError
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the editor UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	statusBar := m.renderStatusBar()
	return m.textarea.View() + "\n" + statusBar
}

// renderStatusBar builds the bottom status bar.
func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	// Left side: title + modified indicator.
	left := m.config.Title
	if m.modified() {
		left += " [modified]"
	}

	// Right side: status message or keybinding hints.
	var right string
	if m.status != "" {
		right = m.status
	} else {
		right = "Ctrl+S save \u00B7 Ctrl+Q quit"
	}

	// Calculate gap between left and right.
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

	// Style based on current status.
	style := lipgloss.NewStyle().Width(width)
	switch m.statusStyle {
	case statusSuccess:
		style = style.Foreground(lipgloss.Color("2")) // green
	case statusError:
		style = style.Foreground(lipgloss.Color("1")) // red
	case statusWarning:
		style = style.Foreground(lipgloss.Color("3")) // yellow
	default:
		style = style.Faint(true)
	}

	return style.Render(bar)
}
