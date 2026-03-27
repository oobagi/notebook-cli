// Package browser provides a fullscreen TUI for browsing notebooks and notes.
package browser

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/clipboard"
	"github.com/oobagi/notebook/internal/model"
	"github.com/oobagi/notebook/internal/render"
	"github.com/oobagi/notebook/internal/storage"
)

// EditFunc is called when the user selects a note to edit.
type EditFunc func(book, note string) error

// Config holds the dependencies needed by the browser.
type Config struct {
	Store       *storage.Store
	EditNote    EditFunc
	InitialBook string // if set, start at L1 in this notebook
}

// Selection represents a note the user chose to open.
type Selection struct {
	Book string
	Note string
}

// Model is the Bubble Tea model for the notebook/note browser.
type Model struct {
	store       *storage.Store
	editNote    EditFunc
	level       int    // 0=notebooks, 1=notes
	notebooks   []notebookItem
	notes       []model.Note
	currentBook string // selected notebook name
	cursor      int    // current selection index
	filter      string // fuzzy search filter text
	filtering   bool   // whether filter mode is active
	filtered    []int  // indices into notebooks/notes after filtering
	width       int
	height      int
	showHelp    bool   // help overlay visible
	quitting    bool
	selected    *Selection // set when user picks a note to edit
	err         error

	// Input mode fields (used by create, rename, delete type-to-confirm).
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputCursor int // cursor position within inputValue
	inputAction func(typed string) tea.Cmd

	// After a rename, this holds the new name so the cursor repositions to it.
	selectAfterReload string

	// Temporary status message shown in the status bar.
	statusText string

	// View mode fields (rendered markdown overlay).
	viewMode    bool   // viewing a note's rendered markdown
	viewContent string // rendered markdown content
	viewScroll  int    // scroll offset for the view
	viewTitle   string // breadcrumb title for the view
}

// notebookItem holds pre-fetched metadata for a notebook.
type notebookItem struct {
	name      string
	noteCount int
	modTime   string
}

// New creates a new browser model.
func New(cfg Config) Model {
	m := Model{
		store:    cfg.Store,
		editNote: cfg.EditNote,
	}
	if cfg.InitialBook != "" {
		m.level = 1
		m.currentBook = cfg.InitialBook
	}
	return m
}

// Selected returns the note selection if the user chose one, or nil.
func (m Model) Selected() *Selection {
	return m.selected
}

// notebooksLoadedMsg carries the loaded notebook list.
type notebooksLoadedMsg struct {
	notebooks []notebookItem
}

// notesLoadedMsg carries the loaded note list.
type notesLoadedMsg struct {
	notes []model.Note
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.level == 1 {
		return m.loadNotes(m.currentBook)
	}
	return m.loadNotebooks()
}

func (m Model) loadNotebooks() tea.Cmd {
	return func() tea.Msg {
		names, err := m.store.ListNotebooks()
		if err != nil {
			return errMsg{err}
		}

		items := make([]notebookItem, 0, len(names))
		for _, name := range names {
			count, err := m.store.NoteCount(name)
			if err != nil {
				return errMsg{err}
			}
			modTime, err := m.store.NotebookModTime(name)
			if err != nil {
				return errMsg{err}
			}

			var timeStr string
			if modTime.IsZero() {
				timeStr = "empty"
			} else {
				timeStr = relativeTime(modTime)
			}

			items = append(items, notebookItem{
				name:      name,
				noteCount: count,
				modTime:   timeStr,
			})
		}
		return notebooksLoadedMsg{notebooks: items}
	}
}

func (m Model) loadNotes(book string) tea.Cmd {
	return func() tea.Msg {
		notes, err := m.store.ListNotes(book)
		if err != nil {
			return errMsg{err}
		}
		return notesLoadedMsg{notes: notes}
	}
}

type errMsg struct{ err error }

// reloadMsg triggers a reload of the current list after a mutation.
type reloadMsg struct{}

