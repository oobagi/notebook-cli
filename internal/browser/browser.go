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
	"github.com/oobagi/notebook/internal/format"
	"github.com/oobagi/notebook/internal/model"
	"github.com/oobagi/notebook/internal/recents"
	"github.com/oobagi/notebook/internal/storage"
	"github.com/oobagi/notebook/internal/theme"
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
	Book     string
	Note     string
	FilePath string // non-empty for external file selections
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

	// Recents view fields.
	recentsView    bool            // whether recents tab is active at L0
	recentEntries  []recents.Entry // loaded recent entries
	filteredRecent []int           // indices into recentEntries after filtering

	// Theme picker fields.
	themeMode      bool   // theme picker overlay visible
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

// recentsLoadedMsg carries the loaded recents list.
type recentsLoadedMsg struct {
	entries []recents.Entry
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
				timeStr = format.RelativeTime(modTime)
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

func (m Model) loadRecents() tea.Cmd {
	return func() tea.Msg {
		entries, err := recents.LoadPruned(m.store.Root)
		if err != nil {
			return errMsg{err}
		}
		return recentsLoadedMsg{entries: entries}
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

	case recentsLoadedMsg:
		m.recentEntries = msg.entries
		m.resetFilter()
		return m, nil

	case reloadMsg:
		if m.recentsView {
			return m, m.loadRecents()
		}
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
	case tea.KeyTab:
		// Toggle between notebooks and recents at L0.
		if m.level == 0 {
			m.recentsView = !m.recentsView
			m.cursor = 0
			m.filter = ""
			m.filtering = false
			m.filtered = nil
			if m.recentsView {
				return m, m.loadRecents()
			}
			return m, m.loadNotebooks()
		}
		return m, nil

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
			if m.recentsView {
				return m.removeRecentEntry()
			}
			return m.startDelete()
		}
		if s == "r" && !m.recentsView {
			return m.startRename()
		}
		if s == "n" && !m.recentsView {
			return m.startCreate()
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

func (m Model) removeRecentEntry() (tea.Model, tea.Cmd) {
	if len(m.filteredRecent) == 0 {
		return m, nil
	}
	if m.cursor >= len(m.filteredRecent) {
		return m, nil
	}
	idx := m.filteredRecent[m.cursor]
	target := m.recentEntries[idx]

	return m, func() tea.Msg {
		entries, err := recents.Load(recents.DefaultPath())
		if err != nil {
			return statusMsg{fmt.Sprintf("Could not load recents: %s", err)}
		}
		entries = recents.Remove(entries, target)
		if err := recents.Save(recents.DefaultPath(), entries); err != nil {
			return statusMsg{fmt.Sprintf("Could not save recents: %s", err)}
		}
		return recentsLoadedMsg{entries: entries}
	}
}

func (m Model) startThemePicker() (tea.Model, tea.Cmd) {
	m.themeMode = true

	// Initialize UI theme cursor to match current config value.
	m.uiThemeCursor = 0
	presets := theme.Presets()
	if cfg, err := config.Load(); err == nil && cfg.Theme != "" {
		for i, p := range presets {
			if p.Name == cfg.Theme {
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

	case tea.KeyUp:
		if m.uiThemeCursor > 0 {
			m.uiThemeCursor--
			presets := theme.Presets()
			if len(presets) > 0 {
				m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
			}
		}
		return m, nil

	case tea.KeyDown:
		presets := theme.Presets()
		if m.uiThemeCursor < len(presets)-1 {
			m.uiThemeCursor++
			m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
		}
		return m, nil

	case tea.KeyEnter:
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
		if err := config.Set(&cfg, "theme", selected.Name); err != nil {
			m.statusText = fmt.Sprintf("Config error: %s", err)
			return m, m.scheduleStatusDismiss()
		}
		if err := config.Save(cfg); err != nil {
			m.statusText = fmt.Sprintf("Config save error: %s", err)
			return m, m.scheduleStatusDismiss()
		}
		theme.SetTheme(selected)
		m.statusText = fmt.Sprintf("UI theme set to %s", selected.Name)
		return m, m.scheduleStatusDismiss()

	case tea.KeyRunes:
		if string(msg.Runes) == "q" || string(msg.Runes) == "t" {
			m.themeMode = false
			return m, nil
		}
	}

	return m, nil
}

// renderUIThemePreview generates a block-style preview for a UI theme preset.
func (m Model) renderUIThemePreview(presetName string) string {
	preset, ok := theme.PresetByName(presetName)
	if !ok {
		return "(unknown preset)"
	}

	bs := preset.Blocks
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Accent))
	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder

	// Heading
	h1Style := bs.Heading1.Text.ToLipgloss(preset.Accent)
	b.WriteString(h1Style.Render("My Notebook"))
	b.WriteString("\n\n")

	// Bullet
	bulletColor := bs.Bullet.MarkerColor
	if bulletColor == "" {
		bulletColor = preset.Muted
	}
	bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(bulletColor))
	b.WriteString(bulletStyle.Render(bs.Bullet.Marker) + "First item\n")
	b.WriteString(bulletStyle.Render(bs.Bullet.Marker) + "Second item\n\n")

	// Numbered
	numColor := bs.Numbered.MarkerColor
	if numColor == "" {
		numColor = preset.Muted
	}
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(numColor))
	b.WriteString(numStyle.Render(fmt.Sprintf(bs.Numbered.Format, 1)) + "Step one\n")
	b.WriteString(numStyle.Render(fmt.Sprintf(bs.Numbered.Format, 2)) + "Step two\n\n")

	// Checklist
	uncheckedColor := bs.Checklist.UncheckedColor
	if uncheckedColor == "" {
		uncheckedColor = preset.Muted
	}
	checkedColor := bs.Checklist.CheckedColor
	if checkedColor == "" {
		checkedColor = preset.Accent
	}
	checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(checkedColor))
	if bs.Checklist.CheckedBold {
		checkedStyle = checkedStyle.Bold(true)
	}
	uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(uncheckedColor))

	checkedText := "Done task"
	if bs.Checklist.CheckedTextFaint {
		checkedText = faint.Render(checkedText)
	}
	b.WriteString(checkedStyle.Render(bs.Checklist.Checked) + checkedText + "\n")
	b.WriteString(uncheckedStyle.Render(bs.Checklist.Unchecked) + "Pending task\n\n")

	// Code block (small sample)
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Border))
	codeW := 16
	codeLine := "fmt.Println()"
	codePad := codeW - len(codeLine)
	if codePad < 0 {
		codePad = 0
	}
	paddedCode := codeLine + strings.Repeat(" ", codePad)

	topBorder := bc.Render("╭" + strings.Repeat("─", codeW+2) + "╮")
	bottomBorder := bc.Render("╰" + strings.Repeat("─", codeW+2) + "╯")
	lang := "go"
	langLabel := faint.Render(lang)
	labelW := lipgloss.Width(langLabel)

	// Label always in top border, aligned per theme.
	dashes := codeW + 2 - labelW - 3
	if dashes < 1 {
		dashes = 1
	}
	switch bs.Code.LabelAlign {
	case "center":
		total := codeW - lipgloss.Width(lang)
		if total < 2 {
			total = 2
		}
		left := total / 2
		right := total - left
		topBorder = bc.Render("╭"+strings.Repeat("─", left)+" ") + langLabel + bc.Render(" "+strings.Repeat("─", right)+"╮")
	case "right":
		topBorder = bc.Render("╭"+strings.Repeat("─", dashes)+" ") + langLabel + bc.Render(" ─╮")
	default: // "left"
		topBorder = bc.Render("╭─ ") + langLabel + bc.Render(" "+strings.Repeat("─", dashes)+"╮")
	}
	b.WriteString(topBorder + "\n")
	b.WriteString(bc.Render("│") + " " + paddedCode + " " + bc.Render("│") + "\n")
	b.WriteString(bottomBorder + "\n")
	b.WriteString("\n")

	// Quote
	barColor := bs.Quote.BarColor
	if barColor == "" {
		barColor = preset.Muted
	}
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor))
	b.WriteString(barStyle.Render(bs.Quote.Bar) + "A wise quote\n\n")

	// Divider
	divColor := bs.Divider.Color
	if divColor == "" {
		divColor = preset.Accent
	}
	maxW := bs.Divider.MaxWidth
	if maxW <= 0 {
		maxW = 40
	}
	divW := 18
	if divW > maxW {
		divW = maxW
	}
	divStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(divColor))
	b.WriteString(divStyle.Render(strings.Repeat(bs.Divider.Char, divW)) + "\n\n")

	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(preset.StatusFg)).
		Background(lipgloss.Color(preset.StatusBg))
	b.WriteString(statusStyle.Render(" saved ✓ ") + " " + accent.Render(preset.Name))

	return b.String()
}

