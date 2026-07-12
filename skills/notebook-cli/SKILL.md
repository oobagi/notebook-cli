---
name: notebook-cli
description: Create, edit, reorganize, or review polished notes for notebook-cli using its round-trip-safe markdown block grammar. Use when a user asks to make or update a notebook, organize information in notebook-cli, author a managed note or compatible .md/.txt file, choose notebook-cli blocks, build a kanban/table/checklist note, or improve an existing note's structure.
---

# Notebook CLI

Create useful notebook-cli notes as plain markdown while preserving the user's storage choices and existing material.

## Load the right references

- Read [references/format.md](references/format.md) completely before creating or structurally editing a note. Treat it as the legal, round-trip-safe format contract.
- Read [references/composition.md](references/composition.md) when choosing a layout, reorganizing content, or improving the note's readability.

## Choose the target

1. Confirm whether the user means a managed notebook-cli note or an arbitrary `.md`/`.txt` file.
2. For a managed note, run `notebook config` to inspect `storage_dir`. Respect an explicit `--dir` or configured storage directory; do not assume `~/.notebook`.
3. Inspect the target note before changing it. For a new managed note, prefer `notebook <book> new "Title"` so notebook-cli performs naming and auto-creation.
4. Resolve the exact generated `.md` path from the storage directory and actual notebook listing before writing. Do not guess around punctuation or path separators.
5. If the `notebook` binary is unavailable, ask for or infer a direct file target only when the user's request makes that safe.

Do not launch the interactive browser or editor during unattended work.

## Compose the note

1. Identify the note's job: capture, explain, decide, plan, track, or reference.
2. Select only blocks that help that job. Use headings for hierarchy, paragraphs for context, lists for compact facts, checklists for actions, tables for comparisons, kanban for changing workflow state, callouts for genuinely important context, and embeds for reusable notes.
3. Keep the user's wording, facts, links, and unfinished items. Do not silently summarize away information unless the user requested condensation.
4. Emit canonical syntax from the format reference. Treat Link as inline markdown, not a persistent block type.
5. Prefer one coherent note over decorative complexity. Split into embedded notes only when the material has independent value or ownership.

## Write safely

- Preserve unrelated content and existing front-to-back order unless reorganization is requested.
- Read again immediately before patching if another process or agent may be editing the file.
- Create or edit the smallest set of files needed for the request.
- Never delete or overwrite an existing note merely because a title collides. Stop and inspect it.
- Keep managed note filenames and notebook directories in their existing slug form.

## Verify

1. Re-read the final markdown.
2. Check every structured block against `references/format.md`, especially list indentation, table row prefixes, callout markers, embed targets, and kanban fences.
3. Confirm code and kanban fences close and that unsupported syntax is not being presented as a first-class block.
4. Confirm links use `http://` or `https://` destinations and embeds reference real note slugs when the targets are available.
5. Report the exact notebook/note or file path changed and mention any unresolved target or syntax uncertainty.
