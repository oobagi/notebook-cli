# notebook

CLI-focused note manager for storing, managing, displaying, and editing markdown notes with live terminal preview.

## Tech Stack

- **Language**: Go (latest stable)
- **Key Dependencies**: Cobra (CLI framework), Bubble Tea (TUI), Glamour (markdown rendering), Lip Gloss (styling), BurntSushi/toml (config)
- **Storage**: Local filesystem — notes stored as `.md` files organized in notebook directories

## Build, Run, Test

```bash
go build -o notebook .       # Build
go run . <command>            # Run
go test ./...                 # Test
go vet ./...                  # Lint
```

## Project Structure

```
cmd/           CLI command definitions (Cobra commands)
internal/
  browser/     TUI notebook/note browser with fuzzy search (Bubble Tea)
  clipboard/   System clipboard and OSC 52 clipboard support
  config/      TOML configuration file (~/.config/notebook/config.toml)
  editor/      Terminal editor with live markdown preview (Bubble Tea)
  model/       Core types: Notebook, Note
  picker/      TUI selection prompt for choosing from a list (Bubble Tea)
  render/      Markdown rendering, OSC 8 links, admonition blocks (Glamour)
  storage/     Filesystem-based notebook/note CRUD operations
  theme/       Auto light/dark theme detection
docs/          Documentation and design system
assets/        Icons and media
```

## Conventions

- Commands follow a noun-verb pattern: `notebook [resource] [sub-resource] [verb]` (e.g., `notebook ideas new "My Note"`)
- Bare commands do useful things: `notebook` opens TUI, `notebook ideas` scopes to that book, `notebook ideas "My Note"` views the note
- Notebook auto-creation: if a note targets a non-existent notebook, create it silently
- Notes are plain markdown files on disk — no proprietary format
- Error messages should be concise and actionable — always suggest a next action
- Use Go standard library where possible; minimize dependencies
- See `docs/design.md` for the full CLI design system (colors, symbols, output formatting)
