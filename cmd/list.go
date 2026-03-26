package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notebooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation in issue #5
		fmt.Println("  No books yet.")
		fmt.Println()
		fmt.Println("  Create one with:  notebook new \"My First Book\"")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
