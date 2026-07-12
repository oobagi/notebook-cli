# notebook-cli format contract

Use this reference when creating or structurally editing notebook-cli notes. It describes the canonical, round-trip-safe subset understood by the block parser and serializer.

## Contents

- [Storage model](#storage-model)
- [Block boundaries](#block-boundaries)
- [Persistent block types](#persistent-block-types)
- [Inline syntax](#inline-syntax)
- [Canonicalization and unsupported syntax](#canonicalization-and-unsupported-syntax)
- [Complete example](#complete-example)

## Storage model

Managed notes live at `<storage_dir>/<notebook>/<note>.md`. The default storage directory is `~/.notebook`, but `notebook config` or `--dir` can override it.

Notebook and note names are stored as slugs. CLI creation lowercases names, changes runs of spaces, tabs, or underscores to `-`, and trims trailing `-`. Display names replace `-` with spaces. Prefer the CLI for creation and inspect the resulting path before writing; do not invent paths containing `/`, `\`, or `..`.

Notes remain plain markdown. Unsupported Markdown usually remains readable paragraph text, but only the grammar below receives first-class block editing and canonical serialization.

## Block boundaries

- Start headings, quotes, callouts, tables, code fences, kanban fences, dividers, and embeds at column zero.
- Separate conceptual blocks with one blank line. Blank lines are preserved as empty paragraph blocks.
- Consecutive ordinary lines form one multiline paragraph until a recognized block start appears.
- Indent list items with exactly four spaces per nesting level. Canonical output uses spaces, not tabs.
- Close every fenced block. An unclosed fence consumes the rest of the note as code or kanban content.

## Persistent block types

notebook-cli has 14 persistent block types when Heading 1, Heading 2, and Heading 3 are counted separately.

### Paragraph

Use ordinary text. Consecutive lines remain one paragraph block.

```markdown
A short paragraph with a second line
that belongs to the same block.
```

### Headings 1-3

Use a hash, a space, and the title. Heading 4 and deeper are not first-class heading blocks.

```markdown
# Note title
## Section
### Subsection
```

### Bullet list

Use `- ` or `* `. Canonical output uses `- `.

```markdown
- Parent item
    - Nested item
        - Nested two levels
```

### Numbered list

Use digits followed by `. `. Numbers are recalculated when notebook-cli serializes consecutive numbered items.

```markdown
1. First step
2. Second step
    1. Nested step
```

Do not depend on handwritten numbering remaining unchanged.

### Checklist

Use `- [ ] ` for open and `- [x] ` for complete. Uppercase `X` is accepted and canonicalized to lowercase.

```markdown
- [ ] Open task
- [x] Completed task
    - [ ] Nested task
```

An optional priority marker may appear immediately after the checkbox:

```markdown
- [ ] ! Low priority
- [ ] !! Medium priority
- [ ] !!! High priority
```

The priority marker is recognized only when followed by a space.

### Code block

Use a triple-backtick fence with an optional language identifier.

````markdown
```go
fmt.Println("hello")
```
````

The language is stored as the code block's title and drives syntax highlighting. Use ` ``` ` with no language for plain text.

### Table

Start every row with `|`. Use a header row and GFM separator row; columns may be left, center, or right aligned.

```markdown
| Item | Owner | Status |
| :--- | :---: | ---: |
| Draft | Sam | Ready |
```

Canonical output pads cells and adds a separator after the first row if one is missing. Keep each row's column count consistent. Literal `|` characters inside cells are not escaped by the parser, so rewrite them or use another block.

### Quote

Prefix every line with `> `. Use bare `>` for a blank quoted line.

```markdown
> The important thing is to keep the important thing important.
> — Stephen Covey
```

### Callout

Begin a quote block with one supported marker. Prefix every body line with `> `.

```markdown
> [!IMPORTANT]
> Confirm the migration before removing the old files.
```

Supported variants are `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, and `CAUTION`, case-insensitively. Unknown variants remain ordinary quotes.

### Divider

Use `---`. Runs of at least three `-`, `*`, or `_` are accepted and canonicalized to `---`.

```markdown
---
```

### Embed

Put an embed on its own line with an exact note path.

```markdown
![[projects/release-decisions]]
```

Use `notebook-slug/note-slug` across notebooks. A single `note-slug` resolves within the current notebook. Prefer explicit cross-notebook paths and confirm the target exists.

### Kanban

Use a `kanban` fence. Columns are headings; cards are bullet items.

````markdown
```kanban
## Backlog
- [feature] Draft onboarding flow

## Todo
- !!! [bug] Fix login redirect

## In Progress
- [documentation] Update setup guide
  Add Linux instructions too.

## Done
- [x] Publish release notes
```
````

Legal column markers are `# `, `## `, or `### `; canonical output uses `## `. A card may use `- ` or `* ` and may include, in this order:

1. Optional `[ ] ` or `[x] ` checkbox.
2. Optional priority: `! `, `!! `, or `!!! `.
3. Optional tag: `[bug] `, `[feature] `, `[documentation] `, or `[question] `.
4. Card text.

`[enhancement]` is accepted as `feature`, and `[docs]` is accepted as `documentation`; canonical output uses the long names. Indent continuation lines by two spaces. A blank line ends continuation.

Completion is column-derived on save: cards in a column named `Done` case-insensitively serialize with `[x]`; cards in other columns serialize open. Put completed cards in `Done` instead of relying on a checked marker elsewhere.

## Inline syntax

The renderer supports these inline forms inside paragraphs, headings, list items, checklists, and quotes:

```markdown
**bold**
*italic*
~~strikethrough~~
__underline__
[Notebook docs](https://example.com/docs)
```

Inline links require an `http://` or `https://` URL. Link is an editor creation affordance, not a persistent block: after saving, `[label](url)` remains ordinary inline markdown inside its containing block. A standalone link reloads as a paragraph.

Do not assume images, footnotes, task metadata, HTML, YAML front matter, definition lists, Heading 4+, or arbitrary fenced extensions have special editor behavior. They may remain as text or be interpreted as a paragraph/code block.

## Canonicalization and unsupported syntax

Opening and saving a note may intentionally normalize syntax:

- Bullet markers become `-`.
- Checked boxes become lowercase `[x]`.
- Numbered lists are resequenced.
- Dividers become `---`.
- Tables are padded and receive a separator row.
- Kanban headings, aliases, completion markers, spacing, and continuations are normalized.
- Link affordances serialize to ordinary inline markdown.

Avoid relying on exact whitespace inside structured blocks. Preserve unusual or unsupported Markdown only when the user values its source form, and warn that notebook-cli may treat it as plain text or normalize it after editing.

## Complete example

````markdown
# Release Plan

A compact plan for shipping version 2.0.

> [!IMPORTANT]
> Do not publish until the migration rehearsal passes.

## Outcomes

- Existing notes open without conversion errors
- Search results remain accurate
- [ ] !!! Run the final backup
- [ ] Publish the release

## Owners

| Area | Owner | Ready |
| :--- | :---: | ---: |
| Migration | Alex | Yes |
| Docs | Riley | No |

## Workflow

```kanban
## Todo
- !! [documentation] Finish upgrade guide

## In Progress
- !!! [bug] Verify migrated links

## Done
- [x] Tag the release candidate
```

## Supporting notes

![[releases/version-2-decisions]]

See the [release checklist](https://example.com/releases/2.0) for external sign-off.
````
