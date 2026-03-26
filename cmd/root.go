package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

var (
	store   *storage.Store
	dirFlag string
)

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Use:           "notebook",
	Short:         "A dead-simple CLI note manager with live markdown preview",
	Long:          "Notebook is a CLI tool for managing markdown notes organized into notebooks, with a live-preview editor mode right in your terminal.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		root := dirFlag
		if root == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve home directory: %w", err)
			}
			root = filepath.Join(home, ".notebook")
		}
		store = storage.NewStore(root)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return dispatch(cmd, args)
	},
	// Accept arbitrary args so the dispatcher can handle noun-verb routing.
	Args: cobra.ArbitraryArgs,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dirFlag, "dir", "", "root directory for notebook storage (default ~/.notebook)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
