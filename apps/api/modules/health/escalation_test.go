package health

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type sentAlert struct {
	targetID string
	message  string
}

func escalatorFixture(states map[string]State, targets []settings.DeliveryTarget, failSends bool) (*Escalator, *[]sentAlert, *bytes.Buffer, *[]Notice) {
	var logged bytes.Buffer
	sent := make([]sentAlert, 0)
	notices := make([]Notice, 0)

	escalator := NewEscalator(
		slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug})),
		func() []settings.DeliveryTarget { return targets },
		func(context.Context) map[string]State { return states },
		func(_ context.Context, target settings.DeliveryTarget, message string) error {
			if failSends {
				return context.DeadlineExceeded
			}
			sent = append(sent, sentAlert{targetID: target.ID, message: message})
			return nil
		},
		func(notice Notice) { notices = append(notices, notice) },
	)
	return escalator, &sent, &logged, &notices
}

func downDecision(now time.Time) Decision {
	failedAt := now.Add(-20 * time.Minute)
	return Decision{
		Alert: AlertBroke,
		Next: Record{
			TargetID:            "broken",
			State:               StateDown,
			ConsecutiveFailures: 3,
			LastObservedAt:      &now,
			LastFailureAt:       &now,
			LastSource:          SourceProbe,
			LastKind:            KindDenied,
			LastDetail:          "M_FORBIDDEN: You don't have permission to post that to the room",
			DownSince:           &failedAt,
		},
	}
}

// The whole point of the escalation: an alert about a target never travels
// through that target.
func TestEscalateExcludesTheSubjectTarget(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	broken := settings.DeliveryTarget{ID: "broken", Type: settings.TargetMatrix, Name: "Ops room", Enabled: true}
	targets := []settings.DeliveryTarget{
		broken,
		{ID: "healthy", Type: settings.TargetNtfy, Name: "Phone", Enabled: true},
		{ID: "also-down", Type: settings.TargetDiscord, Name: "Discord", Enabled: true},
		{ID: "switched-off", Type: settings.TargetSMTP, Name: "Mail", Enabled: false},
	}
	states := map[string]State{
		"broken":       StateDown,
		"healthy":      StateOK,
		"also-down":    StateDown,
		"switched-off": StateOK,
	}

	// The subject's own state is varied on purpose. StateDown would be filtered by
	// the Deliverable check regardless, so it proves nothing about the exclusion;
	// flapping and recovered are both Deliverable, and those are exactly the alerts
	// — "unstable", "back up" — that a naive router would send through the target
	// they are about.
	for _, subjectState := range []State{StateDown, StateFlapping, StateOK, StatePending} {
		t.Run(string(subjectState), func(t *testing.T) {
			states["broken"] = subjectState

			escalator, sent, _, notices := escalatorFixture(states, targets, false)
			escalator.now = func() time.Time { return now }
			escalator.RecordUndelivered("broken")
			escalator.RecordUndelivered("broken")

			escalator.Escalate(context.Background(), broken, downDecision(now))

			if len(*sent) != 1 {
				t.Fatalf("sent %d alerts, want 1: %+v", len(*sent), *sent)
			}
			if (*sent)[0].targetID != "healthy" {
				t.Fatalf("alert went to %q; it must never go to the subject, to another down target, or to a disabled one", (*sent)[0].targetID)
			}
			if len(*notices) != 1 || len((*notices)[0].DeliveredTo) != 1 {
				t.Fatalf("the activity log did not record one delivered notice: %+v", *notices)
			}
			if len(escalator.Standing()) != 0 {
				t.Fatal("a delivered alert must not leave a standing banner")
			}
		})
	}
}

// When nothing is healthy, the only remaining escalation path is the log — and it
// is teed to Journal, so it has to be ERROR, not a shrug at WARN.
func TestEscalateFallsBackToTheLogWhenNothingIsHealthy(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	broken := settings.DeliveryTarget{ID: "broken", Type: settings.TargetMatrix, Name: "Ops room", Enabled: true}
	targets := []settings.DeliveryTarget{broken}
	states := map[string]State{"broken": StateDown}

	escalator, sent, logged, notices := escalatorFixture(states, targets, false)
	escalator.now = func() time.Time { return now }

	escalator.Escalate(context.Background(), broken, downDecision(now))

	if len(*sent) != 0 {
		t.Fatalf("sent %d alerts with nothing healthy", len(*sent))
	}
	if !strings.Contains(logged.String(), "level=ERROR") {
		t.Fatalf("the fallback was not logged at ERROR:\n%s", logged.String())
	}
	standing := escalator.Standing()
	if len(standing) != 1 || standing[0].TargetID != "broken" {
		t.Fatalf("no standing banner recorded: %+v", standing)
	}
	if len(*notices) != 1 || len((*notices)[0].DeliveredTo) != 0 {
		t.Fatalf("the undeliverable notice was not recorded in the activity log: %+v", *notices)
	}
}

