package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Alert is one rendered health notice. It exists as a value with a Title and an
// HTML method so the content checklist can be pinned by a test without a
// network, a clock or a delivery target.
type Alert struct {
	Target      settings.DeliveryTarget
	Decision    Decision
	Undelivered int

	// NextAttemptAt is zero when there will not be one, which is itself the
	// answer: a generic webhook is never probed, so nothing will change until an
	// event routes there.
	NextAttemptAt time.Time
	Now           time.Time
}

// Title is the one line a phone notification shows. Symptom first, in the
// operator's vocabulary — a target name and a channel, never a raw HTTP status.
func (a Alert) Title() string {
	switch a.Decision.Alert {
	case AlertRecovered:
		return fmt.Sprintf("%s is delivering again", a.Target.Name)
	case AlertUnstable:
		return fmt.Sprintf("%s is unstable", a.Target.Name)
	case AlertStillBroken:
		return fmt.Sprintf("%s is still not delivering", a.Target.Name)
	default:
		return fmt.Sprintf("%s is not delivering", a.Target.Name)
	}
}

// HTML renders the notice in the same style as the rest of Antenne's own events.
func (a Alert) HTML() string {
	record := a.Decision.Next
	lines := []string{
		fmt.Sprintf("<b>%s</b><br/>Delivery target <b>%s</b> (%s)",
			envelope.EscapeHTML(a.Title()),
			envelope.EscapeHTML(a.Target.Name),
			envelope.EscapeHTML(string(a.Target.Type))),
	}

	switch a.Decision.Alert {
	case AlertRecovered:
		lines = append(lines, fmt.Sprintf("Recovered after %s.", humanDuration(a.Decision.Downtime)))
	case AlertStillBroken:
		lines = append(lines, fmt.Sprintf("Reminder %d — down for %s.",
			a.Decision.Reminder, humanDuration(a.Decision.Downtime)))
	case AlertUnstable:
		lines = append(lines, "It has changed state too often to trust, so per-transition notices are suppressed until it settles.")
	default:
		lines = append(lines, fmt.Sprintf("First notice — down for %s.",
			humanDuration(record.Downtime(a.Now))))
	}

	if a.Decision.Alert != AlertRecovered {
		if record.ConsecutiveFailures > 0 {
			lines = append(lines, fmt.Sprintf("%d consecutive failed %s.",
				record.ConsecutiveFailures, plural("check", record.ConsecutiveFailures)))
		}
		if record.LastDetail != "" {
			// The far side's own words, verbatim. This is what an operator pastes
			// into a search or forwards to their provider, so it is never rewritten.
			lines = append(lines, fmt.Sprintf("<code>%s</code>", envelope.EscapeHTML(record.LastDetail)))
		}
	}

	lines = append(lines, a.blastRadius())

	switch {
	case a.Decision.Alert == AlertRecovered:
	case a.NextAttemptAt.IsZero():
		lines = append(lines, "This target type has no safe automatic check, so nothing will re-test it until an event routes there.")
	default:
		lines = append(lines, fmt.Sprintf("Next check at %s.", a.NextAttemptAt.UTC().Format("15:04 MST")))
	}

	return strings.Join(lines, "<br/>")
}

// blastRadius answers the question operators ask first and alerting systems
// almost never answer: how much did this cost, and is any of it coming back.
func (a Alert) blastRadius() string {
	switch {
	case a.Decision.Alert == AlertRecovered && a.Undelivered > 0:
		return fmt.Sprintf("<b>%d %s</b> was lost while it was down. Nothing is re-sent automatically; replay from the activity log.",
			a.Undelivered, plural("event", a.Undelivered))
	case a.Decision.Alert == AlertRecovered:
		return "No events were lost."
	case a.Undelivered == 0:
		return "No events have been lost yet — this was found by a check, before any traffic hit it."
	default:
		return fmt.Sprintf("<b>%d %s</b> exhausted every retry and will not be re-sent automatically. They are in the activity log and can be replayed.",
			a.Undelivered, plural("event", a.Undelivered))
	}
}

func plural(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

func humanDuration(span time.Duration) string {
	switch {
	case span < time.Minute:
		return fmt.Sprintf("%ds", int(span.Seconds()))
	case span < time.Hour:
		return fmt.Sprintf("%dm", int(span.Minutes()))
	case span < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(span.Hours()), int(span.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(span.Hours())/24, int(span.Hours())%24)
}
