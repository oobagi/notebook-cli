package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/oobagi/notebook/internal/recents"
	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recently edited notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		entries, err := recents.Load(recents.DefaultPath())
		if err != nil {
			return fmt.Errorf("load recents: %w", err)
		}

		// Prune stale entries whose files no longer exist.
		entries = recents.Prune(entries, store.Root)
		// Persist the pruned list (best-effort).
		_ = recents.Save(recents.DefaultPath(), entries)

		if len(entries) == 0 {
			fmt.Fprintln(w, "  No recent notes.")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  Notes appear here after you edit and save them.")
			return nil
		}

		var rows [][]string
		for _, e := range entries {
			var label, timeStr string
			switch e.Type {
			case "store":
				label = storage.DisplayName(e.Notebook) + " \u203A " + storage.DisplayName(e.Name)
			case "external":
				label = shortenHome(e.Path)
			default:
				continue
			}
			timeStr = relativeTime(e.LastEdited)
			rows = append(rows, []string{label, timeStr})
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

// shortenHome replaces the home directory prefix with ~/ for display.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func init() {
	recentCmd.AddCommand(recentClearCmd)
	rootCmd.AddCommand(recentCmd)
}
