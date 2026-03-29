// Package browser provides a fullscreen TUI for browsing notebooks and notes.
package browser

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oobagi/notebook/internal/clipboard"
	"github.com/oobagi/notebook/internal/config"
	"github.com/oobagi/notebook/internal/model"
	"github.com/oobagi/notebook/internal/render"
	"github.com/oobagi/notebook/internal/storage"
	"github.com/oobagi/notebook/internal/theme"
	"github.com/oobagi/notebook/styles"
)

// EditFunc is called when the user selects a note to edit.
type EditFunc func(book, note string) error

// Config holds the dependencies needed by the browser.
type Config struct {
	Store         *storage.Store
	EditNote      EditFunc
	InitialBook        string // if set, start at L1 in this notebook
	InitialCursor      int    // cursor position to restore within the initial view
	InitialSavedCursor int    // L0 cursor to restore when returning from L1
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
	savedCursor int    // notebook-level cursor restored on Esc
	filter       string // fuzzy search filter text
	filtering    bool   // whether filter mode is active
	filterCursor int    // cursor position within filter
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
	statusGen  int // generation counter for auto-dismiss

	// View mode fields (rendered markdown overlay).
	viewMode    bool   // viewing a note's rendered markdown
	viewContent string // rendered markdown content
	viewScroll  int    // scroll offset for the view
	viewTitle   string // breadcrumb title for the view

	// Theme picker fields.
	themeMode    bool     // theme picker overlay visible
	themeStyles  []string // available glamour style names
	themeCursor  int      // current selection in theme list
	themePreview string   // rendered preview for currently highlighted style
	darkTerminal bool     // true when terminal has dark background

	// Theme picker tab fields (0=glamour style, 1=UI theme).
	themeTab       int    // active tab in theme picker
	uiThemeCursor  int    // cursor in UI theme preset list
	uiThemePreview string // preview for highlighted UI preset
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
	m.cursor = cfg.InitialCursor
	m.savedCursor = cfg.InitialSavedCursor
	return m
}

// Selected returns the note selection if the user chose one, or nil.
func (m Model) Selected() *Selection {
	return m.selected
}

// Cursor returns the current cursor position.
func (m Model) Cursor() int {
	return m.cursor
}

// SavedCursor returns the saved L0 cursor position.
func (m Model) SavedCursor() int {
	return m.savedCursor
}

