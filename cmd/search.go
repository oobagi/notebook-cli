package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

var (
	searchNotebookFlag    string
	searchCaseSensitive   bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all notebooks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd.OutOrStdout(), args[0], searchNotebookFlag, searchCaseSensitive)
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchNotebookFlag, "notebook", "", "limit search to a specific notebook")
	searchCmd.Flags().BoolVar(&searchCaseSensitive, "case-sensitive", false, "use case-sensitive matching")
	rootCmd.AddCommand(searchCmd)
}

// runSearch executes the search and prints formatted results.
func runSearch(w io.Writer, query, notebook string, caseSensitive bool) error {
	results, err := store.SearchNotes(query, notebook, caseSensitive)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		printInfo(w, fmt.Sprintf("No matches for %q", query))
		return nil
	}

	printSearchResults(w, results, query, caseSensitive)
	return nil
}

// printSearchResults formats and prints search results according to the
// design system: breadcrumb path, snippet with bold match, summary count.
func printSearchResults(w io.Writer, results []storage.SearchResult, query string, caseSensitive bool) {
	// Group results by notebook+note to show one breadcrumb per note,
	// using the first matching line as the snippet.
	type key struct{ notebook, note string }
	seen := make(map[key]bool)
	var unique []storage.SearchResult
	for _, r := range results {
		k := key{r.Notebook, r.Note}
		if !seen[k] {
			seen[k] = true
			unique = append(unique, r)
		}
	}

	books := make(map[string]bool)
	for i, r := range unique {
		if i > 0 {
			fmt.Fprintln(w)
		}
		books[r.Notebook] = true

		// Breadcrumb: "  Work > Meeting Notes"
		fmt.Fprintf(w, "  %s \u203A %s\n", r.Notebook, r.Note)

		// Snippet: "    ...discussed the quarterly plan and..."
		snippet := formatSnippet(r.Text, query, caseSensitive)
		fmt.Fprintf(w, "    %s\n", snippet)
	}

	// Summary line
	fmt.Fprintln(w)
	matchWord := "match"
	if len(unique) != 1 {
		matchWord = "matches"
	}
	bookWord := "book"
	if len(books) != 1 {
		bookWord = "books"
	}
	fmt.Fprintf(w, "  \x1b[2m%d %s across %d %s\x1b[0m\n", len(unique), matchWord, len(books), bookWord)
}

// formatSnippet creates a truncated snippet line with the matched term in bold.
// The snippet is at most ~60 chars, with ellipsis on sides if truncated.
func formatSnippet(line, query string, caseSensitive bool) string {
	const maxLen = 60

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "\u2026"
	}

	// Find the match position in the line for centering.
	haystack := trimmed
	needle := query
	if !caseSensitive {
		haystack = strings.ToLower(trimmed)
		needle = strings.ToLower(query)
	}

	idx := strings.Index(haystack, needle)
	if idx < 0 {
		// Should not happen, but handle gracefully.
		if len(trimmed) > maxLen {
			return "\u2026" + trimmed[:maxLen-1] + "\u2026"
		}
		return trimmed
	}

	// Extract a window of ~maxLen chars centered on the match.
	matchLen := len(query)
	// Available space for context around the match (account for ANSI bold codes which are invisible).
	contextBudget := maxLen - matchLen
	if contextBudget < 0 {
		contextBudget = 0
	}
	leftCtx := contextBudget / 2
	rightCtx := contextBudget - leftCtx

	start := idx - leftCtx
	end := idx + matchLen + rightCtx

	prefixEllipsis := false
	suffixEllipsis := false

	if start < 0 {
		// Shift surplus to the right side.
		end += -start
		start = 0
	}
	if end > len(trimmed) {
		// Shift surplus to the left side.
		start -= end - len(trimmed)
		end = len(trimmed)
	}
	if start < 0 {
		start = 0
	}

	if start > 0 {
		prefixEllipsis = true
		start++ // make room for the ellipsis character
		if start > idx {
			start = idx
		}
	}
	if end < len(trimmed) {
		suffixEllipsis = true
		end-- // make room for the ellipsis character
		if end < idx+matchLen {
			end = idx + matchLen
		}
	}

	window := trimmed[start:end]

	// Rebuild with bold match. We need to find the match in the window.
	winHaystack := window
	winNeedle := query
	if !caseSensitive {
		winHaystack = strings.ToLower(window)
		winNeedle = strings.ToLower(query)
	}
	winIdx := strings.Index(winHaystack, winNeedle)

	var b strings.Builder
	if prefixEllipsis {
		b.WriteString("\u2026")
	}
	if winIdx >= 0 {
		b.WriteString(window[:winIdx])
		b.WriteString("\x1b[1m")
		b.WriteString(window[winIdx : winIdx+matchLen])
		b.WriteString("\x1b[0m")
		b.WriteString(window[winIdx+matchLen:])
	} else {
		b.WriteString(window)
	}
	if suffixEllipsis {
		b.WriteString("\u2026")
	}

	return b.String()
}
