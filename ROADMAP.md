# Roadmap

This roadmap orders open work by leverage and dependency. Each item links to a GitHub issue with full details. Checkboxes are updated automatically when issues are closed or reopened.

## Phase 1: Foundation

- [x] [#1 Project scaffolding and CI setup](https://github.com/oobagi/notebook/issues/1)
- [x] [#2 Core data model and filesystem storage](https://github.com/oobagi/notebook/issues/2)
- [x] [#3 CLI framework with Cobra](https://github.com/oobagi/notebook/issues/3)

## Phase 2: Core Commands

- [x] [#4 notebook new — create notes and notebooks](https://github.com/oobagi/notebook/issues/4)
- [x] [#5 notebook list — list notebooks and notes](https://github.com/oobagi/notebook/issues/5)
- [x] [#6 notebook view — render markdown in terminal](https://github.com/oobagi/notebook/issues/6)
- [x] [#7 notebook rm — delete notes and notebooks](https://github.com/oobagi/notebook/issues/7)
- [x] [#8 Auto-create notebooks on note creation](https://github.com/oobagi/notebook/issues/8)

## Phase 3: Editor Mode

- [x] [#9 Basic terminal editor with Bubble Tea](https://github.com/oobagi/notebook/issues/9)
- [x] [#10 Live split-pane markdown preview](https://github.com/oobagi/notebook/issues/10)
- [x] [#11 Editor keybindings and save/quit flow](https://github.com/oobagi/notebook/issues/11)
- [x] [#16 Interactive checkboxes, lists, and links in editor](https://github.com/oobagi/notebook/issues/16)

## Phase 4: Polish

- [x] [#12 Search across notes and notebooks](https://github.com/oobagi/notebook/issues/12)
- [x] [#13 Note metadata (created/modified timestamps)](https://github.com/oobagi/notebook/issues/13)
- [x] [#14 Configuration file support (~/.config/notebook)](https://github.com/oobagi/notebook/issues/14)
- [x] [#15 Shell completions (bash, zsh, fish)](https://github.com/oobagi/notebook/issues/15)

## Phase 5: Rich Experience

- [x] [#17 TUI notebook/note browser with fuzzy search](https://github.com/oobagi/notebook/issues/17)
- [x] [#18 Auto light/dark theme detection](https://github.com/oobagi/notebook/issues/18)
- [x] [#19 Clickable hyperlinks in rendered output (OSC 8)](https://github.com/oobagi/notebook/issues/19)
- [x] [#20 Clipboard copy support](https://github.com/oobagi/notebook/issues/20)
- [x] [#21 GitHub-style admonition blocks in rendering](https://github.com/oobagi/notebook/issues/21)

## Phase 6: TUI as Complete App

- [x] [#49 Strip TUI pickers from CLI commands](https://github.com/oobagi/notebook/issues/49)
- [x] [#50 TUI help overlay](https://github.com/oobagi/notebook/issues/50)
- [x] [#51 TUI inline text input and create operation](https://github.com/oobagi/notebook/issues/51)
- [x] [#52 TUI delete with type-to-confirm](https://github.com/oobagi/notebook/issues/52)
- [x] [#53 TUI rename notebooks and notes](https://github.com/oobagi/notebook/issues/53)
- [x] [#54 TUI view rendered markdown and copy to clipboard](https://github.com/oobagi/notebook/issues/54)

## Phase 7: Editor Polish

- [x] [#78 Fix editor cursor viewport + Ctrl+E preview mode](https://github.com/oobagi/notebook/issues/78)

## Phase 8: Theming

- [x] [#79 Add glamour markdown styles to theme system](https://github.com/oobagi/notebook/issues/79)
- [x] [#80 TUI theme picker with live markdown preview](https://github.com/oobagi/notebook/issues/80)

## Phase 9: TUI Fixes

- [x] [#84 Fix theme picker overlay top cut off](https://github.com/oobagi/notebook/issues/84)
- [x] [#85 Remove notty and ascii from theme picker](https://github.com/oobagi/notebook/issues/85)
- [x] [#86 Show dark/light background hints in theme picker](https://github.com/oobagi/notebook/issues/86)
- [x] [#91 Fix filter search spaces and cursor movement](https://github.com/oobagi/notebook/issues/91)

## Phase 10: CLI Color Theming

- [x] [#87 Add named UI color presets to theme package](https://github.com/oobagi/notebook/issues/87)
- [x] [#88 Add ui_theme config key](https://github.com/oobagi/notebook/issues/88)
- [x] [#89 Wire theme colors into all lipgloss styles](https://github.com/oobagi/notebook/issues/89)
- [x] [#90 Add UI theme tab to theme picker overlay](https://github.com/oobagi/notebook/issues/90)

## Phase 11: Community Glamour Styles

- [x] [#94 Embed community Glamour styles into the binary](https://github.com/oobagi/notebook/issues/94)
- [x] [#95 Wire community styles into ResolveGlamourStyle](https://github.com/oobagi/notebook/issues/95)
- [x] [#96 Show community styles in theme picker](https://github.com/oobagi/notebook/issues/96)
- [x] [#97 CI validation for community style JSON files](https://github.com/oobagi/notebook/issues/97)
- [x] [#98 Contributing guide for community styles](https://github.com/oobagi/notebook/issues/98)

## Phase 12: Block Data Model

- [ ] [#117 Fix Cmd+Backspace should delete to start of line in editor](https://github.com/oobagi/notebook/issues/117)
- [ ] [#118 Fix checklists don't work in editor](https://github.com/oobagi/notebook/issues/118)
- [ ] [#119 Block type definition and markdown parser](https://github.com/oobagi/notebook/issues/119)
- [ ] [#120 Block-to-markdown serializer with round-trip tests](https://github.com/oobagi/notebook/issues/120)

## Phase 13: Block Editor Core

- [ ] [#121 Replace textarea with block list widget](https://github.com/oobagi/notebook/issues/121)
- [ ] [#122 Block operations — create, delete, merge, reorder](https://github.com/oobagi/notebook/issues/122)

## Phase 14: Command Palette & Polish

- [ ] [#123 / command palette for block type insertion](https://github.com/oobagi/notebook/issues/123)
- [ ] [#124 Checklist toggle, visual polish, help overlay update](https://github.com/oobagi/notebook/issues/124)
