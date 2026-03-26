package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := store.CreateNotebook(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
