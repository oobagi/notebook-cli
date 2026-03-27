package cmd

import (
	"fmt"

	"github.com/oobagi/notebook/internal/config"
	"github.com/oobagi/notebook/internal/picker"
	"github.com/oobagi/notebook/internal/theme"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme [dark|light|auto]",
	Short: "Show or set the color theme",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		if len(args) == 0 {
			// No arg: show a picker to choose a theme.
			picked, err := picker.Run(picker.Config{
				Title: "Pick a theme",
				Items: []string{"auto", "dark", "light"},
			})
			if err != nil {
				return err
			}
			if picked == "" {
				printInfo(w, "Cancelled")
				return nil
			}
			args = []string{picked}
		}

		// Set theme.
		name := args[0]
		if name != "dark" && name != "light" && name != "auto" {
			return fmt.Errorf("unknown theme %q (choose: auto, dark, light)", name)
		}

		current, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := config.Set(&current, "theme", name); err != nil {
			return err
		}

		if err := config.Save(current); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		theme.SetTheme(theme.FromName(name))
		printSuccess(w, fmt.Sprintf("Theme set to %q", name))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(themeCmd)
}
