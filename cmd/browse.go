package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/browser"
	"github.com/oobagi/notebook-cli/internal/config"
)

// runBrowser launches the TUI browser in a loop. When the user selects
// a note, the browser quits, the editor launches, and the browser
// re-enters after the editor exits.
func runBrowser() error {
	// Seed the welcome note on first launch (no-op after the first time).
	if err := store.EnsureWelcome(); err != nil {
		return fmt.Errorf("seed welcome note: %w", err)
	}

	var lastBook string
	var lastSel *browser.Selection
	for {
		m := browser.New(browser.Config{
			Store:          store,
			InitialBook:    lastBook,
			RestoreSel:     lastSel,
			DismissedHints: config.LoadDismissedHints(),
		})

		p := tea.NewProgram(m)
		result, err := p.Run()
		if err != nil {
			return fmt.Errorf("run browser: %w", err)
		}

		final := result.(browser.Model)
		sel := final.Selected()
		if sel == nil {
			// User quit without selecting a note.
			return nil
		}

		lastSel = sel

		if sel.FromRecent {
			// Entered from recents — return to L0.
			lastBook = ""
		} else {
			// Entered from notebook list — return to L1 inside the book.
			lastBook = sel.Book
		}

		if sel.FilePath != "" {
			if err := openFile(sel.FilePath); err != nil {
				return fmt.Errorf("open file: %w", err)
			}
		} else {
			if err := editNote(os.Stderr, sel.Book, sel.Note); err != nil {
				return fmt.Errorf("edit note: %w", err)
			}
		}

		// Loop back to re-enter the browser.
	}
}
