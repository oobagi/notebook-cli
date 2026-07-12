# Composing polished notebooks

Use this reference to turn information into a note that is pleasant to scan and easy to maintain. Follow the format contract for exact syntax.

## Contents

- [Start with the note's job](#start-with-the-notes-job)
- [Default shape](#default-shape)
- [Choose blocks by meaning](#choose-blocks-by-meaning)
- [Keep the page calm](#keep-the-page-calm)
- [Reusable recipes](#reusable-recipes)
- [Final editing pass](#final-editing-pass)

## Start with the note's job

Decide what should be easier after reading the note:

- **Capture:** preserve facts, observations, or source material.
- **Explain:** build understanding in a deliberate order.
- **Decide:** compare options and record a conclusion with reasons.
- **Plan:** define outcomes, actions, owners, and sequencing.
- **Track:** show changing state through checklists or kanban.
- **Reference:** make stable information quick to retrieve.

If a note tries to do several unrelated jobs, split it only when each resulting note remains independently useful. Connect split notes with embeds.

## Default shape

Use this shape unless the material suggests something better:

```markdown
# Clear title

One or two sentences explaining the purpose or current state.

## Most important section

The content the reader came for.

## Supporting section

Details, evidence, or next actions.
```

Use one Heading 1 as the note title. Prefer Heading 2 for major sections and Heading 3 only when a section truly needs subsections. A short lead paragraph is usually more useful than a decorative callout.

## Choose blocks by meaning

- Use **paragraphs** for context, reasoning, narrative, and conclusions.
- Use **bullet lists** for parallel facts that do not have state or order.
- Use **numbered lists** only when sequence matters.
- Use **checklists** for actions someone can complete.
- Use **tables** for repeated fields or direct comparisons, usually three or more rows or columns.
- Use **quotes** for attributed language or source excerpts.
- Use **callouts** for warnings, constraints, tips, or decisions that would be costly to miss.
- Use **code blocks** for commands, code, configuration, or exact machine-readable text.
- Use **kanban** only when cards will move between states over time.
- Use **embeds** when another note owns the detail and should stay reusable.
- Use **dividers** for a real shift in context, not between every section.
- Use **inline links** where the label explains why the destination matters.

Do not turn informational bullets into checkboxes. Do not use a table for prose. Do not build a kanban for a fixed sequence that a checklist communicates better.

## Keep the page calm

- Put the most useful section first.
- Keep section names concrete: `Decision`, `Next actions`, `Risks`, `Evidence`, `Open questions`.
- Keep list items grammatically parallel.
- Prefer one strong callout over several competing callouts.
- Keep tables narrow enough to read in a terminal; move lengthy explanations below them.
- Keep kanban card text action-oriented and concise. Put supporting detail on continuation lines.
- Avoid empty sections, repeated summaries, ornamental dividers, and deep heading trees.
- Preserve uncertainty. Label assumptions and open questions instead of converting them into facts.

## Reusable recipes

### Meeting note

```markdown
# Product Review — 2026-07-12

Purpose and attendees in one short paragraph.

## Decisions

- Decision and reason

## Notes

- Important discussion point

## Actions

- [ ] Owner — action — due date

## Open questions

- Question that still needs an answer
```

### Project plan

```markdown
# Project Name

Desired outcome and current status.

> [!IMPORTANT]
> The constraint that governs the plan.

## Outcomes

- Observable result

## Milestones

1. First meaningful checkpoint
2. Second meaningful checkpoint

## Next actions

- [ ] !!! Highest-risk action
- [ ] Regular action

## Risks

| Risk | Impact | Mitigation |
| :--- | :---: | :--- |
| Example | High | Concrete response |
```

Use kanban instead of `Next actions` only when the project needs ongoing state movement.

### Research brief

```markdown
# Research Question

## Bottom line

A direct answer with the present confidence level.

## Evidence

### Finding

Explanation with an inline [source](https://example.com).

## Tensions and gaps

- Conflicting evidence or missing information

## Next research

- [ ] Specific follow-up
```

### Decision record

```markdown
# Decision: Short name

## Decision

The chosen option in one direct paragraph.

## Context

Why a decision was needed.

## Options

| Option | Benefits | Costs |
| :--- | :--- | :--- |
| A | ... | ... |
| B | ... | ... |

## Consequences

- What becomes easier
- What tradeoff remains
```

### Reference note

```markdown
# Topic

One-sentence definition or purpose.

## Quick reference

| Item | Meaning |
| :--- | :--- |
| Term | Concise explanation |

## Details

Explanatory paragraphs and examples.

## Related notes

![[reference/related-topic]]
```

## Final editing pass

Before writing:

1. Confirm the title names the actual subject.
2. Remove sections that contain no useful content.
3. Put the answer, decision, state, or next action near the top.
4. Convert blocks whose form does not match their meaning.
5. Check that actions have clear verbs and owners when known.
6. Check that sources and embeds have descriptive context.
7. Read the note once as a terminal-sized page; shorten wide tables and oversized cards.
8. Preserve any source material the user did not authorize you to discard.
