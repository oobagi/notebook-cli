package cmd

import (
	"fmt"

	"github.com/oobagi/notebook-cli/internal/format"
	"github.com/oobagi/notebook-cli/internal/recents"
	"github.com/oobagi/notebook-cli/internal/storage"
	"github.com/spf13/cobra"
)

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recently edited notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		entries, err := recents.LoadPruned(store.Root)
		if err != nil {
			return fmt.Errorf("load recents: %w", err)
		}

		if len(entries) == 0 {
			fmt.Fprintln(w, "  No recent notes.")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  Notes appear here after you edit and save them.")
			return nil
		}

		var rows [][]string
		for _, e := range entries {
			var label string
			switch e.Type {
			case recents.TypeStore:
				label = storage.DisplayName(e.Notebook) + " \u203A " + storage.DisplayName(e.Name)
			case recents.TypeExternal:
				label = format.ShortenHome(e.Path)
			default:
				continue
			}
			rows = append(rows, []string{label, format.RelativeTime(e.LastEdited)})
		}

		for _, line := range alignColumns(rows) {
			fmt.Fprintln(w, line)
		}
		return nil
	},
}

var recentClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the recent notes list",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		if err := recents.Save(recents.DefaultPath(), nil); err != nil {
			return fmt.Errorf("clear recents: %w", err)
		}

		printSuccess(w, "Cleared recent notes")
		return nil
	},
}

func init() {
	recentCmd.AddCommand(recentClearCmd)
	rootCmd.AddCommand(recentCmd)
}
