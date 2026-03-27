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

// linkOpenMsg is sent after attempting to open a link in the browser.
type linkOpenMsg struct {
	url string
	err error
}

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

// savedQuitMsg is sent after a successful save-then-quit.
type savedQuitMsg struct{ content string }

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
	textarea       textarea.Model
	config         Config
	initial        string
	width          int
	height         int
	status         string
	statusStyle    statusKind
	quitPrompt     bool
	quitting       bool
	showPreview    bool
	showHelp       bool
	clipboard      string
	preview        string
	previewDirty   bool
	previewSeq     int
	previewMode    bool
	elements       []Element
	focusedElement int
	previewScroll  int
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

// renderPreviewWithFocus builds a preview pane that shows an element list
// with the currently focused element highlighted. This is displayed when
// the editor is in preview mode.
func (m Model) renderPreviewWithFocus(width, height int) string {
	var lines []string
	for i, el := range m.elements {
		prefix := "  "
		if i == m.focusedElement {
			prefix = "> "
		}
		line := prefix + elementDisplayText(el)
		// Truncate to fit width.
		if len(line) > width {
			line = line[:width-1] + "\u2026"
		}
		lines = append(lines, line)
	}

	// If there are more lines than height, scroll to keep focused element visible.
	if len(lines) > height {
		start := m.focusedElement - height/2
		if start < 0 {
			start = 0
		}
		end := start + height
		if end > len(lines) {
			end = len(lines)
			start = end - height
			if start < 0 {
				start = 0
			}
		}
		lines = lines[start:end]
	}

	return strings.Join(lines, "\n")
}

// refreshElements re-parses interactive elements from the current content.
// It clamps the focused element index to remain valid.
func (m *Model) refreshElements() {
	m.elements = parseElements(m.textarea.Value())
	if m.focusedElement >= len(m.elements) {
		m.focusedElement = len(m.elements) - 1
	}
	if m.focusedElement < 0 {
		m.focusedElement = 0
	}
}

// enterPreviewMode switches to preview mode with interactive element navigation.
func (m *Model) enterPreviewMode() {
	m.refreshElements()
	if len(m.elements) == 0 {
		m.status = "No interactive elements"
		m.statusStyle = statusWarning
		return
	}
	m.previewMode = true
	m.focusedElement = 0
	m.textarea.Blur()
}

