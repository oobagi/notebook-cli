package model

import "time"

// Note represents a single markdown note within a notebook.
type Note struct {
	Name      string
	Notebook  string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Notebook represents a collection of notes.
type Notebook struct {
	Name  string
	Notes []Note
}
