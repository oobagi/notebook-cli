package render

import (
	"fmt"
	"strings"
	"testing"
)

func TestWrapLink(t *testing.T) {
	got := WrapLink("example", "https://example.com")
	want := "\x1b]8;;https://example.com\x1b\\example\x1b]8;;\x1b\\"
	if got != want {
		t.Errorf("WrapLink mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestLinkifyFindsURLs(t *testing.T) {
	input := "Visit https://example.com for details."
	got := LinkifyMarkdown(input)

	wrapped := WrapLink("https://example.com", "https://example.com")
	want := fmt.Sprintf("Visit %s for details.", wrapped)
	if got != want {
		t.Errorf("LinkifyMarkdown mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestLinkifyFindsHTTPURLs(t *testing.T) {
	input := "Visit http://example.com for info."
	got := LinkifyMarkdown(input)

	if !strings.Contains(got, "\x1b]8;;http://example.com\x1b\\") {
		t.Errorf("expected OSC 8 wrapping for http URL, got %q", got)
	}
}

func TestLinkifyPreservesNonURLText(t *testing.T) {
	input := "Hello world, no links here."
	got := LinkifyMarkdown(input)
	if got != input {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestLinkifyHandlesMultipleURLs(t *testing.T) {
	input := "See https://one.com and https://two.com for more."
	got := LinkifyMarkdown(input)

	if !strings.Contains(got, WrapLink("https://one.com", "https://one.com")) {
		t.Errorf("expected first URL wrapped, got %q", got)
	}
	if !strings.Contains(got, WrapLink("https://two.com", "https://two.com")) {
		t.Errorf("expected second URL wrapped, got %q", got)
	}
}

func TestLinkifyIgnoresPartialURLs(t *testing.T) {
	input := "The word http alone is not a URL."
	got := LinkifyMarkdown(input)
	if got != input {
		t.Errorf("expected unchanged text for partial URL, got %q", got)
	}
}
