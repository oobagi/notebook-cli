package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook/internal/clipboard"
	"github.com/oobagi/notebook/internal/config"
	"github.com/oobagi/notebook/internal/editor"
	"github.com/oobagi/notebook/internal/format"
	"github.com/oobagi/notebook/internal/recents"
	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

// dispatch handles the noun-verb routing for commands like:
//
//	notebook <book>                    -> list notes in book
//	notebook <book> list               -> list notes in book
//	notebook <book> new "Title"        -> create note in book
//	notebook <book> delete "Title"     -> delete note from book
//	notebook <book> search "query"     -> search within book (stub)
//	notebook <book> <note>             -> edit note
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
			printError(w, fmt.Sprintf("Missing note title. Try: notebook %s new \"My Note\"", book))
			return nil
		}
		return createNoteInBook(w, book, rest[1])
	case "delete":
		if len(rest) < 2 {
			printError(w, fmt.Sprintf("Missing note title. Try: notebook %s delete \"Note Title\"", book))
			return nil
		}
		return deleteNoteFromBook(w, cmd.InOrStdin(), book, rest[1], false)
	case "edit":
		if len(rest) < 2 {
			printError(w, fmt.Sprintf("Missing note title. Try: notebook %s edit \"Note Title\"", book))
			return nil
		}
		return editNote(w, book, rest[1])
	case "search":
		if len(rest) < 2 {
			printError(w, fmt.Sprintf("Missing search query. Try: notebook %s search \"query\"", book))
			return nil
		}
		return searchInBook(w, book, rest[1])
	}

	// second arg is not a book-level verb, so treat it as a note name
	note := second
	noteRest := rest[1:]

	// notebook <book> <note> — no verb: edit the note
	if len(noteRest) == 0 {
		return editNote(w, book, note)
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
		return deleteNoteFromBook(w, cmd.InOrStdin(), book, note, false)
	case "copy":
		return copyNote(w, book, note)
	default:
		printError(w, fmt.Sprintf("Unknown command %q. Try: edit, delete, or copy.", verb))
		return nil
	}
}

// --- Book-scoped operations ---

func listNotesInBook(w io.Writer, book string) error {
	notes, err := store.ListNotes(book)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "does not exist") {
			printError(w, fmt.Sprintf("Notebook %q doesn't exist. See your notebooks with: notebook list", book))
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
		sizeStr := format.HumanSize(info.Size())
		timeStr := format.RelativeTime(info.ModTime())
		rows = append(rows, []string{n.Name, sizeStr, timeStr})
	}

	for _, line := range alignColumns(rows) {
		fmt.Fprintln(w, line)
	}
	return nil
}

func createNoteInBook(w io.Writer, book, title string) error {
	title = storage.Slugify(title)
	if title == "" {
		printError(w, "Note title can't be empty.")
		return nil
	}
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

func deleteNoteFromBook(w io.Writer, r io.Reader, book, note string, force bool) error {
	// Verify the note exists before prompting for confirmation.
	if _, err := store.GetNote(book, note); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			printError(w, fmt.Sprintf("Note %q not found in %q. Run notebook %s list to see your notes.", note, book, book))
			return nil
		}
		printError(w, err.Error())
		return nil
	}
	if !force {
		prompt := fmt.Sprintf("  Delete %q from %s? This cannot be undone.", note, book)
		if !confirmByName(w, r, prompt, note) {
			printInfo(w, "Cancelled")
			return nil
		}
	}
	if err := store.DeleteNote(book, note); err != nil {
		printError(w, err.Error())
		return nil
	}
	printSuccess(w, fmt.Sprintf("Deleted %q from %s", note, book))
	return nil
}

func searchInBook(w io.Writer, book, query string) error {
	return runSearch(w, query, book, false)
}

// --- Note-scoped operations ---

func editNote(w io.Writer, book, note string) error {
	n, err := store.GetNote(book, note)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			printError(w, fmt.Sprintf("Note %q not found in %q. Run notebook %s list to see your notes.", note, book, book))
			return nil
		}
		return fmt.Errorf("edit note %q/%q: %w", book, note, err)
	}

	var fileSize int64
	notePath := filepath.Join(store.NotebookDir(book), note+".md")
	if fi, err := os.Stat(notePath); err == nil {
		fileSize = fi.Size()
	}

	cfg := editor.Config{
		Title:    book + " \u203A " + note,
		FilePath: notePath,
		FileSize: fileSize,
		Content:  n.Content,
		Save: func(content string) error {
			if err := store.UpdateNote(book, note, content); err != nil {
				return err
			}
			recents.RecordStore(book, note)
			return nil
		},
		DismissedHints: config.LoadDismissedHints(),
	}

	m := editor.New(cfg)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run editor: %w", err)
	}
	return nil
}

func copyNote(w io.Writer, book, note string) error {
	n, err := store.GetNote(book, note)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			printError(w, fmt.Sprintf("Note %q not found in %q. Run notebook %s list to see your notes.", note, book, book))
			return nil
		}
		return fmt.Errorf("copy note %q/%q: %w", book, note, err)
	}

	if err := clipboard.Copy(n.Content); err != nil {
		printError(w, fmt.Sprintf("Could not copy to clipboard: %s", err))
		return nil
	}
	printSuccess(w, fmt.Sprintf("Copied %q to clipboard", note))
	return nil
}
