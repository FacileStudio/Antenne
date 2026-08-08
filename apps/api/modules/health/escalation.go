package health

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// repeatJitter is the fraction the next reminder is spread by. Applied here and
// never in Evaluate: the policy stays deterministic so the cadence can be tested
// with a table, and the randomness lives with the thing that actually sends.
const repeatJitter = 0.10

// JitterRepeat returns the alert timestamp to persist, moved by up to ±10% of
// the repeat interval so a fleet of reminders does not converge onto one minute
// four hours from now.
//
// Nudging the *recorded* time rather than the threshold is deliberate: it leaves
// Evaluate's "has RepeatInterval elapsed since LastAlertAt" comparison exact.
func JitterRepeat(alertAt time.Time, repeat time.Duration, random func() float64) *time.Time {
	if repeat <= 0 || random == nil {
		at := alertAt
		return &at
	}
	spread := float64(repeat) * repeatJitter
	at := alertAt.Add(time.Duration((random()*2 - 1) * spread))
	return &at
}

// Notice is one health alert, recorded in the activity log whether or not it
// reached a human.
type Notice struct {
	TargetID    string
	TargetName  string
	TargetType  string
	Kind        AlertKind
	Title       string
	MessageHTML string

	// DeliveredTo is the targets the notice actually reached, which is empty
	// exactly when Antenne could not announce its own failure.
	DeliveredTo []string
}

// StandingAlert is an alert nothing could carry. The dashboard shows these as a
// banner, because a monitoring system whose only record of total failure is a
// line in stdout has not reported anything.
type StandingAlert struct {
	TargetID   string    `json:"targetId"`
	TargetName string    `json:"targetName"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	Since      time.Time `json:"since"`
}

// SendFunc delivers one rendered notice straight to one target, bypassing the
// routing rules and the retry queue.
//
// Bypassing them is what keeps the loop guard: an escalation send never enters
// the delivery pipeline, so it can never produce an event, a retry, an exhausted
// callback or another observation. The old guard in App.notifyDeliveryExhausted
// had to cut the cycle on the event type; this one cannot form it.
type SendFunc func(ctx context.Context, target settings.DeliveryTarget, messageHTML string) error

// Escalator decides where an alert about a target goes. The answer is never
// "through that target".
type Escalator struct {
	log     *slog.Logger
	targets func() []settings.DeliveryTarget
	states  func(ctx context.Context) map[string]State
	send    SendFunc
	record  func(Notice)
	now     func() time.Time

	mu sync.Mutex
	// undelivered counts events that ran out of retries per target since the
	// outage began. It is the blast radius, and it is the field operators ask for
	// first and alerting systems never include.
	undelivered map[string]int
	standing    map[string]StandingAlert
}

// NewEscalator builds the escalator. record may be nil.
func NewEscalator(log *slog.Logger, targets func() []settings.DeliveryTarget, states func(context.Context) map[string]State, send SendFunc, record func(Notice)) *Escalator {
	return &Escalator{
		log:         log,
		targets:     targets,
		states:      states,
		send:        send,
		record:      record,
		now:         time.Now,
		undelivered: map[string]int{},
		standing:    map[string]StandingAlert{},
	}
}

// RecordUndelivered notes that one event has permanently failed to a target. The
// delivery pipeline calls this when the retry queue gives up.
func (e *Escalator) RecordUndelivered(targetID string) {
	e.mu.Lock()
	e.undelivered[targetID]++
	e.mu.Unlock()
}

// Standing is every alert that could not be delivered anywhere.
func (e *Escalator) Standing() []StandingAlert {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]StandingAlert, 0, len(e.standing))
	for _, alert := range e.standing {
		out = append(out, alert)
	}
	return out
}

// Escalate sends one alert about one target through the other targets.
func (e *Escalator) Escalate(ctx context.Context, target settings.DeliveryTarget, decision Decision) {
	now := e.now()

	e.mu.Lock()
	undelivered := e.undelivered[target.ID]
	if decision.Alert == AlertRecovered {
		delete(e.undelivered, target.ID)
	}
	e.mu.Unlock()

	notice := Alert{
		Target:        target,
		Decision:      decision,
		Undelivered:   undelivered,
		NextAttemptAt: nextAttempt(target, decision.Next),
		Now:           now,
	}
	title, message := notice.Title(), notice.HTML()

	recipients := e.recipients(ctx, target.ID)
	delivered := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		if err := e.send(ctx, recipient, message); err != nil {
			e.log.Warn("a target health alert could not be delivered",
				slog.String("about", target.Name),
				slog.String("via", recipient.Name),
				slog.Any("error", err))
			continue
		}
		delivered = append(delivered, recipient.Name)
	}

	if e.record != nil {
		e.record(Notice{
			TargetID:    target.ID,
			TargetName:  target.Name,
			TargetType:  string(target.Type),
			Kind:        decision.Alert,
			Title:       title,
			MessageHTML: message,
			DeliveredTo: delivered,
		})
	}

	if len(delivered) > 0 {
		e.clearStanding(target.ID)
		e.log.Info("target health alert sent",
			slog.String("about", target.Name),
			slog.String("alert", string(decision.Alert)),
			slog.String("via", strings.Join(delivered, ", ")))
		return
	}

	// Nothing healthy is left, so this is the watchman problem in its pure form.
	// ERROR rather than WARN because the application log is teed to Journal, which
	// makes this a real escalation path off this host and not a shrug into stdout.
	e.log.Error("no healthy delivery target could carry an alert about a broken one",
		slog.String("about", target.Name),
		slog.String("target_type", string(target.Type)),
		slog.String("alert", string(decision.Alert)),
		slog.Int("candidates", len(recipients)),
		slog.String("detail", decision.Next.LastDetail))

	if decision.Alert == AlertRecovered {
		e.clearStanding(target.ID)
		return
	}
	e.mu.Lock()
	if _, held := e.standing[target.ID]; !held {
		e.standing[target.ID] = StandingAlert{
			TargetID:   target.ID,
			TargetName: target.Name,
			Title:      title,
			Detail:     decision.Next.LastDetail,
			Since:      now,
		}
	}
	e.mu.Unlock()
}

// recipients is every target that could carry this alert: healthy, sendable, and
// not the one the alert is about. Routing an alert about a target through that
// target is how an alerting system stays silent about the only thing it knows.
func (e *Escalator) recipients(ctx context.Context, subjectID string) []settings.DeliveryTarget {
	states := e.states(ctx)
	out := make([]settings.DeliveryTarget, 0, 4)
	for _, candidate := range e.targets() {
		if candidate.ID == subjectID || !candidate.Enabled {
			continue
		}
		// A target with no record yet is StateUnknown, which is Deliverable: an
		// unmeasured channel is worth trying when the alternative is silence.
		if !states[candidate.ID].Deliverable() {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func (e *Escalator) clearStanding(targetID string) {
	e.mu.Lock()
	delete(e.standing, targetID)
	e.mu.Unlock()
}

func nextAttempt(target settings.DeliveryTarget, record Record) time.Time {
	interval := ProbeInterval(target.Type)
	if interval <= 0 || record.LastObservedAt == nil {
		return time.Time{}
	}
	if record.ProbeSuspendedUntil != nil && record.ProbeSuspendedUntil.After(*record.LastObservedAt) {
		return *record.ProbeSuspendedUntil
	}
	return NextProbeAt(target.ID, interval, *record.LastObservedAt)
}

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
