package cmd

import (
	"fmt"

	"github.com/oobagi/notebook/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		displayValue := func(key, value string) {
			if value == "" {
				fmt.Fprintf(w, "  %-14s (default)\n", key)
			} else {
				fmt.Fprintf(w, "  %-14s %s\n", key, value)
			}
		}

		displayValue("storage_dir", cfg.StorageDir)
		displayValue("editor", cfg.Editor)
		displayValue("theme", cfg.Theme)
		displayValue("date_format", cfg.DateFormat)
		displayValue("glamour_style", cfg.GlamourStyle)
		displayValue("ui_theme", cfg.UITheme)

		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Config file: %s\n", config.Path())

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		key, value := args[0], args[1]

		if !config.ValidKeys[key] {
			return fmt.Errorf("unknown config key: %q (valid keys: storage_dir, editor, theme, date_format, glamour_style, ui_theme)", key)
		}

		// Load current config, apply change, save.
		current, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := config.Set(&current, key, value); err != nil {
			return err
		}

		if err := config.Save(current); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		printSuccess(w, fmt.Sprintf("Set %s to %q", key, value))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
