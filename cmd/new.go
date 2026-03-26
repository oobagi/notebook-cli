package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		name := args[0]

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
