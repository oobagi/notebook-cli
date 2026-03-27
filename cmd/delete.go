package cmd

import (
	"fmt"
	"os"

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
			printError(w, "Missing name. Try: notebook delete \"My Notebook\"")
			return nil
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
			var prompt string
			if count > 0 {
				prompt = fmt.Sprintf("  Delete %q and %s? This cannot be undone.", name, pluralize(count, "note", "notes"))
			} else {
				prompt = fmt.Sprintf("  Delete %q? This cannot be undone.", name)
			}
			if !confirmByName(w, cmd.InOrStdin(), prompt, name) {
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
