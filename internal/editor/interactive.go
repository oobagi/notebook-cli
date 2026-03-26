package editor

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ElementKind distinguishes interactive element types found in the markdown.
type ElementKind int

const (
	// ElementCheckbox represents a markdown checkbox (- [ ] or - [x]).
	ElementCheckbox ElementKind = iota
	// ElementLink represents a markdown link ([text](url)).
	ElementLink
)

// Element represents an interactive element found in markdown content.
type Element struct {
	Kind    ElementKind
	Line    int    // zero-based line number in the source markdown
	Col     int    // column offset in the source line
	Text    string // display text
	URL     string // for links
	Checked bool   // for checkboxes
}

// checkboxRe matches markdown checkboxes: optional whitespace, dash, space,
// bracket-space/x-bracket, then text.
var checkboxRe = regexp.MustCompile(`^(\s*)-\s+\[([ xX])\]\s*(.*)$`)

// linkRe matches inline markdown links: [text](url).
var linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// parseElements scans markdown content and extracts interactive elements
// (checkboxes and links). It returns them in document order.
func parseElements(content string) []Element {
	if content == "" {
		return nil
	}

	var elements []Element
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for checkboxes first.
		if m := checkboxRe.FindStringSubmatchIndex(line); m != nil {
			// m[0..1] = full match
			// m[2..3] = leading whitespace
			// m[4..5] = check character (space or x/X)
			// m[6..7] = text after checkbox
			checkChar := line[m[4]:m[5]]
			text := line[m[6]:m[7]]
			col := m[0] // start of the full match (same as leading whitespace start)

			elements = append(elements, Element{
				Kind:    ElementCheckbox,
				Line:    i,
				Col:     col,
				Text:    text,
				Checked: checkChar == "x" || checkChar == "X",
			})
		}

		// Find all links in the line.
		linkMatches := linkRe.FindAllStringSubmatchIndex(line, -1)
		for _, lm := range linkMatches {
			// lm[0..1] = full match
			// lm[2..3] = link text
			// lm[4..5] = URL
			text := line[lm[2]:lm[3]]
			url := line[lm[4]:lm[5]]
			col := lm[0]

			elements = append(elements, Element{
				Kind: ElementLink,
				Line: i,
				Col:  col,
				Text: text,
				URL:  url,
			})
		}
	}

	return elements
}

// toggleCheckbox toggles the checkbox on the given line in the content string.
// It replaces "- [ ]" with "- [x]" or vice versa. Returns the modified content
// and the new checked state.
func toggleCheckbox(content string, line int) (string, bool) {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return content, false
	}

	l := lines[line]
	var newChecked bool

	if strings.Contains(l, "- [ ]") {
		lines[line] = strings.Replace(l, "- [ ]", "- [x]", 1)
		newChecked = true
	} else if strings.Contains(l, "- [x]") {
		lines[line] = strings.Replace(l, "- [x]", "- [ ]", 1)
		newChecked = false
	} else if strings.Contains(l, "- [X]") {
		lines[line] = strings.Replace(l, "- [X]", "- [ ]", 1)
		newChecked = false
	} else {
		return content, false
	}

	return strings.Join(lines, "\n"), newChecked
}

// openLink opens a URL in the default browser (macOS only).
func openLink(url string) error {
	return exec.Command("open", url).Start()
}

// elementDisplayText returns a short label for an element, used in the preview
// to indicate the focused item.
func elementDisplayText(el Element) string {
	switch el.Kind {
	case ElementCheckbox:
		mark := " "
		if el.Checked {
			mark = "x"
		}
		return fmt.Sprintf("[%s] %s", mark, el.Text)
	case ElementLink:
		return fmt.Sprintf("%s (%s)", el.Text, el.URL)
	default:
		return el.Text
	}
}
