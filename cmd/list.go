package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notebooks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		notebooks, err := store.ListNotebooks()
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		for _, name := range notebooks {
			fmt.Fprintf(w, "  %s\n", name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