// renderThemeOverlay builds the theme picker overlay with preset list and preview.
func (m Model) renderThemeOverlay() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	dim := lipgloss.NewStyle().Faint(true)

	// Left pane: UI theme preset list.
	listWidth := 30
	var left strings.Builder
	left.WriteString("  UI Theme\n")
	left.WriteString("  \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n")

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

	// Right pane: preview.
	previewWidth := w - listWidth - 10
	if previewWidth < 20 {
		previewWidth = 20
	}

	previewContent := m.uiThemePreview

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
	statusHint := dim.Render("  \u2191/\u2193 navigate \u00b7 Enter apply \u00b7 Esc cancel")

	full := rendered + "\n" + statusHint

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, full)
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter)

	if m.recentsView {
		m.filteredRecent = nil
		for i, e := range m.recentEntries {
			label := recentEntryLabel(e)
			if query == "" || strings.Contains(strings.ToLower(label), query) {
				m.filteredRecent = append(m.filteredRecent, i)
			}
		}
		if m.cursor >= len(m.filteredRecent) {
			m.cursor = len(m.filteredRecent) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return
	}

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
	m.filteredRecent = nil

	if m.recentsView {
		m.filteredRecent = make([]int, len(m.recentEntries))
		for i := range m.recentEntries {
			m.filteredRecent[i] = i
		}
		if m.cursor >= len(m.filteredRecent) {
			m.cursor = len(m.filteredRecent) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return
	}

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
	if m.recentsView {
		return len(m.filteredRecent)
	}
	return len(m.filtered)
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	// Handle recents view selection.
	if m.recentsView {
		if len(m.filteredRecent) == 0 {
			return m, nil
		}
		if m.cursor >= len(m.filteredRecent) {
			return m, nil
		}
		idx := m.filteredRecent[m.cursor]
		entry := m.recentEntries[idx]
		switch entry.Type {
		case recents.TypeStore:
			m.selected = &Selection{
				Book: entry.Notebook,
				Note: entry.Name,
			}
		case recents.TypeExternal:
			m.selected = &Selection{
				FilePath: entry.Path,
			}
		}
		return m, tea.Quit
	}

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
	if m.recentsView {
		m.recentsView = false
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		m.filtered = nil
		m.filteredRecent = nil
		return m, m.loadNotebooks()
	}
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
	if m.recentsView {
		help = `  Keybindings
  ───────────────────────────

  ↑/↓       Navigate
  Enter      Open note/file
  d          Remove from recents
  Tab        Switch to notebooks
  t          Theme picker
  /          Search
  Esc        Back to notebooks
  q          Quit
  ?          Toggle help

  Press ? or Esc to close`
	} else if m.level == 0 {
		help = `  Keybindings
  ───────────────────────────

  ↑/↓       Navigate
  Enter      Open notebook
  n          New notebook
  d          Delete notebook
  r          Rename notebook
  Tab        Switch to recents
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

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.renderHelpOverlay()
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
	dim := lipgloss.NewStyle().Faint(true)

	if m.recentsView {
		active := style.Render("Recent")
		inactive := dim.Render("Notebooks")
		return fmt.Sprintf(" %s  %s", active, inactive)
	}

	if m.level == 0 {
		active := style.Render("Notebooks")
		inactive := dim.Render("Recent")
		return fmt.Sprintf(" %s  %s", active, inactive)
	}
	return fmt.Sprintf(" %s", style.Render(fmt.Sprintf("notebook \u203A %s", storage.DisplayName(m.currentBook))))
}

func (m Model) renderContent(maxLines int) string {
	if m.recentsView {
		return m.renderRecentList(maxLines)
	}
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

func (m Model) renderRecentList(maxLines int) string {
	if len(m.recentEntries) == 0 {
		return m.renderEmptyRecents()
	}

	if len(m.filteredRecent) == 0 && m.filtering {
		return "  No matches\n"
	}

	var b strings.Builder
	visible := m.visibleRange(len(m.filteredRecent), maxLines)

	for vi, fi := range visible {
		idx := m.filteredRecent[fi]
		e := m.recentEntries[idx]
		selected := fi == m.cursor
		line := m.formatRecentLine(e, selected)
		b.WriteString(line)
		if vi < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) formatRecentLine(e recents.Entry, selected bool) string {
	bullet := "  "
	label := recentEntryLabel(e)
	display := label
	timeStr := format.RelativeTime(e.LastEdited)

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		display = nameStyle.Render(label)
	}

	return fmt.Sprintf("%s%s    %s",
		padRight(bullet, 2),
		padRight(display, m.recentNameColWidth()),
		timeStr,
	)
}

func (m Model) recentNameColWidth() int {
	maxLen := 0
	for _, e := range m.recentEntries {
		if dl := len(recentEntryLabel(e)); dl > maxLen {
			maxLen = dl
		}
	}
	if maxLen < 10 {
		maxLen = 10
	}
	return maxLen
}

// recentEntryLabel returns the display label for a recent entry.
func recentEntryLabel(e recents.Entry) string {
	switch e.Type {
	case recents.TypeStore:
		return storage.DisplayName(e.Notebook) + " \u203A " + storage.DisplayName(e.Name)
	case recents.TypeExternal:
		return format.ShortenHome(e.Path)
	}
	return ""
}

func (m Model) renderEmptyRecents() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	dim := lipgloss.NewStyle().Faint(true)
	msg := "No recent notes.\n\nNotes appear here after you edit and save them."
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, dim.Render(msg))
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
		sizeStr = format.HumanSize(info.Size())
		timeStr = format.RelativeTime(info.ModTime())
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

	if m.recentsView {
		return dim.Render("  \u2191/\u2193 navigate \u00B7 Enter open \u00B7 d remove \u00B7 Tab notebooks \u00B7 / search \u00B7 q quit \u00B7 ? help")
	}

	if m.level == 0 {
		return dim.Render("  \u2191/\u2193 navigate \u00B7 Enter open \u00B7 Tab recents \u00B7 / search \u00B7 q quit \u00B7 ? help")
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
