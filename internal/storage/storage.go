package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oobagi/notebook/internal/model"
)

// Store manages notebooks and notes on the local filesystem.
// Notes are stored as plain .md files in notebook directories.
type Store struct {
	Root string
}

// New creates a Store rooted at the given directory.
// The root directory is created if it does not exist.
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Store{Root: root}, nil
}

// DefaultRoot returns the default storage path (~/.notebook).
func DefaultRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".notebook")
}

// CreateNotebook creates a new notebook directory.
func (s *Store) CreateNotebook(name string) error {
	path := filepath.Join(s.Root, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create notebook %q: %w", name, err)
	}
	return nil
}

// CreateNote creates a new note in a notebook. The notebook directory
// is auto-created if it does not exist.
func (s *Store) CreateNote(notebook, name, content string) error {
	dir := filepath.Join(s.Root, notebook)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create notebook dir %q: %w", notebook, err)
	}
	path := s.notePath(notebook, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write note %q: %w", name, err)
	}
	return nil
}

// GetNote reads a note from disk and returns it with metadata.
func (s *Store) GetNote(notebook, name string) (model.Note, error) {
	path := s.notePath(notebook, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Note{}, fmt.Errorf("read note %q: %w", name, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.Note{}, fmt.Errorf("stat note %q: %w", name, err)
	}
	return model.Note{
		Name:      name,
		Notebook:  notebook,
		Content:   string(data),
		UpdatedAt: info.ModTime(),
	}, nil
}

// UpdateNote overwrites the content of an existing note.
func (s *Store) UpdateNote(notebook, name, content string) error {
	path := s.notePath(notebook, name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("note %q not found: %w", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write note %q: %w", name, err)
	}
	return nil
}

// DeleteNote removes a single note file.
func (s *Store) DeleteNote(notebook, name string) error {
	path := s.notePath(notebook, name)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete note %q: %w", name, err)
	}
	return nil
}

// DeleteNotebook removes a notebook directory and all its notes.
func (s *Store) DeleteNotebook(name string) error {
	path := filepath.Join(s.Root, name)
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("delete notebook %q: %w", name, err)
	}
	return nil
}

// ListNotebooks returns the names of all notebook directories.
func (s *Store) ListNotebooks() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, fmt.Errorf("list notebooks: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListNotes returns the names of all notes in a notebook (without .md extension).
func (s *Store) ListNotes(notebook string) ([]string, error) {
	dir := filepath.Join(s.Root, notebook)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list notes in %q: %w", notebook, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return names, nil
}

// NoteCount returns the number of notes in a notebook.
func (s *Store) NoteCount(notebook string) (int, error) {
	notes, err := s.ListNotes(notebook)
	if err != nil {
		return 0, err
	}
	return len(notes), nil
}

// NotebookExists returns true if the notebook directory exists.
func (s *Store) NotebookExists(name string) bool {
	info, err := os.Stat(filepath.Join(s.Root, name))
	return err == nil && info.IsDir()
}

// NoteExists returns true if the note file exists.
func (s *Store) NoteExists(notebook, name string) bool {
	_, err := os.Stat(s.notePath(notebook, name))
	return err == nil
}

func (s *Store) notePath(notebook, name string) string {
	return filepath.Join(s.Root, notebook, name+".md")
}
