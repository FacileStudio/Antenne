package webhooks

import (
	"fmt"
	"strings"
	"time"
)

// formatTimestamp renders an ISO timestamp the way the old dashboard did:
// `2 Aug 2026, 14:05 UTC`. It is pinned to UTC rather than the container's zone
// so the same event reads identically in Matrix, in an email and in the
// activity log.
func formatTimestamp(raw string) string {
	parsed, err := parseTime(raw)
	if err != nil {
		return raw
	}
	return parsed.UTC().Format("2 Jan 2006, 15:04") + " UTC"
}

func parseTime(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("webhooks: unrecognised timestamp %q", raw)
}

// humanDuration renders a span the way the alerts read: `2h 15m`, `45m`, `30s`.
func humanDuration(span time.Duration) string {
	switch {
	case span < time.Minute:
		return fmt.Sprintf("%ds", int(span.Seconds()))
	case span < time.Hour:
		return fmt.Sprintf("%dm", int(span.Minutes()))
	case span < 24*time.Hour:
		hours := int(span.Hours())
		minutes := int(span.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(span.Hours()) / 24
	hours := int(span.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
