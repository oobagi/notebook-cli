# notebook-cli

Terminal-native note manager with a block-based editor supporting 14 block types, interactive browser, 16 themes, and search. Notes are plain markdown files on disk.

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
go test ./...                 # Test (649 tests across 13 packages)
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
  format/      Output formatting: status bars, footer inputs, relative time, file sizes
  ui/          Shared UI components: picker, help overlay, key handler
  model/       Core types: Note, Notebook
  recents/     Recently edited notes tracking (~/.config/notebook/recent.json)
  storage/     Filesystem CRUD, search, welcome note seeding, slug/display name conversion
  theme/       16 themes, per-block styling
fork/
  bubbles/     Forked Bubbles components (textarea, viewport) with project-specific patches
assets/        Icons and media for README
```

## Conventions

- **CLI pattern**: `notebook [book] [note] [verb]` — bare commands do useful things (`notebook` opens TUI, `notebook ideas` scopes to a book, `notebook ideas "My Note"` opens the editor)
- **Auto-creation**: targeting a non-existent notebook creates it silently
- **Plain markdown**: notes are `.md` files, no proprietary format, readable outside the app
- **Block editor**: 14 block types (paragraph, h1-h3, bullet, numbered, checklist, code, table, quote, definition, callout, divider, embed). `/` at start of block opens command palette
- **Search**: `/` in browser searches notebook names (L0) and note titles (L0 Notes section, L1). Title-only, synchronous, lazy-loaded note name cache
- **Themes**: 16 built-in, selectable via `t` in browser or `notebook theme <name>` CLI
- **Settings**: `,` in browser opens settings screen for storage, theme, date format, word wrap, etc.
- **View mode**: `Ctrl+R` toggles read-only rendered view with clickable checklists and expandable embeds
- **Storage names**: filesystem uses slugs (`my-note.md`), display uses spaces (`My Note`). See `storage.Slugify()` and `storage.DisplayName()`
- **Error messages**: concise and actionable — always suggest a next action
- **Dependencies**: prefer Go standard library; minimize external deps
- **Testing**: table-driven tests, `setupTestStore()` helper for browser tests, `processCmd()` for Bubble Tea message processing
