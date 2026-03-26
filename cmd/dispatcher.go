package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// dispatch handles the noun-verb routing for commands like:
//
//	notebook <book>                    -> list notes in book
//	notebook <book> list               -> list notes in book
//	notebook <book> new "Title"        -> create note in book
//	notebook <book> delete "Title"     -> delete note from book
//	notebook <book> search "query"     -> search within book (stub)
//	notebook <book> <note>             -> view note (stub)
//	notebook <book> <note> edit        -> edit note (stub)
//	notebook <book> <note> delete      -> delete note
//	notebook <book> <note> copy        -> copy note (stub)
func dispatch(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	book := args[0]
	rest := args[1:]

	// notebook <book> — no further args: list notes in the book
	if len(rest) == 0 {
		return listNotesInBook(w, book)
	}

	// notebook <book> <verb> [args...]
	second := rest[0]
	switch second {
	case "list":
		return listNotesInBook(w, book)
	case "new":
		if len(rest) < 2 {
			return fmt.Errorf("usage: notebook %s new <title>", book)
		}
		return createNoteInBook(w, book, rest[1])
	case "delete":
		if len(rest) < 2 {
			return fmt.Errorf("usage: notebook %s delete <title>", book)
		}
		return deleteNoteFromBook(w, book, rest[1])
	case "search":
		if len(rest) < 2 {
			return fmt.Errorf("usage: notebook %s search <query>", book)
		}
		return searchInBook(w, book, rest[1])
	}

	// second arg is not a book-level verb, so treat it as a note name
	note := second
	noteRest := rest[1:]

	// notebook <book> <note> — no verb: view the note
	if len(noteRest) == 0 {
		return viewNote(w, book, note)
	}

	// notebook <book> <note> <verb>
	verb := noteRest[0]
	trailing := noteRest[1:]
	switch verb {
	case "edit", "delete", "copy":
		if len(trailing) > 0 {
			return fmt.Errorf("unexpected arguments after %q", verb)
		}
	}
	switch verb {
	case "edit":
		return editNote(w, book, note)
	case "delete":
		return deleteNoteFromBook(w, book, note)
	case "copy":
		return copyNote(w, book, note)
	default:
		return fmt.Errorf("unknown command %q for note %q in %q", verb, note, book)
	}
}

// --- Book-scoped operations ---

func listNotesInBook(w io.Writer, book string) error {
	notes, err := store.ListNotes(book)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "does not exist") {
			printError(w, fmt.Sprintf("Notebook %q not found", book))
			return nil
		}
		return fmt.Errorf("list notes in %q: %w", book, err)
	}

	if len(notes) == 0 {
		fmt.Fprintln(w, "  No notes yet.")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Create one with:  notebook %s new \"My First Note\"\n", book)
		return nil
	}

	var rows [][]string
	for _, n := range notes {
		info, err := os.Stat(store.NotebookDir(book) + "/" + n.Name + ".md")
		if err != nil {
			return fmt.Errorf("stat note %q: %w", n.Name, err)
		}
		sizeStr := humanSize(info.Size())
		timeStr := relativeTime(info.ModTime())
		rows = append(rows, []string{n.Name, sizeStr, timeStr})
	}

	for _, line := range alignColumns(rows) {
		fmt.Fprintln(w, line)
	}
	return nil
}

func createNoteInBook(w io.Writer, book, title string) error {
	if err := store.CreateNote(book, title, ""); err != nil {
		// Check for "already exists" from the storage layer.
		if strings.Contains(err.Error(), "already exists") {
			printError(w, fmt.Sprintf("Note %q already exists in %q", title, book))
			return nil
		}
		printError(w, err.Error())
		return nil
	}
	printSuccess(w, fmt.Sprintf("Created %q in %s", title, book))
	return nil
}

func deleteNoteFromBook(w io.Writer, book, note string) error {
	if err := store.DeleteNote(book, note); err != nil {
		printError(w, err.Error())
		return nil
	}
	printSuccess(w, fmt.Sprintf("Deleted %q from %s", note, book))
	return nil
}

func searchInBook(w io.Writer, book, query string) error {
	fmt.Fprintln(w, "search: not implemented yet")
	return nil
}

// --- Note-scoped operations ---

func viewNote(w io.Writer, book, note string) error {
	fmt.Fprintln(w, "view: not implemented yet")
	return nil
}

func editNote(w io.Writer, book, note string) error {
	fmt.Fprintln(w, "edit: not implemented yet")
	return nil
}

func copyNote(w io.Writer, book, note string) error {
	fmt.Fprintln(w, "copy: not implemented yet")
	return nil
}
