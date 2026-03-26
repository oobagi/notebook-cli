# glow -- Terminal Markdown Viewer

> Source: https://github.com/charmbracelet/glow

## Overview

Glow is a terminal-based markdown reader by Charmbracelet, written in Go (~24k GitHub stars). It renders markdown with syntax highlighting, colors, and layout formatting directly in the terminal. It operates in two modes: a **CLI mode** for piping/scripting (render a single file to stdout) and a **TUI mode** (interactive file browser + pager). It handles local files, stdin, and can fetch READMEs from GitHub/GitLab URLs.

Glow v2 (current) is a local-first tool. The earlier v1 had cloud-sync "stash" features via Charm Cloud, but these were removed entirely in the v2.0.0 release.

## Key Features

- **Markdown rendering in terminal** -- Uses Glamour (their own rendering library) to produce styled, colorized markdown output with word wrapping, heading formatting, code block highlighting, and link display.
- **TUI file browser** -- Running `glow` with no arguments launches an interactive browser that discovers markdown files in the current directory (and subdirectories / git repo). Supports fuzzy search (`/` key) via the `sahilm/fuzzy` library.
- **High-performance pager** -- Built on the Bubbles viewport component. Supports vim-style navigation (j/k, g/G, d/u, b/f), scroll percentage in status bar, and optional line numbers.
- **File watching** -- Uses `fsnotify` to detect on-disk changes and auto-reload the currently viewed document.
- **Editor integration** -- Press `e` to open the current document in `$EDITOR`, then returns to the pager view.
- **Clipboard support** -- Press `c` to copy document contents via OSC 52 and system clipboard (`atotto/clipboard`).
- **Multiple input sources** -- Local files, stdin (`glow -`), GitHub/GitLab repo URLs (`glow github.com/owner/repo`), and arbitrary HTTP URLs to `.md` files.
- **Auto light/dark detection** -- Detects terminal background color and selects appropriate color scheme. Can be overridden with `-s dark`, `-s light`, or a custom JSON stylesheet.
- **Word wrap control** -- `-w 60` sets column width for wrapping.
- **Pager mode** -- `-p` pipes output through `$PAGER` (defaults to `less -r`).
- **Non-TTY awareness** -- Gracefully degrades when output is piped (disables styling for clean text output).
- **YAML configuration** -- `glow.yml` config file for persistent settings (style, mouse support, pager, width, line numbers, newline preservation, show hidden files).

## Notable Implementation Details

**Tech stack:**
- Go (99.9% of codebase), requires Go 1.21+
- Bubble Tea (Elm-architecture TUI framework) for the interactive mode
- Glamour for markdown-to-ANSI rendering
- Lipgloss for component-level terminal styling
- Bubbles for reusable TUI components (viewport, etc.)
- Cobra + Viper for CLI framework and config management

**Architecture:**
- State machine pattern with two top-level states: `stateShowStash` (file browser) and `stateShowDocument` (pager view). Transitions managed by a parent `model` struct that delegates to child `stashModel` and `pagerModel`.
- A `commonModel` struct holds shared state (config, terminal dimensions, working directory) shared across sub-models.
- Source abstraction: a lightweight `source` struct wraps `io.ReadCloser` + URL metadata, enabling uniform handling of files, stdin, and HTTP sources.
- Separate execution paths: `executeCLI` (render and print), `runTUI` (interactive mode), `executeArg` (resolve source argument).
- Platform-specific file ignoring (`ignore_darwin.go`, `ignore_general.go`).

**Styling system:**
- Adaptive color palette using Lipgloss's `AdaptiveColor` (auto-switches values for light vs dark terminals).
- Primary accent: green `#04B575`. Error: red with cream background. Multiple gray levels for visual hierarchy.
- Styles defined as functional variables -- `lipgloss.NewStyle().Foreground(color).Render` assigned directly as render functions.
- Tab styles for active/inactive states, subtle styles for secondary info, pagination styles.

**Keybindings (pager):**
- `j/k` or arrows: line-by-line scroll
- `d/u`: half-page scroll
- `b/f`: full-page scroll
- `g/G` or Home/End: jump to start/end
- `e`: open in editor
- `c`: copy to clipboard
- `?`: toggle help overlay
- `esc`: return to file browser

**Keybindings (file browser):**
- `j/k` or arrows: navigate list
- `/`: fuzzy search/filter
- `enter`: open selected document
- `e`: open in editor
- `?`: toggle help
- `q`: quit

**Historical note on cloud features:**
Glow v1 included a "stash" feature backed by Charm Cloud -- users could save/sync markdown documents to a remote server. This was removed in v2.0.0 as a clean break. The `stash` terminology remains in the codebase but now refers to the local file browser only.

## Relevance to Notebook

Several patterns from Glow are directly applicable:

- **TUI file browser with fuzzy search** -- The pattern of discovering markdown files in a directory tree and presenting them in a filterable list is exactly what a CLI note manager needs. The `sahilm/fuzzy` integration for instant filtering is worth adopting.
- **Bubble Tea architecture** -- The state machine approach (browser view vs document view) with shared state in a `commonModel` is a clean pattern for a multi-view TUI. Worth studying as a reference even if Notebook uses a different language.
- **Glamour rendering** -- If building in Go, Glamour is the obvious choice for rendering notes. If in another language, its approach (JSON-configurable stylesheets, light/dark auto-detection) is worth emulating.
- **File watching with auto-reload** -- Useful for a note manager where the user might edit in an external editor and want to see changes reflected immediately.
- **Editor integration** -- The `e`-to-edit pattern (launch `$EDITOR`, return to viewer on close) is essential for a note manager.
- **Clipboard copy** -- One-key copy of note contents is a good UX pattern.
- **Dual-mode operation** -- CLI mode for scripting/piping + TUI mode for browsing is a strong design. Notes should be renderable to stdout for use in pipelines (`notebook show <note> | ...`) and also browsable interactively.
- **YAML config with sensible defaults** -- The config approach (auto-detect everything, override with config file or flags) is user-friendly.
- **What NOT to copy** -- Glow's cloud sync was removed, suggesting that for a personal tool, local-first with optional git-based sync is the better path. Avoid building bespoke sync infrastructure.
