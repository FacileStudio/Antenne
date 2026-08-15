package health

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// probeIntervals is how often each target type may be *actively* probed. One
// global tick would be wrong in both directions: five minutes is careless for
// SMTP and lazy for Matrix.
//
// Every number here is a cost, not a preference:
//
//   - Matrix: /account/whoami is a cheap authenticated GET.
//   - Discord: a webhook GET is cheap, but ~10k 401/403/429 in ten minutes gets
//     the instance's IP blocked for about an hour, so it gets a wider gap than
//     its cost alone would suggest.
//   - SMTP: every probe is a TCP connect, a full TLS handshake and a credential
//     presentation. Repeated auth failures feed fail2ban, Google's 454 throttle
//     and Microsoft smart lockout — a monitor on a five-minute tick can lock the
//     account it is monitoring. Half an hour to an hour is the defensible range,
//     and the per-target phase below spreads 30 into 30-45.
//   - ntfy: a topic GET is as cheap as it gets.
//   - Perception: an authenticated HTTP API, no published rate limit; treated
//     like Discord rather than like ntfy because a token rejection there is
//     logged on the far side.
//   - Generic webhook: zero, meaning never. There is no read-only call that an
//     arbitrary URL is guaranteed to answer, and a GET against a POST-only
//     endpoint proves nothing. It stays StateUnknown until a real delivery says
//     otherwise, and painting it green would be a lie.
var probeIntervals = map[settings.DeliveryTargetType]time.Duration{
	settings.TargetMatrix:     5 * time.Minute,
	settings.TargetDiscord:    10 * time.Minute,
	settings.TargetSMTP:       30 * time.Minute,
	settings.TargetNtfy:       5 * time.Minute,
	settings.TargetPerception: 10 * time.Minute,
	settings.TargetWebhook:    0,
}

// ProbeInterval is how often this target type may be probed, or zero when it has
// no safe read-only probe.
func ProbeInterval(kind settings.DeliveryTargetType) time.Duration {
	return probeIntervals[kind]
}

// sweepInterval is how often the scheduler wakes up to ask which targets are
// due. It is not a probe interval: the shortest of those is five minutes, so a
// one-minute sweep resolves every slot with room to spare while keeping one
// goroutine for the whole subsystem instead of one per target.
const sweepInterval = time.Minute

// Records is the slice of health persistence the scheduler needs. Narrow on
// purpose: the scheduler decides *when* to look, so it has no business reading
// anything it does not write back.
type Records interface {
	Load(ctx context.Context, targetID string) (Record, error)
	LoadAll(ctx context.Context) ([]Record, error)
	Upsert(ctx context.Context, record Record) error
	Delete(ctx context.Context, targetID string) error
}

// ProbeFunc performs one read-only check of a target.
//
// It is a function rather than an interface method so the probe implementation
// can live in the delivery package (which owns the per-provider calls) without
// health having to import it — the dependency runs delivery -> health, and
// closing the loop would be a build error.
type ProbeFunc func(ctx context.Context, target settings.DeliveryTarget) Observation

// Policy is Evaluate. It is injected rather than called directly so the
// scheduler is testable without the policy, and so the wiring site shows at a
// glance that there is exactly one policy in the process.
type Policy func(current Record, obs Observation, cfg Thresholds, now time.Time) Decision

// Escalation delivers an alert about a target somewhere other than that target.
type Escalation interface {
	Escalate(ctx context.Context, target settings.DeliveryTarget, decision Decision)
}

// Monitor decides when to look at a target, and nothing else. Every outcome goes
// through the policy; there is deliberately no threshold, no hysteresis and no
// notification cadence in this file. Two places that decide when a target is
// down will disagree, and finding out which one was right costs a weekend.
type Monitor struct {
	log        *slog.Logger
	records    Records
	probe      ProbeFunc
	policy     Policy
	escalation Escalation
	thresholds Thresholds

	now    func() time.Time
	random func() float64

	// observeMu serializes the load-decide-store cycle. A real delivery outcome
	// and a probe can land on the same target at the same moment, and a lost
	// update there loses a consecutive-failure count, which is the one number the
	// hysteresis is counting.
	observeMu sync.Mutex

	mu        sync.Mutex
	cancel    context.CancelFunc
	startedAt time.Time
	lastSweep time.Time
	group     sync.WaitGroup
}

