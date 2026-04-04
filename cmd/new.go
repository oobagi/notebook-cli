package cmd

import (
	"fmt"
	"os"

	"github.com/oobagi/notebook-cli/internal/storage"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new notebook",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Missing name. Try: notebook new \"My Notebook\"")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments. Try: notebook new \"My Notebook\"")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		name := storage.Slugify(args[0])
		if name == "" {
			printError(w, "Notebook name can't be empty.")
			return nil
		}

		// Check if the notebook already exists before creating.
		dir := store.NotebookDir(name)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			printError(w, fmt.Sprintf("Notebook %q already exists", name))
			return nil
		}

		if err := store.CreateNotebook(name); err != nil {
			printError(w, err.Error())
			return nil
		}
		printSuccess(w, fmt.Sprintf("Created %q", name))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
