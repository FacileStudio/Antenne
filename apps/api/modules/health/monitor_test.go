package health

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// The persistent store has to satisfy the narrow interface the scheduler asks
// for. Asserted here rather than in store.go so the two stay independently
// owned, and so a signature drift breaks at this line instead of at a wiring
// site three packages away.
var _ Records = (*Store)(nil)

// memoryRecords is the store the scheduler is written against, with none of the
// persistence. The scheduler's job is timing, and timing needs no database.
type memoryRecords struct {
	byID map[string]Record
}

func newMemoryRecords(records ...Record) *memoryRecords {
	store := &memoryRecords{byID: map[string]Record{}}
	for _, record := range records {
		store.byID[record.TargetID] = record
	}
	return store
}

func (m *memoryRecords) Load(_ context.Context, targetID string) (Record, error) {
	record, ok := m.byID[targetID]
	if !ok {
		return Record{TargetID: targetID, State: StateUnknown}, nil
	}
	return record, nil
}

func (m *memoryRecords) LoadAll(context.Context) ([]Record, error) {
	out := make([]Record, 0, len(m.byID))
	for _, record := range m.byID {
		out = append(out, record)
	}
	return out, nil
}

func (m *memoryRecords) Upsert(_ context.Context, record Record) error {
	m.byID[record.TargetID] = record
	return nil
}

func (m *memoryRecords) Delete(_ context.Context, targetID string) error {
	delete(m.byID, targetID)
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func observedAt(base time.Time, offset time.Duration) *time.Time {
	moment := base.Add(offset)
	return &moment
}

func TestShouldProbe(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	started := now.Add(-time.Hour)
	const targetID = "target-1"

	matrix := ProbeInterval(settings.TargetMatrix)

	cases := []struct {
		name     string
		record   Record
		interval time.Duration
		want     bool
	}{
		{
			name:     "never observed and past its phase",
			record:   Record{TargetID: targetID},
			interval: matrix,
			want:     true,
		},
		{
			name:     "a type with no safe probe is never probed",
			record:   Record{TargetID: targetID},
			interval: ProbeInterval(settings.TargetWebhook),
			want:     false,
		},
		{
			name:     "a fresh observation defers the probe",
			record:   Record{TargetID: targetID, LastObservedAt: observedAt(now, -time.Minute)},
			interval: matrix,
			want:     false,
		},
		{
			name:     "a stale observation is due",
			record:   Record{TargetID: targetID, LastObservedAt: observedAt(now, -2*matrix)},
			interval: matrix,
			want:     true,
		},
		{
			name: "a suspended target is left alone",
			record: Record{
				TargetID:            targetID,
				LastObservedAt:      observedAt(now, -2*matrix),
				ProbeSuspendedUntil: observedAt(now, time.Hour),
			},
			interval: matrix,
			want:     false,
		},
		{
			name: "an expired suspension no longer holds",
			record: Record{
				TargetID:            targetID,
				LastObservedAt:      observedAt(now, -2*matrix),
				ProbeSuspendedUntil: observedAt(now, -time.Minute),
			},
			interval: matrix,
			want:     true,
		},
		{
			name: "a confirmed-dead credential is never probed again",
			record: Record{
				TargetID:       targetID,
				LastObservedAt: observedAt(now, -24*time.Hour),
				LastKind:       KindDead,
			},
			interval: matrix,
			want:     false,
		},
		{
			name: "a rate-limited target is still probed once the wait is over",
			record: Record{
				TargetID:       targetID,
				LastObservedAt: observedAt(now, -2*matrix),
				LastKind:       KindRateLimited,
			},
			interval: matrix,
			want:     true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ShouldProbe(testCase.record, targetID, testCase.interval, started, now)
			if got != testCase.want {
				t.Fatalf("ShouldProbe = %v, want %v", got, testCase.want)
			}
		})
	}
}

// The gap between probes must never be shorter than the interval, or a delivery
// that landed just before a slot boundary would be followed straight away by a
// probe — the opposite of passive-first.
func TestNextProbeAtNeverShortensTheGap(t *testing.T) {
	base := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	interval := ProbeInterval(settings.TargetSMTP)

	for _, id := range []string{"a", "b", "target-42", "", "zzzzzzzzzzzz"} {
		for offset := time.Duration(0); offset < interval; offset += time.Minute {
			since := base.Add(offset)
			gap := NextProbeAt(id, interval, since).Sub(since)
			if gap < interval {
				t.Fatalf("target %q at +%s: gap %s is shorter than the interval %s", id, offset, gap, interval)
			}
			if gap >= interval+interval/2 {
				t.Fatalf("target %q at +%s: gap %s exceeds interval + half", id, offset, gap)
			}
		}
	}
}

// The phase has to come from the target id, not from process start, or every
// restart re-aligns every target onto one tick and the deploy gets the blame.
func TestProbePhaseIsPerTargetAndStable(t *testing.T) {
	interval := ProbeInterval(settings.TargetMatrix)
	first := probePhase("target-a", interval)
	if first != probePhase("target-a", interval) {
		t.Fatal("the phase is not stable for the same target")
	}
	if first == probePhase("target-b", interval) {
		t.Fatal("two targets share a phase, so a restart re-aligns them")
	}
}

