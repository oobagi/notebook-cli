// Package editor provides a Bubble Tea TUI for editing markdown notes.
package editor

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/render"
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

// previewTickMsg is sent after the debounce interval to trigger preview rendering.
// The seq field is compared against Model.previewSeq to implement true debounce:
// only the most recent tick (matching the current seq) triggers a render.
type previewTickMsg struct{ seq int }

// previewDebounce is the delay before re-rendering the preview after a text change.
const previewDebounce = 150 * time.Millisecond

// minSplitWidth is the minimum terminal width required to show the split pane.
// Below this, the preview is auto-hidden.
const minSplitWidth = 40

// Model is the Bubble Tea model for the editor.
type Model struct {
	textarea     textarea.Model
	config       Config
	initial      string
	width        int
	height       int
	status       string
	statusStyle  statusKind
	quitWarning  bool
	quitting     bool
	showPreview  bool
	preview      string
	previewDirty bool
	previewSeq   int
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
		textarea:    ta,
		config:      cfg,
		initial:     cfg.Content,
		showPreview: true,
	}
}

// Init returns the initial command for the editor (start cursor blinking).
// The first preview render is triggered by the initial WindowSizeMsg.
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

// textareaWidth returns the width the textarea should use given the current
// preview visibility and total terminal width.
func (m Model) textareaWidth() int {
	if m.showPreview && m.width >= minSplitWidth {
		return m.width / 2
	}
	return m.width
}

// previewWidth returns the width available for the preview pane.
func (m Model) previewWidth() int {
	if !m.showPreview || m.width < minSplitWidth {
		return 0
	}
	// Total width minus left pane minus 1-column border.
	return m.width - m.width/2 - 1
}

// resizeTextarea updates the textarea dimensions for the current layout.
func (m *Model) resizeTextarea() {
	m.textarea.SetWidth(m.textareaWidth())
	// Reserve 1 line for the status bar.
	if m.height > 1 {
		m.textarea.SetHeight(m.height - 1)
	}
}

// renderPreview renders the current content as markdown for the preview pane.
func (m *Model) renderPreview() {
	pw := m.previewWidth()
	if pw <= 0 {
		m.preview = ""
		return
	}
	content := m.textarea.Value()
	if content == "" {
		m.preview = ""
		return
	}
	m.preview = render.RenderMarkdown(content, pw)
	m.previewDirty = false
}

// schedulePreviewTick increments the sequence counter and returns a tick command.
// Only the tick matching the current sequence will trigger a render, implementing
// a true debounce (renders after the last change, not the first).
func (m *Model) schedulePreviewTick() tea.Cmd {
	m.previewSeq++
	seq := m.previewSeq
	return tea.Tick(previewDebounce, func(t time.Time) tea.Msg {
		return previewTickMsg{seq: seq}
	})
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTextarea()
		m.previewDirty = true
		return m, m.schedulePreviewTick()

	case previewTickMsg:
		if msg.seq == m.previewSeq && m.previewDirty {
			m.renderPreview()
		}
		return m, nil

	case tea.KeyMsg:
		// Clear transient status on any keypress, unless quitting.
		if m.statusStyle == statusSuccess {
			m.status = ""
			m.statusStyle = statusNone
		}

		switch msg.String() {
		case "ctrl+p":
			m.showPreview = !m.showPreview
			m.resizeTextarea()
			if m.showPreview {
				m.previewDirty = true
				return m, m.schedulePreviewTick()
			}
			return m, nil

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

	// Track content before the textarea processes the message so we can
	// detect changes and schedule a preview re-render.
	contentBefore := m.textarea.Value()

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	if m.textarea.Value() != contentBefore {
		m.previewDirty = true
		cmd = tea.Batch(cmd, m.schedulePreviewTick())
	}

	return m, cmd
}

// View renders the editor UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	statusBar := m.renderStatusBar()

	if !m.showPreview || m.width < minSplitWidth {
		return m.textarea.View() + "\n" + statusBar
	}

	// Split-pane layout: editor left, border, preview right.
	editorHeight := m.height - 1
	if editorHeight < 1 {
		editorHeight = 1
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 1 // 1 col for border

	// Build the vertical border (thin dim line of │ characters).
	borderStyle := lipgloss.NewStyle().Faint(true)
	var borderLines []string
	for i := 0; i < editorHeight; i++ {
		borderLines = append(borderLines, borderStyle.Render("│"))
	}
	border := strings.Join(borderLines, "\n")

	// Constrain the preview pane.
	previewStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(editorHeight)

	leftPane := m.textarea.View()
	rightPane := previewStyle.Render(m.preview)

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, border, rightPane)

	return split + "\n" + statusBar
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
		right = "Ctrl+S save \u00B7 Ctrl+P preview \u00B7 Ctrl+Q quit"
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
