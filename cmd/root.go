package cmd

import (
	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

var storageDir string

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Use:           "notebook",
	Short:         "A dead-simple CLI note manager with live markdown preview",
	Long:          "Notebook is a CLI tool for managing markdown notes organized into notebooks, with a live-preview editor mode right in your terminal.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&storageDir, "dir", storage.DefaultRoot(), "storage directory for notebooks")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
