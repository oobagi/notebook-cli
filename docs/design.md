# CLI Design System

The rules that make notebook feel like one cohesive tool. Every command, every color, every symbol follows this document.

---

## Command Hierarchy

Commands follow a **noun-verb** pattern. The mental model is: navigate to a thing, then act on it.

```
notebook [resource] [sub-resource] [verb] [flags]
```

### Full Command Tree

```
notebook                              # Launch TUI browser (browse everything)
notebook list                         # List all books
notebook new "Book Name"              # Create a book
notebook delete "Book Name"           # Delete a book (requires confirmation)

notebook <book>                       # List notes in a book
notebook <book> list                  # List notes in a book
notebook <book> new "Note Title"      # Create a note (auto-creates book if needed)
notebook <book> <note>                # View a note (rendered markdown)
notebook <book> <note> edit           # Open note in editor with live preview
notebook <book> <note> delete         # Delete a note
notebook <book> <note> copy           # Copy note contents to clipboard

notebook search "query"               # Search across all books
notebook <book> search "query"        # Search within a book
notebook config                       # Show current config
notebook config set <key> <value>     # Set a config value
```

### Design Decisions

**Bare commands do useful things.** Every level of the hierarchy has a meaningful default:
- `notebook` → TUI browser
- `notebook work` → TUI scoped to "work"
- `notebook work meeting` → render the note

No verb required for the most common action at each level.

**Noun-verb, not verb-noun.** `notebook work list` reads like English: "notebook, work, list it." It also means `notebook work` alone can launch the TUI scoped to that book, which verb-first patterns can't do.

**`new` not `add`.** "Add" implies appending to a list. "New" implies creation.

**`delete` not `rm`.** Spell it out. The target user doesn't live in a shell. Short aliases (`rm`, `ls`) work silently but are never shown in help text or docs.

**Fuzzy matching on resource names.** `notebook wrk` resolves to `work` if unambiguous. If ambiguous, show a selection prompt. Never error with "not found" when the user was close.

**Auto-create books.** `notebook ideas new "shower thought"` creates the `ideas` book silently if it doesn't exist. No extra step, no error.

---

## Output Design

### Spacing Rules

- Left-align everything. Never center text in a terminal.
- One blank line between logical sections. Never two.
- No trailing whitespace. No leading blank lines before output.
- Two-space left indent for all list items (leaves room for selection indicators in TUI).

### List Output

`notebook list` (all books):

```
  Personal    4 notes    2h ago
  Work        12 notes   just now
  Research    1 note     3d ago
```

`notebook work list` (notes in a book):

```
  Meeting Notes       3.2 KB   10m ago
  Project Roadmap     8.1 KB   2d ago
  Weekly Standup      1.4 KB   1w ago
```

Rules:
- Columns aligned with minimum 4 spaces between.
- Natural language counts: "1 note" not "1 notes".
- No header row. The content is self-evident.
- No bullet characters in static lists — whitespace creates the grouping.

### Timestamps

Human-relative time in all list views:

| Condition           | Format       |
|---------------------|--------------|
| < 1 minute          | just now     |
| 1–59 minutes        | 10m ago      |
| 1–23 hours          | 2h ago       |
| 1–6 days            | 3d ago       |
| 7–29 days           | 2w ago       |
| 30+ days, same year | Mar 15       |
| Different year      | Mar 15, 2025 |

In detail/view mode, show both: `Mar 15, 2026 at 2:30 PM (3d ago)`

### Status Messages

```
  ✓ Created "Meeting Notes" in Work
  ✗ Book "xyz" not found. Did you mean "xylophone"?
  ! Note "Draft" has unsaved changes
  → Opening in editor...
```

| Prefix | Color  | Tense            | When                          |
|--------|--------|------------------|-------------------------------|
| `✓`    | Green  | Past ("Created") | Success confirmation          |
| `✗`    | Red    | Present          | Errors                        |
| `!`    | Yellow | Present          | Warnings, caution states      |
| `→`    | Dim    | Participle       | Transitions ("Opening...")    |

Rules:
- One line for simple operations. Never multi-line success messages.
- Errors always suggest a next action. "Not found" alone is never acceptable.
- Counts summarize batch operations: `✓ Deleted 3 notes from Work`

### Empty States

Never show blank output. Empty states are onboarding:

```
  No books yet.

  Create one with:  notebook new "My First Book"
```

No error symbol. The suggestion line is dimmed.

### Search Results

```
  Work › Meeting Notes
    ...discussed the quarterly plan and...

  Personal › Journal
    ...reminded me of the quarterly review...

  2 matches across 2 books
```

- Breadcrumb path with `›` separator
- One-line snippet (max 60 chars), matched term in **bold**, `...` on sides
- Summary count at bottom, dimmed

### Truncation

Truncate long text with `…` (U+2026). Never use `...`. Max content width: 80 characters.

---

## Color Palette

Six colors. Each has exactly one job.

