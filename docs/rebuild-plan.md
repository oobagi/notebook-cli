# Notebook Rebuild Plan

This plan is for a ground-up local rebuild of the TUI app while preserving the current storage format and the behaviors that already feel right. The goal is not a visual redesign first. The goal is a cleaner architecture where editor mode is a real markdown editing surface, view mode is the polished interactive surface, and components share deterministic APIs for rendering, focus, scrolling, hit testing, undo, and dialogs.

## Current Diagnosis

The current codebase has accreted behavior in a few large files:

- `internal/editor/editor.go` is the document model, input router, mode manager, overlay stack, undo manager, persistence bridge, and scroll coordinator.
- `internal/editor/render.go` contains active editor rendering, inactive editor rendering, and read-mode rendering as separate switch-heavy paths.
- `internal/editor/table.go` and `internal/editor/kanban.go` each implement their own edit state, rendering, cursor movement, serialization, and scroll rules.
- `internal/browser/browser.go` has a separate shell implementation for chrome, status, footer inputs, settings, help, and theme picking.
- `internal/ui` has useful primitives, but some are placement-specific, especially footer pickers and footer inputs.

The dominant problem is parallel truth. The editor stores `[]block.Block`, one `textarea.Model` per block, active table state, active kanban state, view-mode line offsets, embed-modal block state, hover state, and undo snapshots. Many code paths must remember to sync these sources manually.

The second problem is duplicated geometry. Rendering computes visual output, but hit testing and scroll anchoring often recompute wrapping and line offsets separately. That is why checklist links, table cells, kanban cards, embeds, and modal content are easy to break independently.

The third problem is implicit mode priority. Help, quit prompt, palette, link prompt, embed modal, definition preview, view mode, kanban mode, table mode, and textarea editing are all ordered by branches inside `Update`.

## Product Direction

### Editor Mode

Editor mode should feel like a markdown text editor.

- Raw markdown source is the source of truth.
- Inline formatting syntax remains visible while editing.
- `/link` should insert markdown text like `[](https://)`, not create a custom link wizard.
- Rich components can still have editor affordances, but the default should be raw markdown text first.
- The editor should avoid mouse capture unless a focused component explicitly needs it, so terminal selection still works.

### View Mode

View mode should feel like the complete rendered app.

- Markdown is rendered.
- Links look and behave like links.
- Checklists, embeds, tables, kanbans, and future widgets expose explicit hit targets.
- All user-visible mutations in view mode go through the same undo/redo transaction layer as edit mode.
- Mouse hit testing is based on render output metadata, not separately reconstructed coordinates.

## Component Taxonomy

Target block model:

- `TextBlock`: paragraph and heading, with optional heading level.
- `ListItem`: bullet, ordered, and task/checklist, with indent, checked state, ordinal, and priority.
- `QuoteBlock`: quote and callout/admonition as one component; callout is `QuoteBlock{Variant}`.
- `FenceBlock`: fenced code and fenced components. `kanban` is a fenced component.
- `TableBlock`: GFM pipe table with rows, alignment, and original raw source preservation.
- `ThematicBreak`.
- `EmbedBlock`.

Recommended decision:

- Keep definition-style `: text` syntax as ordinary paragraph text. No dedicated Definition block or migration layer.
- Merge Quote and Callout.
- Keep Kanban, Table, Embed, Checklist, Code, Divider, lists, headings, paragraphs.
- Treat standalone `[title](url)` and bare URLs as normal paragraph markdown. Inline markdown links are the link model.

## Architecture

### Packages

Proposed package layout:

- `internal/app`: top-level Bubble Tea host and screen switching.
- `internal/shell`: chrome, layout, status, dialogs, overlay stack, mouse policy.
- `internal/document`: markdown source, parsed projection, transactions, undo/redo.
- `internal/markdown`: block parser, inline parser, source ranges, serializer/preserver.
- `internal/render`: render primitives, layout tree, hit boxes, ANSI-safe measurement.
- `internal/components`: block/component controllers for text, list, quote, table, kanban, embed, code, divider.
- `internal/editor2`: raw markdown editor screen.
- `internal/viewer`: rendered interactive note screen.
- `internal/browser`: browser screen, gradually adapted to `internal/shell`.

This can be done incrementally; the package names can be adjusted before implementation.

### Shell API

The shell should own layout and overlays so browser/editor do not each hand-roll header/footer/status math.

