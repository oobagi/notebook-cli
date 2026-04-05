# notebook-cli

Terminal-native note manager with a block-based editor, interactive browser, 17 themes, and search. Notes are plain markdown files on disk.

## Tech Stack

- **Language**: Go 1.25+
- **TUI**: Bubble Tea v2, Bubbles v2, Lip Gloss v2
- **CLI**: Cobra
- **Syntax Highlighting**: Chroma
- **Config**: BurntSushi/toml
- **Storage**: Local filesystem — notes stored as `.md` files in `~/.notebook/`

## Build, Run, Test

```bash
go install ./cmd/notebook/    # Install as "notebook" binary
go run ./cmd/notebook/        # Run from source
go test ./...                 # Test (592 tests across 12 packages)
go vet ./...                  # Lint
```

## Project Structure

```
cmd/
  notebook/    Main entry point (go install produces "notebook" binary)
cmd/*.go       Cobra command definitions (root, search, theme, config)
internal/
  block/       Block types, markdown↔block parser and serializer
  browser/     TUI browser: notebook/note list, search, recents, preview pane, theme picker
  clipboard/   System clipboard + OSC 52 fallback
  config/      TOML config (~/.config/notebook/config.toml), hint tracking
  editor/      Block editor: per-block textareas, / command palette, undo/redo, view mode
  format/      Output formatting: status bars, relative time, file sizes
  model/       Core types: Note, Notebook
  recents/     Recently edited notes tracking (~/.config/notebook/recent.json)
  storage/     Filesystem CRUD, search, welcome note seeding, slug/display name conversion
  theme/       17 themes, auto light/dark detection, per-block styling
fork/
  bubbles/     Forked Bubbles components (textarea, viewport) with project-specific patches
docs/          Design system documentation
assets/        Icons and media for README
```

## Conventions

- **CLI pattern**: `notebook [book] [note] [verb]` — bare commands do useful things (`notebook` opens TUI, `notebook ideas` scopes to a book, `notebook ideas "My Note"` opens the editor)
- **Auto-creation**: targeting a non-existent notebook creates it silently
- **Plain markdown**: notes are `.md` files, no proprietary format, readable outside the app
- **Block editor**: 10 block types (paragraph, h1-h3, bullet, numbered, checklist, code, quote, divider). `/` on empty block opens command palette
- **Search**: `/` in browser searches notebook names (L0) and note titles (L0 Notes section, L1). Title-only, synchronous, lazy-loaded note name cache
- **Themes**: 17 built-in, selectable via `t` in browser or `notebook theme <name>` CLI
- **View mode**: `Ctrl+R` toggles read-only rendered view with clickable checklists
- **Storage names**: filesystem uses slugs (`my-note.md`), display uses spaces (`My Note`). See `storage.Slugify()` and `storage.DisplayName()`
- **Error messages**: concise and actionable — always suggest a next action
- **Dependencies**: prefer Go standard library; minimize external deps
- **Testing**: table-driven tests, `setupTestStore()` helper for browser tests, `processCmd()` for Bubble Tea message processing
