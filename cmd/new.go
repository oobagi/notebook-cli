package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new notebook",
	Long: `Create a new notebook.

Examples:
  notebook new "Work"              Create a notebook called Work
  notebook new "Personal Journal"  Create a notebook with spaces in the name`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !validName(name) {
			return invalidNameError(name)
		}

		store, err := getStore()
		if err != nil {
			return err
		}

		if store.NotebookExists(name) {
			return fmt.Errorf("notebook %q already exists", name)
		}

		if err := store.CreateNotebook(name); err != nil {
			return err
		}

		fmt.Printf("  ✓ Created \"%s\"\n", name)
		return nil
	},
}

// newNoteCmd handles: notebook <book> new <note-title>
// This is registered dynamically by the notebook subcommand dispatcher.
func runNewNote(book string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: notebook %s new <note-title>", book)
	}
	name := args[0]
	if !validName(name) {
		return invalidNameError(name)
	}

	store, err := getStore()
	if err != nil {
		return err
	}

	if store.NoteExists(book, name) {
		return fmt.Errorf("note %q already exists in %q", name, book)
	}

	if err := store.CreateNote(book, name, ""); err != nil {
		return err
	}

	fmt.Printf("  ✓ Created \"%s\" in %s\n", name, book)
	return nil
}

func init() {
	rootCmd.AddCommand(newCmd)
}
