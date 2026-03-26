package browser

import (
	"fmt"
	"strings"
	"time"
)

// relativeTime returns a human-friendly relative timestamp string.
func relativeTime(t time.Time) string {
	return relativeTimeFrom(t, time.Now())
}

// relativeTimeFrom returns a relative timestamp using the given reference time.
func relativeTimeFrom(t time.Time, now time.Time) string {
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

// humanSize formats a byte count into a human-readable string.
func humanSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		kb := float64(bytes) / 1024
		return formatFloat(kb) + " KB"
	case bytes < 1024*1024*1024:
		mb := float64(bytes) / (1024 * 1024)
		return formatFloat(mb) + " MB"
	default:
		gb := float64(bytes) / (1024 * 1024 * 1024)
		return formatFloat(gb) + " GB"
	}
}

func formatFloat(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	if strings.HasSuffix(s, ".0") {
		return s[:len(s)-2]
	}
	return s
}
