package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	Use:   "notebook",
	Short: "A dead-simple CLI note manager with live markdown preview",
	Long:  "Notebook is a CLI tool for managing markdown notes organized into notebooks, with a live-preview editor mode right in your terminal.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
