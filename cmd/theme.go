package cmd

import (
	"fmt"
	"strings"

	"github.com/oobagi/notebook/internal/config"
	"github.com/oobagi/notebook/internal/theme"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme [name]",
	Short: "Show or set the color theme",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		if len(args) == 0 {
			// List available themes and show current.
			presets := theme.Presets()
			current := theme.Current()
			for _, p := range presets {
				marker := "  "
				if p.Name == current.Name {
					marker = "* "
				}
				fmt.Fprintf(w, "%s%s\n", marker, p.Name)
			}
			return nil
		}

		name := args[0]
		if name != "auto" {
			if _, ok := theme.PresetByName(name); !ok {
				presets := theme.Presets()
				var names []string
				for _, p := range presets {
					names = append(names, p.Name)
				}
				return fmt.Errorf("unknown theme %q (choose: auto, %s)", name, strings.Join(names, ", "))
			}
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