```go
type ViewSpec struct {
    Header *HeaderSpec
    Body   BodySpec
    Footer *FooterSpec
    Dialog *DialogSpec
    Status StatusSpec
    MouseMode tea.MouseMode
}

type DialogSpec struct {
    ID        string
    Kind      DialogKind
    Placement Placement
    Title     string
    Body      Renderable
    Actions   []Action
    Input     *InputSpec
    Picker    *PickerSpec
}
```

Dialog routing gets first chance at keys, paste, and mouse. Unsaved exit, destructive delete, save-before-jump, picker selection, input prompts, and help should all use the same overlay stack. Full-screen or centered confirm dialogs replace footer-only warnings.

### Document API

Raw markdown should be the editable source. Parsed blocks are a projection used for view rendering and component interactions.

```go
type Document struct {
    Source string
    Blocks []Block
}

type Block struct {
    ID     string
    Kind   BlockKind
    Range  SourceRange
    Raw    string
    Props  BlockProps
    Inline []InlineNode
}

type Transaction struct {
    Label   string
    Before  Snapshot
    Apply   func(*Document) error
    Effects []Effect
}
```

All mutations go through transactions. Text edits can be coalesced. Structural actions, checklist toggles, table edits, kanban moves, embed edits, and sort actions are discrete undo entries.

### Component API

Every component should speak the same controller language.

```go
type ComponentController interface {
    Render(RenderContext) RenderResult
    HandleKey(KeyContext, tea.KeyPressMsg) ComponentResult
    HandleMouse(MouseContext, tea.MouseMsg) ComponentResult
    MoveFocus(Direction, EntryEdge) NavResult
    Serialize() SourcePatch
}

type RenderResult struct {
    Lines           []string
    Boxes           []HitBox
    Cursor          *Rect
    PreferredScroll *Rect
}

type HitBox struct {
    Rect          Rect
    BlockID       string
    ComponentPath string
    Role          HitRole
    Payload       any
    Z             int
}
```

Hit boxes should cover inline links, checkbox markers, embed cards, table cells, kanban cards, modal backdrops, picker rows, and buttons. Mouse handling uses hit boxes emitted by the last render.

Focus should be path-based, not index-only:

- `block:<id>`
- `table:<id>/cell/<row>/<col>`
- `kanban:<id>/card/<col>/<card>`
- `kanban:<id>/editor/<col>/<card>`
- `dialog:<id>/input`
- `picker:<id>/row/<n>`

Scroll containers should also be named and stable:

- `document`
- `view`
- `dialog:<id>`
- `table:<id>`
- `kanban:<id>/col/<n>`

This removes index-shifting maps like the current kanban offset storage.

## Terminal UI Policy

Bubble Tea v2 supports setting alternate screen, mouse mode, focus reporting, and keyboard enhancements on `tea.View`. The current app already uses `tea.NewView(content)` and sets `AltScreen = true`; view mode sets mouse capture. The rebuild should make that explicit in `internal/shell`.

Policy:

- Use alternate screen for browser and editor screens.
- Use `MouseModeNone` in edit mode by default.
- Use `MouseModeAllMotion` only when hover feedback is needed, usually view mode and dialogs with hoverable hit boxes.
- Use `MouseModeCellMotion` for click-only surfaces where hover is not needed.
- Keep bracketed paste enabled unless a specific view opts out.
- Use viewport components where possible for scroll containers; custom scroll math must be backed by render geometry.

Reference docs:

- Bubble Tea v2 `tea.View` supports `AltScreen`, `MouseMode`, `ReportFocus`, and layered rendered content: https://pkg.go.dev/charm.land/bubbletea/v2
- Bubbles viewport supports soft wrap, mouse wheel, explicit width/height, offsets, and visibility helpers: https://pkg.go.dev/charm.land/bubbles/v2/viewport

## Test And Benchmark Strategy

Before replacing the current app, build a dual-app harness.

### Baseline

Fix test hermeticity first. Some tests can read real user config unless `HOME` and `XDG_CONFIG_HOME` are isolated.

Useful commands:

```bash
HOME="$(mktemp -d)" XDG_CONFIG_HOME="$(mktemp -d)" NO_COLOR=1 go test ./...
HOME="$(mktemp -d)" XDG_CONFIG_HOME="$(mktemp -d)" go vet ./...
```

### Golden Fixtures

Create deterministic fixtures under `testdata/corpus`:

