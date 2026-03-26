package cmd

import (
	"strings"
	"testing"

	"github.com/oobagi/notebook/internal/storage"
	"github.com/spf13/cobra"
)

func TestCompletionBash(t *testing.T) {
	out, err := executeCapture([]string{"completion", "bash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "notebook") {
		t.Errorf("expected bash completion script to contain %q, got %q", "notebook", out)
	}
	if !strings.Contains(out, "bash") {
		t.Errorf("expected bash completion script to contain %q, got %q", "bash", out)
	}
}

func TestCompletionZsh(t *testing.T) {
	out, err := executeCapture([]string{"completion", "zsh"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "notebook") {
		t.Errorf("expected zsh completion script to contain %q, got %q", "notebook", out)
	}
}

func TestCompletionFish(t *testing.T) {
	out, err := executeCapture([]string{"completion", "fish"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "notebook") {
		t.Errorf("expected fish completion script to contain %q, got %q", "notebook", out)
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	_, err := executeCapture([]string{"completion", "nushell"})
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected 'unsupported shell' in error, got %q", err.Error())
	}
}

func TestCompletionNoArgs(t *testing.T) {
	_, err := executeCapture([]string{"completion"})
	if err == nil {
		t.Fatal("expected error when no shell argument provided")
	}
}

func TestDynamicCompletionNotebooks(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("work")
	_ = st.CreateNotebook("personal")

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Should contain notebook names.
	found := map[string]bool{"work": false, "personal": false}
	for _, c := range completions {
		if _, ok := found[c]; ok {
			found[c] = true
		}
	}
	for name, ok := range found {
		if !ok {
			t.Errorf("expected completion to contain %q", name)
		}
	}

	// Should also contain known subcommands.
	subcommands := []string{"list", "new", "delete", "search", "config", "version", "completion"}
	for _, sub := range subcommands {
		contains := false
		for _, c := range completions {
			if c == sub {
				contains = true
				break
			}
		}
		if !contains {
			t.Errorf("expected completion to contain subcommand %q", sub)
		}
	}
}

func TestDynamicCompletionNotes(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "todo", "buy milk")
	_ = st.CreateNote("work", "meeting", "standup notes")

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{"work"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Should contain note names.
	found := map[string]bool{"todo": false, "meeting": false}
	for _, c := range completions {
		if _, ok := found[c]; ok {
			found[c] = true
		}
	}
	for name, ok := range found {
		if !ok {
			t.Errorf("expected completion to contain note %q", name)
		}
	}

	// Should also contain book-level verbs.
	verbs := []string{"list", "new", "delete", "search"}
	for _, v := range verbs {
		contains := false
		for _, c := range completions {
			if c == v {
				contains = true
				break
			}
		}
		if !contains {
			t.Errorf("expected completion to contain verb %q", v)
		}
	}
}

func TestDynamicCompletionNoteVerbs(t *testing.T) {
	dir := setupTestStore(t)
	_ = dir

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{"work", "todo"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	expected := []string{"edit", "delete", "copy"}
	if len(completions) != len(expected) {
		t.Fatalf("expected %d completions, got %d: %v", len(expected), len(completions), completions)
	}
	for _, e := range expected {
		contains := false
		for _, c := range completions {
			if c == e {
				contains = true
				break
			}
		}
		if !contains {
			t.Errorf("expected completion to contain %q", e)
		}
	}
}

func TestDynamicCompletionNoNotebooks(t *testing.T) {
	_ = setupTestStore(t) // empty store

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// With no notebooks, should still return subcommands.
	subcommands := []string{"list", "new", "delete", "search", "config", "version", "completion"}
	for _, sub := range subcommands {
		contains := false
		for _, c := range completions {
			if c == sub {
				contains = true
				break
			}
		}
		if !contains {
			t.Errorf("expected completion to contain subcommand %q", sub)
		}
	}
}

func TestDynamicCompletionInvalidBook(t *testing.T) {
	_ = setupTestStore(t) // empty store, no "nonexistent" notebook

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{"nonexistent"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Should fall back to book-level verbs when notebook doesn't exist.
	expected := []string{"list", "new", "delete", "search"}
	if len(completions) != len(expected) {
		t.Fatalf("expected %d completions, got %d: %v", len(expected), len(completions), completions)
	}
}

func TestDynamicCompletionBeyondThreeArgs(t *testing.T) {
	_ = setupTestStore(t)

	fn := rootCmd.ValidArgsFunction
	if fn == nil {
		t.Fatal("ValidArgsFunction should be set on rootCmd")
	}

	completions, directive := fn(rootCmd, []string{"work", "todo", "edit"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
	if completions != nil {
		t.Errorf("expected nil completions for 3+ args, got %v", completions)
	}
}
