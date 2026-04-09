package storage

import (
	"os"
	"path/filepath"
)

// welcomeContent is the markdown seeded into the "getting-started" notebook
// on first launch. It's intentionally lean — the feature showcase is a
// separate note linked via embed.
const welcomeContent = "" +
	"             \u2588\u2584       \u2588\u2584\n" +
	" \u2584          \u2584\u2588\u2588\u2584      \u2588\u2588                \u2584\u2584\n" +
	" \u2588\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2588\u2588 \u2584\u2588\u2580\u2588\u2584 \u2588\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2584\u2588\u2588\u2588\u2584 \u2588\u2588 \u2584\u2588\u2580\n" +
	" \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588\u2584\u2588\u2580 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588 \u2588\u2588\u2588\u2588\n" +
	"\u2584\u2588\u2588 \u2580\u2588\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2588\u2588\u2584\u2580\u2588\u2584\u2584\u2584\u2584\u2588\u2588\u2588\u2588\u2580\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2580\u2588\u2588\u2588\u2580\u2584\u2588\u2588 \u2580\u2588\u2584\n\n" +
	"v1.2.0\n\n" +
	"Welcome to your terminal notebook.\n\n" +
	"> Press **Ctrl+G** for help. Press **Ctrl+R** for view mode.\n\n" +
	"---\n\n" +
	"## Try it\n\n" +
	"- [x] Open this note in the editor\n" +
	"- [ ] Press **Enter** to create a new block\n" +
	"- [ ] Type **/** to open the command palette\n" +
	"- [ ] Press **Ctrl+X** to interact (toggle checkboxes, open embeds, search definitions)\n" +
	"- [ ] Press **t** in the browser to try a theme\n\n" +
	"## Go build something\n\n" +
	"**Esc** twice to reach the main screen, then **n** to create a notebook.\n\n" +
	"Delete this notebook anytime with **d**.\n\n" +
	"---\n\n" +
	"> [!TIP]\n" +
	"> Want a tour of every feature? Open the showcase below. In view mode (**Ctrl+R**), click the embed or press **Ctrl+X** to open it.\n\n" +
	"![[getting-started/2-feature-tour]]"

// showcaseContent is the feature showcase note linked from the welcome note.
const showcaseContent = "" +
	"# Feature Showcase\n\n" +
	"A tour of everything notebook can do.\n\n" +
	"---\n\n" +
	"## Lists\n\n" +
	"- Bullet lists support nesting with **Tab** / **Shift+Tab**\n" +
	"    - Nested item\n\n" +
	"1. Numbered lists auto-renumber\n" +
	"2. Reorder with **Alt+Up** / **Alt+Down**\n\n" +
	"- [x] Checklists: click in view mode or **Ctrl+X** in edit mode to toggle\n" +
	"- [ ] **Ctrl+H** sorts checked items to the bottom\n\n" +
	"## Code Blocks\n\n" +
	"```go\nfunc main() {\n    fmt.Println(\"Hello from notebook!\")\n}\n```\n\n" +
	"## Tables\n\n" +
	"| Shortcut | Action |\n| --- | --- |\n| Alt+R | Add row |\n| Alt+C | Add column |\n\n" +
	"## Quotes\n\n" +
	"> The best way to predict the future is to invent it.\n> \u2014 Alan Kay\n\n" +
	"## Definitions\n\n" +
	"Press **:** on an empty block to search definitions.\n\n" +
	"TUI\n: Terminal User Interface\n\n" +
	"## Callouts\n\n" +
	"> [!TIP]\n> Five variants. Press **Ctrl+T** to cycle: Note, Tip, Important, Warning, Caution.\n\n" +
	"---\n\n" +
	"Press **Esc** to go back and start building your own notes."

// markerFileName is the name of the hidden file that indicates the store
// has already been initialised (so deleted welcome notes don't reappear).
const markerFileName = ".initialized"

// EnsureWelcome seeds welcome notes on first launch. It is safe to call
// on every startup: if the marker file exists the function returns
// immediately. New notes are only created when the store contains
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
		if err := s.CreateNote("getting-started", "1-welcome", welcomeContent); err != nil {
			return err
		}
		if err := s.CreateNote("getting-started", "2-feature-tour", showcaseContent); err != nil {
			return err
		}
	}

	// Write the marker regardless of whether we created the note, so we
	// never attempt the seed again.
	return os.WriteFile(markerPath, []byte("initialized\n"), 0o644)
}
