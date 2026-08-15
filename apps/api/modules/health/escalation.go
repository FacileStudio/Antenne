package health

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

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
	delivered := e.fanOut(ctx, target, recipients, message)

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

	e.raiseWatchman(target, decision, title, len(recipients), now)
}

// fanOut sends the message to every recipient and returns the ones that took
// it. A refusal is logged and skipped rather than aborting: the whole point of
// having several recipients is that one of them being broken is survivable.
func (e *Escalator) fanOut(ctx context.Context, target settings.DeliveryTarget, recipients []settings.DeliveryTarget, message string) []string {
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
	return delivered
}

// raiseWatchman handles an alert nothing could carry: it is logged loudly and
// then held, so the next healthy target reports the backlog instead of the
// outage vanishing with the only channel that could have announced it.
func (e *Escalator) raiseWatchman(target settings.DeliveryTarget, decision Decision, title string, candidates int, now time.Time) {
	// Nothing healthy is left, so this is the watchman problem in its pure form.
	// ERROR rather than WARN because the application log is teed to Journal, which
	// makes this a real escalation path off this host and not a shrug into stdout.
	e.log.Error("no healthy delivery target could carry an alert about a broken one",
		slog.String("about", target.Name),
		slog.String("target_type", string(target.Type)),
		slog.String("alert", string(decision.Alert)),
		slog.Int("candidates", candidates),
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
