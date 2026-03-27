package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oobagi/notebook/internal/browser"
)

// runBrowser launches the TUI browser in a loop. When the user selects
// a note, the browser quits, the editor launches, and the browser
// re-enters after the editor exits.
func runBrowser() error {
	var lastBook string
	for {
		m := browser.New(browser.Config{
			Store: store,
			EditNote: func(book, note string) error {
				return editNote(nil, book, note)
			},
			InitialBook: lastBook,
		})

		p := tea.NewProgram(m, tea.WithAltScreen())
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

		// Launch editor for the selected note.
		lastBook = sel.Book
		if err := editNote(os.Stderr, sel.Book, sel.Note); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		// Loop back to re-enter the browser.
	}
}
