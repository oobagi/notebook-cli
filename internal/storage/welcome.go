package storage

import (
	"os"
	"path/filepath"
)

// welcomeContent is the markdown seeded into the "getting-started" notebook
// on first launch. It showcases every block type the editor supports.
const welcomeContent = "" +
	"             \u2588\u2584       \u2588\u2584\n" +
	" \u2584          \u2584\u2588\u2588\u2584      \u2588\u2588                \u2584\u2584\n" +
	" \u2588\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2588\u2588 \u2584\u2588\u2580\u2588\u2584 \u2588\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2588\u2588 \u2584\u2588\u2580\n" +
	" \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588\u2584\u2588\u2580 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588\u2588\u2588\n" +
	"\u2584\u2588\u2588 \u2580\u2588\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2588\u2588\u2584\u2580\u2588\u2584\u2584\u2584\u2584\u2588\u2588\u2588\u2588\u2580\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2588\u2588 \u2580\u2588\u2584\n\n" +
	"Welcome to your terminal notebook. Notes live in notebooks, and everything you write is plain markdown.\n\n" +
	"> Press **Esc** to go back to the browser. Press **Ctrl+R** to toggle view mode.\n\n" +
	"---\n\n" +
	"## Blocks\n\n" +
	"Press **/** on an empty block to pick a type. Press **Enter** to create a new block below.\n\n" +
	"- Paragraphs for prose\n" +
	"- Headings (H1, H2, H3) for structure\n" +
	"- Bullet and numbered lists\n" +
	"- Checklists with toggleable items\n" +
	"- Code blocks with syntax highlighting\n" +
	"- Quotes for callouts\n" +
	"- Dividers to separate sections\n\n" +
	"## Try it\n\n" +
	"- [x] Open this note in the editor\n" +
	"- [ ] Press **Enter** at the end of a line to make a new block\n" +
	"- [ ] Type **/** on the empty block and pick a type\n" +
	"- [ ] Press **Ctrl+X** to toggle this checkbox\n" +
	"- [ ] Press **t** in the browser to change themes\n\n" +
	"## Keys\n\n" +
	"- **Ctrl+S** \u2014 Save\n" +
	"- **Ctrl+R** \u2014 View mode\n" +
	"- **Ctrl+Z / Ctrl+Y** \u2014 Undo / Redo\n" +
	"- **Ctrl+G** \u2014 Help\n" +
	"- **/** \u2014 Command palette (empty block)\n" +
	"- **Esc** \u2014 Back to browser\n\n" +
	"## Go build something\n\n" +
	"Press **Esc** twice to get back to the main screen (once to leave this note, once to leave this notebook). Then press **n** to create your first notebook.\n\n" +
	"You can delete this welcome notebook anytime \u2014 highlight it and press **d**."

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
