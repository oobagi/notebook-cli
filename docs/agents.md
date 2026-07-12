# Using notebook-cli with agents

notebook-cli stores ordinary markdown files, so coding agents can create and edit notes without driving the interactive TUI. The repo-owned [`notebook-cli` skill](../skills/notebook-cli/SKILL.md) teaches an agent how to locate notes, use notebook-cli's first-class block grammar, preserve existing material, and choose an appropriate structure.

## Install the skill

For Codex, symlink the skill into the personal skills directory so updates in this checkout remain the source of truth:

```bash
mkdir -p "${CODEX_HOME:-$HOME/.codex}/skills"
ln -s "$(pwd)/skills/notebook-cli" "${CODEX_HOME:-$HOME/.codex}/skills/notebook-cli"
```

Run that command from the repository root. If the destination already exists, remove or rename it only after confirming it is not a separate customized skill.

Other agents can consume the same `SKILL.md` and references when they support repository or user skills. If they do not, point their project instructions at the two reference files directly.

## What the skill contains

- [`SKILL.md`](../skills/notebook-cli/SKILL.md) defines when the skill should trigger and the safe authoring workflow.
- [`format.md`](../skills/notebook-cli/references/format.md) is the exact agent-facing contract for persistent blocks, inline syntax, storage names, and round-trip behavior.
- [`composition.md`](../skills/notebook-cli/references/composition.md) gives restrained layout guidance and reusable note recipes.

The detailed knowledge lives in the references instead of being repeated in AGENTS.md. This keeps routine repository context small and gives note-authoring agents the full contract only when they need it.

## Example requests

```text
Use $notebook-cli to turn these interview notes into a research brief.
Create a project notebook with a roadmap note and a separate decisions note.
Clean up my existing weekly review note without deleting any content.
Make a kanban note for this release plan using only legal notebook-cli syntax.
```

Agents should inspect an existing note before editing it, avoid opening the interactive TUI during automation, and report the exact file or managed notebook/note they changed.
