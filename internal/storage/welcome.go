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
	"Welcome to your terminal notebook.\n\n" +
	"> Press **Ctrl+G** for help. Press **Ctrl+R** for view mode.\n\n" +
	"---\n\n" +
	"## Try it\n\n" +
	"- [x] Open this note in the editor\n" +
	"- [ ] Press **Enter** to create a new block\n" +
	"- [ ] Type **/** on the empty block to pick a type\n" +
	"- [ ] Press **Ctrl+X** to toggle this checkbox\n" +
	"- [ ] Press **t** in the browser to try a theme\n\n" +
	"## Go build something\n\n" +
	"**Esc** twice to reach the main screen, then **n** to create a notebook.\n\n" +
	"Delete this notebook anytime with **d**."

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