- block zoo note
- long note
- many notebooks and notes
- unicode and wide characters
- kanban-heavy note
- embeds
- inline links
- standalone links as paragraph text
- tables
- empty states

Render fixed frames at `40x16`, `80x24`, and `120x40`. Store ANSI-aware goldens and stripped layout goldens. Normalize absolute paths, timestamps, cursor blink state, and transient status text.

### Dual-App Comparison

Build both binaries:

```bash
go build -o .tmp/notebook-current ./cmd/notebook
go build -o .tmp/notebook-next ./cmd/notebook
```

The second command runs inside the rebuild worktree. The harness should copy the same fixture store to two temp dirs, run identical commands, and compare exit code, stdout, stderr, resulting markdown, config, and recents.

PTY scenarios should cover:

- browser L0/L1 navigation
- search
- help
- settings
- theme picker
- create/rename/delete
- editor open/save/quit
- view mode
- checklist toggle and undo
- inline link hover/click
- embed modal
- table navigation
- kanban navigation/edit/move/sort

### Benchmarks

Add benchmarks for:

- markdown parse and render
- serialize unchanged source
- storage list/search on large stores
- browser search and preview rendering
- editor raw render
- view render and hit-box generation
- table render/navigation
- kanban render/navigation

Performance gate: no more than an agreed regression, for example 10 percent mean or p95 on the same machine, unless a deliberate tradeoff is documented.

## Migration Plan

### Phase 0: Freeze Current Behavior

- Keep the current dirty work separate from the rebuild.
- Add hermetic config setup to tests.
- Add characterization tests for the current tricky behavior: view-mode links/checklists, embed undo/save, table cursor moves, kanban scroll/selection, quit prompts, and picker routing.
- Add a small golden frame harness before changing architecture.

### Phase 1: Shell Extraction

- Add `internal/shell`.
- Move header/status/footer layout into shell specs.
- Convert browser help/settings/input/delete prompts to shell overlays.
- Convert editor quit prompt, help, pickers, link prompt, and embed sheet to shell overlays.
- Introduce full-screen/center confirm dialogs for exit/save/delete.

### Phase 2: Document And Markdown Core

- Introduce `internal/document` with source-first `Document`.
- Parser returns blocks with source ranges and stable IDs.
- Serializer preserves unchanged raw source.
- Implement inline parser once and use it everywhere.
- Keep removed definition syntax as ordinary markdown text.
- Merge Quote and Callout internally.

### Phase 3: Raw Editor

- Replace per-block textareas with a raw markdown editing surface.
- Keep current markdown shortcuts as source edits, not hidden block transformations.
- `/` palette inserts markdown snippets or component templates.
- Keep component-specific editing only where it is clearly better, such as kanban/table optional focused controllers.

### Phase 4: Rendered Viewer

- Build view rendering from parsed blocks.
- Every render pass emits `RenderResult` with hit boxes.
- Links, checkbox toggles, embeds, table cells, and kanban cards use hit boxes.
- View-mode mutations use document transactions and main undo/redo.

### Phase 5: Components

- Port table to a component controller.
- Port kanban to a component controller.
- Port embeds to nested document/view controller with explicit save results.
- Remove global component state from the top-level editor model.

### Phase 6: Dual-App Validation

- Create a rebuild worktree, for example:

```bash
git worktree add ../notebook-rebuild -b rebuild/v2
```

- Build current and candidate binaries.
- Run unit tests, golden frame tests, PTY tests, file-tree diffs, and benchmarks.
- Replace the current app only when parity gates pass.

## Acceptance Gates

- `go test ./...` and `go vet ./...` pass with isolated `HOME` and `XDG_CONFIG_HOME`.
- Existing markdown files open and save without data loss.
- Existing config and recents continue to work.
- Current behavior tests pass or intentional changes are explicitly documented.
- Visual goldens either match or have reviewed diffs.
- View-mode hit boxes match rendered output.
- Undo/redo works for text edits, checklist toggles, table changes, kanban changes, embed edits, sort actions, and view-mode interactions.
- Runtime harness shows no unacceptable performance regression.

## Open Decisions

- Whether table and kanban should be raw markdown only in editor mode, optional interactive edit components, or full interactive components in both modes.
- Whether browser and editor should remain separate Bubble Tea programs or become screens inside one app host.
- Exact visual treatment for full-screen confirm dialogs.
