package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oobagi/notebook/internal/browser"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse notebooks and notes in a fullscreen TUI",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowser()
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}

// runBrowser launches the TUI browser in a loop. When the user selects
// a note, the browser quits, the editor launches, and the browser
// re-enters after the editor exits.
func runBrowser() error {
	for {
		m := browser.New(browser.Config{
			Store: store,
			EditNote: func(book, note string) error {
				return editNote(nil, book, note)
			},
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
		if err := editNote(os.Stderr, sel.Book, sel.Note); err != nil {
			return fmt.Errorf("edit note: %w", err)
		}

		// Loop back to re-enter the browser.
	}
}
