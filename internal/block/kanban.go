package block

import "strings"

// KanbanCard is a single card on a kanban board.
type KanbanCard struct {
	Text     string
	Priority Priority
	Done     bool
}

// KanbanColumn is a single column on a kanban board.
type KanbanColumn struct {
	Title string
	Cards []KanbanCard
}

// DefaultKanbanContent is the markdown body inserted when a fresh Kanban
// block is created from the command palette. Four columns starting at
// Backlog so users can park ideas before promoting them to Todo.
const DefaultKanbanContent = "## Backlog\n" +
	"- New card\n" +
	"\n" +
	"## Todo\n" +
	"\n" +
	"## In Progress\n" +
	"\n" +
	"## Done\n"

// ParseKanban interprets the inner body of a kanban fence into columns
// and cards. The body uses lightweight markdown:
//
//	## Column Title
//	- Card text             (open card)
//	- [x] Card text         (done card)
//	- !! Card text          (priority marker before text)
//	- [ ] !!! Card text     (combined: open + high priority)
//	  continuation line     (indented continuation of previous card)
//
// Lines that do not match a column header or card are treated as
// continuation lines for the most recent card if any, otherwise ignored.
// An empty body produces an empty slice.
func ParseKanban(body string) []KanbanColumn {
	var cols []KanbanColumn
	if strings.TrimSpace(body) == "" {
		return cols
	}

	var current *KanbanColumn
	var lastCard *KanbanCard
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, " \t")
		if line == "" {
			lastCard = nil // blank line breaks card continuation
			continue
		}

		// Column header: ## Title (also accept # and ###).
		if strings.HasPrefix(line, "### ") {
			cols = append(cols, KanbanColumn{Title: strings.TrimSpace(line[4:])})
			current = &cols[len(cols)-1]
			lastCard = nil
			continue
		}
		if strings.HasPrefix(line, "## ") {
			cols = append(cols, KanbanColumn{Title: strings.TrimSpace(line[3:])})
			current = &cols[len(cols)-1]
			lastCard = nil
			continue
		}
		if strings.HasPrefix(line, "# ") {
			cols = append(cols, KanbanColumn{Title: strings.TrimSpace(line[2:])})
			current = &cols[len(cols)-1]
			lastCard = nil
			continue
		}

		// Card: - text, - [ ] text, - [x] text, optionally with priority.
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			body := line[2:]
			done := false
			if strings.HasPrefix(body, "[x] ") || strings.HasPrefix(body, "[X] ") {
				done = true
				body = body[4:]
			} else if strings.HasPrefix(body, "[ ] ") {
				body = body[4:]
			}
			prio, text := ParsePriorityMarker(body)
			if current == nil {
				cols = append(cols, KanbanColumn{Title: "Untitled"})
				current = &cols[len(cols)-1]
			}
			current.Cards = append(current.Cards, KanbanCard{
				Text:     text,
				Priority: prio,
				Done:     done,
			})
			lastCard = &current.Cards[len(current.Cards)-1]
			continue
		}

		// Otherwise: treat as a continuation of the previous card. We strip
		// up to two leading spaces (the indent SerializeKanban emits) so
		// the user-visible text doesn't accumulate whitespace on round-trip.
		if lastCard != nil {
			cont := strings.TrimPrefix(line, "  ")
			if lastCard.Text == "" {
				lastCard.Text = cont
			} else {
				lastCard.Text += "\n" + cont
			}
		}
	}
	return cols
}

// SerializeKanban renders columns and cards back to the lightweight
// markdown format consumed by ParseKanban. It is the inverse of
// ParseKanban for the canonical form.
//
// "Done" is column-derived: cards in any column whose title is "Done"
// (case-insensitive) are emitted as `- [x] ...` so the markdown stays
// readable outside the app. Cards spanning multiple lines have their
// continuation lines indented two spaces, matching list-continuation
// conventions and ensuring round-trip via ParseKanban.
func SerializeKanban(cols []KanbanColumn) string {
	var lines []string
	for i, col := range cols {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "## "+col.Title)
		done := strings.EqualFold(strings.TrimSpace(col.Title), "Done")
		for _, c := range col.Cards {
			textLines := strings.Split(c.Text, "\n")
			first := textLines[0]
			// Drop fully-empty cards on save: no text and no
			// continuation lines means there's nothing to round-trip
			// (and serializing `- ` would round-trip to "no card"
			// anyway since the parser strips trailing whitespace).
			if first == "" && len(textLines) == 1 {
				continue
			}
			marker := ""
			if m := c.Priority.Marker(); m != "" {
				marker = m + " "
			}
			if done {
				lines = append(lines, "- [x] "+marker+first)
			} else {
				lines = append(lines, "- "+marker+first)
			}
			for _, cont := range textLines[1:] {
				lines = append(lines, "  "+cont)
			}
		}
	}
	return strings.Join(lines, "\n")
}
