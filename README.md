<p align="center">
  <img src="assets/notebook-icon.png" width="128" height="128" alt="notebook icon">
</p>

<h1 align="center">notebook-cli</h1>

<p align="center">
  <strong>A terminal-native note editor with a block-based editing experience.</strong>
</p>

<p align="center">
  <a href="https://github.com/oobagi/notebook-cli/releases/latest"><img src="https://img.shields.io/github/v/release/oobagi/notebook-cli?color=blue" alt="Latest Release"></a>
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" alt="Go 1.24+">
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
  <a href="ROADMAP.md"><strong>Roadmap</strong></a>
</p>

---

Notebook is a TUI note manager that organizes markdown notes into notebooks. It comes with a block editor, an interactive browser with fuzzy search, 17 themes, inline markdown formatting, undo/redo, and a view mode for reading.

Everything runs in your terminal. No account, no sync, no config required.

## Install

```bash
go install github.com/oobagi/notebook-cli@latest
```

## Quick Start

```bash
# Launch the interactive browser
notebook

# Create a note (auto-creates the notebook if it doesn't exist)
notebook ideas new "First Thought"

# Open it in the editor
notebook ideas "First Thought"

# Open any markdown or text file directly (.md, .txt, .markdown)
notebook path/to/file.md
```

## Features

### Block Editor

The editor works with structured blocks instead of a single text buffer. Each block has a type — paragraph, heading, list, code, quote, or divider — and you edit one block at a time.

Press **/** on an empty block to open the command palette and switch its type. Press **Enter** to create a new block below.

**10 block types:** Paragraph, Heading 1/2/3, Bullet List, Numbered List, Checklist, Code Block, Quote, Divider.

#### Editor keybindings

| Key | Action |
|---|---|
| **Enter** | New block below |
| **/** | Command palette (on empty block) |
| **Ctrl+S** | Save |
| **Ctrl+Z / Ctrl+Y** | Undo / Redo (100 levels) |
| **Ctrl+K** | Cut block |
| **Alt+Up / Alt+Down** | Move block up/down |
| **Ctrl+X** | Toggle checkbox (checklist blocks) |
| **Ctrl+R** | Switch to view mode |
| **Ctrl+J / Shift+Enter** | Newline within block |
| **Ctrl+W** | Toggle word wrap |
| **Ctrl+G** | Help overlay |
| **Esc** | Back to browser |

### Inline Formatting

Inactive blocks render inline markdown — **bold**, *italic*, ~~strikethrough~~, and __underline__. Formatting nests correctly and won't interfere with block-level styles like headings.

| Syntax | Result |
|---|---|
| `**text**` | **Bold** |
| `*text*` | *Italic* |
| `~~text~~` | ~~Strikethrough~~ |
| `__text__` | Underline |
| `***text***` | Bold + Italic |
| `**__text__**` | Bold + Underline |

`snake_case` is left alone. The active block you're editing always shows raw markdown.

### Interactive Browser

Launch `notebook` with no arguments to open the browser. Navigate notebooks and notes with arrow keys, open with Enter, go back with Escape.

| Key | Action |
|---|---|
| **Up/Down** | Navigate |
| **Enter** | Open |
| **Esc** | Back / Quit |
| **/** | Filter (fuzzy search) |
| **n** | New notebook/note |
| **d** | Delete (with confirmation) |
| **r** | Rename |
| **c** | Copy note to clipboard |
| **p** | Toggle preview pane |
| **t** | Theme picker |
| **Tab** | Switch to recents |
| **?** | Help |

The **recents** section (Tab) shows recently edited notes across all notebooks.

The **preview pane** (p) shows note content in a side column while you browse.

### View Mode

Press **Ctrl+R** in the editor to switch to a read-only view. Content is rendered with full theme styling — no cursor, no editing chrome. Scroll with arrow keys or Page Up/Down. Press **Ctrl+R** again to return to editing where you left off.

### Themes

17 built-in themes with per-block customization (markers, heading styles, code block layouts, colors). Press **t** in the browser to open the theme picker and preview them live.

Themes: Dark, Light, Ocean, Forest, Sunset, Monochrome, Rose, Cyberpunk, Minimal, Retro, Nord, Solarized, Dracula, Tokyo, Lavender, Ember, Catppuccin.

Set a theme from the CLI:

```bash
notebook theme sunset
notebook theme auto    # auto-detect dark/light terminal
```

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
notebook path/to/file.md              Open any .md, .txt, or .markdown file
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
| `theme` | `auto` | Theme name or `auto` for terminal detection |
| `date_format` | `relative` | How dates are displayed |

## License

MIT
