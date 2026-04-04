package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oobagi/notebook-cli/internal/config"
	"github.com/oobagi/notebook-cli/internal/storage"
	"github.com/oobagi/notebook-cli/internal/theme"
	"github.com/spf13/cobra"
)

var (
	store   *storage.Store
	dirFlag string
	cfg     config.Config
)

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       Version,
	Use:           "notebook",
	Short:         "A dead-simple CLI note manager with live markdown preview",
	Long:          "Notebook is a CLI tool for managing markdown notes organized into notebooks, with a live-preview editor mode right in your terminal.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Resolve the active theme from config (or auto-detect).
		theme.SetTheme(theme.FromName(cfg.Theme))

		root := dirFlag
		if root == "" {
			root = cfg.StorageDir
		}

		// Expand ~ to home directory.
		if strings.HasPrefix(root, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve home directory: %w", err)
			}
			root = filepath.Join(home, root[2:])
		}

		store = storage.NewStore(root)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runBrowser()
		}

		// If the first arg looks like a file path, open it directly
		// instead of treating it as a notebook name.
		if isFilePath(args[0]) {
			return openFile(args[0])
		}

		return dispatch(cmd, args)
	},
	// Accept arbitrary args so the dispatcher can handle noun-verb routing.
	Args: cobra.ArbitraryArgs,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dirFlag, "dir", "", "root directory for notebook storage (default ~/.notebook)")
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.SetVersionTemplate("notebook {{.Version}}\n")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
