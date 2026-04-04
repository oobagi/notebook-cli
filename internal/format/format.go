package format

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// StatusBar builds a status bar string with left status, an optional centered
// hint, and right-aligned keybinds. Width constrains the total bar width.
// All three sections always render — nothing is dropped at narrow widths.
func StatusBar(left, hint, right string, width int) string {
	if width <= 0 {
		width = 80
	}

	if hint != "" {
		left += " " + hint
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// ShortenHome replaces the home directory prefix with ~/ for display.
func ShortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// RelativeTime returns a human-friendly relative timestamp string.
// It follows the design system rules from docs/design.md.
func RelativeTime(t time.Time) string {
	return RelativeTimeFrom(t, time.Now())
}

// RelativeTimeFrom returns a relative timestamp using the given reference time.
func RelativeTimeFrom(t time.Time, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	default:
		if t.Year() == now.Year() {
			return t.Format("Jan 2")
		}
		return t.Format("Jan 2, 2006")
	}
}

// HumanSize formats a byte count into a human-readable string.
func HumanSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		kb := float64(bytes) / 1024
		return FormatFloat(kb) + " KB"
	case bytes < 1024*1024*1024:
		mb := float64(bytes) / (1024 * 1024)
		return FormatFloat(mb) + " MB"
	default:
		gb := float64(bytes) / (1024 * 1024 * 1024)
		return FormatFloat(gb) + " GB"
	}
}

// FormatFloat formats a float to one decimal place, dropping the ".0" suffix.
func FormatFloat(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	if strings.HasSuffix(s, ".0") {
		return s[:len(s)-2]
	}
	return s
}
