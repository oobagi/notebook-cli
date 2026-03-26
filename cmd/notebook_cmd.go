package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// notebookSubcommands maps verb names to handler functions.
// Each handler receives the book name and remaining args.
var notebookSubcommands = map[string]func(book string, args []string) error{
	"new": runNewNote,
	// "list", "delete", "search" handlers added in their respective files
}

// registerNotebookVerb registers a verb that can be used in the
// `notebook <book> <verb> [args]` pattern.
func registerNotebookVerb(verb string, fn func(book string, args []string) error) {
	notebookSubcommands[verb] = fn
}

// knownSubcommands returns the set of top-level subcommand names
// so the dispatcher can distinguish `notebook list` from `notebook mybook`.
func knownSubcommands() map[string]bool {
	names := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		names[c.Name()] = true
		for _, alias := range c.Aliases {
			names[alias] = true
		}
	}
	// Also include built-in names
	names["help"] = true
	names["completion"] = true
	return names
}

func init() {
	// Override the root command's RunE to handle dynamic book routing.
	// When args don't match a known subcommand, treat the first arg as a
	// notebook name and dispatch based on subsequent args.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		book := args[0]
		known := knownSubcommands()
		if known[book] {
			// This shouldn't happen — Cobra routes known subcommands.
			// But just in case, show help.
			return cmd.Help()
		}

		if !validName(book) {
			return invalidNameError(book)
		}

		// notebook <book> — with no verb, view/browse the notebook
		if len(args) == 1 {
			// For now, show notes in the notebook (list behavior)
			return runNotebookList(book)
		}

		// notebook <book> <verb> [args...]
		verb := args[1]
		if handler, ok := notebookSubcommands[verb]; ok {
			return handler(book, args[2:])
		}

		// notebook <book> <note> [verb] — two-level resource
		note := args[1]
		if !validName(note) {
			return invalidNameError(note)
		}

		if len(args) == 2 {
			// notebook <book> <note> — view the note
			return runViewNote(book, note)
		}

		// notebook <book> <note> <verb>
		noteVerb := args[2]
		switch noteVerb {
		case "edit":
			return fmt.Errorf("editor not yet implemented (coming in Phase 3)")
		case "delete", "rm":
			return runDeleteNote(book, note)
		case "copy":
			return fmt.Errorf("clipboard copy not yet implemented (coming in Phase 5)")
		default:
			return fmt.Errorf("unknown command %q — try: edit, delete, copy", noteVerb)
		}
	}

	// Allow arbitrary args on root for the dynamic routing
	rootCmd.Args = cobra.ArbitraryArgs

	// Disable Cobra's built-in subcommand suggestions for the root
	// so it doesn't confuse book names with typos of subcommands
	rootCmd.DisableSuggestions = true
}

// runNotebookList lists notes in a notebook. Stub until issue #5.
func runNotebookList(book string) error {
	store, err := getStore()
	if err != nil {
		return err
	}
	if !store.NotebookExists(book) {
		return fmt.Errorf("notebook %q not found", book)
	}
	notes, err := store.ListNotes(book)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Println("  No notes yet.")
		fmt.Println()
		fmt.Printf("  Create one with:  notebook %s new \"My First Note\"\n", book)
		return nil
	}
	for _, n := range notes {
		fmt.Printf("  %s\n", n)
	}
	return nil
}

// runViewNote views a note. Stub until issue #6.
func runViewNote(book, note string) error {
	store, err := getStore()
	if err != nil {
		return err
	}
	n, err := store.GetNote(book, note)
	if err != nil {
		return fmt.Errorf("note %q not found in %q", note, book)
	}
	fmt.Print(n.Content)
	if len(n.Content) > 0 && !strings.HasSuffix(n.Content, "\n") {
		fmt.Println()
	}
	return nil
}

// runDeleteNote deletes a note. Stub until issue #7.
func runDeleteNote(book, note string) error {
	store, err := getStore()
	if err != nil {
		return err
	}
	if !store.NoteExists(book, note) {
		return fmt.Errorf("note %q not found in %q", note, book)
	}
	if err := store.DeleteNote(book, note); err != nil {
		return err
	}
	fmt.Printf("  ✓ Deleted \"%s\" from %s\n", note, book)
	return nil
}
