package cmd

import (
	"fmt"
	"regexp"

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

// getStore creates a Store using the configured storage directory.
func getStore() (*storage.Store, error) {
	return storage.New(storageDir)
}

// validName checks that a name contains only safe characters.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 _-]*$`)

func validName(name string) bool {
	return validNamePattern.MatchString(name)
}

func invalidNameError(name string) error {
	return fmt.Errorf("invalid name %q — use letters, numbers, hyphens, underscores, or spaces", name)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
