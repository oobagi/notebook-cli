package render

import (
	"fmt"
	"regexp"
)

// urlPattern matches http:// and https:// URLs while avoiding ANSI escape
// sequences and closing brackets that are part of surrounding syntax.
var urlPattern = regexp.MustCompile(`https?://[^\s\x1b\]]+`)

// WrapLink wraps text in an OSC 8 hyperlink escape sequence.
func WrapLink(text, url string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// LinkifyMarkdown scans rendered output for URLs and wraps them in OSC 8
// hyperlink escape sequences so they are clickable in supported terminals.
// It matches http:// and https:// URLs and leaves non-URL text unchanged.
func LinkifyMarkdown(content string) string {
	return urlPattern.ReplaceAllStringFunc(content, func(rawURL string) string {
		return WrapLink(rawURL, rawURL)
	})
}
