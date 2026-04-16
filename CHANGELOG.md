# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Tables: Home/End now jump between cells when already at the cell's line start/end.
- Tables: Alt+Left/Right jump to the previous/next cell when the cursor is at the cell's start/end (previously only plain Left/Right did this — Alt variants now skip the walk-to-edge step).
- Tables: Alt+Up/Down move the focused cell up/down within the table when the cursor is on the first/last line of the cell, falling through to block-swap only on the top/bottom row.
- Tables: Enter on a fully empty row exits the table and drops the row.
- Tables: Alt+Shift+Backspace / Alt+Shift+D delete row / column.
- Definition lookup (`:`) now searches definitions across every note in every notebook, not just the current one. Cross-note matches are read-only and labelled with their `notebook/note` source.
- Browser: `c` now copies the contents of the highlighted recent entry to the clipboard, matching the existing notebook-level `c` binding.

### Changed
- Tables: Alt+Backspace / Alt+D now delete word (matching all other blocks); row/column delete moved to the Alt+Shift variants above.
- View mode (Ctrl+R) trims leading and trailing empty paragraph blocks so accidental top/bottom whitespace doesn't pad the rendered view. The file on disk is untouched.
- Command palette (/) and other pickers now keep a constant footer height as you filter, so the layout no longer jumps.

### Fixed
- Inline markdown (**bold**, *italic*, __underline__, ~~strike~~) now renders correctly when a delimiter pair is split across wrapped lines.
- Alt+Up / Alt+Down on a focused table no longer duplicates the entire table into cell (0,0).
- Undo no longer needs to be pressed twice in tables. Previously cursor-only key presses inside a cell registered as edits because the dirty-check compared per-cell text against the full serialized table.

## [1.2.1] - 2026-04-09

### Added
- Placeholder text for empty blocks: callout ("Empty callout"), definition ("Term" / "Definition"), code block ("language"), and embed ("Link or note path") now show placeholder hints in edit, inactive, and view modes.
- Keyboard controls for browser inputs: word jump (Alt+Left/Right), word delete (Alt+Backspace), line start/end (Home/End, Ctrl+A/E), and line delete (Ctrl+U/K) in search, rename, settings, and picker inputs.
- Picker inputs support Alt+Backspace (delete word) and Ctrl+U (clear filter).

### Changed
- **Unified empty-block behavior**: Enter on an empty code block, quote, callout, or embed now converts it to a paragraph (previously inserted a newline or did nothing).
- **Unified Backspace at position 0**: any non-paragraph block type converts to paragraph on Backspace (unwrap), matching list item behavior. Headings, code blocks, quotes, callouts, embeds, and definitions all behave consistently. A second Backspace merges or deletes the paragraph.
- Code block language cursor and all placeholder cursors use plain reverse styling instead of accent color.
- Default `hide_checked` setting changed from `true` to `false`.
- Table headers no longer use forced bold styling or "Header 1" / "Header 2" default text.
- Definition list Enter on term line moves cursor to existing definition line instead of inserting a duplicate.
- Code block cursor starts on the language line when created via palette.
- Navigating up into a table places the cursor at the last data cell instead of the top-left.

### Fixed
- Browser input fields (search, filter, rename, settings, palette) now support standard cursor controls instead of character-by-character only.
- Code block Backspace at position 0 no longer merges newline content into the previous block.
- Empty embed blocks can now be dismissed with Enter (converts to paragraph).

## [1.2.0] - 2026-04-07

### Added
- Table block type with cell editing, per-column widths, row/column operations.
- Callout block type with five admonition variants (Note, Tip, Important, Warning, Caution).
- Definition list block type with term/definition pairs and lookup palette.
- Embed block type for cross-referencing notes inline.
- Ctrl+X as universal interact/activate key.
- Large ASCII-art headings in view mode.
- Directory browser mode for filesystem browsing with split-pane preview.
- Block type transformation via palette on non-empty blocks.
