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

Notebook is a CLI tool for people who think in markdown. Organize notes into notebooks, write with live preview, and never leave your terminal.

- **No config needed** — just start writing
- **Notebooks auto-create** — reference a notebook that doesn't exist and it appears
- **Plain markdown files** — your notes are just `.md` files on disk, no lock-in
- **Live editor mode** — split-pane TUI with real-time markdown rendering as you type

## Features

- **[Create & organize](#usage)** — Group notes into notebooks. Create as many as you need.
- **[Live markdown preview](#usage)** — Editor mode renders markdown in real-time alongside your text.
- **[Terminal-native](#usage)** — Beautifully rendered markdown right in your terminal. No browser, no GUI.
- **[Zero lock-in](#usage)** — Notes are plain `.md` files in directories. Take them anywhere.

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
notebook search "query"                # Search across all notebooks
notebook config                        # Show current configuration
notebook completion bash               # Generate shell completions
```

## License

MIT