// scheduleStatusDismiss increments the generation counter and returns a tick
// command that will clear the status text after statusTimeout.
func (m *Model) scheduleStatusDismiss() tea.Cmd {
	m.statusGen++
	gen := m.statusGen
	return tea.Tick(statusTimeout, func(t time.Time) tea.Msg {
		return statusTimeoutMsg{generation: gen}
	})
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

// statusTimeoutMsg is sent after the status auto-dismiss delay.
// The generation field is compared against Model.statusGen to ensure
// only the most recent status message is cleared.
type statusTimeoutMsg struct{ generation int }

// statusTimeout is the delay before auto-dismissing a transient status message.
const statusTimeout = 4 * time.Second

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
		return m, m.scheduleStatusDismiss()

	case statusTimeoutMsg:
		if msg.generation == m.statusGen {
			m.statusText = ""
		}
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

	// When theme picker is showing, handle navigation/selection.
	if m.themeMode {
		return m.handleThemeKey(msg)
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
			m.filterCursor = 0
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
		if s == "t" {
			return m.startThemePicker()
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
		m.filterCursor = 0
		m.resetFilter()
		return m, nil

	case tea.KeyEnter:
		m.filtering = false
		return m.handleEnter()

	case tea.KeyLeft:
		if m.filterCursor > 0 {
			m.filterCursor--
		}
		return m, nil

	case tea.KeyRight:
		if m.filterCursor < len(m.filter) {
			m.filterCursor++
		}
		return m, nil

	case tea.KeyBackspace:
		if m.filterCursor > 0 {
			m.filter = m.filter[:m.filterCursor-1] + m.filter[m.filterCursor:]
			m.filterCursor--
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

	case tea.KeySpace:
		m.filter = m.filter[:m.filterCursor] + " " + m.filter[m.filterCursor:]
		m.filterCursor++
		m.applyFilter()
		return m, nil

	case tea.KeyRunes:
		ch := string(msg.Runes)
		m.filter = m.filter[:m.filterCursor] + ch + m.filter[m.filterCursor:]
		m.filterCursor += len(ch)
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
		m.inputPrompt = fmt.Sprintf("Delete %q? Type the name to confirm:", storage.DisplayName(name))
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			if typed != storage.DisplayName(name) {
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
		m.inputPrompt = fmt.Sprintf("Delete %q from %s? Type the name to confirm:", storage.DisplayName(name), storage.DisplayName(m.currentBook))
		m.inputValue = ""
		m.inputAction = func(typed string) tea.Cmd {
			if typed != storage.DisplayName(name) {
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
		m.inputValue = storage.DisplayName(name)
		m.inputCursor = len(m.inputValue)
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
		m.inputValue = storage.DisplayName(name)
		m.inputCursor = len(m.inputValue)
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
		m.inputPrompt = fmt.Sprintf("New note in %s:", storage.DisplayName(m.currentBook))
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
			title:   fmt.Sprintf("%s \u203A %s", storage.DisplayName(m.currentBook), storage.DisplayName(note.Name)),
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
		return statusMsg{fmt.Sprintf("Copied %q to clipboard", storage.DisplayName(note.Name))}
	}
}

// themePreviewSample is the hardcoded markdown snippet used to preview glamour styles.
const themePreviewSample = `# Heading

**Bold text** and regular text.

` + "```go" + `
fmt.Println("hello world")
` + "```" + `

- Item one
- Item two

- [x] Completed task
- [ ] Pending task

> A blockquote for emphasis.

> [!NOTE]
> This is an admonition note.
`

// buildThemeStyles returns the glamour style names shown in the picker,
// combining built-in styles with community styles separated by a divider.
func buildThemeStyles() []string {
	builtin := []string{"auto", "dark", "light", "dracula", "tokyo-night", "pink"}
	community := styles.List() // returns sorted community names

	result := make([]string, 0, len(builtin)+1+len(community))
	result = append(result, builtin...)
	if len(community) > 0 {
		result = append(result, "---") // divider marker
		result = append(result, community...)
	}
	return result
}

// buildStyleHints returns background compatibility hints for all styles.
func buildStyleHints() map[string]string {
	hints := map[string]string{
		"auto":        "",
		"dark":        "(dark bg)",
		"light":       "(light bg)",
		"dracula":     "(dark bg)",
		"tokyo-night": "(dark bg)",
		"pink":        "(dark bg)",
	}
	// All community styles are designed for dark backgrounds.
	for _, name := range styles.List() {
		hints[name] = "(dark bg)"
	}
	return hints
}

// styleHints maps each style name to its intended background compatibility.
var styleHints = buildStyleHints()

func (m Model) startThemePicker() (tea.Model, tea.Cmd) {
	m.themeMode = true
	m.themeTab = 0
	m.themeStyles = buildThemeStyles()
	m.themeCursor = 0
	m.darkTerminal = lipgloss.HasDarkBackground()
	// Pre-select the currently active glamour style if set.
	if cfg, err := config.Load(); err == nil && cfg.GlamourStyle != "" {
		for i, s := range m.themeStyles {
			if s == "---" {
				continue
			}
			if s == cfg.GlamourStyle {
				m.themeCursor = i
				break
			}
		}
	}
	m.themePreview = m.renderThemePreview(m.themeStyles[m.themeCursor])

	// Initialize UI theme cursor to match current config value.
	m.uiThemeCursor = 0
	presets := theme.Presets()
	if cfg, err := config.Load(); err == nil && cfg.UITheme != "" {
		for i, p := range presets {
			if p.Name == cfg.UITheme {
				m.uiThemeCursor = i
				break
			}
		}
	}
	if len(presets) > 0 {
		m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
	}
	return m, nil
}

func (m Model) handleThemeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.themeMode = false
		return m, nil

	case tea.KeyTab:
		// Toggle between glamour (0) and UI theme (1) tabs.
		if m.themeTab == 0 {
			m.themeTab = 1
		} else {
			m.themeTab = 0
		}
		return m, nil

	case tea.KeyUp:
		if m.themeTab == 0 {
			if m.themeCursor > 0 {
				m.themeCursor--
				// Skip the divider.
				if m.themeStyles[m.themeCursor] == "---" && m.themeCursor > 0 {
					m.themeCursor--
				}
				m.themePreview = m.renderThemePreview(m.themeStyles[m.themeCursor])
			}
		} else {
			if m.uiThemeCursor > 0 {
				m.uiThemeCursor--
				presets := theme.Presets()
				if len(presets) > 0 {
					m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
				}
			}
		}
		return m, nil

	case tea.KeyDown:
		if m.themeTab == 0 {
			if m.themeCursor < len(m.themeStyles)-1 {
				m.themeCursor++
				// Skip the divider.
				if m.themeStyles[m.themeCursor] == "---" && m.themeCursor < len(m.themeStyles)-1 {
					m.themeCursor++
				}
				m.themePreview = m.renderThemePreview(m.themeStyles[m.themeCursor])
			}
		} else {
			presets := theme.Presets()
			if m.uiThemeCursor < len(presets)-1 {
				m.uiThemeCursor++
				m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
			}
		}
		return m, nil

	case tea.KeyEnter:
		if m.themeTab == 0 {
			selected := m.themeStyles[m.themeCursor]
			m.themeMode = false

			// Persist to config and apply.
			cfg, err := config.Load()
			if err != nil {
				m.statusText = fmt.Sprintf("Config load error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			if err := config.Set(&cfg, "glamour_style", selected); err != nil {
				m.statusText = fmt.Sprintf("Config error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			if err := config.Save(cfg); err != nil {
				m.statusText = fmt.Sprintf("Config save error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			render.SetGlamourStyle(selected)
			m.statusText = fmt.Sprintf("Theme set to %q", selected)
		} else {
			presets := theme.Presets()
			if len(presets) == 0 {
				return m, nil
			}
			selected := presets[m.uiThemeCursor]
			m.themeMode = false

			// Persist to config and apply.
			cfg, err := config.Load()
			if err != nil {
				m.statusText = fmt.Sprintf("Config load error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			if err := config.Set(&cfg, "ui_theme", selected.Name); err != nil {
				m.statusText = fmt.Sprintf("Config error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			if err := config.Save(cfg); err != nil {
				m.statusText = fmt.Sprintf("Config save error: %s", err)
				return m, m.scheduleStatusDismiss()
			}
			theme.SetTheme(selected)
			m.statusText = fmt.Sprintf("UI theme set to %s", selected.Name)
		}
		return m, m.scheduleStatusDismiss()

	case tea.KeyRunes:
		if string(msg.Runes) == "q" || string(msg.Runes) == "t" {
			m.themeMode = false
			return m, nil
		}
	}

	return m, nil
}

func (m Model) renderThemePreview(styleName string) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	// Use roughly half the width for the preview pane.
	previewWidth := w/2 - 6
	if previewWidth < 20 {
		previewWidth = 20
	}
	return render.RenderMarkdownWithStyle(themePreviewSample, previewWidth, styleName)
}

// renderUIThemePreview generates a mock TUI chrome preview for a UI theme preset.
func (m Model) renderUIThemePreview(presetName string) string {
	preset, ok := theme.PresetByName(presetName)
	if !ok {
		return "(unknown preset)"
	}

	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Accent))
	border := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Border))
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(preset.StatusFg)).
		Background(lipgloss.Color(preset.StatusBg))

	var b strings.Builder
	b.WriteString(border.Render("\u250c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510"))
	b.WriteString("\n")
	b.WriteString(border.Render("\u2502") + " " + accent.Render("\u25cf") + " My notebook   " + border.Render("\u2502"))
	b.WriteString("\n")
	b.WriteString(border.Render("\u2502") + "   Another note   " + border.Render("\u2502"))
	b.WriteString("\n")
	b.WriteString(border.Render("\u2502") + "   Third item     " + border.Render("\u2502"))
	b.WriteString("\n")
	b.WriteString(border.Render("\u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518"))
	b.WriteString("\n")
	b.WriteString("  " + statusStyle.Render(" Status: saved \u2713 "))

	return b.String()
}

// renderThemeOverlay builds the theme picker overlay with style list and preview.
func (m Model) renderThemeOverlay() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	// Tab bar.
	var tabBar strings.Builder
	glamourTab := "Glamour Style"
	uiTab := "UI Theme"
	if m.themeTab == 0 {
		tabBar.WriteString("  " + bold.Render("["+glamourTab+"]"))
		tabBar.WriteString("  " + dim.Render("["+uiTab+"]"))
	} else {
		tabBar.WriteString("  " + dim.Render("["+glamourTab+"]"))
		tabBar.WriteString("  " + bold.Render("["+uiTab+"]"))
	}
	tabBar.WriteString("\n")
	tabBar.WriteString("  \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n")

	// Left pane: list for active tab.
	listWidth := 30
	var left strings.Builder
	left.WriteString(tabBar.String())

	if m.themeTab == 0 {
		// Glamour style list.
		for i, s := range m.themeStyles {
			if s == "---" {
				left.WriteString(dim.Render("  ── community ──") + "\n")
				continue
			}
			hint := styleHints[s]
			hintStr := ""
			if hint != "" {
				hintStr = " " + dim.Render(hint)
			}
			if i == m.themeCursor {
				sel := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
				left.WriteString(fmt.Sprintf("  %s %s%s\n", sel.Render("\u25cf"), sel.Render(s), hintStr))
			} else {
				left.WriteString(fmt.Sprintf("    %s%s\n", s, hintStr))
			}
		}

		// Show warning when highlighted style conflicts with terminal background.
		if cur := m.themeStyles[m.themeCursor]; cur != "auto" {
			curHint := styleHints[cur]
			conflict := false
			if m.darkTerminal && curHint == "(light bg)" {
				conflict = true
			}
			if !m.darkTerminal && curHint == "(dark bg)" {
				conflict = true
			}
			if conflict {
				left.WriteString("\n")
				left.WriteString(dim.Render("  \u26a0 May be hard to read\n    on your terminal"))
				left.WriteString("\n")
			}
		}
	} else {
		// UI theme preset list.
		presets := theme.Presets()
		for i, p := range presets {
			bgHint := ""
			if p.Background != "" {
				bgHint = " " + dim.Render("("+p.Background+" bg)")
			}
			if i == m.uiThemeCursor {
				sel := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
				left.WriteString(fmt.Sprintf("  %s %s%s\n", sel.Render("\u25cf"), sel.Render(p.Name), bgHint))
			} else {
				left.WriteString(fmt.Sprintf("    %s%s\n", p.Name, bgHint))
			}
		}
	}

	// Right pane: preview for active tab.
	previewWidth := w - listWidth - 10
	if previewWidth < 20 {
		previewWidth = 20
	}

	var previewContent string
	if m.themeTab == 0 {
		previewContent = m.themePreview
	} else {
		previewContent = m.uiThemePreview
	}

	// Clamp preview lines to fit the overlay height.
	previewLines := strings.Split(previewContent, "\n")
	maxPreview := h - 8
	if maxPreview < 1 {
		maxPreview = 1
	}
	if len(previewLines) > maxPreview {
		previewLines = previewLines[:maxPreview]
	}

	previewBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Current().Border)).
		Padding(0, 1).
		Width(previewWidth)

	preview := previewBox.Render(strings.Join(previewLines, "\n"))

	// Combine left + right.
	leftBox := lipgloss.NewStyle().
		Width(listWidth).
		Padding(1, 0)

	rightBox := lipgloss.NewStyle().
		Padding(1, 0)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox.Render(left.String()), rightBox.Render(preview))

	// Outer container.
	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Current().Border)).
		Padding(0, 1).
		MaxHeight(h - 2)

	rendered := outer.Render(combined)

	// Status hint.
	statusHint := dim.Render("  \u2191/\u2193 navigate \u00b7 Tab switch \u00b7 Enter apply \u00b7 Esc cancel")

	full := rendered + "\n" + statusHint

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, full)
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter)
	m.filtered = nil

	if m.level == 0 {
		for i, nb := range m.notebooks {
			if query == "" || strings.Contains(storage.DisplayName(nb.name), query) {
				m.filtered = append(m.filtered, i)
			}
		}
	} else {
		for i, n := range m.notes {
			if query == "" || strings.Contains(storage.DisplayName(n.Name), query) {
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
		m.savedCursor = m.cursor
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		m.notes = nil
		m.filtered = nil
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
		m.cursor = m.savedCursor
		m.filter = ""
		m.filtering = false
		m.notebooks = nil
		m.filtered = nil
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
  t          Theme picker
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
  t          Theme picker
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
		BorderForeground(lipgloss.Color(theme.Current().Border)).
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

	if m.themeMode {
		return m.renderThemeOverlay()
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
	return style.Render(fmt.Sprintf("notebook \u203A %s", storage.DisplayName(m.currentBook)))
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

// padRight pads s with trailing spaces to reach width, using lipgloss.Width
// to measure display width (ignoring ANSI escape sequences).
func padRight(s string, width int) string {
	display := lipgloss.Width(s)
	if display >= width {
		return s
	}
	return s + strings.Repeat(" ", width-display)
}

func (m Model) formatNotebookLine(nb notebookItem, selected bool) string {
	bullet := "  "
	display := storage.DisplayName(nb.name)
	name := display
	countStr := pluralize(nb.noteCount, "note", "notes")

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		name = nameStyle.Render(display)
	}

	return fmt.Sprintf("%s%s    %-*s    %s",
		padRight(bullet, 2),
		padRight(name, m.nameColWidth(0)),
		10, countStr,
		nb.modTime,
	)
}

func (m Model) formatNoteLine(n model.Note, selected bool) string {
	bullet := "  "
	display := storage.DisplayName(n.Name)
	name := display

	// Get file size.
	p := m.store.NotebookDir(m.currentBook) + "/" + n.Name + ".md"
	info, err := os.Stat(p)
	var sizeStr, timeStr string
	if err == nil {
		sizeStr = humanSize(info.Size())
		timeStr = relativeTime(info.ModTime())
	}

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		name = nameStyle.Render(display)
	}

	return fmt.Sprintf("%s%s    %-*s    %s",
		padRight(bullet, 2),
		padRight(name, m.nameColWidth(1)),
		8, sizeStr,
		timeStr,
	)
}

func (m Model) nameColWidth(level int) int {
	maxLen := 0
	if level == 0 {
		for _, nb := range m.notebooks {
			if dl := len(storage.DisplayName(nb.name)); dl > maxLen {
				maxLen = dl
			}
		}
	} else {
		for _, n := range m.notes {
			if dl := len(storage.DisplayName(n.Name)); dl > maxLen {
				maxLen = dl
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
	msg := fmt.Sprintf("No notes in %s.\n\nPress n to create one.", storage.DisplayName(m.currentBook))
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, dim.Render(msg))
}

func (m Model) renderStatusBar() string {
	dim := lipgloss.NewStyle().Faint(true)

	if m.inputMode {
		before := m.inputValue[:m.inputCursor]
		after := m.inputValue[m.inputCursor:]
		cursor := lipgloss.NewStyle().Reverse(true)
		cursorChar := " "
		if m.inputCursor < len(m.inputValue) {
			cursorChar = string(m.inputValue[m.inputCursor])
			after = after[1:]
		}
		return dim.Render(fmt.Sprintf("  %s %s", m.inputPrompt, before)) + cursor.Render(cursorChar) + dim.Render(after) + dim.Render(" · Enter confirm · Esc cancel")
	}

	if m.statusText != "" {
		return dim.Render(fmt.Sprintf("  %s", m.statusText))
	}

	if m.filtering {
		before := m.filter[:m.filterCursor]
		after := m.filter[m.filterCursor:]
		cursor := lipgloss.NewStyle().Reverse(true)
		cursorChar := " "
		if m.filterCursor < len(m.filter) {
			cursorChar = string(m.filter[m.filterCursor])
			after = after[1:]
		}
		return dim.Render("  Filter: "+before) + cursor.Render(cursorChar) + dim.Render(after+" \u00B7 Esc clear \u00B7 Enter select")
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
