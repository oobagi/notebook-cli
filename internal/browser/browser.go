// Package browser provides a fullscreen TUI for browsing notebooks and notes.
package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oobagi/notebook-cli/internal/clipboard"
	"github.com/oobagi/notebook-cli/internal/config"
	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/model"
	"github.com/oobagi/notebook-cli/internal/recents"
	"github.com/oobagi/notebook-cli/internal/storage"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/oobagi/notebook-cli/internal/ui"
)

// Config holds the dependencies needed by the browser.
type Config struct {
	Store          *storage.Store
	InitialBook    string     // if set, start at L1 in this notebook
	RestoreSel     *Selection // if set, reposition cursor to this item after load
	DismissedHints map[string]bool
	ShowPreview    *bool // from config; nil = default (true)
}

// Selection represents a note the user chose to open.
type Selection struct {
	Book       string
	Note       string
	FilePath   string // non-empty for external file selections
	FromRecent bool   // true if selected from the recents section
}

// Model is the Bubble Tea model for the notebook/note browser.
type Model struct {
	store        *storage.Store
	level        int // 0=notebooks, 1=notes
	notebooks    []notebookItem
	notes        []model.Note
	currentBook  string // selected notebook name
	cursor       int    // current selection index
	filter       string // fuzzy search filter text
	filtering    bool   // whether filter mode is active
	filterCursor int    // cursor position within filter
	filtered     []int  // indices into notebooks/notes after filtering
	width        int
	height       int
	showHelp     bool // help overlay visible
	quitting     bool
	selected     *Selection // set when user picks a note to edit
	err          error

	// Input mode fields (used by create, rename, delete type-to-confirm).
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputCursor int // cursor position within inputValue
	inputAction func(typed string) tea.Cmd

	// After a rename, this holds the new name so the cursor repositions to it.
	selectAfterReload string

	// restoreSel holds the previously selected item to reposition the cursor after load.
	restoreSel *Selection

	// Temporary status message shown in the status bar.
	statusText string
	statusGen  int // generation counter for auto-dismiss

	// Recents view fields.
	recentEntries  []recents.Entry // loaded recent entries
	filteredRecent []int           // indices into recentEntries after filtering

	// Theme picker fields.
	themeMode      bool   // theme picker overlay visible
	uiThemeCursor  int    // cursor in UI theme preset list
	uiThemePreview string // preview for highlighted UI preset

	// Settings overlay fields.
	showSettings    bool
	settingsCursor  int
	settingsEditing bool          // true when inline-editing a string setting value
	settingsCfg     config.Config // cached config, loaded on entry, refreshed on change

	// Onboarding hints.
	dismissedHints map[string]bool

	// Cursor blink for input/filter modes.
	inputCur cursor.Model

	// Preview toggle.
	showPreview bool

	// L0 search (flat result list, in-memory).
	allNoteNames  []l0SearchResult // all note names cached on first search
	searchResults []l0SearchResult // flat search results filtered from allNoteNames
}

// notebookItem holds pre-fetched metadata for a notebook.
type notebookItem struct {
	name      string
	noteCount int
	modTime   string
}

// settingKind describes how a setting is edited in the settings overlay.
type settingKind int

const (
	settingBool  settingKind = iota // Enter toggles true/false
	settingCycle                    // Enter cycles through options
	settingText                     // Enter opens inline text input
)

type settingDef struct {
	key     string
	label   string
	desc    string
	kind    settingKind
	options []string // for cycle kind
}

func settingDefs() []settingDef {
	return []settingDef{
		{"date_format", "Date format", "How dates are displayed in the browser", settingCycle,
			[]string{"relative", "2006-01-02", "Jan 2, 2006", "01/02/2006", "02 Jan 2006"}},
		{"editor", "External editor", "Command used when opening notes externally", settingCycle,
			[]string{"", "vim", "nvim", "nano", "code", "subl", "emacs", "micro", "helix"}},
		{"storage_dir", "Storage directory", "Where notebooks and notes are stored on disk", settingText, nil},
		{"hide_checked", "Sort checked", "Sort checked items to the bottom of checklists", settingBool, nil},
		{"cascade_checks", "Cascade checks", "Checking a parent also checks its children", settingBool, nil},
		{"show_preview", "Show preview", "Show the note preview pane in the browser", settingBool, nil},
		{"word_wrap", "Word wrap", "Wrap long lines in the editor", settingBool, nil},
	}
}

func settingDisplayValue(def settingDef, val string) string {
	if def.key == "date_format" && val != "relative" && val != "" {
		return time.Now().Format(val)
	}
	if def.key == "editor" && val == "" {
		return "(default)"
	}
	if val == "" {
		return "(default)"
	}
	return val
}

// New creates a new browser model.
func New(cfg Config) Model {
	m := Model{
		store:       cfg.Store,
		showPreview: config.BoolVal(cfg.ShowPreview, true),
	}
	if cfg.InitialBook != "" {
		m.level = 1
		m.currentBook = cfg.InitialBook
	}
	m.restoreSel = cfg.RestoreSel
	m.dismissedHints = cfg.DismissedHints
	if m.dismissedHints == nil {
		m.dismissedHints = make(map[string]bool)
	}
	m.inputCur = cursor.New()
	m.inputCur.Style = lipgloss.NewStyle()
	m.inputCur.TextStyle = lipgloss.NewStyle().Faint(true)
	return m
}

