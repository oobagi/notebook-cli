package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := store.DeleteNotebook(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