// reloadAndSelectMsg triggers a reload and repositions the cursor on the named item.
type reloadAndSelectMsg struct{ name string }

// statusMsg carries a temporary status message for the status bar.
type statusMsg struct{ text string }

// viewLoadedMsg carries the note content to display in the view overlay.
type viewLoadedMsg struct {
	title   string
	content string
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case notebooksLoadedMsg:
		m.notebooks = msg.notebooks
		m.resetFilter()
		if m.selectAfterReload != "" {
			for i, fi := range m.filtered {
				if m.notebooks[fi].name == m.selectAfterReload {
					m.cursor = i
					break
				}
			}
			m.selectAfterReload = ""
		}
		return m, nil

	case notesLoadedMsg:
		m.notes = msg.notes
		m.resetFilter()
		if m.selectAfterReload != "" {
			for i, fi := range m.filtered {
				if m.notes[fi].Name == m.selectAfterReload {
					m.cursor = i
					break
				}
			}
			m.selectAfterReload = ""
		}
		return m, nil

	case reloadMsg:
		if m.level == 0 {
			return m, m.loadNotebooks()
		}
		return m, m.loadNotes(m.currentBook)

	case reloadAndSelectMsg:
		m.selectAfterReload = msg.name
		if m.level == 0 {
			return m, m.loadNotebooks()
		}
		return m, m.loadNotes(m.currentBook)

	case statusMsg:
		m.statusText = msg.text
		return m, nil

	case viewLoadedMsg:
		w := m.width
		if w <= 0 {
			w = 80
		}
		m.viewMode = true
		m.viewTitle = msg.title
		m.viewContent = render.RenderMarkdown(msg.content, w-4)
		m.viewScroll = 0
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear any lingering status text on next keypress.
	m.statusText = ""

	// Global quit keys.
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	}

	// When view overlay is showing, handle scroll/dismiss keys.
	if m.viewMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.viewMode = false
			return m, nil
		case tea.KeyRunes:
			if string(msg.Runes) == "q" {
				m.viewMode = false
				return m, nil
			}
		case tea.KeyUp:
			if m.viewScroll > 0 {
				m.viewScroll--
			}
			return m, nil
		case tea.KeyDown:
			m.viewScroll++
			return m, nil
		case tea.KeyPgUp:
			m.viewScroll -= 10
			if m.viewScroll < 0 {
				m.viewScroll = 0
			}
			return m, nil
		case tea.KeyPgDown:
			m.viewScroll += 10
			return m, nil
		}
		return m, nil
	}

	// When help overlay is showing, only ? and Esc dismiss it.
	if m.showHelp {
		switch msg.Type {
		case tea.KeyEsc:
			m.showHelp = false
			return m, nil
		case tea.KeyRunes:
			if string(msg.Runes) == "?" {
				m.showHelp = false
				return m, nil
			}
		}
		return m, nil
	}

	// When in input mode, delegate to input handler.
	if m.inputMode {
		return m.handleInputKey(msg)
	}

	// When filtering, handle text input first.
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case tea.KeyDown:
		max := m.listLen() - 1
		if max < 0 {
			max = 0
		}
		if m.cursor < max {
			m.cursor++
		}
		return m, nil

	case tea.KeyEnter, tea.KeyRight:
		return m.handleEnter()

	case tea.KeyEsc, tea.KeyLeft:
		return m.handleEsc()

	case tea.KeyRunes:
		s := string(msg.Runes)
		if s == "q" {
			m.quitting = true
			return m, tea.Quit
		}
		if s == "?" {
			m.showHelp = true
			return m, nil
		}
		if s == "/" {
			m.filtering = true
			m.filter = ""
			m.applyFilter()
			return m, nil
		}
		if s == "d" {
			return m.startDelete()
		}
		if s == "r" {
			return m.startRename()
		}
		if s == "n" {
			return m.startCreate()
		}
		if s == "v" && m.level == 1 {
			return m.startView()
		}
		if s == "c" && m.level == 1 {
			return m.copyNote()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filter = ""
		m.resetFilter()
		return m, nil

	case tea.KeyEnter:
		m.filtering = false
		return m.handleEnter()

	case tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
		return m, nil

	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case tea.KeyDown:
		max := len(m.filtered) - 1
		if max < 0 {
			max = 0
		}
		if m.cursor < max {
			m.cursor++
		}
		return m, nil

	case tea.KeyRunes:
		m.filter += string(msg.Runes)
		m.applyFilter()
		return m, nil
	}

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.inputMode = false
		m.inputPrompt = ""
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		return m, nil

	case tea.KeyEnter:
		action := m.inputAction
		value := m.inputValue
		m.inputMode = false
		m.inputPrompt = ""
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		if action != nil {
			return m, action(value)
		}
		return m, nil

	case tea.KeyLeft:
		if m.inputCursor > 0 {
			m.inputCursor--
		}
		return m, nil

	case tea.KeyRight:
		if m.inputCursor < len(m.inputValue) {
			m.inputCursor++
		}
		return m, nil

	case tea.KeyBackspace:
		if m.inputCursor > 0 {
			m.inputValue = m.inputValue[:m.inputCursor-1] + m.inputValue[m.inputCursor:]
			m.inputCursor--
		}
		return m, nil

	case tea.KeySpace:
		m.inputValue = m.inputValue[:m.inputCursor] + " " + m.inputValue[m.inputCursor:]
		m.inputCursor++
		return m, nil

	case tea.KeyRunes:
		ch := string(msg.Runes)
		m.inputValue = m.inputValue[:m.inputCursor] + ch + m.inputValue[m.inputCursor:]
		m.inputCursor += len(ch)
		return m, nil
	}

	return m, nil
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		// Delete notebook.
		if len(m.filtered) == 0 {
			return m, nil
		}
		idx := m.filtered[m.cursor]
		name := m.notebooks[idx].name
		m.inputMode = true
		m.inputPrompt = fmt.Sprintf("Delete %q? Type the name to confirm:", name)
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			if typed != name {
				return func() tea.Msg {
					return statusMsg{"Name doesn't match \u2014 cancelled"}
				}
			}
			return func() tea.Msg {
				if err := m.store.DeleteNotebook(name); err != nil {
					return errMsg{err}
				}
				return reloadMsg{}
			}
		}
	} else {
		// Delete note.
		if len(m.filtered) == 0 {
			return m, nil
		}
		idx := m.filtered[m.cursor]
		name := m.notes[idx].Name
		m.inputMode = true
		m.inputPrompt = fmt.Sprintf("Delete %q from %s? Type the name to confirm:", name, m.currentBook)
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			if typed != name {
				return func() tea.Msg {
					return statusMsg{"Name doesn't match \u2014 cancelled"}
				}
			}
			return func() tea.Msg {
				if err := m.store.DeleteNote(m.currentBook, name); err != nil {
					return errMsg{err}
				}
				return reloadMsg{}
			}
		}
	}
	return m, nil
}

