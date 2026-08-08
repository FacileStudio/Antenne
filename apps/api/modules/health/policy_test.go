package health

import (
	"testing"
	"time"
)

var base = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

func at(offset time.Duration) *time.Time {
	moment := base.Add(offset)
	return &moment
}

func fail(kind Kind) Observation {
	return Observation{OK: false, Kind: kind, Source: SourceProbe, Detail: "401 M_UNKNOWN_TOKEN"}
}

func ok() Observation {
	return Observation{OK: true, Source: SourceProbe}
}

// filled builds a transition ring of the given length so a case can opt in to
// flap detection, or (all-identical) opt out of it while still having a full ring.
func filled(length int, outcome bool) []bool {
	ring := make([]bool, length)
	for index := range ring {
		ring[index] = outcome
	}
	return ring
}

func TestEvaluate(t *testing.T) {
	cfg := DefaultThresholds()

	cases := []struct {
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
	}{
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
			wantState:     StateDown,
			wantAlert:     AlertNone,
			wantDownSince: true,
		},
		{
			name:          "enough successes without the dwell is not a recovery",
			current:       Record{State: StateDown, ConsecutiveSuccesses: 1, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1},
			obs:           ok(),
			now:           base.Add(time.Minute),
			wantState:     StateDown,
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
			wantState:     StateDown,
			wantAlert:     AlertNone,
			wantDownSince: true,
			check: func(t *testing.T, decision Decision) {
				if decision.Next.ProbeSuspendedUntil != nil || decision.Next.ProbeBackoffLevel != 0 {
					t.Errorf("the suspension survived a success: %+v", decision.Next)
				}
			},
		},
		{
			name:      "a full ring of one outcome is not flapping",
			current:   Record{State: StateOK, ConsecutiveSuccesses: 20, Transitions: filled(FlapWindow, true)},
			obs:       ok(),
			now:       base,
			wantState: StateOK,
			wantAlert: AlertNone,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			decision := Evaluate(testCase.current, testCase.obs, cfg, testCase.now)

			if decision.Next.State != testCase.wantState {
				t.Errorf("state = %q, want %q", decision.Next.State, testCase.wantState)
			}
			if decision.Alert != testCase.wantAlert {
				t.Errorf("alert = %q, want %q", decision.Alert, testCase.wantAlert)
			}
			if decision.Downtime != testCase.wantDowntime {
				t.Errorf("downtime = %v, want %v", decision.Downtime, testCase.wantDowntime)
			}
			if decision.Reminder != testCase.wantReminder {
				t.Errorf("reminder = %d, want %d", decision.Reminder, testCase.wantReminder)
			}
			if testCase.wantDownSince && decision.Next.DownSince == nil {
				t.Error("DownSince was not set")
			}
			if !testCase.wantDownSince && decision.Next.DownSince != nil {
				t.Errorf("DownSince = %v, want nil", decision.Next.DownSince)
			}
			if decision.Next.LastObservedAt == nil || !decision.Next.LastObservedAt.Equal(testCase.now) {
				t.Errorf("LastObservedAt = %v, want %v", decision.Next.LastObservedAt, testCase.now)
			}
			if testCase.check != nil {
				testCase.check(t, decision)
			}
		})
	}
}

// TestRecoveryNeedsBothSuccessesAndDwell is the asymmetry, pinned on its own
// because a symmetric rule passes every other test in this file while producing
// a down/up/down/up storm in production.
func TestRecoveryNeedsBothSuccessesAndDwell(t *testing.T) {
	cfg := DefaultThresholds()
	down := Record{State: StateDown, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1}

	// One success, well past the dwell: the success count alone must not recover.
	first := Evaluate(down, ok(), cfg, base.Add(time.Hour))
	if first.Next.State != StateDown || first.Alert != AlertNone {
		t.Fatalf("recovered on one success: state=%q alert=%q", first.Next.State, first.Alert)
	}

	// The second success, still past the dwell: now it recovers.
	second := Evaluate(first.Next, ok(), cfg, base.Add(time.Hour+time.Minute))
	if second.Next.State != StateOK || second.Alert != AlertRecovered {
		t.Fatalf("did not recover on two successes past the dwell: state=%q alert=%q", second.Next.State, second.Alert)
	}

	// Same two successes, inside the dwell: must stay down.
	early := Evaluate(down, ok(), cfg, base.Add(time.Minute))
	later := Evaluate(early.Next, ok(), cfg, base.Add(2*time.Minute))
	if later.Next.State != StateDown || later.Alert != AlertNone {
		t.Fatalf("recovered inside the dwell: state=%q alert=%q", later.Next.State, later.Alert)
	}
}

