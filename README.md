<p align="center">
  <img src="assets/notebook-icon.png" width="128" height="128" alt="notebook icon">
</p>

<h1 align="center">Notebook</h1>

<p align="center">
  <strong>A dead-simple CLI note manager with live markdown preview in your terminal.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" alt="Go 1.24+">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey" alt="Platform">
</p>

<p align="center">
  <a href="#quickstart"><strong>Quickstart</strong></a>
  ·
  <a href="#usage"><strong>Usage</strong></a>
  ·
  <a href="ROADMAP.md"><strong>Roadmap</strong></a>
</p>

---

## What is Notebook?

Notebook is a terminal-native note editor that renders markdown as you type — split-pane, real-time, no browser needed. Think Google Docs meets your terminal.

- **Block editor** — write in structured blocks (paragraphs, headings, lists, code, quotes) with a `/` command palette to switch types
- **TUI browser** — launch `notebook` and browse everything with arrow keys and fuzzy search, like a GUI
- **Open any file** — `notebook path/to/file.md` opens external files directly in the editor
- **Clickable links** — URLs in your notes are actually clickable in supported terminals (OSC 8)

## Features

- **[Block editor](#usage)** — Structured editing with 10 block types. Press `/` to open the command palette and change block types.
- **[External files](#usage)** — `notebook path/to/file.md` opens any markdown or text file directly in the editor.
- **[Clipboard copy](#usage)** — `notebook <book> <note> copy` to copy note content to your clipboard.
- **[Full-text search](#usage)** — `notebook search "query"` across every note, with highlighted matches and context.
- **[TUI browser](#usage)** — Fuzzy-searchable notebook/note browser. Arrow keys + Enter. No commands to memorize.
- **[GitHub admonitions](#usage)** — `[!NOTE]`, `[!WARNING]`, and friends render with colored borders and icons.
- **[Theme picker](#usage)** — Press `t` in the TUI to browse glamour styles with a live preview. Auto-detects light/dark by default.

## Quickstart

```bash
# Install
go install github.com/oobagi/notebook@latest

# Create your first note (auto-creates the "ideas" notebook)
notebook ideas new "First Thought"

# List your notebooks
notebook list

# Open a note in editor mode (live preview)
notebook ideas "First Thought" edit
```

## Usage

```bash
notebook                               # Launch interactive TUI browser
notebook list                          # List all notebooks
notebook new "Book Name"               # Create a notebook
notebook <book> list                   # List notes in a notebook
notebook <book> new "Note Title"       # Create a note (auto-creates notebook)
notebook <book> <note>                 # View a note (rendered markdown)
notebook <book> <note> edit            # Open editor with live markdown preview
notebook <book> <note> delete          # Delete a note
notebook <book> <note> copy            # Copy note content to clipboard
notebook <book> delete                 # Delete a notebook and all its notes
notebook <book> search "query"         # Search within a notebook
notebook search "query"                # Search across all notebooks
notebook path/to/file.md               # Open any file directly in the editor
notebook config                        # Show current configuration
notebook config set <key> <value>      # Set a config value
```

## Community Styles

The theme picker includes community-contributed Glamour styles for markdown rendering. See [styles/community/CONTRIBUTING.md](styles/community/CONTRIBUTING.md) to add your own.

## License

MIT
