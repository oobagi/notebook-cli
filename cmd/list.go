package cmd

import (
	"fmt"

	"github.com/oobagi/notebook/internal/format"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [notebook]",
	Short: "List all notebooks, or notes in a notebook",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		// If a notebook name is given, list notes in that notebook.
		if len(args) == 1 {
			return listNotesInBook(w, args[0])
		}

		// Otherwise, list all notebooks.
		notebooks, err := store.ListNotebooks()
		if err != nil {
			return err
		}

		if len(notebooks) == 0 {
			fmt.Fprintln(w, "  No books yet.")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  Create one with:  notebook new \"My First Book\"")
			return nil
		}

		var rows [][]string
		for _, name := range notebooks {
			count, err := store.NoteCount(name)
			if err != nil {
				return err
			}
			modTime, err := store.NotebookModTime(name)
			if err != nil {
				return err
			}

			countStr := pluralize(count, "note", "notes")
			var timeStr string
			if modTime.IsZero() {
				timeStr = "empty"
			} else {
				timeStr = format.RelativeTime(modTime)
			}

			rows = append(rows, []string{name, countStr, timeStr})
		}

		for _, line := range alignColumns(rows) {
			fmt.Fprintln(w, line)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