func (m Model) startRename() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		if len(m.filtered) == 0 {
			return m, nil
		}
		idx := m.filtered[m.cursor]
		name := m.notebooks[idx].name
		m.inputMode = true
		m.inputPrompt = "Rename notebook:"
		m.inputValue = name
		m.inputCursor = len(name)
		m.inputAction = func(typed string) tea.Cmd {
			slug := storage.Slugify(typed)
			if slug == "" {
				return func() tea.Msg {
					return statusMsg{"Name must not be empty"}
				}
			}
			if slug == name {
				return func() tea.Msg { return statusMsg{"No change"} }
			}
			return func() tea.Msg {
				if err := m.store.RenameNotebook(name, slug); err != nil {
					return statusMsg{err.Error()}
				}
				return reloadAndSelectMsg{slug}
			}
		}
	} else {
		if len(m.filtered) == 0 {
			return m, nil
		}
		idx := m.filtered[m.cursor]
		name := m.notes[idx].Name
		m.inputMode = true
		m.inputPrompt = "Rename note:"
		m.inputValue = name
		m.inputCursor = len(name)
		m.inputAction = func(typed string) tea.Cmd {
			slug := storage.Slugify(typed)
			if slug == "" {
				return func() tea.Msg {
					return statusMsg{"Name must not be empty"}
				}
			}
			if slug == name {
				return func() tea.Msg { return statusMsg{"No change"} }
			}
			return func() tea.Msg {
				if err := m.store.RenameNote(m.currentBook, name, slug); err != nil {
					return statusMsg{err.Error()}
				}
				return reloadAndSelectMsg{slug}
			}
		}
	}
	return m, nil
}