func TestBreakAndRecoverInsideTheThresholdSaysNothing(t *testing.T) {
	cfg := DefaultThresholds()

	broke := Evaluate(Record{State: StateOK}, fail(KindNone), cfg, base)
	if broke.Alert != AlertNone {
		t.Fatalf("the first failure announced %q", broke.Alert)
	}
	back := Evaluate(broke.Next, ok(), cfg, base.Add(time.Minute))
	if back.Alert != AlertNone {
		t.Fatalf("the recovery announced %q for an outage nobody heard about", back.Alert)
	}
	if back.Next.State != StateOK {
		t.Fatalf("state = %q, want ok", back.Next.State)
	}
}

// TestARepeatIsSuppressedByAStateChange walks a full outage, a recovery, and a
// second outage, then waits out the original repeat interval. The reminder clock
// belongs to the current outage: it must not fire off the first outage's send.
func TestARepeatIsSuppressedByAStateChange(t *testing.T) {
	cfg := DefaultThresholds()

	record := Record{State: StatePending, ConsecutiveFailures: 1}
	broke := Evaluate(record, fail(KindNone), cfg, base)
	if broke.Alert != AlertBroke {
		t.Fatalf("alert = %q, want broke", broke.Alert)
	}

	recovering := Evaluate(broke.Next, ok(), cfg, base.Add(20*time.Minute))
	recovered := Evaluate(recovering.Next, ok(), cfg, base.Add(21*time.Minute))
	if recovered.Alert != AlertRecovered {
		t.Fatalf("alert = %q, want recovered", recovered.Alert)
	}

	// Breaks again more than a repeat interval after the first send.
	pending := Evaluate(recovered.Next, fail(KindNone), cfg, base.Add(5*time.Hour))
	again := Evaluate(pending.Next, fail(KindNone), cfg, base.Add(5*time.Hour+time.Minute))
	if again.Alert != AlertBroke {
		t.Fatalf("alert = %q, want broke for the new outage", again.Alert)
	}
	if again.Next.AlertCount != 1 {
		t.Fatalf("AlertCount = %d, want 1 for a fresh outage", again.Next.AlertCount)
	}

	// A failure right after: no reminder, the new outage's clock just started.
	quiet := Evaluate(again.Next, fail(KindNone), cfg, base.Add(5*time.Hour+2*time.Minute))
	if quiet.Alert != AlertNone {
		t.Fatalf("alert = %q, want silence so soon after the new notice", quiet.Alert)
	}
}

func TestRepeatsAreNumbered(t *testing.T) {
	cfg := DefaultThresholds()
	record := Record{State: StateDown, ConsecutiveFailures: 2, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1}

	first := Evaluate(record, fail(KindNone), cfg, base.Add(4*time.Hour))
	if first.Alert != AlertStillBroken || first.Reminder != 1 {
		t.Fatalf("first reminder: alert=%q reminder=%d", first.Alert, first.Reminder)
	}
	second := Evaluate(first.Next, fail(KindNone), cfg, base.Add(8*time.Hour))
	if second.Alert != AlertStillBroken || second.Reminder != 2 {
		t.Fatalf("second reminder: alert=%q reminder=%d", second.Alert, second.Reminder)
	}
	if second.Downtime != 8*time.Hour {
		t.Errorf("downtime = %v, want 8h", second.Downtime)
	}
}

func TestARateLimitBurstCausesNoTransition(t *testing.T) {
	cfg := DefaultThresholds()
	record := Record{State: StateOK, ConsecutiveSuccesses: 4}

	for burst := 0; burst < 5; burst++ {
		decision := Evaluate(record, Observation{Kind: KindRateLimited, Source: SourceDelivery, Detail: "429"}, cfg,
			base.Add(time.Duration(burst)*time.Minute))
		if decision.Next.State != StateOK || decision.Alert != AlertNone {
			t.Fatalf("burst %d: state=%q alert=%q", burst, decision.Next.State, decision.Alert)
		}
		record = decision.Next
	}
	if record.ConsecutiveFailures != 0 || len(record.Transitions) != 0 {
		t.Fatalf("a rate-limit burst left evidence behind: %+v", record)
	}
}

