# Roadmap

This roadmap orders open work by leverage and dependency. Each item links to a GitHub issue with full details. Checkboxes are updated automatically when issues are closed or reopened.

## Phase 1: Foundation

- [ ] Project scaffolding and CI setup
- [ ] Core data model and filesystem storage
- [ ] CLI framework with Cobra

## Phase 2: Core Commands

- [ ] `notebook new` — create notes and notebooks
- [ ] `notebook list` — list notebooks and notes
- [ ] `notebook view` — render markdown in terminal
- [ ] `notebook rm` — delete notes and notebooks
- [ ] Auto-create notebooks on note creation

## Phase 3: Editor Mode

- [ ] Basic terminal editor with Bubble Tea
- [ ] Live split-pane markdown preview
- [ ] Editor keybindings and save/quit flow

## Phase 4: Polish

- [ ] Search across notes and notebooks
- [ ] Note metadata (created/modified timestamps)
- [ ] Configuration file support (~/.config/notebook)
- [ ] Shell completions (bash, zsh, fish)
