package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/oobagi/notebook/internal/picker"
	"github.com/spf13/cobra"
)

var forceFlag bool

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a notebook",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// No arg provided: show a picker.
			notebooks, err := store.ListNotebooks()
			if err != nil {
				printError(w, err.Error())
				return nil
			}
			if len(notebooks) == 0 {
				printInfo(w, "No notebooks to delete.")
				return nil
			}
			picked, err := picker.Run(picker.Config{
				Title: "Pick a notebook to delete",
				Items: notebooks,
			})
			if err != nil {
				return err
			}
			if picked == "" {
				printInfo(w, "Cancelled")
				return nil
			}
			name = picked
		}

		// Check if notebook exists.
		dir := store.NotebookDir(name)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			printError(w, fmt.Sprintf("Notebook %q doesn't exist. See your notebooks with: notebook list", name))
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
