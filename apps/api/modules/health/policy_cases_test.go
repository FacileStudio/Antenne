package health

import (
	"slices"
	"testing"
	"time"
)

type evaluateCase struct {
	name    string
	current Record
	obs     Observation
	now     time.Time

	wantState     State
	wantAlert     AlertKind
	wantDowntime  time.Duration
	wantReminder  int
	wantDownSince bool
	check         func(*testing.T, Decision)
}

// evaluateCases is the state machine's truth table, grouped by the behaviour
// each block pins. It lives apart from the test that runs it so each stays
// readable on its own.
func evaluateCases(cfg Thresholds) []evaluateCase {
	return slices.Concat(
		entryCases(cfg),
		exitCases(cfg),
		reminderCases(cfg),
		inconclusiveCases(cfg),
		suspensionCases(cfg),
		flapCases(cfg),
	)
}

// entryCases cover the failure threshold that opens an outage and the first
// notice it earns.
func entryCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:      "a first failure is pending and raises nothing",
			current:   Record{State: StateOK},
			obs:       fail(KindNone),
			now:       base,
			wantState: StatePending,
			wantAlert: AlertNone,
		},
		{
			name:          "the second consecutive failure goes down and announces it",
			current:       Record{State: StatePending, ConsecutiveFailures: 1},
			obs:           fail(KindNone),
			now:           base,
			wantState:     StateDown,
			wantAlert:     AlertBroke,
			wantDownSince: true,
		},
		{
			name:          "one success is not enough to recover even after the dwell",
			current:       Record{State: StateDown, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           ok(),
			now:           base.Add(time.Hour),
			wantState:     StateRecovering,
			wantAlert:     AlertNone,
			wantDownSince: true,
		},
	}
}

// exitCases cover the deliberately asymmetric way an outage closes: enough
// consecutive successes and a minimum dwell, or no recovery notice at all.
func exitCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:          "enough successes without the dwell is not a recovery",
			current:       Record{State: StateDown, ConsecutiveSuccesses: 1, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           ok(),
			now:           base.Add(time.Minute),
			wantState:     StateRecovering,
			wantAlert:     AlertNone,
			wantDownSince: true,
		},
		{
			name:         "successes plus the dwell recover and state the downtime",
			current:      Record{State: StateDown, ConsecutiveSuccesses: 1, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:          ok(),
			now:          base.Add(30 * time.Minute),
			wantState:    StateOK,
			wantAlert:    AlertRecovered,
			wantDowntime: 30 * time.Minute,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.AlertCount != 0 || decision.Next.LastAlertAt != nil {
					t.Errorf("the outage bookkeeping survived the recovery: %+v", decision.Next)
				}
			},
		},
		{
			name:      "an outage nobody was told about recovers silently",
			current:   Record{State: StateDown, ConsecutiveSuccesses: 1, DownSince: at(0), AlertCount: 0},
			obs:       ok(),
			now:       base.Add(30 * time.Minute),
			wantState: StateOK,
			wantAlert: AlertNone,
		},
	}
}

// reminderCases cover the still-broken repeat cadence.
func reminderCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:          "a reminder is due one repeat interval after the last send",
			current:       Record{State: StateDown, ConsecutiveFailures: 5, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           fail(KindNone),
			now:           base.Add(4 * time.Hour),
			wantState:     StateDown,
			wantAlert:     AlertStillBroken,
			wantDowntime:  4 * time.Hour,
			wantReminder:  1,
			wantDownSince: true,
		},
		{
			name:          "no reminder before the repeat interval elapses",
			current:       Record{State: StateDown, ConsecutiveFailures: 5, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           fail(KindNone),
			now:           base.Add(3*time.Hour + 59*time.Minute),
			wantState:     StateDown,
			wantAlert:     AlertNone,
			wantDownSince: true,
		},
	}
}