// NewMonitor builds the scheduler. escalation may be nil, which measures health
// without telling anyone — useful in tests, useless in production.
func NewMonitor(log *slog.Logger, records Records, probe ProbeFunc, policy Policy, escalation Escalation) *Monitor {
	return &Monitor{
		log:        log,
		records:    records,
		probe:      probe,
		policy:     policy,
		escalation: escalation,
		thresholds: DefaultThresholds(),
		now:        time.Now,
		random:     rand.Float64,
	}
}

// Apply restarts the sweep to match the current settings. It is called on boot
// and after every dashboard save, like every other poller in this process.
func (m *Monitor) Apply(ctx context.Context, current settings.AppSettings) {
	m.Stop()

	targets := make([]settings.DeliveryTarget, 0, len(current.Delivery.Items))
	for _, target := range current.Delivery.Items {
		// Enabled is admin state, not health. A target switched off is not
		// unhealthy, and probing it would answer a question nobody asked.
		if target.Enabled {
			targets = append(targets, target)
		}
	}

	m.prune(ctx, current.Delivery.Items)

	loopCtx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.cancel = cancel
	m.startedAt = m.now()
	m.mu.Unlock()

	m.log.Info("monitoring delivery targets", slog.Int("targets", len(targets)))

	m.group.Add(1)
	go m.loop(loopCtx, targets)
}

// Stop halts the sweep and waits for the in-flight probe to finish.
func (m *Monitor) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.group.Wait()
}

// LastSweepAt is when the scheduler last completed a full pass. The heartbeat
// asks, because a heartbeat a stalled process can still send is worthless.
func (m *Monitor) LastSweepAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastSweep
}

// Snapshot is the persisted health of every target that has a record.
func (m *Monitor) Snapshot(ctx context.Context) []Record {
	records, err := m.records.LoadAll(ctx)
	if err != nil {
		m.log.Warn("could not read target health", slog.Any("error", err))
		return nil
	}
	return records
}

// States is the same thing keyed by target, for the escalation's "is anything
// else alive" question.
func (m *Monitor) States(ctx context.Context) map[string]State {
	records := m.Snapshot(ctx)
	states := make(map[string]State, len(records))
	for _, record := range records {
		states[record.TargetID] = record.State
	}
	return states
}

// Check probes one target now, whatever the schedule says, and records the
// outcome like any other observation.
//
// It deliberately ignores the suspension and the terminal-kind guards that
// ShouldProbe applies: those exist to stop an unattended loop from hammering a
// dead credential, and this is an operator pressing a button after fixing one. It
// is also the only way out of a terminal kind, which is why the guard can afford
// to be absolute.
func (m *Monitor) Check(ctx context.Context, target settings.DeliveryTarget) Observation {
	obs := m.probe(ctx, target)
	m.Observe(ctx, target, obs)
	return obs
}

// Observe feeds one observation from any source through the policy, persists the
// result and escalates whatever the policy asked for. The delivery pipeline
// calls this with real outcomes; the sweep calls it with probes.
func (m *Monitor) Observe(ctx context.Context, target settings.DeliveryTarget, obs Observation) {
	m.observeMu.Lock()
	defer m.observeMu.Unlock()

	current, err := m.records.Load(ctx, target.ID)
	if err != nil {
		m.log.Warn("could not read target health",
			slog.String("target", target.Name), slog.Any("error", err))
		return
	}
	current.TargetID = target.ID

	now := m.now()
	decision := m.policy(current, obs, m.thresholds, now)

	// The repeat interval is jittered here rather than inside Evaluate, which
	// stays deterministic so the cadence can be tested with a table and no clock.
	switch decision.Alert {
	case AlertBroke, AlertStillBroken:
		decision.Next.LastAlertAt = JitterRepeat(now, m.thresholds.RepeatInterval, m.random)
	}

	if err := m.records.Upsert(ctx, decision.Next); err != nil {
		m.log.Error("could not persist target health",
			slog.String("target", target.Name), slog.Any("error", err))
		return
	}

	if decision.Alert == AlertNone || m.escalation == nil {
		return
	}
	m.escalation.Escalate(ctx, target, decision)
}
