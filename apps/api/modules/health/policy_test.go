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

	cases := evaluateCases(cfg)

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
	// StateRecovering rather than StateDown — the outage is still open and no
	// recovery is announced, but the target is answering and the dashboard says so.
	first := Evaluate(down, ok(), cfg, base.Add(time.Hour))
	if first.Next.State != StateRecovering || first.Alert != AlertNone {
		t.Fatalf("recovered on one success: state=%q alert=%q", first.Next.State, first.Alert)
	}
	if first.Next.DownSince == nil {
		t.Fatal("the outage was closed by a single success")
	}

	// The second success, still past the dwell: now it recovers.
	second := Evaluate(first.Next, ok(), cfg, base.Add(time.Hour+time.Minute))
	if second.Next.State != StateOK || second.Alert != AlertRecovered {
		t.Fatalf("did not recover on two successes past the dwell: state=%q alert=%q", second.Next.State, second.Alert)
	}

	// Same two successes, inside the dwell: the outage must stay open.
	early := Evaluate(down, ok(), cfg, base.Add(time.Minute))
	later := Evaluate(early.Next, ok(), cfg, base.Add(2*time.Minute))
	if later.Next.State != StateRecovering || later.Alert != AlertNone {
		t.Fatalf("recovered inside the dwell: state=%q alert=%q", later.Next.State, later.Alert)
	}
	if later.Next.DownSince == nil {
		t.Fatal("the dwell was skipped: the outage was closed inside it")
	}
}

// TestRecoveringIsNotAnOutageOnTheWayBack pins the report the operator reads.
// The screenshot that produced this test showed a red "Down for 7m" dot beside
// the successful probe response that had just been quoted verbatim underneath
// it, with "Last worked just now" alongside — three widgets, one of them
// disagreeing with the other two and with the operator's own Check now.
func TestRecoveringIsNotAnOutageOnTheWayBack(t *testing.T) {
	cfg := DefaultThresholds()
	down := Record{State: StateDown, DownSince: at(0), LastAlertAt: at(0), AlertCount: 1}

	recovering := Evaluate(down, ok(), cfg, base.Add(7*time.Minute))
	if recovering.Next.State != StateRecovering {
		t.Fatalf("state = %q, want %q while the dwell runs", recovering.Next.State, StateRecovering)
	}
	// The circuit breaker has to be closed: the target just answered, and a
	// probe is a weaker signal than the send this would otherwise refuse.
	if !recovering.Next.State.Deliverable() {
		t.Error("a target that is answering was left undeliverable")
	}

	// Failing again during the wait is the same outage, not a new one. It owes
	// no second first-notice, and must not fall back to pending: an outage that
	// has already been announced is not "not confirmed yet".
	relapse := Evaluate(recovering.Next, fail(KindNone), cfg, base.Add(8*time.Minute))
	if relapse.Next.State != StateDown {
		t.Errorf("state = %q, want %q on a relapse inside an open outage", relapse.Next.State, StateDown)
	}
	if relapse.Alert != AlertNone {
		t.Errorf("alert = %q, want none: the outage never ended", relapse.Alert)
	}
	if relapse.Next.DownSince == nil || !relapse.Next.DownSince.Equal(base) {
		t.Errorf("DownSince = %v, want the original outage start", relapse.Next.DownSince)
	}
	if relapse.Next.AlertCount != 1 {
		t.Errorf("AlertCount = %d, want 1: the relapse announced the same outage twice", relapse.Next.AlertCount)
	}

	// And the real recovery still closes the loop once the dwell is served.
	back := Evaluate(relapse.Next, ok(), cfg, base.Add(20*time.Minute))
	again := Evaluate(back.Next, ok(), cfg, base.Add(21*time.Minute))
	if again.Next.State != StateOK || again.Alert != AlertRecovered {
		t.Fatalf("did not recover: state=%q alert=%q", again.Next.State, again.Alert)
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