func TestRecoveryClearsTheStandingBanner(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	broken := settings.DeliveryTarget{ID: "broken", Type: settings.TargetMatrix, Name: "Ops room", Enabled: true}
	escalator, _, _, _ := escalatorFixture(map[string]State{"broken": StateDown}, []settings.DeliveryTarget{broken}, false)
	escalator.now = func() time.Time { return now }

	escalator.Escalate(context.Background(), broken, downDecision(now))
	if len(escalator.Standing()) != 1 {
		t.Fatal("expected a standing banner after an undeliverable alert")
	}

	recovered := downDecision(now)
	recovered.Alert = AlertRecovered
	recovered.Downtime = 25 * time.Minute
	escalator.Escalate(context.Background(), broken, recovered)

	if len(escalator.Standing()) != 0 {
		t.Fatal("the standing banner outlived the outage")
	}
}

// A send that fails is not a delivery. Counting it would report an alert as
// escalated when nobody saw it.
func TestEscalateTreatsAFailedSendAsUndelivered(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	broken := settings.DeliveryTarget{ID: "broken", Type: settings.TargetMatrix, Name: "Ops room", Enabled: true}
	targets := []settings.DeliveryTarget{broken, {ID: "healthy", Type: settings.TargetNtfy, Name: "Phone", Enabled: true}}

	escalator, _, logged, _ := escalatorFixture(map[string]State{"broken": StateDown, "healthy": StateOK}, targets, true)
	escalator.now = func() time.Time { return now }

	escalator.Escalate(context.Background(), broken, downDecision(now))

	if !strings.Contains(logged.String(), "level=ERROR") {
		t.Fatalf("a failed escalation send was not escalated to the log:\n%s", logged.String())
	}
	if len(escalator.Standing()) != 1 {
		t.Fatal("a failed escalation send left no standing banner")
	}
}

// The research's alert-content checklist, pinned. Every one of these is a field an
// operator needs and a real alerting system leaves out.
func TestAlertHTMLCarriesTheWholeChecklist(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	target := settings.DeliveryTarget{ID: "broken", Type: settings.TargetMatrix, Name: "Ops room"}
	decision := downDecision(now)

	alert := Alert{
		Target:        target,
		Decision:      decision,
		Undelivered:   7,
		NextAttemptAt: now.Add(5 * time.Minute),
		Now:           now,
	}
	html := alert.HTML()

	for _, want := range []string{
		"Ops room",                        // symptom, in the operator's vocabulary
		"matrix",                          // which channel
		"3 consecutive failed checks",     // the triggering value
		"M_FORBIDDEN",                     // the far side's verbatim words
		"down for 20m",                    // how long
		"First notice",                    // which transition this is
		"Next check at 12:05",             // when something changes next
		"<b>7 events</b> exhausted every", // the blast radius
		"replayed",                        // and whether any of it is coming back
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the alert is missing %q:\n%s", want, html)
		}
	}

	// The far side's words survive intact, and the markup they might carry does not.
	if !strings.Contains(html, "You don't have permission to post that to the room") {
		t.Errorf("the far side's words were rewritten:\n%s", html)
	}

	hostile := downDecision(now)
	hostile.Next.LastDetail = `<img src=x onerror="alert(1)">`
	escaped := Alert{Target: target, Decision: hostile, Now: now}.HTML()
	if strings.Contains(escaped, "<img") {
		t.Errorf("a provider error is interpolated into HTML unescaped:\n%s", escaped)
	}
	if !strings.Contains(escaped, "&lt;img") {
		t.Errorf("the escaped detail is missing entirely:\n%s", escaped)
	}
}

func TestAlertReportsWhenNothingWillRetest(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	alert := Alert{
		Target:   settings.DeliveryTarget{ID: "hook", Type: settings.TargetWebhook, Name: "Some hook"},
		Decision: downDecision(now),
		Now:      now,
	}
	html := alert.HTML()
	if !strings.Contains(html, "no safe automatic check") {
		t.Fatalf("a target with no probe must say so rather than promise a next check:\n%s", html)
	}
	if !strings.Contains(html, "No events have been lost yet") {
		t.Fatalf("zero blast radius must be stated, not omitted:\n%s", html)
	}
}

func TestJitterRepeatStaysWithinTenPercent(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	repeat := 4 * time.Hour
	limit := time.Duration(repeatJitter * float64(repeat))

	for _, draw := range []float64{0, 0.25, 0.5, 0.75, 1} {
		got := JitterRepeat(now, repeat, func() float64 { return draw })
		offset := got.Sub(now)
		if offset < -limit || offset > limit {
			t.Fatalf("draw %v moved the reminder by %s, outside ±%s", draw, offset, limit)
		}
	}
}
