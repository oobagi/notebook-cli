package storage

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oobagi/notebook/internal/model"
)

// SearchResult represents a single line that matched a search query.
type SearchResult struct {
	Notebook string
	Note     string
	Line     int    // 1-based line number
	Text     string // the matching line
}

// ErrInvalidName is returned when a notebook or note name contains
// path-traversal sequences or is otherwise unsafe for use as a filename.
var ErrInvalidName = errors.New("invalid name")

// validName rejects names that could escape the storage root via path
// traversal or that are degenerate filesystem entries.
func validName(name string) error {
	if name == "" || name == "." {
		return fmt.Errorf("%w: must not be empty or %q", ErrInvalidName, ".")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("%w: %q contains path separator or traversal sequence", ErrInvalidName, name)
	}
	return nil
}

// Store manages notebook and note persistence on the filesystem.
// Notes are stored as markdown files within notebook directories.
// Layout: <root>/<notebook>/<note>.md
type Store struct {
	Root string
}

// NewStore creates a Store backed by the given root directory.
func NewStore(root string) *Store {
	return &Store{Root: root}
}

// notePath returns the full filesystem path for a note.
func (s *Store) notePath(notebook, name string) string {
	return filepath.Join(s.Root, notebook, name+".md")
}

// notebookPath returns the full filesystem path for a notebook directory.
func (s *Store) notebookPath(name string) string {
	return filepath.Join(s.Root, name)
}

// NotebookDir returns the full filesystem path for a notebook directory.
// This is the exported version of notebookPath for use by the command layer.
func (s *Store) NotebookDir(name string) string {
	return filepath.Join(s.Root, name)
}

// CreateNotebook creates a new notebook directory.
func (s *Store) CreateNotebook(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	dir := s.notebookPath(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create notebook %q: %w", name, err)
	}
	return nil
}

// CreateNote writes a markdown file into the given notebook directory.
// The notebook directory is created automatically if it does not exist.
// It returns an error if a note with the same name already exists.
func (s *Store) CreateNote(notebook, name, content string) error {
	if err := validName(notebook); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}

	dir := s.notebookPath(notebook)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create notebook dir %q: %w", notebook, err)
	}

	p := s.notePath(notebook, name)
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("note %q already exists in %q", name, notebook)
	}

	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("create note %q/%q: %w", notebook, name, err)
	}
	return nil
}

// GetNote reads a note from disk and populates all fields including
// timestamps derived from os.Stat.
func (s *Store) GetNote(notebook, name string) (model.Note, error) {
	if err := validName(notebook); err != nil {
		return model.Note{}, err
	}
	if err := validName(name); err != nil {
		return model.Note{}, err
	}
	p := s.notePath(notebook, name)

	data, err := os.ReadFile(p)
	if err != nil {
		return model.Note{}, fmt.Errorf("get note %q/%q: %w", notebook, name, err)
	}

	info, err := os.Stat(p)
	if err != nil {
		return model.Note{}, fmt.Errorf("stat note %q/%q: %w", notebook, name, err)
	}

	return model.Note{
		Name:      name,
		Notebook:  notebook,
		Content:   string(data),
		CreatedAt: fileCreatedAt(info),
		UpdatedAt: info.ModTime(),
	}, nil
}

// ListNotebooks returns the names of all notebook directories under the root.
// If the root directory does not exist yet, it returns an empty slice.
func (s *Store) ListNotebooks() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("list notebooks: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListNotes returns all notes in a notebook with metadata populated from stat.
func (s *Store) ListNotes(notebook string) ([]model.Note, error) {
	if err := validName(notebook); err != nil {
		return nil, err
	}
	dir := s.notebookPath(notebook)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list notes in %q: %w", notebook, err)
	}

	var notes []model.Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".md")
		note, err := s.GetNote(notebook, name)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, nil
}

// NoteCount returns the number of markdown notes in a notebook directory
// without reading their content.
func (s *Store) NoteCount(notebook string) (int, error) {
	if err := validName(notebook); err != nil {
		return 0, err
	}
	dir := s.notebookPath(notebook)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("count notes in %q: %w", notebook, err)
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count, nil
}