// inconclusiveCases cover the kinds that say nothing about the target: rate limits, unreachability and no probe at all.
func inconclusiveCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:      "a rate limit never moves a target toward down",
			current:   Record{State: StateOK, ConsecutiveFailures: 1},
			obs:       Observation{Kind: KindRateLimited, Source: SourceProbe, RetryAfter: 90 * time.Second},
			now:       base,
			wantState: StateOK,
			wantAlert: AlertNone,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.ConsecutiveFailures != 1 {
					t.Errorf("a rate limit touched the failure counter: %d", decision.Next.ConsecutiveFailures)
				}
				if len(decision.Next.Transitions) != 0 {
					t.Errorf("a rate limit fed the flap ring: %v", decision.Next.Transitions)
				}
				if decision.Next.ProbeSuspendedUntil == nil || !decision.Next.ProbeSuspendedUntil.Equal(base.Add(90*time.Second)) {
					t.Errorf("Retry-After was not binding: %v", decision.Next.ProbeSuspendedUntil)
				}
			},
		},
		{
			name:      "unreachable keeps the last state rather than blaming the target",
			current:   Record{State: StateOK, ConsecutiveSuccesses: 3},
			obs:       fail(KindUnreachable),
			now:       base,
			wantState: StateOK,
			wantAlert: AlertNone,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.ConsecutiveSuccesses != 3 || decision.Next.ConsecutiveFailures != 0 {
					t.Errorf("unreachable moved the counters: %+v", decision.Next)
				}
			},
		},
		{
			name:          "unreachable does not lift an existing outage either",
			current:       Record{State: StateDown, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           fail(KindUnreachable),
			now:           base.Add(time.Hour),
			wantState:     StateDown,
			wantAlert:     AlertNone,
			wantDownSince: true,
		},
		{
			name:      "unsupported is unknown and never alerts",
			current:   Record{State: StatePending, ConsecutiveFailures: 9},
			obs:       fail(KindUnsupported),
			now:       base,
			wantState: StateUnknown,
			wantAlert: AlertNone,
		},
	}
}

// suspensionCases cover the probe backoff a confirmed-dead credential earns, and what lifts it.
func suspensionCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:          "a confirmed-dead credential suspends probing and grows the backoff",
			current:       Record{State: StatePending, ConsecutiveFailures: 1, ProbeBackoffLevel: 1},
			obs:           fail(KindDead),
			now:           base,
			wantState:     StateDown,
			wantAlert:     AlertBroke,
			wantDownSince: true,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.ProbeBackoffLevel != 2 {
					t.Errorf("the backoff level did not grow: %d", decision.Next.ProbeBackoffLevel)
				}
				if decision.Next.ProbeSuspendedUntil == nil || !decision.Next.ProbeSuspendedUntil.Equal(base.Add(2*time.Hour)) {
					t.Errorf("probing was not suspended for the backoff: %v", decision.Next.ProbeSuspendedUntil)
				}
			},
		},
		{
			name:          "a success voids the suspension earned by a dead verdict",
			current:       Record{State: StateDown, DownSince: at(0), ProbeSuspendedUntil: at(time.Hour), ProbeBackoffLevel: 3},
			obs:           ok(),
			now:           base.Add(30 * time.Minute),
			wantState:     StateRecovering,
			wantAlert:     AlertNone,
			wantDownSince: true,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.ProbeSuspendedUntil != nil || decision.Next.ProbeBackoffLevel != 0 {
					t.Errorf("the suspension survived a success: %+v", decision.Next)
				}
			},
		},
	}
}

// flapCases cover flap detection at the edges of the transition ring.
func flapCases(cfg Thresholds) []evaluateCase {
	return []evaluateCase{
		{
			name:      "a full ring of one outcome is not flapping",
			current:   Record{State: StateOK, ConsecutiveSuccesses: 20, Transitions: filled(FlapWindow, true)},
			obs:       ok(),
			now:       base,
			wantState: StateOK,
			wantAlert: AlertNone,
		},
	}
}
