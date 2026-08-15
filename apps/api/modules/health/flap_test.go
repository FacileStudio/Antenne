package health

import (
	"testing"
	"time"
)

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
