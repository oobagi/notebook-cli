package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/oobagi/notebook/internal/storage"
)

func TestRootCommandUseField(t *testing.T) {
	if rootCmd.Use != "notebook" {
		t.Errorf("expected Use to be %q, got %q", "notebook", rootCmd.Use)
	}
}

func TestRootCommandSilenceErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

// setupTestStore creates a temporary store and configures the package-level
// store variable. It returns the temp dir path.
func setupTestStore(t *testing.T) string {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	dir := t.TempDir()
	store = storage.NewStore(dir)
	return dir
}

// executeCapture runs the root command with the given args and captures stdout.
func executeCapture(args []string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// --- Flag tests ---

func TestDirFlagRegistered(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("dir")
	if f == nil {
		t.Fatal("--dir flag not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--dir default should be empty string, got %q", f.DefValue)
	}
}

func TestDirFlagDefaultsToHomeNotebook(t *testing.T) {
	// Reset dirFlag to empty to simulate no --dir passed.
	dirFlag = ""
	rootCmd.SetArgs([]string{"list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	home, _ := os.UserHomeDir()
	if store == nil {
		t.Fatal("store should be initialized after Execute")
	}
	if store.Root != home+"/.notebook" {
		t.Errorf("store.Root = %q, want %q", store.Root, home+"/.notebook")
	}
}

func TestDirFlagCustomPath(t *testing.T) {
	dir := t.TempDir()
	dirFlag = dir
	rootCmd.SetArgs([]string{"list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.Root != dir {
		t.Errorf("store.Root = %q, want %q", store.Root, dir)
	}
	dirFlag = "" // reset
}

// --- Dispatcher routing tests ---

func TestDispatchBookWithNoArgs(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("ideas")
	_ = st.CreateNote("ideas", "spark", "content")

	out, err := executeCapture([]string{"--dir", dir, "ideas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "spark") {
		t.Errorf("expected note 'spark' in output, got %q", out)
	}
}

func TestDispatchBookList(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("work")
	_ = st.CreateNote("work", "todo", "stuff")
	_ = st.CreateNote("work", "meeting", "notes")

	out, err := executeCapture([]string{"--dir", dir, "work", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "todo") {
		t.Errorf("expected %q in output, got %q", "todo", out)
	}
	if !strings.Contains(out, "meeting") {
		t.Errorf("expected %q in output, got %q", "meeting", out)
	}
}

func TestDispatchBookNew(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "journal", "new", "day1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Created \"day1\" in journal\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	// Verify the note was actually created.
	st := storage.NewStore(dir)
	note, err := st.GetNote("journal", "day1")
	if err != nil {
		t.Fatalf("note should exist: %v", err)
	}
	if note.Name != "day1" {
		t.Errorf("note name = %q, want %q", note.Name, "day1")
	}
}

func TestDispatchBookDelete(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "temp", "data")

	// Pipe the note name to confirm deletion.
	rootCmd.SetIn(strings.NewReader("temp\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "work", "delete", "temp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713 Deleted \"temp\" from work") {
		t.Errorf("expected success message in output, got %q", out)
	}

	// Verify the note is gone.
	_, err = st.GetNote("work", "temp")
	if err == nil {
		t.Fatal("note should have been deleted")
	}
}

func TestDispatchBookSearch(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "todo", "hello world")

	out, err := executeCapture([]string{"--dir", dir, "work", "search", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' in output, got %q", out)
	}
	if !strings.Contains(out, "todo") {
		t.Errorf("expected 'todo' in output, got %q", out)
	}
}

func TestDispatchNoteViewNotFound(t *testing.T) {
	dir := setupTestStore(t)
	_ = storage.NewStore(dir)

	out, err := executeCapture([]string{"--dir", dir, "work", "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got %q", out)
	}
}

func TestDispatchNoteEditNotFound(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "work", "readme", "edit"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got %q", out)
	}
}

func TestDispatchNoteDelete(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "doomed", "bye")

	// Pipe the note name to confirm deletion.
	rootCmd.SetIn(strings.NewReader("doomed\n"))
	defer rootCmd.SetIn(nil)

	out, err := executeCapture([]string{"--dir", dir, "work", "doomed", "delete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713 Deleted \"doomed\" from work") {
		t.Errorf("expected success message in output, got %q", out)
	}

	_, err = st.GetNote("work", "doomed")
	if err == nil {
		t.Fatal("note should have been deleted")
	}
}

func TestDispatchNoteCopy(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("work", "readme", "# Hello\n\nSome content.")

	out, err := executeCapture([]string{"--dir", dir, "work", "readme", "copy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should show success message (clipboard copy may use OSC 52 fallback in CI).
	if !strings.Contains(out, "Copied") || !strings.Contains(out, "readme") {
		t.Errorf("expected success message with note name, got %q", out)
	}
}

func TestDispatchNoteCopyNotFound(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "work", "nonexistent", "copy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got %q", out)
	}
}

func TestDispatchUnknownNoteVerb(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "work", "readme", "frobnicate"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Unknown command") {
		t.Errorf("expected 'Unknown command' in output, got %q", out)
	}
}

func TestDispatchBookNewMissingTitle(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "work", "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Missing note title") {
		t.Errorf("expected 'Missing note title' in output, got %q", out)
	}
}

func TestDispatchBookDeleteNoArgShowsError(t *testing.T) {
	dir := setupTestStore(t)

	// No note title given: should print a usage hint error.
	out, err := executeCapture([]string{"--dir", dir, "work", "delete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Missing note title") {
		t.Errorf("expected 'Missing note title' in output, got %q", out)
	}
}

func TestDispatchBookDeleteNoArgEmptyBook(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("empty-book")

	// No note title given: should print a usage hint error (not picker).
	out, err := executeCapture([]string{"--dir", dir, "empty-book", "delete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Missing note title") {
		t.Errorf("expected 'Missing note title' in output, got %q", out)
	}
}

// --- Top-level command tests ---

func TestTopLevelList(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("alpha")
	_ = st.CreateNotebook("beta")

	out, err := executeCapture([]string{"--dir", dir, "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected %q in output, got %q", "alpha", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected %q in output, got %q", "beta", out)
	}
}

func TestTopLevelNew(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "new", "Projects"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Created \"projects\"\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	st := storage.NewStore(dir)
	nbs, _ := st.ListNotebooks()
	found := false
	for _, n := range nbs {
		if n == "projects" {
			found = true
		}
	}
	if !found {
		t.Error("notebook 'projects' should exist after creation (slugified)")
	}
}

func TestTopLevelDelete(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("trash")

	out, err := executeCapture([]string{"--dir", dir, "delete", "--force", "trash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Deleted \"trash\"\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	nbs, _ := st.ListNotebooks()
	for _, n := range nbs {
		if n == "trash" {
			t.Error("notebook 'trash' should be deleted")
		}
	}
}

func TestTopLevelSearch(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "search", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No notebooks exist, so should show "No matches".
	if !strings.Contains(out, "No matches") {
		t.Errorf("expected 'No matches' message, got %q", out)
	}
}

func TestTopLevelConfig(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "config"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "storage_dir") {
		t.Errorf("expected 'storage_dir' in output, got %q", out)
	}
	if !strings.Contains(out, "Config file:") {
		t.Errorf("expected 'Config file:' in output, got %q", out)
	}
}

// --- No args launches browser ---

func TestNoArgsLaunchesBrowser(t *testing.T) {
	dir := setupTestStore(t)

	// Without a TTY the browser cannot open, so we expect a TTY-related error.
	// This confirms the root command dispatches to the browser rather than help.
	_, err := executeCapture([]string{"--dir", dir})
	if err == nil {
		t.Fatal("expected TTY error when launching browser without a terminal")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Errorf("expected TTY error, got %q", err.Error())
	}
}

// --- Issue #4: notebook new output formatting ---

func TestNewNotebookSuccessOutput(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "new", "test-book"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Created \"test-book\"\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	// Verify the notebook was actually created.
	st := storage.NewStore(dir)
	nbs, _ := st.ListNotebooks()
	found := false
	for _, n := range nbs {
		if n == "test-book" {
			found = true
		}
	}
	if !found {
		t.Error("notebook 'test-book' should exist after creation")
	}
}

func TestNewNoteInBookSuccessOutput(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "mybook", "new", "test-note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "  \u2713 Created \"test-note\" in mybook\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	// Verify the note was actually created.
	st := storage.NewStore(dir)
	note, err := st.GetNote("mybook", "test-note")
	if err != nil {
		t.Fatalf("note should exist: %v", err)
	}
	if note.Name != "test-note" {
		t.Errorf("note name = %q, want %q", note.Name, "test-note")
	}
}

func TestNewDuplicateNotebookShowsError(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNotebook("my-notebook")

	out, err := executeCapture([]string{"--dir", dir, "new", "my-notebook"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "already exists") {
		t.Errorf("expected 'already exists' in output, got %q", out)
	}
}

func TestNewNoteAutoCreatesNotebook(t *testing.T) {
	dir := setupTestStore(t)

	// "newbook" does not exist yet; creating a note should auto-create it.
	out, err := executeCapture([]string{"--dir", dir, "newbook", "new", "first-note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2713") {
		t.Errorf("expected success symbol in output, got %q", out)
	}

	// Verify both the notebook and note exist.
	st := storage.NewStore(dir)
	nbs, _ := st.ListNotebooks()
	found := false
	for _, n := range nbs {
		if n == "newbook" {
			found = true
		}
	}
	if !found {
		t.Error("notebook 'newbook' should have been auto-created")
	}
	note, err := st.GetNote("newbook", "first-note")
	if err != nil {
		t.Fatalf("note should exist: %v", err)
	}
	if note.Name != "first-note" {
		t.Errorf("note name = %q, want %q", note.Name, "first-note")
	}
}

func TestNewDuplicateNoteShowsError(t *testing.T) {
	dir := setupTestStore(t)
	st := storage.NewStore(dir)
	_ = st.CreateNote("my-notebook", "my-note", "content")

	out, err := executeCapture([]string{"--dir", dir, "my-notebook", "new", "my-note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error symbol in output, got %q", out)
	}
	if !strings.Contains(out, "already exists") {
		t.Errorf("expected 'already exists' in output, got %q", out)
	}
}