func (m Model) startCreate() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		m.inputMode = true
		m.inputPrompt = "New notebook:"
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			slug := storage.Slugify(typed)
			if slug == "" {
				return func() tea.Msg {
					return statusMsg{"Name must not be empty"}
				}
			}
			return func() tea.Msg {
				if err := m.store.CreateNotebook(slug); err != nil {
					return statusMsg{err.Error()}
				}
				return reloadAndSelectMsg{slug}
			}
		}
	} else {
		m.inputMode = true
		m.inputPrompt = fmt.Sprintf("New note in %s:", m.currentBook)
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			slug := storage.Slugify(typed)
			if slug == "" {
				return func() tea.Msg {
					return statusMsg{"Name must not be empty"}
				}
			}
			return func() tea.Msg {
				if err := m.store.CreateNote(m.currentBook, slug, ""); err != nil {
					return statusMsg{err.Error()}
				}
				return reloadAndSelectMsg{slug}
			}
		}
	}
	return m, nil
}

func (m Model) startView() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	idx := m.filtered[m.cursor]
	note := m.notes[idx]
	return m, func() tea.Msg {
		n, err := m.store.GetNote(m.currentBook, note.Name)
		if err != nil {
			return errMsg{err}
		}
		return viewLoadedMsg{
			title:   fmt.Sprintf("%s \u203A %s", m.currentBook, note.Name),
			content: n.Content,
		}
	}
}