// exitPreviewMode returns to edit mode.
func (m *Model) exitPreviewMode() {
	m.previewMode = false
	m.textarea.Focus()
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

// cursorLine returns the zero-based line index where the textarea cursor is.
func (m Model) cursorLine() int {
	return m.textarea.Line()
}

// cutLine removes the current line from the textarea and stores it in the
// clipboard. It returns the cut text.
func (m *Model) cutLine() string {
	value := m.textarea.Value()
	lines := strings.Split(value, "\n")

	line := m.cursorLine()
	if line < 0 || line >= len(lines) {
		return ""
	}

	cut := lines[line]

	// Remove the line.
	newLines := make([]string, 0, len(lines)-1)
	newLines = append(newLines, lines[:line]...)
	newLines = append(newLines, lines[line+1:]...)

	m.textarea.SetValue(strings.Join(newLines, "\n"))

	// Reposition cursor: stay on the same line index, clamped to the new
	// line count. Move to the start of the line for simplicity.
	newLineCount := len(newLines)
	targetLine := line
	if targetLine >= newLineCount {
		targetLine = newLineCount - 1
	}
	if targetLine < 0 {
		targetLine = 0
	}
	m.textarea.SetCursor(0)
	for m.textarea.Line() > targetLine {
		m.textarea.CursorUp()
	}
	for m.textarea.Line() < targetLine {
		m.textarea.CursorDown()
	}

	return cut
}

// pasteLine inserts the clipboard content at the current cursor line.
func (m *Model) pasteLine() {
	if m.clipboard == "" {
		return
	}
	value := m.textarea.Value()
	lines := strings.Split(value, "\n")

	line := m.cursorLine()
	if line < 0 {
		line = 0
	}
	if line > len(lines) {
		line = len(lines)
	}

	// Insert the clipboard as a new line before the current line.
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:line]...)
	newLines = append(newLines, m.clipboard)
	newLines = append(newLines, lines[line:]...)

	m.textarea.SetValue(strings.Join(newLines, "\n"))

	// Position cursor on the pasted line.
	m.textarea.SetCursor(0)
	for m.textarea.Line() < line {
		m.textarea.CursorDown()
	}
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
		// When help overlay is showing, only Ctrl+G and Esc dismiss it.
		if m.showHelp {
			switch msg.String() {
			case "ctrl+g", "esc":
				m.showHelp = false
			}
			return m, nil
		}

		// When quit prompt is showing, handle the 3-option response.
		if m.quitPrompt {
			switch msg.String() {
			case "y", "Y", "enter":
				m.quitPrompt = false
				// Save then quit.
				if m.config.Save != nil {
					content := m.textarea.Value()
					return m, func() tea.Msg {
						if err := m.config.Save(content); err != nil {
							return saveErrMsg{err: err}
						}
						return savedQuitMsg{content: content}
					}
				}
				// No save function configured — just quit.
				m.quitting = true
				return m, tea.Quit
			case "n", "N":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.quitPrompt = false
				m.status = ""
				m.statusStyle = statusNone
				return m, nil
			}
			// Ignore all other keys while in the prompt.
			return m, nil
		}

		// When in preview mode, handle interactive navigation.
		if m.previewMode {
			switch msg.String() {
			case "ctrl+e", "esc":
				m.exitPreviewMode()
				return m, nil

			case "up", "shift+tab":
				if m.focusedElement > 0 {
					m.focusedElement--
				}
				return m, nil

			case "down", "tab":
				if m.focusedElement < len(m.elements)-1 {
					m.focusedElement++
				}
				return m, nil

			case "enter", " ":
				if m.focusedElement >= 0 && m.focusedElement < len(m.elements) {
					el := m.elements[m.focusedElement]
					switch el.Kind {
					case ElementCheckbox:
						newContent, checked := toggleCheckbox(m.textarea.Value(), el.Line)
						m.textarea.SetValue(newContent)
						m.elements[m.focusedElement].Checked = checked
						m.refreshElements()
						m.previewDirty = true
						return m, m.schedulePreviewTick()
					case ElementLink:
						url := el.URL
						m.status = "Opening link..."
						m.statusStyle = statusSuccess
						return m, func() tea.Msg {
							err := openLink(url)
							return linkOpenMsg{url: url, err: err}
						}
					}
				}
				return m, nil

			case "ctrl+q":
				if m.modified() {
					m.exitPreviewMode()
					m.quitPrompt = true
					m.status = "Save before quitting? [Y/n/Esc]"
					m.statusStyle = statusWarning
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit

			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
			// Ignore all other keys in preview mode.
			return m, nil
		}

		// Clear transient status on any keypress.
		if m.statusStyle == statusSuccess {
			m.status = ""
			m.statusStyle = statusNone
		}

		switch msg.String() {
		case "ctrl+e":
			if m.showPreview && m.width >= minSplitWidth {
				m.enterPreviewMode()
			}
			return m, nil

		case "ctrl+g":
			m.showHelp = true
			return m, nil

		case "ctrl+k":
			m.clipboard = m.cutLine()
			m.previewDirty = true
			return m, m.schedulePreviewTick()

		case "ctrl+u":
			m.pasteLine()
			m.previewDirty = true
			return m, m.schedulePreviewTick()

		case "ctrl+p":
			m.showPreview = !m.showPreview
			m.resizeTextarea()
			if m.showPreview {
				m.previewDirty = true
				return m, m.schedulePreviewTick()
			}
			return m, nil

		case "ctrl+s":
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
			if m.modified() {
				m.quitPrompt = true
				m.status = "Save before quitting? [Y/n/Esc]"
				m.statusStyle = statusWarning
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "ctrl+c":
			if m.modified() {
				m.quitPrompt = true
				m.status = "Save before quitting? [Y/n/Esc]"
				m.statusStyle = statusWarning
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		}

	case linkOpenMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to open link: %s", msg.err)
			m.statusStyle = statusError
		}
		return m, nil

	case savedMsg:
		m.initial = msg.content
		m.status = "Saved"
		m.statusStyle = statusSuccess
		return m, nil

	case savedQuitMsg:
		m.initial = msg.content
		m.quitting = true
		return m, tea.Quit

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

// renderHelpOverlay builds the full-screen help panel.
func (m Model) renderHelpOverlay() string {
	help := `  Keybindings
  ───────────────────────────

  Ctrl+S    Save
  Ctrl+Q    Quit
  Ctrl+C    Force quit (no save)
  Ctrl+P    Toggle preview
  Ctrl+E    Preview mode
  Ctrl+G    Toggle this help
  Ctrl+K    Cut line
  Ctrl+U    Paste line

  Press Ctrl+G or Esc to close`

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2).
		Width(36).
		Align(lipgloss.Left)

	rendered := box.Render(help)

	// Center the box in the terminal.
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, rendered)
}

// View renders the editor UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.renderHelpOverlay()
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

	// When in preview mode, overlay a focused-element indicator on the preview.
	previewContent := m.preview
	if m.previewMode && len(m.elements) > 0 {
		previewContent = m.renderPreviewWithFocus(rightWidth, editorHeight)
	}

	rightPane := previewStyle.Render(previewContent)

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, border, rightPane)

	return split + "\n" + statusBar
}

// renderStatusBar builds the bottom status bar.
func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	// Left side: title + mode indicator + modified indicator.
	left := m.config.Title
	if m.previewMode {
		left += " PREVIEW"
	}
	if m.modified() {
		left += " [modified]"
	}

	// Right side: status message or keybinding hints.
	var right string
	if m.status != "" {
		right = m.status
	} else if m.previewMode {
		right = "\u2191/\u2193 navigate \u00B7 Enter toggle/open \u00B7 Ctrl+E edit \u00B7 Ctrl+Q quit"
	} else {
		right = "Ctrl+S save \u00B7 Ctrl+E preview \u00B7 Ctrl+G help \u00B7 Ctrl+Q quit"
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
