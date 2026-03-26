package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for notebook.

Bash:
  source <(notebook completion bash)

  # To load completions for each session, execute once:
  notebook completion bash > /etc/bash_completion.d/notebook

Zsh:
  source <(notebook completion zsh)

  # To load completions for each session, execute once:
  notebook completion zsh > "${fpath[1]}/_notebook"

Fish:
  notebook completion fish | source

  # To load completions for each session, execute once:
  notebook completion fish > ~/.config/fish/completions/notebook.fish`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	rootCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Need store initialized for completions.
		if store == nil {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			store = storage.NewStore(filepath.Join(home, ".notebook"))
		}

		if len(args) == 0 {
			// Complete notebook names + known subcommands.
			notebooks, err := store.ListNotebooks()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			completions := append(notebooks, "list", "new", "delete", "search", "config", "version", "completion")
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 1 {
			book := args[0]
			// Complete note names in the notebook + book-level verbs.
			notes, err := store.ListNotes(book)
			if err != nil {
				return []string{"list", "new", "delete", "search"}, cobra.ShellCompDirectiveNoFileComp
			}
			completions := make([]string, 0, len(notes)+4)
			for _, n := range notes {
				completions = append(completions, n.Name)
			}
			completions = append(completions, "list", "new", "delete", "search")
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 2 {
			// Complete note-level verbs.
			return []string{"edit", "delete", "copy"}, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
