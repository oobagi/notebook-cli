# Changelog

All notable changes to this project will be documented in this file.

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
