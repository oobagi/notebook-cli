package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var forceFlag bool

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		name := args[0]

		// Check if notebook exists.
		dir := store.NotebookDir(name)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			printError(w, fmt.Sprintf("Notebook %q not found", name))
			return nil
		}

		// Count notes for the confirmation message.
		count, err := store.NoteCount(name)
		if err != nil {
			printError(w, err.Error())
			return nil
		}

		if !forceFlag {
			// Show confirmation prompt.
			if count > 0 {
				fmt.Fprintf(w, "  Delete %q and %s? This cannot be undone. [y/N] ",
					name, pluralize(count, "note", "notes"))
			} else {
				fmt.Fprintf(w, "  Delete %q? This cannot be undone. [y/N] ", name)
			}

			scanner := bufio.NewScanner(cmd.InOrStdin())
			if !scanner.Scan() {
				return nil
			}
			answer := strings.TrimSpace(scanner.Text())
			if answer != "y" && answer != "Y" {
				printInfo(w, "Cancelled")
				return nil
			}
		}

		if err := store.DeleteNotebook(name); err != nil {
			printError(w, err.Error())
			return nil
		}

		if count > 0 {
			printSuccess(w, fmt.Sprintf("Deleted %q and %s", name, pluralize(count, "note", "notes")))
		} else {
			printSuccess(w, fmt.Sprintf("Deleted %q", name))
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}