// TestFlapThenSettle is the sequence no single row can catch: an oscillation has
// to fill the window before it is called unstable, must be announced exactly
// once, and must then be hard to leave.
func TestFlapThenSettle(t *testing.T) {
	cfg := DefaultThresholds()
	record := Record{State: StateOK}
	moment := base

	unstable := 0
	other := 0
	for step := 0; step < FlapWindow; step++ {
		decision := Evaluate(record, alternating(step), cfg, moment)
		switch decision.Alert {
		case AlertNone:
		case AlertUnstable:
			unstable++
		default:
			other++
		}
		record = decision.Next
		moment = moment.Add(time.Minute)
	}
	if record.State != StateFlapping {
		t.Fatalf("state = %q after a full window of oscillation, want flapping", record.State)
	}
	if unstable != 1 {
		t.Fatalf("unstable notices = %d, want exactly 1", unstable)
	}
	if other != 0 {
		t.Fatalf("%d per-transition notices escaped while flapping", other)
	}

	// It keeps flapping for a while, then settles once the window is clean.
	recovered := 0
	for step := 0; step < FlapWindow+2; step++ {
		decision := Evaluate(record, ok(), cfg, moment)
		if decision.Alert == AlertRecovered {
			recovered++
		} else if decision.Alert != AlertNone {
			t.Fatalf("step %d announced %q while settling", step, decision.Alert)
		}
		record = decision.Next
		moment = moment.Add(time.Minute)
	}
	if record.State != StateOK {
		t.Fatalf("state = %q after a clean window, want ok", record.State)
	}
	if recovered != 1 {
		t.Fatalf("recovery notices = %d, want exactly 1 to close the unstable notice", recovered)
	}
}

func alternating(step int) Observation {
	if step%2 == 0 {
		return fail(KindNone)
	}
	return ok()
}

func TestFlapPercentIgnoresAPartialRing(t *testing.T) {
	if _, measurable := flapPercent(filled(FlapWindow-1, true)); measurable {
		t.Error("flapPercent answered on a partial ring")
	}
	percent, measurable := flapPercent(alternatingRing(FlapWindow))
	if !measurable {
		t.Fatal("flapPercent refused a full ring")
	}
	if percent < 99 {
		t.Errorf("a fully alternating ring measured %.1f%%, want ~100%%", percent)
	}
}

func alternatingRing(length int) []bool {
	ring := make([]bool, length)
	for index := range ring {
		ring[index] = index%2 == 0
	}
	return ring
}

/*
Regression from the first production deploy. A Matrix account over its device
limit answered 403 M_FORBIDDEN, which is KindDead — terminal, so probing was
suspended for an hour, correctly. But one failure is below the entry threshold,
so the target sat in StatePending with AlertCount 0, and the suspension meant the
second failure that would have promoted it never came. Antenne knew a channel was
dead, and told nobody.
*/
func TestTerminalKindAlertsOnTheFirstObservation(t *testing.T) {
	cfg := DefaultThresholds()
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	dead := Observation{
		Kind:   KindDead,
		Source: SourceProbe,
		Detail: `Matrix API POST /_matrix/client/v3/login -> HTTP 403 {"errcode":"M_FORBIDDEN"}`,
	}

	decision := Evaluate(Record{TargetID: "matrix-1", State: StateOK}, dead, cfg, now)

	if decision.Next.State != StateDown {
		t.Errorf("state = %q, want %q on a terminal kind", decision.Next.State, StateDown)
	}
	if decision.Alert != AlertBroke {
		t.Errorf("alert = %q, want %q — a rejected credential must be reported at once", decision.Alert, AlertBroke)
	}
	if decision.Next.DownSince == nil {
		t.Error("DownSince was not set, so no downtime can ever be reported")
	}
	if !decision.Next.ProbeSuspended(now) {
		t.Error("probing was not suspended; a rejected credential must not be retried on a schedule")
	}

	// The blip case must be untouched: a non-terminal failure still waits.
	blip := Observation{Kind: KindUnsupported, Source: SourceProbe}
	blip.Kind = "" // an ordinary failure with no special classification
	ordinary := Evaluate(Record{TargetID: "webhook-1", State: StateOK}, blip, cfg, now)
	if ordinary.Next.State != StatePending {
		t.Errorf("an ordinary first failure gave %q, want %q", ordinary.Next.State, StatePending)
	}
	if ordinary.Alert != AlertNone {
		t.Errorf("an ordinary first failure alerted %q, want silence", ordinary.Alert)
	}
}
