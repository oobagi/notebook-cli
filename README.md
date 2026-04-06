<p align="center">
  <img src="assets/notebook-icon.png" width="128" height="128" alt="notebook icon">
</p>

<h1 align="center">notebook</h1>

<p align="center">
  <strong>A terminal-native note editor with a block-based editing experience.</strong>
</p>

<p align="center">
  <a href="https://github.com/oobagi/notebook-cli/releases/latest"><img src="https://img.shields.io/github/v/release/oobagi/notebook-cli?color=blue" alt="Latest Release"></a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey" alt="Platform">
</p>

<p align="center">
  <a href="#install"><strong>Install</strong></a>
  ·
  <a href="#quick-start"><strong>Quick Start</strong></a>
  ·
  <a href="#features"><strong>Features</strong></a>
  ·
  <a href="#cli-reference"><strong>CLI Reference</strong></a>
  ·
  <a href="#configuration"><strong>Configuration</strong></a>
  ·
  <a href="ROADMAP.md"><strong>Roadmap</strong></a>
</p>

---

Notebook is a TUI note manager that organizes markdown notes into notebooks. It comes with a block editor, an interactive browser with search, 17 themes, inline markdown formatting, undo/redo, and a view mode for reading.

Everything runs in your terminal. No account, no sync, no config required.
<p align="center">
  <img src="assets/hero.gif" alt="notebook demo" width="1010">
</p>

## Install

```bash
# Homebrew
brew install oobagi/tap/notebook

# Go (1.25+)
go install github.com/oobagi/notebook-cli/cmd/notebook@latest

```

Alternatively, prebuilt binaries for macOS and Linux are available on the [releases page](https://github.com/oobagi/notebook-cli/releases/latest).

## Quick Start

```bash
# Launch the interactive browser
notebook

# Create a note (auto-creates the notebook if it doesn't exist)
notebook ideas new "First Thought"

# Open it in the editor
notebook ideas "First Thought"

# Open any .md or .txt file directly
notebook path/to/file.md
```

## Features

- **Block editor** — 10 block types: paragraphs, headings, lists, checklists, code blocks, quotes, dividers. Press **/** to switch types.
- **Syntax highlighting** — 500+ languages via Chroma. Name the language on the first line of a code block.
- **Inline formatting** — `**bold**`, `*italic*`, `~~strikethrough~~`, `__underline__` render live in inactive blocks.
- **View mode** — Ctrl+R for a clean, read-only view. Click checklists to toggle them without editing.
- **Search** — Press **/** in the browser to search across all notebooks by title.
- **Preview pane** — Press **p** to see note content while browsing.
- **16 themes** — Dark, Ocean, Forest, Sunset, Monochrome, Rose, Cyberpunk, Minimal, Retro, Nord, Solarized, Dracula, Tokyo, Lavender, Ember, Catppuccin.
- **Undo/redo** — 100 levels, tracks content changes only.
- **Mouse support** — Click checklists in view mode, native text selection in the editor.

<details>
<summary><strong>Editor keybindings</strong></summary>

| Key | Action |
|---|---|
| **Enter** | New block below |
| **/** | Command palette (on empty block) |
| **Ctrl+S** | Save |
| **Ctrl+Z / Ctrl+Y** | Undo / Redo |
| **Ctrl+K** | Cut block |
| **Alt+Up / Alt+Down** | Move block up/down |
| **Ctrl+X** | Toggle checkbox |
| **Ctrl+R** | View mode |
| **Ctrl+J / Shift+Enter** | Newline within block |
| **Ctrl+W** | Toggle word wrap |
| **Ctrl+G** | Help |
| **Esc** | Back to browser |

</details>

<details>
<summary><strong>Browser keybindings</strong></summary>

| Key | Action |
|---|---|
| **Up/Down** | Navigate |
| **Enter** | Open |
| **Esc** | Back / Quit |
| **/** | Search |
| **n** | New notebook/note |
| **d** | Delete |
| **r** | Rename |
| **c** | Copy to clipboard |
| **p** | Preview pane |
| **t** | Theme picker |
| **Tab** | Cycle sections |
| **?** | Help |

</details>

## CLI Reference

```
notebook                               Launch interactive browser
notebook list                          List all notebooks
notebook <book> list                   List notes in a notebook
notebook <book> new "Title"            Create a note
notebook <book> "Title"                Open a note in the editor
notebook <book> "Title" edit           Open a note in the editor
notebook <book> "Title" copy           Copy note to clipboard
notebook <book> "Title" delete         Delete a note
notebook <book> delete                 Delete a notebook
notebook search "query"                Search across all notebooks
notebook <book> search "query"         Search within a notebook
notebook path/to/file.md               Open any .md or .txt file
notebook theme [name]                  List or set theme
notebook config                        Show current config
notebook config set <key> <value>      Set a config value
```

## Configuration

Config lives at `~/.config/notebook/config.toml`. Set values with `notebook config set`:

| Key | Default | Description |
|---|---|---|
| `storage_dir` | `~/.notebook` | Where notebooks are stored |
| `editor` | built-in | External editor command (leave empty for built-in) |
| `theme` | `dark` | Theme name (see `notebook theme` for options) |
| `date_format` | `relative` | How dates are displayed |

## License

MIT
