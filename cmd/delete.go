package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm"},
	Short:   "Delete a notebook",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation in issue #7
		fmt.Printf("  ✗ Not implemented yet\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
