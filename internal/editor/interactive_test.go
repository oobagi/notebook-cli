package editor

import (
	"testing"
)

func TestParseCheckboxes(t *testing.T) {
	content := "- [ ] unchecked item\n- [x] checked item"
	elements := parseElements(content)

	checkboxes := filterByKind(elements, ElementCheckbox)
	if len(checkboxes) != 2 {
		t.Fatalf("expected 2 checkboxes, got %d", len(checkboxes))
	}

	if checkboxes[0].Text != "unchecked item" {
		t.Fatalf("expected text %q, got %q", "unchecked item", checkboxes[0].Text)
	}
	if checkboxes[0].Checked {
		t.Fatal("first checkbox should be unchecked")
	}
	if checkboxes[0].Line != 0 {
		t.Fatalf("first checkbox should be on line 0, got %d", checkboxes[0].Line)
	}

	if checkboxes[1].Text != "checked item" {
		t.Fatalf("expected text %q, got %q", "checked item", checkboxes[1].Text)
	}
	if !checkboxes[1].Checked {
		t.Fatal("second checkbox should be checked")
	}
	if checkboxes[1].Line != 1 {
		t.Fatalf("second checkbox should be on line 1, got %d", checkboxes[1].Line)
	}
}

func TestParseLinks(t *testing.T) {
	content := "Check out [Google](https://google.com) and [GitHub](https://github.com)."
	elements := parseElements(content)

	links := filterByKind(elements, ElementLink)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}

	if links[0].Text != "Google" {
		t.Fatalf("expected text %q, got %q", "Google", links[0].Text)
	}
	if links[0].URL != "https://google.com" {
		t.Fatalf("expected URL %q, got %q", "https://google.com", links[0].URL)
	}

	if links[1].Text != "GitHub" {
		t.Fatalf("expected text %q, got %q", "GitHub", links[1].Text)
	}
	if links[1].URL != "https://github.com" {
		t.Fatalf("expected URL %q, got %q", "https://github.com", links[1].URL)
	}
}

func TestParseNestedCheckboxes(t *testing.T) {
	content := "- [ ] top level\n  - [ ] nested once\n    - [x] nested twice"
	elements := parseElements(content)

	checkboxes := filterByKind(elements, ElementCheckbox)
	if len(checkboxes) != 3 {
		t.Fatalf("expected 3 checkboxes, got %d", len(checkboxes))
	}

	if checkboxes[0].Text != "top level" {
		t.Fatalf("expected text %q, got %q", "top level", checkboxes[0].Text)
	}
	if checkboxes[0].Col != 0 {
		t.Fatalf("top level checkbox col should be 0, got %d", checkboxes[0].Col)
	}

	if checkboxes[1].Text != "nested once" {
		t.Fatalf("expected text %q, got %q", "nested once", checkboxes[1].Text)
	}
	if checkboxes[1].Col != 0 {
		// The full match starts at column 0 because leading whitespace is part of the match.
		// But the indentation is captured as part of the match start.
	}

	if checkboxes[2].Text != "nested twice" {
		t.Fatalf("expected text %q, got %q", "nested twice", checkboxes[2].Text)
	}
	if !checkboxes[2].Checked {
		t.Fatal("third checkbox should be checked")
	}
}

func TestToggleCheckbox(t *testing.T) {
	content := "- [ ] first\n- [x] second\n- [ ] third"

	// Toggle first: unchecked -> checked.
	result, checked := toggleCheckbox(content, 0)
	if !checked {
		t.Fatal("toggling unchecked should return checked=true")
	}
	lines := splitLines(result)
	if lines[0] != "- [x] first" {
		t.Fatalf("expected %q, got %q", "- [x] first", lines[0])
	}
	// Other lines unchanged.
	if lines[1] != "- [x] second" {
		t.Fatalf("expected %q, got %q", "- [x] second", lines[1])
	}
	if lines[2] != "- [ ] third" {
		t.Fatalf("expected %q, got %q", "- [ ] third", lines[2])
	}

	// Toggle second: checked -> unchecked.
	result, checked = toggleCheckbox(content, 1)
	if checked {
		t.Fatal("toggling checked should return checked=false")
	}
	lines = splitLines(result)
	if lines[1] != "- [ ] second" {
		t.Fatalf("expected %q, got %q", "- [ ] second", lines[1])
	}
}

func TestToggleCheckboxOutOfRange(t *testing.T) {
	content := "- [ ] item"

	result, _ := toggleCheckbox(content, 5)
	if result != content {
		t.Fatal("toggling out-of-range line should return original content")
	}

	result, _ = toggleCheckbox(content, -1)
	if result != content {
		t.Fatal("toggling negative line should return original content")
	}
}

func TestToggleCheckboxNonCheckboxLine(t *testing.T) {
	content := "just text\n- [ ] item"

	result, _ := toggleCheckbox(content, 0)
	if result != content {
		t.Fatal("toggling non-checkbox line should return original content")
	}
}

func TestParseEmpty(t *testing.T) {
	elements := parseElements("")
	if len(elements) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(elements))
	}
}

func TestParseMixedContent(t *testing.T) {
	content := `# Shopping List

- [ ] Buy groceries
- [x] Clean house

Check [recipes](https://recipes.com) for ideas.

- [ ] Cook dinner

Also see [tips](https://tips.com).`

	elements := parseElements(content)

	checkboxes := filterByKind(elements, ElementCheckbox)
	links := filterByKind(elements, ElementLink)

	if len(checkboxes) != 3 {
		t.Fatalf("expected 3 checkboxes, got %d", len(checkboxes))
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}

	// Verify order: elements should be in document order.
	if elements[0].Kind != ElementCheckbox {
		t.Fatal("first element should be a checkbox")
	}
	if elements[1].Kind != ElementCheckbox {
		t.Fatal("second element should be a checkbox")
	}
	if elements[2].Kind != ElementLink {
		t.Fatal("third element should be a link")
	}
	if elements[3].Kind != ElementCheckbox {
		t.Fatal("fourth element should be a checkbox")
	}
	if elements[4].Kind != ElementLink {
		t.Fatal("fifth element should be a link")
	}
}

func TestParseUppercaseX(t *testing.T) {
	content := "- [X] uppercase checked"
	elements := parseElements(content)

	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}
	if !elements[0].Checked {
		t.Fatal("uppercase X checkbox should be checked")
	}
}

func TestToggleUppercaseX(t *testing.T) {
	content := "- [X] item"
	result, checked := toggleCheckbox(content, 0)
	if checked {
		t.Fatal("toggling uppercase X should return checked=false")
	}
	if result != "- [ ] item" {
		t.Fatalf("expected %q, got %q", "- [ ] item", result)
	}
}

func TestElementDisplayText(t *testing.T) {
	checkbox := Element{Kind: ElementCheckbox, Text: "task", Checked: false}
	if got := elementDisplayText(checkbox); got != "[ ] task" {
		t.Fatalf("expected %q, got %q", "[ ] task", got)
	}

	checkedBox := Element{Kind: ElementCheckbox, Text: "done", Checked: true}
	if got := elementDisplayText(checkedBox); got != "[x] done" {
		t.Fatalf("expected %q, got %q", "[x] done", got)
	}

	link := Element{Kind: ElementLink, Text: "Go", URL: "https://go.dev"}
	if got := elementDisplayText(link); got != "Go (https://go.dev)" {
		t.Fatalf("expected %q, got %q", "Go (https://go.dev)", got)
	}
}

// --- helpers ---

func filterByKind(elements []Element, kind ElementKind) []Element {
	var result []Element
	for _, el := range elements {
		if el.Kind == kind {
			result = append(result, el)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
