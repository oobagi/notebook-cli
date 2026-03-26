# see -- CLI Markdown Viewer

> Source: https://github.com/guilhermeprokisch/see
> Stars: 269 | Language: Rust | License: MIT
> Crate: `see-cat` v0.9.1

## Overview

`see` is a terminal file visualization tool positioned as a modern replacement for `cat`. Its core purpose is rendering files with rich formatting directly in the terminal -- syntax-highlighted code, styled markdown, inline images, and more. Written in Rust, it uses tree-sitter (via the `lumis` crate) for syntax highlighting and a JSON AST approach for markdown rendering. It also exposes a library API (`render_markdown_to_html`) for programmatic HTML export.

## Key Features

- **Tree-sitter syntax highlighting**: Uses `lumis` (tree-sitter wrapper) for grammar-based highlighting rather than regex, supporting a wide range of languages out of the box. The `all-languages` feature flag bundles every grammar into the binary.
- **Markdown rendering**: Parses markdown into a JSON AST, then walks the tree to emit ANSI-styled terminal output. Supports headers (color-coded by depth), bold, italic, strikethrough, inline code, code blocks, tables, lists (ordered, unordered, task lists with checkboxes), blockquotes, footnotes, and reference-style links.
- **Admonition/alert support**: Detects GitHub-style `[!NOTE]`, `[!TIP]`, `[!WARNING]` etc. inside blockquotes and renders them with distinct icons and colors.
- **Inline image rendering**: Renders images (local, remote URL, or base64 data URI) directly in the terminal via the `viuer` crate. Remote images are downloaded to a temp directory (hashed filenames for caching).
- **Clickable hyperlinks**: Uses OSC 8 terminal escape sequences to create real clickable links in supported terminals.
- **Built-in pager**: An internal pager (no dependency on `less` or `bat`) with vim-style keybindings (j/k, g/G, Space/PageUp/PageDown, q to quit). Handles ANSI-aware soft-wrapping and preserves scroll position across reloads.
- **File watch mode**: Monitors files for changes and auto-reloads the display, with configurable poll interval (default 250ms, minimum 50ms). Useful for live-previewing markdown while editing.
- **Directory tree rendering**: When pointed at a directory, displays a tree view with file icons (via `devicons` crate).
- **Emoji support**: Converts `:emoji_name:` shortcodes to Unicode emoji in rendered output.
- **HTML-to-markdown conversion**: Converts inline HTML in markdown documents back to markdown for consistent terminal rendering (via `htmd` crate).
- **Configurable via TOML**: Config file at `~/.config/see/config.toml` with options for theme, image dimensions, table borders, line numbers, color toggle, watch interval, and custom file extension-to-language mappings.
- **Piped input support**: Works with stdin, so `curl ... | see` or `git diff | see` works.
- **Library mode**: The crate exports `render_markdown_to_html()` and `render_file_to_html()` for use as a Rust library.

## Notable Implementation Details

- **Rendering architecture**: Markdown is parsed to JSON AST (using the `markdown` crate with its `json` feature), then a recursive `render_node()` function pattern-matches on node type and emits ANSI-formatted output via `termcolor`. This is a two-pass approach -- first collect link definitions and footnotes, then render.
- **Global mutable state**: Uses `lazy_static!` mutexes for rendering state (heading level, indent depth, list stack, link definitions). This is pragmatic but means rendering is not re-entrant.
- **Viewer plugin system**: A trait-based `Viewer` abstraction with `MarkdownViewer`, `CodeViewer`, and `ImageViewer` implementations. `determine_viewer()` routes by file extension. The `ViewerManager` coordinates output.
- **Pager subprocess trick**: When the pager needs to reload (watch mode), it re-invokes itself as a subprocess with special flags to capture fresh rendered output, avoiding infinite recursion with flag detection.
- **ANSI-aware line wrapping**: The pager's `wrap_ansi_line()` function tracks active SGR sequences and hyperlink state across line breaks, ensuring formatting is correctly continued on wrapped lines. Uses `unicode-width` for accurate character width calculation.
- **Table rendering**: Box-drawing characters for borders, auto-calculated column widths, header rows in red/bold, first columns in cyan.
- **Language detection**: Uses `hyperpolyglot` crate for identifying programming languages from file paths.
- **Theme options**: Includes github_light, tokyonight, dracula, catppuccin_mocha, and others via the lumis highlighting engine.
- **Dependencies of note**: `crossterm` for terminal control, `viuer` for image protocols, `reqwest` (with rustls) for fetching remote images, `image` for processing, `base64` for data URIs.

## Relevance to Notebook

Several aspects of `see` are directly applicable to a CLI note manager:

- **Markdown rendering approach**: The JSON AST walker pattern is clean and extensible. For notebook, we could adopt a similar recursive renderer that maps markdown nodes to styled terminal output -- particularly the heading color hierarchy, task list checkboxes, and admonition blocks.
- **Built-in pager with watch mode**: The internal pager that avoids depending on external tools (`less`, `bat`) is appealing for a self-contained CLI. The watch mode could enable a "live preview" workflow where users edit notes in their editor while seeing rendered output update in real time.
- **OSC 8 hyperlinks**: Making links in notes actually clickable in the terminal is a nice UX touch worth adopting.
- **Admonition rendering**: The `[!NOTE]`/`[!TIP]`/`[!WARNING]` blocks could be useful for structured note metadata or callouts.
- **ANSI-aware text wrapping**: If notebook ever implements a pager or preview mode, the technique of tracking active ANSI sequences across line breaks is essential for correct rendering.
- **Configuration pattern**: The TOML config with sensible defaults and CLI flag overrides is a solid pattern for user customization.
- **Viewer trait abstraction**: The pluggable viewer system could inspire a similar pattern in notebook for rendering different content types (markdown notes, code snippets, images, structured data).
- **What to avoid**: The `lazy_static!` global mutable state approach works but is fragile. A context struct passed through the render tree would be cleaner and more testable.
