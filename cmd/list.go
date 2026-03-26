package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notebooks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
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
				timeStr = relativeTime(modTime)
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
