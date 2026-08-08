package health

import "time"

// Thresholds are the policy's tunables, gathered in one place so the numbers are
// reviewable rather than scattered through the logic.
type Thresholds struct {
	// FailuresToDown is the entry threshold. Two is the "at least two rule
	// evaluation cycles" floor from Google's practical-alerting chapter: one
	// failure is a blip.
	FailuresToDown int

	// SuccessesToRecover is the exit threshold, and must be >= FailuresToDown.
	// This asymmetry is the whole point.
	SuccessesToRecover int

	// MinDownDwell is how long a target must have been down before a recovery is
	// believed, on top of SuccessesToRecover. Without it a target failing every
	// other probe recovers and breaks forever, and every one of those is a
	// notification.
	MinDownDwell time.Duration

	// RepeatInterval is the fixed gap between reminders about a target that is
	// still down. Fixed, not escalating: every primary source (Alertmanager's
	// repeat_interval, Nagios' notification_interval, PagerDuty's ack timeout)
	// uses a fixed gap, and escalation in practice means widening the audience,
	// not lengthening the wait. An operator has to be able to answer "if I do
	// nothing, when do I hear again".
	RepeatInterval time.Duration

	// FlapEnter and FlapLeave are percentages of state changes across the
	// FlapWindow, mirroring Nagios' 20.0/5.0 defaults. Two thresholds, so
	// leaving the flapping state is harder than entering it.
	FlapEnter float64
	FlapLeave float64
}

// DefaultThresholds are the researched defaults. Every value here has a source
// behind it; see the package comment and the wiki page it points at.
func DefaultThresholds() Thresholds {
	return Thresholds{
		FailuresToDown:     2,
		SuccessesToRecover: 2,
		MinDownDwell:       10 * time.Minute,
		RepeatInterval:     4 * time.Hour,
		FlapEnter:          20.0,
		FlapLeave:          5.0,
	}
}

// AlertKind is what the policy wants said, if anything.
type AlertKind string

const (
	// AlertNone is the overwhelmingly common case, and the reason this is a
	// decision rather than a side effect.
	AlertNone AlertKind = ""

	// AlertBroke is the first notice for an outage.
	AlertBroke AlertKind = "broke"

	// AlertStillBroken is a reminder, carrying which one it is.
	AlertStillBroken AlertKind = "still_broken"

	// AlertRecovered closes the loop and states the downtime. Its absence is why
	// operators stop trusting an alerting system.
	AlertRecovered AlertKind = "recovered"

	// AlertUnstable is sent once when a target starts flapping, in place of the
	// per-transition notices it suppresses.
	AlertUnstable AlertKind = "unstable"
)

// Decision is the policy's whole output: the record to persist and the alert to
// raise. Returning it instead of acting keeps Evaluate pure, which is what makes
// the hysteresis and the repeat cadence testable with a table and no clock, no
// database and no network.
type Decision struct {
	Next  Record
	Alert AlertKind

	// Downtime is set on AlertRecovered and AlertStillBroken.
	Downtime time.Duration

	// Reminder is which repeat this is, set on AlertStillBroken.
	Reminder int
}
