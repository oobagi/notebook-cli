package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all notebooks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation in issue #12
		fmt.Printf("  ✗ Not implemented yet\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