// NotebookModTime returns the most recent modification time of any note
// in the notebook. If the notebook is empty or does not exist, it returns
// the zero time.
func (s *Store) NotebookModTime(notebook string) (time.Time, error) {
	if err := validName(notebook); err != nil {
		return time.Time{}, err
	}
	dir := s.notebookPath(notebook)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("notebook mod time %q: %w", notebook, err)
	}

	var latest time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return time.Time{}, fmt.Errorf("stat %q in %q: %w", e.Name(), notebook, err)
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

// RenameNotebook renames a notebook directory atomically.
func (s *Store) RenameNotebook(oldName, newName string) error {
	if err := validName(oldName); err != nil {
		return err
	}
	if err := validName(newName); err != nil {
		return err
	}
	oldDir := s.notebookPath(oldName)
	if _, err := os.Stat(oldDir); err != nil {
		return fmt.Errorf("rename notebook %q to %q: %w", oldName, newName, err)
	}
	newDir := s.notebookPath(newName)
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("notebook %q already exists", newName)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("rename notebook %q to %q: %w", oldName, newName, err)
	}
	return nil
}

// RenameNote renames a note file within a notebook atomically.
func (s *Store) RenameNote(notebook, oldName, newName string) error {
	if err := validName(notebook); err != nil {
		return err
	}
	if err := validName(oldName); err != nil {
		return err
	}
	if err := validName(newName); err != nil {
		return err
	}
	oldPath := s.notePath(notebook, oldName)
	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("rename note %q/%q to %q: %w", notebook, oldName, newName, err)
	}
	newPath := s.notePath(notebook, newName)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("note %q already exists in %q", newName, notebook)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename note %q/%q to %q: %w", notebook, oldName, newName, err)
	}
	return nil
}

// DeleteNote removes a single note file.
func (s *Store) DeleteNote(notebook, name string) error {
	if err := validName(notebook); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	p := s.notePath(notebook, name)
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("delete note %q/%q: %w", notebook, name, err)
	}
	return nil
}

// DeleteNotebook removes a notebook directory and all of its contents.
func (s *Store) DeleteNotebook(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	dir := s.notebookPath(name)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete notebook %q: %w", name, err)
	}
	return nil
}

// UpdateNote overwrites the content of an existing note.
func (s *Store) UpdateNote(notebook, name, content string) error {
	if err := validName(notebook); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	p := s.notePath(notebook, name)

	// Verify the file exists before overwriting.
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("update note %q/%q: %w", notebook, name, err)
	}

	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("update note %q/%q: %w", notebook, name, err)
	}
	return nil
}

// SearchNotes searches for a query string across notes. If notebook is
// non-empty, only that notebook is searched. Results are sorted by
// notebook, note name, then line number.
func (s *Store) SearchNotes(query string, notebook string, caseSensitive bool) ([]SearchResult, error) {
	var notebooks []string
	if notebook != "" {
		if err := validName(notebook); err != nil {
			return nil, err
		}
		notebooks = []string{notebook}
	} else {
		var err error
		notebooks, err = s.ListNotebooks()
		if err != nil {
			return nil, fmt.Errorf("search: list notebooks: %w", err)
		}
	}

	q := query
	if !caseSensitive {
		q = strings.ToLower(query)
	}

	var results []SearchResult
	for _, nb := range notebooks {
		dir := s.notebookPath(nb)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("search: read dir %q: %w", nb, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			noteName := strings.TrimSuffix(e.Name(), ".md")
			p := filepath.Join(dir, e.Name())

			f, err := os.Open(p)
			if err != nil {
				return nil, fmt.Errorf("search: open %q/%q: %w", nb, noteName, err)
			}

			scanner := bufio.NewScanner(f)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				haystack := line
				if !caseSensitive {
					haystack = strings.ToLower(line)
				}
				if strings.Contains(haystack, q) {
					results = append(results, SearchResult{
						Notebook: nb,
						Note:     noteName,
						Line:     lineNum,
						Text:     line,
					})
				}
			}
			f.Close()
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("search: scan %q/%q: %w", nb, noteName, err)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Notebook != results[j].Notebook {
			return results[i].Notebook < results[j].Notebook
		}
		if results[i].Note != results[j].Note {
			return results[i].Note < results[j].Note
		}
		return results[i].Line < results[j].Line
	})

	return results, nil
}