| Role         | Color        | ANSI     | Usage                                         |
|--------------|--------------|----------|-----------------------------------------------|
| **Primary**  | Cyan         | `\e[36m` | Resource names, selections, links              |
| **Success**  | Green        | `\e[32m` | Confirmations, checkmarks, saved               |
| **Error**    | Red          | `\e[31m` | Errors, destructive warnings                   |
| **Warning**  | Yellow       | `\e[33m` | Caution states, unsaved changes                |
| **Muted**    | Dim          | `\e[2m`  | Metadata, timestamps, hints, secondary info    |
| **Emphasis** | Bold         | `\e[1m`  | Headings, counts, primary focus                |

### Why These

- **Cyan over blue.** Blue is unreadable on dark terminals at normal weight. Cyan works on both light and dark.
- **No magenta/purple.** Clashes with most prompt themes, hard to distinguish from blue for colorblind users.
- **Dim over hardcoded gray.** `\e[2m` inherits the user's foreground color. Hardcoded gray (`\e[90m`) is invisible on some dark themes.
- **Bold as emphasis, not a color.** Bold white works on every terminal theme.

### Accessibility

- Never convey meaning through color alone. `✓ ✗ ! →` carry meaning without color.
- Support `NO_COLOR` env var (https://no-color.org). Strip all ANSI when set. Output must remain parseable.
- No background colors in CLI output. Background colors only in TUI mode for selection highlighting.

---

## Symbols

Eight symbols. Use nothing else.

| Symbol | Unicode | Name             | Usage                                       |
|--------|---------|------------------|---------------------------------------------|
| `✓`    | U+2713  | Check mark       | Success confirmation                        |
| `✗`    | U+2717  | Ballot X         | Error                                       |
| `!`    | U+0021  | Exclamation      | Warning (plain ASCII, intentional)           |
| `→`    | U+2192  | Right arrow      | Navigation, transitions                      |
| `›`    | U+203A  | Guillemet        | Breadcrumb separator: `Work › Meeting Notes` |
| `…`    | U+2026  | Ellipsis         | Truncation                                   |
| `●`    | U+25CF  | Filled circle    | Selected/active state in TUI                 |
| `○`    | U+25CB  | Empty circle     | Unselected state in TUI (multi-select only)  |

### Excluded on purpose

- **No emoji.** Inconsistent widths across terminals, breaks column alignment.
- **No `⚠` triangle.** ASCII `!` is more readable and doesn't require font support.
- **No box-drawing in CLI output.** `┌─┐│└─┘` are reserved for TUI panels and modals only. CLI command output uses whitespace and color.

---

## Interactive Patterns

### Navigation

```
  ↑/↓        Move up/down
  Enter       Open / Confirm
  Esc         Go back one level / Cancel
  q           Quit entirely
  /           Start search (filters in real-time)
  ?           Toggle help overlay
  Ctrl+S      Save (editor)
  Ctrl+Q      Quit (editor)
```

Arrow keys are always the primary documented navigation. `Esc` is always safe — it never destroys data.

### Single Selection

```
    Personal
  ● Work
    Research
```

Active item gets `●` and renders in cyan. Inactive items get two-space indent (no `○`, that implies multi-select).

### Multi-Selection

```
  ● Meeting Notes
  ○ Project Roadmap
  ● Weekly Standup

  Space to toggle  ·  Enter to confirm  ·  Esc to cancel
```

`●`/`○` for selected/unselected. Status bar uses `·` (middle dot) as separator.

### Confirmations

**Destructive actions** (delete a book or note):

```
  Delete "Meeting Notes" from Work? This cannot be undone.

  Type the note name to confirm: _
```

Require typing the resource name. Never `y/n` for deletes.

**Significant but reversible actions** (rename, move):

```
  Rename "Draft" to "Final"? [y/N]
```

Default is always the safe option (No). Capitalized letter = default.

### Editor Cascade

When opening a note for editing, resolve the editor in this order:

1. `$NOTEBOOK_EDITOR` (app-specific override)
2. `editor` in notebook config
3. `$VISUAL`
4. `$EDITOR`
5. `micro` if installed, then `nano`
6. Error with instructions

Never fall back to `vi`/`vim`.

---

## Box Drawing

Box-drawing characters (`┌──┐│└──┘`) are used in exactly two contexts:

1. **TUI mode** — panels, modals, selection boxes in the full-screen interface.
2. **Single-resource detail views** — metadata display before opening a note.

Never in list output, tables, or standard CLI command responses.

---

## Summary

Five rules that govern everything:

1. **Whitespace over decoration.** Indentation and blank lines create structure. Color and symbols are seasoning, not the meal.

2. **Every color earns its spot.** Cyan = actionable thing. Green = done. Red = problem. Yellow = careful. Dim = context. Bold = look here.

3. **Symbols work without color.** `✓ ✗ ! →` carry meaning for colorblind users and `NO_COLOR` environments.

4. **Bare commands do useful things.** No verb needed for the most common action at every level of the hierarchy.

5. **Friction matches consequence.** `notebook new "Idea"` creates instantly. `notebook delete "Idea"` requires typing the name.
