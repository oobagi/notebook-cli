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
		// Implementation in issue #4
		fmt.Printf("  ✓ Created \"%s\"\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
