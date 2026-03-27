package render

import (
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/oobagi/notebook/internal/theme"
	"golang.org/x/term"
)

// RenderMarkdown renders a markdown string for terminal display using Glamour.
// When running on a TTY, the glamour style is resolved from the user's
// glamour_style config (which may be a built-in name or a JSON file path).
// When output is not a terminal, glamour's NoTTY style is used via
// WithAutoStyle to keep output free of ANSI escape codes.
func RenderMarkdown(content string, width int) string {
	var styleOpt glamour.TermRendererOption
	if term.IsTerminal(int(os.Stdout.Fd())) {
		styleOpt = resolveStyleOption()
	} else {
		styleOpt = glamour.WithAutoStyle()
	}

	opts := []glamour.TermRendererOption{styleOpt}
	if width > 0 {
		opts = append(opts, glamour.WithWordWrap(width))
	}

	renderer, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return content
	}

	out, err := renderer.Render(content)
	if err != nil {
		return content
	}

	// Style GitHub-style admonition blocks ([!NOTE], [!TIP], etc.) when
	// outputting to a TTY with colors enabled.
	_, noColor := os.LookupEnv("NO_COLOR")
	if term.IsTerminal(int(os.Stdout.Fd())) && !noColor {
		out = RenderAdmonitions(out)
	}

	// Add clickable hyperlinks via OSC 8 when outputting to a TTY.
	// Skip when NO_COLOR is set to keep output free of escape sequences.
	if term.IsTerminal(int(os.Stdout.Fd())) && !noColor {
		out = LinkifyMarkdown(out)
	}

	return out
}

// glamourStyleOverride holds the user's glamour_style config value.
// It is set during initialization via SetGlamourStyle.
var glamourStyleOverride string

// SetGlamourStyle stores the user's glamour_style config value so that
// RenderMarkdown can resolve it at render time.
func SetGlamourStyle(style string) {
	glamourStyleOverride = style
}

// RenderMarkdownWithStyle renders markdown using a specific named glamour style.
// This is used by the theme picker to preview different styles without
// changing the global glamour_style setting.
func RenderMarkdownWithStyle(content string, width int, styleName string) string {
	style, isFile := theme.ResolveGlamourStyle(styleName)
	var styleOpt glamour.TermRendererOption
	if isFile {
		styleOpt = glamour.WithStylesFromJSONFile(style)
	} else {
		styleOpt = glamour.WithStandardStyle(style)
	}

	opts := []glamour.TermRendererOption{styleOpt}
	if width > 0 {
		opts = append(opts, glamour.WithWordWrap(width))
	}

	renderer, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return content
	}

	out, err := renderer.Render(content)
	if err != nil {
		return content
	}

	out = RenderAdmonitions(out)
	return out
}

// resolveStyleOption returns the appropriate glamour TermRendererOption
// based on the user's glamour_style config. It supports built-in style
// names and custom JSON file paths.
func resolveStyleOption() glamour.TermRendererOption {
	style, isFile := theme.ResolveGlamourStyle(glamourStyleOverride)
	if isFile {
		return glamour.WithStylesFromJSONFile(style)
	}
	return glamour.WithStandardStyle(style)
}