func (m Model) copyNote() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	idx := m.filtered[m.cursor]
	note := m.notes[idx]
	return m, func() tea.Msg {
		n, err := m.store.GetNote(m.currentBook, note.Name)
		if err != nil {
			return errMsg{err}
		}
		if err := clipboard.Copy(n.Content); err != nil {
			return statusMsg{fmt.Sprintf("Could not copy: %s", err)}
		}
		return statusMsg{fmt.Sprintf("Copied %q to clipboard", note.Name)}
	}
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter)
	m.filtered = nil

	if m.level == 0 {
		for i, nb := range m.notebooks {
			if query == "" || strings.Contains(strings.ToLower(nb.name), query) {
				m.filtered = append(m.filtered, i)
			}
		}
	} else {
		for i, n := range m.notes {
			if query == "" || strings.Contains(strings.ToLower(n.Name), query) {
				m.filtered = append(m.filtered, i)
			}
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) resetFilter() {
	m.filter = ""
	m.filtering = false
	m.filtered = nil

	if m.level == 0 {
		m.filtered = make([]int, len(m.notebooks))
		for i := range m.notebooks {
			m.filtered[i] = i
		}
	} else {
		m.filtered = make([]int, len(m.notes))
		for i := range m.notes {
			m.filtered[i] = i
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) listLen() int {
	return len(m.filtered)
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	if m.cursor >= len(m.filtered) {
		return m, nil
	}

	idx := m.filtered[m.cursor]

	if m.level == 0 {
		m.currentBook = m.notebooks[idx].name
		m.level = 1
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		return m, m.loadNotes(m.currentBook)
	}

	// Level 1: quit the browser with the selection so the caller can
	// launch the editor and then re-enter the browser.
	note := m.notes[idx]
	m.selected = &Selection{
		Book: m.currentBook,
		Note: note.Name,
	}
	return m, tea.Quit
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	if m.level == 1 {
		m.level = 0
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		return m, m.loadNotebooks()
	}
	m.quitting = true
	return m, tea.Quit
}

// renderHelpOverlay builds the centered help panel.
func (m Model) renderHelpOverlay() string {
	var help string
	if m.level == 0 {
		help = `  Keybindings
  ───────────────────────────

  ↑/↓       Navigate
  Enter      Open notebook
  n          New notebook
  d          Delete notebook
  r          Rename notebook
  /          Search
  q          Quit
  ?          Toggle help

  Press ? or Esc to close`
	} else {
		help = `  Keybindings
  ───────────────────────────

  ↑/↓       Navigate
  Enter      Edit note
  v          View note
  n          New note
  d          Delete note
  r          Rename note
  c          Copy to clipboard
  /          Search
  Esc        Back to notebooks
  q          Quit
  ?          Toggle help

  Press ? or Esc to close`
	}

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

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, rendered)
}

// renderViewOverlay builds the scrollable rendered markdown view.
func (m Model) renderViewOverlay() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	// Split content into lines and apply scroll.
	lines := strings.Split(m.viewContent, "\n")
	viewHeight := h - 4 // space for title + status

	// Clamp scroll.
	maxScroll := len(lines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.viewScroll > maxScroll {
		m.viewScroll = maxScroll
	}

	start := m.viewScroll
	end := start + viewHeight
	if end > len(lines) {
		end = len(lines)
	}

	visible := strings.Join(lines[start:end], "\n")

	var b strings.Builder
	// Title bar.
	bold := lipgloss.NewStyle().Bold(true)
	b.WriteString("\n  ")
	b.WriteString(bold.Render(m.viewTitle))
	b.WriteString("\n\n")
	b.WriteString(visible)
	b.WriteString("\n\n")

	// Status bar.
	dim := lipgloss.NewStyle().Faint(true)
	scrollInfo := ""
	if len(lines) > viewHeight {
		scrollInfo = fmt.Sprintf(" (%d/%d)", m.viewScroll+1, len(lines))
	}
	b.WriteString(dim.Render(fmt.Sprintf("  \u2191/\u2193 scroll \u00B7 Esc close%s", scrollInfo)))

	return b.String()
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.renderHelpOverlay()
	}

	if m.viewMode {
		return m.renderViewOverlay()
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}

	var b strings.Builder

	// Breadcrumb / path.
	breadcrumb := m.renderBreadcrumb()
	b.WriteString(breadcrumb)
	b.WriteString("\n\n")

	// Content area.
	contentHeight := m.height - 4 // breadcrumb + blank + status bar + blank
	if contentHeight < 1 {
		contentHeight = 1
	}

	content := m.renderContent(contentHeight)
	b.WriteString(content)

	// Pad to push status bar to bottom.
	contentLines := strings.Count(content, "\n")
	if contentLines < contentHeight {
		for i := 0; i < contentHeight-contentLines; i++ {
			b.WriteString("\n")
		}
	}

	// Status bar.
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m Model) renderBreadcrumb() string {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if m.level == 0 {
		return style.Render("notebook")
	}
	return style.Render(fmt.Sprintf("notebook \u203A %s", m.currentBook))
}

func (m Model) renderContent(maxLines int) string {
	if m.level == 0 {
		return m.renderNotebookList(maxLines)
	}
	return m.renderNoteList(maxLines)
}

func (m Model) renderNotebookList(maxLines int) string {
	if len(m.notebooks) == 0 {
		return m.renderEmptyNotebooks()
	}

	if len(m.filtered) == 0 && m.filtering {
		return "  No matches\n"
	}

	var b strings.Builder
	visible := m.visibleRange(len(m.filtered), maxLines)

	for vi, fi := range visible {
		idx := m.filtered[fi]
		nb := m.notebooks[idx]
		selected := fi == m.cursor
		line := m.formatNotebookLine(nb, selected)
		b.WriteString(line)
		if vi < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderNoteList(maxLines int) string {
	if len(m.notes) == 0 {
		return m.renderEmptyNotes()
	}

	if len(m.filtered) == 0 && m.filtering {
		return "  No matches\n"
	}

	var b strings.Builder
	visible := m.visibleRange(len(m.filtered), maxLines)

	for vi, fi := range visible {
		idx := m.filtered[fi]
		n := m.notes[idx]
		selected := fi == m.cursor
		line := m.formatNoteLine(n, selected)
		b.WriteString(line)
		if vi < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// visibleRange returns the slice of filtered indices to display,
// implementing scrolling so the cursor stays visible.
func (m Model) visibleRange(total, maxLines int) []int {
	if total <= maxLines {
		indices := make([]int, total)
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	// Scroll so cursor is visible.
	start := 0
	if m.cursor >= maxLines {
		start = m.cursor - maxLines + 1
	}
	end := start + maxLines
	if end > total {
		end = total
		start = end - maxLines
	}

	indices := make([]int, end-start)
	for i := range indices {
		indices[i] = start + i
	}
	return indices
}

func (m Model) formatNotebookLine(nb notebookItem, selected bool) string {
	bullet := "  "
	name := nb.name
	countStr := pluralize(nb.noteCount, "note", "notes")

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		name = nameStyle.Render(nb.name)
	}

	return fmt.Sprintf("%s%-*s    %-*s    %s",
		bullet,
		m.nameColWidth(0), name,
		10, countStr,
		nb.modTime,
	)
}

func (m Model) formatNoteLine(n model.Note, selected bool) string {
	bullet := "  "
	name := n.Name

	// Get file size.
	p := m.store.NotebookDir(m.currentBook) + "/" + n.Name + ".md"
	info, err := os.Stat(p)
	var sizeStr, timeStr string
	if err == nil {
		sizeStr = humanSize(info.Size())
		timeStr = relativeTime(info.ModTime())
	}

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		name = nameStyle.Render(n.Name)
	}

	return fmt.Sprintf("%s%-*s    %-*s    %s",
		bullet,
		m.nameColWidth(1), name,
		8, sizeStr,
		timeStr,
	)
}

func (m Model) nameColWidth(level int) int {
	maxLen := 0
	if level == 0 {
		for _, nb := range m.notebooks {
			if len(nb.name) > maxLen {
				maxLen = len(nb.name)
			}
		}
	} else {
		for _, n := range m.notes {
			if len(n.Name) > maxLen {
				maxLen = len(n.Name)
			}
		}
	}
	if maxLen < 10 {
		maxLen = 10
	}
	return maxLen
}

func (m Model) renderEmptyNotebooks() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	dim := lipgloss.NewStyle().Faint(true)
	msg := "No notebooks yet.\n\nPress n to create one."
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, dim.Render(msg))
}

func (m Model) renderEmptyNotes() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	dim := lipgloss.NewStyle().Faint(true)
	msg := fmt.Sprintf("No notes in %s.\n\nPress n to create one.", m.currentBook)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, dim.Render(msg))
}

func (m Model) renderStatusBar() string {
	dim := lipgloss.NewStyle().Faint(true)

	if m.inputMode {
		before := m.inputValue[:m.inputCursor]
		after := m.inputValue[m.inputCursor:]
		return dim.Render(fmt.Sprintf("  %s %s", m.inputPrompt, before)) + "█" + dim.Render(after)
	}

	if m.statusText != "" {
		return dim.Render(fmt.Sprintf("  %s", m.statusText))
	}

	if m.filtering {
		return dim.Render(fmt.Sprintf("  Filter: %s_ \u00B7 Esc clear \u00B7 Enter select", m.filter))
	}

	return dim.Render("  \u2191/\u2193 navigate \u00B7 Enter open \u00B7 / search \u00B7 Esc back \u00B7 q quit \u00B7 ? help")
}

// pluralize returns "1 note" or "3 notes" style strings.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