func TestSweepSkipsFreshAndSuspendedTargets(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	matrix := ProbeInterval(settings.TargetMatrix)

	targets := []settings.DeliveryTarget{
		{ID: "fresh", Type: settings.TargetMatrix, Name: "fresh", Enabled: true},
		{ID: "suspended", Type: settings.TargetMatrix, Name: "suspended", Enabled: true},
		{ID: "webhook", Type: settings.TargetWebhook, Name: "webhook", Enabled: true},
		{ID: "due", Type: settings.TargetMatrix, Name: "due", Enabled: true},
	}

	store := newMemoryRecords(
		Record{TargetID: "fresh", LastObservedAt: observedAt(now, -time.Minute), LastSource: SourceDelivery},
		Record{TargetID: "suspended", LastObservedAt: observedAt(now, -2*matrix), ProbeSuspendedUntil: observedAt(now, time.Hour)},
		Record{TargetID: "webhook", LastObservedAt: observedAt(now, -24*time.Hour)},
		Record{TargetID: "due", LastObservedAt: observedAt(now, -2*matrix)},
	)

	probed := make([]string, 0)
	monitor := NewMonitor(quietLogger(), store,
		func(_ context.Context, target settings.DeliveryTarget) Observation {
			probed = append(probed, target.ID)
			return Observation{OK: true, Source: SourceProbe}
		},
		func(current Record, obs Observation, _ Thresholds, when time.Time) Decision {
			current.LastObservedAt = &when
			current.LastSource = obs.Source
			return Decision{Next: current}
		},
		nil,
	)
	monitor.now = func() time.Time { return now }
	monitor.startedAt = now.Add(-time.Hour)

	monitor.sweep(context.Background(), targets)

	if len(probed) != 1 || probed[0] != "due" {
		t.Fatalf("probed %v, want only [due]: a fresh observation, a suspension and a type with no safe probe must all be skipped", probed)
	}
	if monitor.LastSweepAt() != now {
		t.Fatal("the sweep did not record its completion, so the heartbeat cannot tell whether the loop is alive")
	}
}

func TestObserveJittersTheRepeatWithoutTouchingThePolicy(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	target := settings.DeliveryTarget{ID: "t", Type: settings.TargetMatrix, Name: "t", Enabled: true}
	store := newMemoryRecords()

	monitor := NewMonitor(quietLogger(), store,
		func(context.Context, settings.DeliveryTarget) Observation { return Observation{} },
		func(current Record, _ Observation, _ Thresholds, _ time.Time) Decision {
			current.State = StateDown
			// A deterministic policy: it never sets LastAlertAt itself.
			return Decision{Next: current, Alert: AlertBroke}
		},
		nil,
	)
	monitor.now = func() time.Time { return now }
	monitor.random = func() float64 { return 1 }

	monitor.Observe(context.Background(), target, Observation{OK: false, Source: SourceProbe})

	stored, _ := store.Load(context.Background(), "t")
	if stored.LastAlertAt == nil {
		t.Fatal("the alert timestamp was not recorded, so the reminder cadence has nothing to measure from")
	}
	want := now.Add(time.Duration(repeatJitter * float64(DefaultThresholds().RepeatInterval)))
	if !stored.LastAlertAt.Equal(want) {
		t.Fatalf("LastAlertAt = %s, want %s (the jitter belongs here, not in Evaluate)", stored.LastAlertAt, want)
	}
}

func TestPruneDropsHealthForRemovedTargets(t *testing.T) {
	store := newMemoryRecords(
		Record{TargetID: "kept"},
		Record{TargetID: "deleted"},
	)
	monitor := NewMonitor(quietLogger(), store, nil, nil, nil)

	monitor.prune(context.Background(), []settings.DeliveryTarget{{ID: "kept"}})

	if _, ok := store.byID["deleted"]; ok {
		t.Fatal("health for a removed target survived, so the dashboard keeps a red row nobody can fix")
	}
	if _, ok := store.byID["kept"]; !ok {
		t.Fatal("prune dropped a live target")
	}
}

// Every type either has a defensible interval or an explicit zero. A type added
// without a decision would silently inherit zero and never be probed, which is a
// choice nobody made.
func TestEveryTargetTypeHasAnIntervalDecision(t *testing.T) {
	for _, kind := range settings.DeliveryTargetTypes {
		if _, ok := probeIntervals[kind]; !ok {
			t.Fatalf("no probe interval decision for target type %q", kind)
		}
	}
	if ProbeInterval(settings.TargetWebhook) != 0 {
		t.Fatal("a generic webhook has no safe read-only probe and must never be scheduled")
	}
	if got := ProbeInterval(settings.TargetSMTP); got < 30*time.Minute {
		t.Fatalf("SMTP probe interval %s is under 30m; every probe is a credential presentation and fail2ban is counting", got)
	}
}
