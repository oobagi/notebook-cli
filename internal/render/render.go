package render

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders a markdown string for terminal display using Glamour.
// On rendering failure, the raw content is returned as a fallback.
func RenderMarkdown(content string, width int) string {
	opts := []glamour.TermRendererOption{glamour.WithAutoStyle()}
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

	return out
}