// Selected returns the note selection if the user chose one, or nil.
func (m Model) Selected() *Selection {
	return m.selected
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

// l0SearchResult represents a single note in the flat L0 search list.
type l0SearchResult struct {
	notebook string
	note     string
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.level == 1 {
		return m.loadNotes(m.currentBook)
	}
	return tea.Batch(m.loadNotebooks(), m.loadRecents())
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

// importFileMsg signals the user confirmed an external file path to open.
type importFileMsg struct{ path string }

// statusMsg carries a temporary status message for the status bar.
type statusMsg struct{ text string }

// statusTimeoutMsg is sent after the status auto-dismiss delay.
// The generation field is compared against Model.statusGen to ensure
// only the most recent status message is cleared.
type statusTimeoutMsg struct{ generation int }

type clipboardReadErrMsg struct{ err error }

// statusTimeout is the delay before auto-dismissing a transient status message.
const statusTimeout = 4 * time.Second

func readClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.Read()
		if err != nil {
			return clipboardReadErrMsg{err: err}
		}
		return tea.PasteMsg{Content: text}
	}
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
		return m, nil

	case notesLoadedMsg:
		m.notes = msg.notes
		m.resetFilter()
		return m, nil

	case recentsLoadedMsg:
		m.recentEntries = msg.entries
		m.resetFilter()
		return m, nil

	case reloadMsg:
		m.allNoteNames = nil // invalidate search cache
		if m.level == 0 {
			return m, tea.Batch(m.loadNotebooks(), m.loadRecents())
		}
		return m, m.loadNotes(m.currentBook)

	case reloadAndSelectMsg:
		m.allNoteNames = nil // invalidate search cache
		m.selectAfterReload = msg.name
		if m.level == 0 {
			return m, m.loadNotebooks()
		}
		return m, m.loadNotes(m.currentBook)

	case importFileMsg:
		m.selected = &Selection{
			FilePath:   msg.path,
			FromRecent: true,
		}
		return m, tea.Quit

	case statusMsg:
		m.statusText = msg.text
		return m, m.scheduleStatusDismiss()

	case settingsEditDoneMsg:
		m.settingsEditing = false
		m.inputMode = false
		_ = config.Set(&m.settingsCfg, msg.key, msg.value)
		m.saveSettingsCfg()
		return m, nil

	case statusTimeoutMsg:
		if msg.generation == m.statusGen {
			m.statusText = ""
		}
		return m, nil

	case clipboardReadErrMsg:
		m.statusText = "Paste failed: " + msg.err.Error()
		return m, m.scheduleStatusDismiss()

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.PasteMsg:
		return m.handlePaste(msg.Content)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Forward to cursor blink model when in input/filter mode.
	if m.inputMode || m.filtering {
		var cmd tea.Cmd
		m.inputCur, cmd = m.inputCur.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handlePaste(text string) (tea.Model, tea.Cmd) {
	if m.showSettings && m.settingsEditing {
		m.inputCur.IsBlinked = false
		m.inputValue, m.inputCursor = ui.InsertLineText(m.inputValue, m.inputCursor, text)
		return m, nil
	}
	if m.inputMode {
		m.inputCur.IsBlinked = false
		m.inputValue, m.inputCursor = ui.InsertLineText(m.inputValue, m.inputCursor, text)
		return m, nil
	}
	if m.filtering {
		m.inputCur.IsBlinked = false
		m.filter, m.filterCursor = ui.InsertLineText(m.filter, m.filterCursor, text)
		m.applyFilter()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Clear any lingering status text on next keypress.
	m.statusText = ""
	// Clear deferred cursor positioning once the user interacts.
	m.selectAfterReload = ""
	m.restoreSel = nil

	// Global quit keys.
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// When help overlay is showing, ?/Esc close it, Ctrl+C quits.
	if m.showHelp {
		switch msg.String() {
		case "esc":
			m.showHelp = false
			return m, nil
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			if msg.Text == "?" {
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

	// When settings overlay is showing.
	if m.showSettings {
		if m.settingsEditing {
			return m.handleSettingsInputKey(msg)
		}
		return m.handleSettingsKey(msg)
	}

	// When in input mode, delegate to input handler.
	if m.inputMode {
		return m.handleInputKey(msg)
	}

	// When filtering, handle text input first.
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	switch msg.String() {
	case "tab":
		m.tabNextSection()
		return m, nil

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down":
		max := m.listLen() - 1
		if max < 0 {
			max = 0
		}
		if m.cursor < max {
			m.cursor++
		}
		return m, nil

	case "enter":
		return m.handleEnter()

	case "esc":
		return m.handleEsc()

	default:
		s := msg.Text
		if s == "?" {
			m.showHelp = true
			return m, nil
		}
		if s == "h" {
			if hint := m.currentHintID(); hint != "" {
				if m.dismissedHints == nil {
					m.dismissedHints = make(map[string]bool)
				}
				m.dismissedHints[hint] = true
				config.DismissHint(hint)
			}
			return m, nil
		}
		if s == "/" {
			m.filtering = true
			m.filter = ""
			m.filterCursor = 0
			m.applyFilter()
			return m, m.inputCur.Focus()
		}
		if s == "d" {
			if m.level == 0 && m.inRecentsSection() {
				return m.removeRecentEntry()
			}
			return m.startDelete()
		}
		if s == "r" && !(m.level == 0 && m.inRecentsSection()) {
			return m.startRename()
		}
		if s == "n" {
			return m.startCreate()
		}
		if s == "i" && m.level == 0 {
			return m.startImport()
		}
		if s == "c" {
			if m.level == 0 && m.inRecentsSection() {
				return m.copyRecentEntry()
			}
			if m.level == 1 {
				return m.copyNote()
			}
		}
		if s == "t" {
			return m.startThemePicker()
		}
		if s == "," {
			m.showSettings = true
			m.settingsCursor = 0
			m.settingsCfg, _ = config.Load()
			return m, nil
		}
		if s == "p" {
			m.showPreview = !m.showPreview
			if m.showPreview {
				m.statusText = "Preview on"
			} else {
				m.statusText = "Preview off"
			}
			if globalCfg, err := config.Load(); err == nil {
				globalCfg.ShowPreview = config.BoolPtr(m.showPreview)
				_ = config.Save(globalCfg)
			}
			return m, m.scheduleStatusDismiss()
		}
		return m, nil
	}
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.inputCur.IsBlinked = false
	// "/" with empty filter dismisses search (toggle behavior).
	if msg.Text == "/" && m.filter == "" {
		m.filtering = false
		m.resetFilter()
		m.inputCur.Blur()
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.filterCursor = 0
		m.resetFilter()
		m.inputCur.Blur()
		return m, nil
	case "enter":
		m.filtering = false
		m.inputCur.Blur()
		return m.handleEnter()
	case "tab":
		m.tabNextSection()
		return m, nil
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down":
		max := m.listLen() - 1
		if max < 0 {
			max = 0
		}
		if m.cursor < max {
			m.cursor++
		}
		return m, nil
	case "ctrl+v":
		return m, readClipboardCmd()
	default:
		prev := m.filter
		m.filter, m.filterCursor, _ = ui.EditLine(m.filter, m.filterCursor, msg)
		if m.filter != prev {
			m.applyFilter()
		}
	}
	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.inputCur.IsBlinked = false
	switch msg.String() {
	case "esc":
		m.inputMode = false
		m.inputPrompt = ""
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		m.inputCur.Blur()
		return m, nil
	case "enter":
		action := m.inputAction
		value := m.inputValue
		m.inputMode = false
		m.inputPrompt = ""
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		m.inputCur.Blur()
		if action != nil {
			return m, action(value)
		}
		return m, nil
	case "ctrl+v":
		return m, readClipboardCmd()
	default:
		m.inputValue, m.inputCursor, _ = ui.EditLine(m.inputValue, m.inputCursor, msg)
	}
	return m, nil
}

func (m Model) handleSettingsInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.inputCur.IsBlinked = false
	switch msg.String() {
	case "esc":
		m.settingsEditing = false
		m.inputMode = false
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		m.inputCur.Blur()
		return m, nil
	case "enter":
		action := m.inputAction
		value := m.inputValue
		m.settingsEditing = false
		m.inputMode = false
		m.inputValue = ""
		m.inputCursor = 0
		m.inputAction = nil
		m.inputCur.Blur()
		if action != nil {
			return m, action(value)
		}
		return m, nil
	case "ctrl+v":
		return m, readClipboardCmd()
	default:
		m.inputValue, m.inputCursor, _ = ui.EditLine(m.inputValue, m.inputCursor, msg)
	}
	return m, nil
}

func (m Model) handleSettingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	defs := settingDefs()
	if msg.Text == "," {
		m.showSettings = false
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.showSettings = false
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "up":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
		return m, nil
	case "down":
		if m.settingsCursor < len(defs)-1 {
			m.settingsCursor++
		}
		return m, nil
	case "enter", "space", "right":
		if m.settingsCursor < len(defs) {
			return m.applySettingChange(defs[m.settingsCursor])
		}
		return m, nil
	case "left":
		if m.settingsCursor < len(defs) {
			return m.applySettingChangeReverse(defs[m.settingsCursor])
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) saveSettingsCfg() {
	_ = config.Save(m.settingsCfg)
}

func (m Model) applySettingChange(def settingDef) (tea.Model, tea.Cmd) {
	current, _ := config.Get(m.settingsCfg, def.key)

	switch def.kind {
	case settingBool:
		newVal := "true"
		if current == "true" {
			newVal = "false"
		}
		_ = config.Set(&m.settingsCfg, def.key, newVal)
		m.saveSettingsCfg()
		if def.key == "show_preview" {
			m.showPreview = newVal == "true"
		}

	case settingCycle:
		idx := 0
		for i, opt := range def.options {
			if opt == current {
				idx = i
				break
			}
		}
		next := def.options[(idx+1)%len(def.options)]
		_ = config.Set(&m.settingsCfg, def.key, next)
		m.saveSettingsCfg()

	case settingText:
		m.settingsEditing = true
		m.inputMode = true
		m.inputPrompt = def.label
		m.inputValue = current
		m.inputCursor = len(current)
		m.inputAction = func(typed string) tea.Cmd {
			return func() tea.Msg {
				return settingsEditDoneMsg{key: def.key, value: typed}
			}
		}
		return m, m.inputCur.Focus()
	}

	return m, nil
}

func (m Model) applySettingChangeReverse(def settingDef) (tea.Model, tea.Cmd) {
	current, _ := config.Get(m.settingsCfg, def.key)

	switch def.kind {
	case settingBool:
		return m.applySettingChange(def)
	case settingCycle:
		idx := 0
		for i, opt := range def.options {
			if opt == current {
				idx = i
				break
			}
		}
		prev := def.options[(idx-1+len(def.options))%len(def.options)]
		_ = config.Set(&m.settingsCfg, def.key, prev)
		m.saveSettingsCfg()
	case settingText:
		return m.applySettingChange(def) // text editing is the same
	}
	return m, nil
}

type settingsEditDoneMsg struct {
	key   string
	value string
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		// Delete notebook.
		if len(m.filtered) == 0 {
			return m, nil
		}
		_, localIdx := m.cursorSection()
		idx := m.filtered[localIdx]
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
	return m, m.inputCur.Focus()
}

func (m Model) startRename() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		if len(m.filtered) == 0 {
			return m, nil
		}
		_, localIdx := m.cursorSection()
		idx := m.filtered[localIdx]
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
	return m, m.inputCur.Focus()
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
	return m, m.inputCur.Focus()
}

func (m Model) startImport() (tea.Model, tea.Cmd) {
	m.inputMode = true
	m.inputPrompt = "Open file:"
	m.inputValue = ""
	m.inputAction = func(typed string) tea.Cmd {
		path := strings.TrimSpace(typed)
		if path == "" {
			return func() tea.Msg {
				return statusMsg{"Path must not be empty"}
			}
		}
		// Expand ~ to home directory.
		if strings.HasPrefix(path, "~/") || path == "~" {
			home, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(home, path[1:])
			}
		}
		// Resolve to absolute path.
		abs, err := filepath.Abs(path)
		if err != nil {
			return func() tea.Msg {
				return statusMsg{fmt.Sprintf("Invalid path: %s", err)}
			}
		}
		// Validate file exists and is readable.
		info, err := os.Stat(abs)
		if err != nil {
			return func() tea.Msg {
				return statusMsg{fmt.Sprintf("Cannot open: %s", err)}
			}
		}
		if info.IsDir() {
			return func() tea.Msg {
				return statusMsg{"Cannot open a directory"}
			}
		}
		return func() tea.Msg {
			return importFileMsg{path: abs}
		}
	}
	return m, m.inputCur.Focus()
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

func (m Model) copyRecentEntry() (tea.Model, tea.Cmd) {
	if len(m.filteredRecent) == 0 {
		return m, nil
	}
	_, localIdx := m.cursorSection()
	if localIdx >= len(m.filteredRecent) {
		return m, nil
	}
	idx := m.filteredRecent[localIdx]
	entry := m.recentEntries[idx]

	return m, func() tea.Msg {
		var content, label string
		switch entry.Type {
		case recents.TypeStore:
			n, err := m.store.GetNote(entry.Notebook, entry.Name)
			if err != nil {
				return statusMsg{fmt.Sprintf("Could not load: %s", err)}
			}
			content = n.Content
			label = storage.DisplayName(entry.Name)
		case recents.TypeExternal:
			data, err := os.ReadFile(entry.Path)
			if err != nil {
				return statusMsg{fmt.Sprintf("Could not read: %s", err)}
			}
			content = string(data)
			label = filepath.Base(entry.Path)
		default:
			return statusMsg{"Unknown recent entry type"}
		}
		if err := clipboard.Copy(content); err != nil {
			return statusMsg{fmt.Sprintf("Could not copy: %s", err)}
		}
		return statusMsg{fmt.Sprintf("Copied %q to clipboard", label)}
	}
}

func (m Model) removeRecentEntry() (tea.Model, tea.Cmd) {
	if len(m.filteredRecent) == 0 {
		return m, nil
	}
	_, localIdx := m.cursorSection()
	if localIdx >= len(m.filteredRecent) {
		return m, nil
	}
	idx := m.filteredRecent[localIdx]
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

func (m Model) handleThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.themeMode = false
		return m, nil

	case "up":
		if m.uiThemeCursor > 0 {
			m.uiThemeCursor--
			presets := theme.Presets()
			if len(presets) > 0 {
				m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
			}
		}
		return m, nil

	case "down":
		presets := theme.Presets()
		if m.uiThemeCursor < len(presets)-1 {
			m.uiThemeCursor++
			m.uiThemePreview = m.renderUIThemePreview(presets[m.uiThemeCursor].Name)
		}
		return m, nil

	case "enter":
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

	default:
		if msg.Text == "t" {
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

	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent)).Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	// Left pane: UI theme preset list.
	listWidth := 30
	var left strings.Builder
	left.WriteString("  " + accent.Render("UI Theme") + "\n")
	left.WriteString("  " + dim.Render("─────────────────") + "\n")

	presets := theme.Presets()
	for i, p := range presets {
		if i == m.uiThemeCursor {
			sel := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
			left.WriteString(fmt.Sprintf("  %s %s\n", sel.Render("\u25cf"), sel.Render(p.Name)))
		} else {
			left.WriteString(fmt.Sprintf("    %s\n", p.Name))
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
	statusHint := dim.Render("↑/↓ navigate · Enter apply · Esc cancel")

	full := rendered + "\n" + statusHint

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, full)
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter)

	if m.level == 0 {
		if query != "" {
			// Filter notebooks by name.
			m.filteredRecent = nil
			m.filtered = nil
			for i, nb := range m.notebooks {
				if strings.Contains(strings.ToLower(storage.DisplayName(nb.name)), query) {
					m.filtered = append(m.filtered, i)
				}
			}

			// Filter notes by title (lazy-load names on first search).
			if m.allNoteNames == nil {
				m.allNoteNames = m.loadNoteNames()
			}
			m.searchResults = nil
			for _, r := range m.allNoteNames {
				if strings.Contains(strings.ToLower(storage.DisplayName(r.note)), query) {
					m.searchResults = append(m.searchResults, r)
				}
			}

			total := m.totalL0Items()
			if m.cursor >= total {
				m.cursor = total - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			return
		}

		// No query — normal browsing mode.
		m.searchResults = nil
		m.filteredRecent = nil
		for i := range m.recentEntries {
			m.filteredRecent = append(m.filteredRecent, i)
		}
		m.filtered = nil
		for i := range m.notebooks {
			m.filtered = append(m.filtered, i)
		}
		total := m.totalL0Items()
		if m.cursor >= total {
			m.cursor = total - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return
	}

	// L1: title-only filter.
	m.filtered = nil
	for i, n := range m.notes {
		if query == "" || strings.Contains(strings.ToLower(storage.DisplayName(n.Name)), query) {
			m.filtered = append(m.filtered, i)
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// loadNoteNames reads all note names from all notebooks (no content).
func (m *Model) loadNoteNames() []l0SearchResult {
	var results []l0SearchResult
	for _, nb := range m.notebooks {
		dir := m.store.NotebookDir(nb.name)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			results = append(results, l0SearchResult{
				notebook: nb.name,
				note:     name,
			})
		}
	}
	return results
}

func (m *Model) resetFilter() {
	m.filter = ""
	m.filtering = false
	m.filtered = nil
	m.filteredRecent = nil
	m.searchResults = nil

	if m.level == 0 {
		m.filteredRecent = make([]int, len(m.recentEntries))
		for i := range m.recentEntries {
			m.filteredRecent[i] = i
		}
		m.filtered = make([]int, len(m.notebooks))
		for i := range m.notebooks {
			m.filtered[i] = i
		}
		// Re-apply deferred cursor positioning every time data loads,
		// so whichever async source arrives last gets the final say.
		if m.restoreSel != nil {
			for i, fi := range m.filteredRecent {
				e := m.recentEntries[fi]
				if m.restoreSel.FilePath != "" {
					if e.Path == m.restoreSel.FilePath {
						m.cursor = i
						break
					}
				} else if e.Notebook == m.restoreSel.Book && e.Name == m.restoreSel.Note {
					m.cursor = i
					break
				}
			}
		}
		if m.selectAfterReload != "" && len(m.notebooks) > 0 {
			for i, fi := range m.filtered {
				if m.notebooks[fi].name == m.selectAfterReload {
					m.cursor = len(m.filteredRecent) + i
					break
				}
			}
		}
		total := m.totalL0Items()
		if m.cursor >= total {
			m.cursor = total - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return
	}

	m.filtered = make([]int, len(m.notes))
	for i := range m.notes {
		m.filtered[i] = i
	}

	// Restore cursor to previously-opened note.
	if m.restoreSel != nil && m.restoreSel.Note != "" && len(m.notes) > 0 {
		for i, fi := range m.filtered {
			if m.notes[fi].Name == m.restoreSel.Note {
				m.cursor = i
				break
			}
		}
	}
	// Re-apply deferred cursor positioning for note list.
	if m.selectAfterReload != "" && len(m.notes) > 0 {
		for i, fi := range m.filtered {
			if m.notes[fi].Name == m.selectAfterReload {
				m.cursor = i
				break
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

func (m Model) totalL0Items() int {
	return len(m.filteredRecent) + len(m.filtered) + len(m.searchResults)
}

func (m Model) cursorSection() (string, int) {
	if m.cursor < len(m.filteredRecent) {
		return "recent", m.cursor
	}
	nbStart := len(m.filteredRecent)
	if m.cursor < nbStart+len(m.filtered) {
		return "notebook", m.cursor - nbStart
	}
	return "search", m.cursor - nbStart - len(m.filtered)
}

func (m Model) inRecentsSection() bool {
	return len(m.filteredRecent) > 0 && m.cursor < len(m.filteredRecent)
}

// tabNextSection cycles the cursor to the first item of the next L0 section.
// Sections (recents, notebooks, notes) are skipped if empty.
func (m *Model) tabNextSection() {
	if m.level != 0 {
		return
	}

	// Build the list of section start indices (only non-empty sections).
	type section struct{ start int }
	var sections []section
	if len(m.filteredRecent) > 0 {
		sections = append(sections, section{0})
	}
	if len(m.filtered) > 0 {
		sections = append(sections, section{len(m.filteredRecent)})
	}
	if len(m.searchResults) > 0 {
		sections = append(sections, section{len(m.filteredRecent) + len(m.filtered)})
	}
	if len(sections) < 2 {
		return // nothing to cycle
	}

	// Find which section the cursor is currently in, then jump to the next.
	for i, s := range sections {
		next := sections[(i+1)%len(sections)]
		if i == len(sections)-1 || m.cursor < sections[i+1].start {
			m.cursor = next.start
			return
		}
		_ = s
	}
}

func (m Model) listLen() int {
	if m.level == 0 {
		return m.totalL0Items()
	}
	return len(m.filtered)
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.level == 0 {
		section, localIdx := m.cursorSection()

		// Flat search results mode.
		if section == "search" {
			if localIdx >= len(m.searchResults) {
				return m, nil
			}
			r := m.searchResults[localIdx]
			m.selected = &Selection{
				Book: r.notebook,
				Note: r.note,
			}
			return m, tea.Quit
		}

		if section == "recent" {
			if len(m.filteredRecent) == 0 {
				return m, nil
			}
			if localIdx >= len(m.filteredRecent) {
				return m, nil
			}
			idx := m.filteredRecent[localIdx]
			entry := m.recentEntries[idx]
			switch entry.Type {
			case recents.TypeStore:
				m.selected = &Selection{
					Book:       entry.Notebook,
					Note:       entry.Name,
					FromRecent: true,
				}
			case recents.TypeExternal:
				m.selected = &Selection{
					FilePath:   entry.Path,
					FromRecent: true,
				}
			}
			return m, tea.Quit
		}

		// Notebook section.
		if len(m.filtered) == 0 {
			return m, nil
		}
		if localIdx >= len(m.filtered) {
			return m, nil
		}
		idx := m.filtered[localIdx]
		m.currentBook = m.notebooks[idx].name
		m.level = 1
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		m.notes = nil
		m.filtered = nil
		m.filteredRecent = nil
		m.searchResults = nil
		return m, m.loadNotes(m.currentBook)
	}

	// Level 1.
	if len(m.filtered) == 0 {
		return m, nil
	}
	if m.cursor >= len(m.filtered) {
		return m, nil
	}

	idx := m.filtered[m.cursor]
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
		m.selectAfterReload = m.currentBook
		m.cursor = 0
		m.filter = ""
		m.filtering = false
		m.notebooks = nil
		m.filtered = nil
		m.filteredRecent = nil
		return m, tea.Batch(m.loadNotebooks(), m.loadRecents())
	}
	m.quitting = true
	return m, tea.Quit
}

// renderSettingsView builds the full-screen settings view matching the
// browser's list layout with section header, bullet selection, and status bar.
// renderSettingsFooter builds the settings panel as a footer popup.
func (m Model) renderSettingsFooter() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	th := theme.Current()
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent))
	dim := lipgloss.NewStyle().Faint(true)

	cfg := m.settingsCfg
	defs := settingDefs()

	border := dim.Render(strings.Repeat("\u2500", width))

	var b strings.Builder
	b.WriteString(border + "\n")

	maxLabel := 0
	for _, d := range defs {
		if len(d.label) > maxLabel {
			maxLabel = len(d.label)
		}
	}

	for i, d := range defs {
		val, _ := config.Get(cfg, d.key)
		displayVal := settingDisplayValue(d, val)

		if d.kind == settingCycle && len(d.options) > 1 {
			displayVal = "◂ " + displayVal + " ▸"
		}

		label := d.label
		pad := strings.Repeat(" ", maxLabel-len(label)+4)

		if i == m.settingsCursor {
			bullet := accentStyle.Render("\u25CF") + " "
			b.WriteString(bullet + accentStyle.Render(label) + pad + accentStyle.Render(displayVal) + "\n")
		} else {
			b.WriteString("  " + label + pad + dim.Render(displayVal) + "\n")
		}
	}

	return b.String()
}

// renderHelpFooter builds the help footer panel with keybind reference.
func (m Model) renderHelpFooter() string {
	return ui.RenderHelpFooter([]ui.HelpBinding{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Open"},
		{Key: "Esc", Desc: "Back / quit"},
		{Key: "Tab", Desc: "Jump section"},
		{Key: "/", Desc: "Search"},
		{Key: "n", Desc: "New"},
		{Key: "d", Desc: "Delete"},
		{Key: "r", Desc: "Rename"},
		{Key: "c", Desc: "Copy note"},
		{Key: "i", Desc: "Import file"},
		{Key: "t", Desc: "Theme"},
		{Key: "p", Desc: "Preview"},
		{Key: ",", Desc: "Settings"},
		{Key: "?", Desc: "Close"},
	}, m.width)
}

// currentHintID returns the hint ID relevant to the current browser state,
// or "" if no hint applies.
func (m Model) currentHintID() string {
	if m.level == 0 && !m.filtering {
		return "browser.theme"
	}
	return ""
}

// statusBarHeight returns the number of terminal lines the rendered status
// bar will occupy. When the bar text is wider than the terminal, lipgloss
// wraps it onto multiple lines; this helper accounts for that.
func (m Model) statusBarHeight() int {
	rendered := m.renderStatusBar()
	return strings.Count(rendered, "\n") + 1
}

// View implements tea.Model.
func (m Model) View() tea.View {
	if m.quitting {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	var content string

	if m.themeMode {
		content = m.renderThemeOverlay()
	} else if m.err != nil {
		content = fmt.Sprintf("\n  Error: %v\n", m.err)
	} else {
		var b strings.Builder

		// Breadcrumb / path (hidden at L0 — section headers serve as titles).
		breadcrumb := m.renderBreadcrumb()
		if breadcrumb != "" {
			b.WriteString(breadcrumb)
			b.WriteString("\n")
		}

		// Footer panel (help or settings).
		var footer string
		if m.showSettings {
			footer = m.renderSettingsFooter()
		} else if m.showHelp {
			footer = m.renderHelpFooter()
		}

		// Content area.
		headerLines := 1 // breadcrumb line
		if breadcrumb == "" {
			headerLines = 0
		}
		footerH := 1 // gap line before status bar
		if footer != "" {
			footerH = strings.Count(footer, "\n")
		}
		contentHeight := m.height - headerLines - footerH - m.statusBarHeight()
		if contentHeight < 1 {
			contentHeight = 1
		}

		c := m.renderContent(contentHeight)
		b.WriteString(c)

		// Pad to push status bar to bottom.
		contentLines := strings.Count(c, "\n")
		if contentLines < contentHeight {
			for i := 0; i < contentHeight-contentLines; i++ {
				b.WriteString("\n")
			}
		}

		// Footer and status bar.
		if footer != "" {
			b.WriteString(footer)
		} else {
			b.WriteString("\n")
		}
		b.WriteString(m.renderStatusBar())
		content = b.String()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderBreadcrumb() string {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	if m.level == 0 {
		return ""
	}
	return fmt.Sprintf(" %s", style.Render(fmt.Sprintf("notebook \u203A %s", storage.DisplayName(m.currentBook))))
}

func (m Model) renderContent(maxLines int) string {
	if m.level == 0 {
		return m.renderUnifiedList(maxLines)
	}
	return m.renderNoteList(maxLines)
}

func (m Model) renderUnifiedList(maxLines int) string {
	type vrow struct {
		text      string
		dataIndex int // combined cursor index, or -1 for header/separator
	}

	var rows []vrow
	cursorRow := 0
	isSearching := m.filtering && m.filter != ""

	hasRecents := len(m.filteredRecent) > 0
	hasNotebooks := len(m.filtered) > 0
	hasNotes := len(m.searchResults) > 0

	// Recent section (hidden during search).
	if hasRecents && !isSearching {
		rows = append(rows, vrow{text: m.renderSectionHeader("Recent"), dataIndex: -1})
		for i, fi := range m.filteredRecent {
			combinedIdx := i
			selected := combinedIdx == m.cursor
			if selected {
				cursorRow = len(rows)
			}
			line := m.formatRecentLine(m.recentEntries[fi], selected)
			rows = append(rows, vrow{text: line, dataIndex: combinedIdx})
		}
		rows = append(rows, vrow{text: "", dataIndex: -1})
	}

	// Notebooks section.
	rows = append(rows, vrow{text: m.renderSectionHeader("Notebooks"), dataIndex: -1})

	if hasNotebooks {
		for i, fi := range m.filtered {
			combinedIdx := len(m.filteredRecent) + i
			selected := combinedIdx == m.cursor
			if selected {
				cursorRow = len(rows)
			}
			line := m.formatNotebookLine(m.notebooks[fi], selected)
			rows = append(rows, vrow{text: line, dataIndex: combinedIdx})
		}
	} else if !m.filtering {
		// Empty state when not filtering.
		var b strings.Builder
		for i, row := range rows {
			b.WriteString(row.text)
			if i < len(rows)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
		remainingHeight := maxLines - len(rows) - 1
		if remainingHeight < 1 {
			remainingHeight = 1
		}
		b.WriteString(m.renderEmptyNotebooks(remainingHeight))
		return b.String()
	} else if !hasNotes {
		rows = append(rows, vrow{text: "  No matches", dataIndex: -1})
	}

	// Notes section (only during search).
	if isSearching && hasNotes {
		rows = append(rows, vrow{text: "", dataIndex: -1})
		rows = append(rows, vrow{text: m.renderSectionHeader("Notes"), dataIndex: -1})

		searchBase := len(m.filteredRecent) + len(m.filtered)
		for i, r := range m.searchResults {
			combinedIdx := searchBase + i
			selected := combinedIdx == m.cursor
			if selected {
				cursorRow = len(rows)
			}
			line := m.formatL0SearchResult(r, selected)
			rows = append(rows, vrow{text: line, dataIndex: combinedIdx})
		}
	}

	// Scroll window.
	totalRows := len(rows)
	start := 0
	if totalRows > maxLines {
		start = cursorRow - maxLines/2
		if start < 0 {
			start = 0
		}
		if start+maxLines > totalRows {
			start = totalRows - maxLines
		}
	}
	end := start + maxLines
	if end > totalRows {
		end = totalRows
	}

	var b strings.Builder
	for vi, row := range rows[start:end] {
		b.WriteString(row.text)
		if vi < end-start-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderSectionHeader(title string) string {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	return " " + style.Render(title)
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

func (m Model) formatRecentLine(e recents.Entry, selected bool) string {
	bullet := "  "
	label := recentEntryLabel(e)
	// Recents don't have the count column, so the name can use that space.
	const recentNameMax = nameColMax + 4 + 10 // 38
	display := truncAt(label, recentNameMax)
	timeStr := format.RelativeTime(e.LastEdited)

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		display = nameStyle.Render(truncAt(label, recentNameMax))
	}

	const timeColMax = 8
	const fixedCols = 2 + recentNameMax + 4 + timeColMax + 4

	line := fmt.Sprintf("%s%s    %s",
		padRight(bullet, 2),
		padRight(display, recentNameMax),
		padRight(timeStr, timeColMax),
	)

	if m.showPreview {
		if remain := m.width - fixedCols; remain > 0 {
			content := m.recentContent(e)
			if content != "" {
				preview := notePreview(content, remain)
				if preview != "" {
					dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Muted))
					line += "    " + dim.Render(preview)
				}
			}
		}
	}

	return line
}

// formatL0SearchResult renders a single result in the flat L0 search list.
func (m Model) formatL0SearchResult(r l0SearchResult, selected bool) string {
	bullet := "  "
	noteName := truncAt(storage.DisplayName(r.note), nameColMax)
	bookName := storage.DisplayName(r.notebook)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Muted))

	var styledName string
	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		styledName = highlightMatch(noteName, m.filter, accentStyle)
	} else {
		styledName = highlightMatch(noteName, m.filter, lipgloss.NewStyle())
	}

	display := styledName + "  " + dim.Render(bookName)
	const searchNameMax = nameColMax + 4 + 10 // 38

	return fmt.Sprintf("%s%s",
		padRight(bullet, 2),
		padRight(display, searchNameMax),
	)
}

// recentContent returns the text content for a recent entry (best-effort).
func (m Model) recentContent(e recents.Entry) string {
	switch e.Type {
	case recents.TypeStore:
		n, err := m.store.GetNote(e.Notebook, e.Name)
		if err == nil {
			return n.Content
		}
	case recents.TypeExternal:
		data, err := os.ReadFile(e.Path)
		if err == nil {
			return string(data)
		}
	}
	return ""
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
	display := truncName(storage.DisplayName(nb.name))
	name := display
	countStr := pluralize(nb.noteCount, "note", "notes")

	if selected {
		bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		bullet = bulletStyle.Render("\u25CF") + " "
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		if m.filtering && m.filter != "" {
			name = highlightMatch(display, m.filter, accentStyle)
		} else {
			name = accentStyle.Render(display)
		}
	} else if m.filtering && m.filter != "" {
		name = highlightMatch(display, m.filter, lipgloss.NewStyle())
	}

	return fmt.Sprintf("%s%s    %-10s    %s",
		padRight(bullet, 2),
		padRight(name, nameColMax),
		countStr,
		nb.modTime,
	)
}

func (m Model) formatNoteLine(n model.Note, selected bool) string {
	bullet := "  "
	display := truncName(storage.DisplayName(n.Name))
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
		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Accent))
		if m.filtering && m.filter != "" {
			name = highlightMatch(display, m.filter, accentStyle)
		} else {
			name = accentStyle.Render(display)
		}
	} else if m.filtering && m.filter != "" {
		name = highlightMatch(display, m.filter, lipgloss.NewStyle())
	}

	const timeColMax = 8
	const fixedCols = 2 + nameColMax + 4 + 10 + 4 + timeColMax + 4 // bullet+name+gap+size+gap+time+gap

	line := fmt.Sprintf("%s%s    %-10s    %s",
		padRight(bullet, 2),
		padRight(name, nameColMax),
		sizeStr,
		padRight(timeStr, timeColMax),
	)

	if m.showPreview {
		if remain := m.width - fixedCols; remain > 0 && n.Content != "" {
			preview := notePreview(n.Content, remain)
			if preview != "" {
				dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Current().Muted))
				line += "    " + dim.Render(preview)
			}
		}
	}

	return line
}

// notePreview returns a one-line plain-text snippet of a note's content,
// truncated to maxWidth runes.
func notePreview(content string, maxWidth int) string {
	snippet := strings.Join(strings.Fields(content), " ")
	runes := []rune(snippet)
	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}
	return string(runes)
}

const nameColMax = 24

// truncName truncates a display name to nameColMax, adding "..." if needed.
func truncName(name string) string {
	return truncAt(name, nameColMax)
}

// truncAt truncates a string to max characters, adding "..." if needed.
func truncAt(name string, max int) string {
	if len(name) <= max {
		return name
	}
	return name[:max-3] + "..."
}

// highlightMatch renders a name with the matching query substring underlined.
// baseStyle is applied to non-matching characters; matching characters get
// the same style plus underline.
func highlightMatch(name, query string, baseStyle lipgloss.Style) string {
	if query == "" {
		return baseStyle.Render(name)
	}
	lower := strings.ToLower(name)
	lowerQ := strings.ToLower(query)
	idx := strings.Index(lower, lowerQ)
	if idx < 0 {
		return baseStyle.Render(name)
	}

	matchStyle := baseStyle.Underline(true)
	before := name[:idx]
	matched := name[idx : idx+len(query)]
	after := name[idx+len(query):]

	var b strings.Builder
	if before != "" {
		b.WriteString(baseStyle.Render(before))
	}
	b.WriteString(matchStyle.Render(matched))
	if after != "" {
		b.WriteString(baseStyle.Render(after))
	}
	return b.String()
}

func (m Model) renderEmptyNotebooks(maxHeight int) string {
	return m.renderStarEmpty("No notebooks yet.", "Press n to create one.", maxHeight)
}

func (m Model) renderEmptyNotes() string {
	return m.renderStarEmpty("No notes in "+storage.DisplayName(m.currentBook)+".", "Press n to create one.", 0)
}

// renderStarEmpty renders a centered empty state with a star/dot pattern.
// If maxHeight > 0, it constrains the height to that value.
func (m Model) renderStarEmpty(title, subtitle string, maxHeight int) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := maxHeight
	if h <= 0 {
		h = m.height - 4
	}
	if h < 1 {
		h = 1
	}

	dim := lipgloss.NewStyle().Faint(true)
	accent := lipgloss.NewStyle().Faint(true)

	stars := []string{
		"        *              .        ",
		" .                *             ",
		"             .            *     ",
		"    *                           ",
	}
	bottom := []string{
		"                       *        ",
		"   *            .               ",
		"          *               .     ",
		" .              *              .",
	}

	var b strings.Builder
	for _, line := range stars {
		b.WriteString(dim.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(accent.Render(title))
	b.WriteString("\n")
	b.WriteString(dim.Render(subtitle))
	b.WriteString("\n\n")
	for i, line := range bottom {
		b.WriteString(dim.Render(line))
		if i < len(bottom)-1 {
			b.WriteString("\n")
		}
	}

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, b.String())
}

func (m Model) renderStatusBar() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	if m.inputMode && m.showSettings {
		// Settings text edit: use input bar without border (settings footer provides its own).
		input := ui.NewTextInput(m.inputValue)
		input.SetCursor(m.inputCursor)
		return input.RenderStatus(ui.TextInputProps{
			Prompt:        m.inputPrompt,
			Hints:         "Enter confirm \u00B7 Esc cancel",
			Width:         width,
			CursorVisible: !m.inputCur.IsBlinked,
		})
	}
	if m.inputMode {
		input := ui.NewTextInput(m.inputValue)
		input.SetCursor(m.inputCursor)
		return input.RenderFooter(ui.TextInputProps{
			Prompt:        m.inputPrompt,
			Hints:         "Enter confirm \u00B7 Esc cancel",
			Width:         width,
			CursorVisible: !m.inputCur.IsBlinked,
		})
	}

	if m.filtering {
		input := ui.NewTextInput(m.filter)
		input.SetCursor(m.filterCursor)
		return input.RenderFooter(ui.TextInputProps{
			Prompt:        "Search:",
			Hints:         "Esc clear \u00B7 Enter select",
			Width:         width,
			CursorVisible: !m.inputCur.IsBlinked,
		})
	}

	left := " "
	var hint string
	var right string

	if m.showSettings {
		defs := settingDefs()
		if m.settingsCursor < len(defs) {
			if desc := defs[m.settingsCursor].desc; desc != "" {
				left = "  " + desc
			}
			switch defs[m.settingsCursor].kind {
			case settingBool:
				right = "Enter toggle \u00B7 Esc close"
			case settingCycle:
				right = "←/→ cycle \u00B7 Esc close"
			case settingText:
				right = "Enter edit \u00B7 Esc close"
			}
		}
	} else if m.showHelp {
		right = "? close"
	} else if m.statusText != "" {
		left = "  " + m.statusText
	} else if m.level == 0 {
		if !m.dismissedHints["browser.theme"] {
			hint = "press t to change theme!  [h]ide"
		}
		if len(m.filteredRecent) > 0 && len(m.filtered) > 0 {
			right = "Tab jump \u00B7 / search \u00B7 ? help"
		} else {
			right = "/ search \u00B7 ? help"
		}
	} else {
		right = "/ search \u00B7 Esc back \u00B7 ? help"
	}

	bar := format.StatusBar(left, hint, right, width)
	return lipgloss.NewStyle().Faint(true).Width(width).Render(bar)
}

// pluralize returns "1 note" or "3 notes" style strings.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
