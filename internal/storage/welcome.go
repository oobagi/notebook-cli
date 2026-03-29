package storage

import (
	"os"
	"path/filepath"
)

// welcomeContent is the markdown seeded into the "getting-started" notebook
// on first launch. It showcases every block type the editor supports.
const welcomeContent = `# Welcome to Notebook

A simple, beautiful note-taking app for your terminal.

## Block Types

Your notes are made of blocks. Each block has a type. Press ` + "`/`" + ` in the editor to change a block's type. Here are all the types you can use:

### Headings

You're looking at three heading levels right now — H1, H2, and H3.

- Bullet lists organize ideas without order
- Each line starting with ` + "`-`" + ` becomes a bullet item
- Nest your thoughts naturally

1. Numbered lists add sequence
2. Steps, rankings, priorities
3. Each line starting with a number becomes an item

- [x] Checklists track progress
- [ ] Toggle items with the ` + "`/`" + ` palette
- [ ] Try it — change this block's type

> Quotes highlight important ideas or capture someone's words.

---

` + "```" + `
Code blocks preserve formatting exactly.
Great for snippets, commands, or ASCII art.
` + "```" + `

## Keyboard Shortcuts

- **Ctrl+S** — Save
- **Ctrl+Q** — Quit
- **Ctrl+G** — Help overlay
- **Ctrl+K** — Cut block
- **Enter** — New block
- **Backspace** — Merge with previous block (when at start)
- **Alt+Up/Down** — Reorder blocks
- **/** — Command palette (change block type)

## Getting Started

Create notebooks and notes from the main screen. Open this note in the editor to see how it's built — every block type above is something you can use in your own notes.`

// markerFileName is the name of the hidden file that indicates the store
// has already been initialised (so deleted welcome notes don't reappear).
const markerFileName = ".initialized"

// EnsureWelcome seeds a welcome note on first launch. It is safe to call
// on every startup: if the marker file exists the function returns
// immediately. A new welcome note is only created when the store contains
// zero notebooks.
func (s *Store) EnsureWelcome() error {
	markerPath := filepath.Join(s.Root, markerFileName)

	// Fast path: marker already exists — nothing to do.
	if _, err := os.Stat(markerPath); err == nil {
		return nil
	}

	// Ensure the root directory exists before listing notebooks.
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return err
	}

	notebooks, err := s.ListNotebooks()
	if err != nil {
		return err
	}

	if len(notebooks) == 0 {
		if err := s.CreateNote("getting-started", "welcome", welcomeContent); err != nil {
			return err
		}
	}

	// Write the marker regardless of whether we created the note, so we
	// never attempt the seed again.
	return os.WriteFile(markerPath, []byte("initialized\n"), 0o644)
}
