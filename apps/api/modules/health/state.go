// Package health tracks whether each delivery target can actually be reached,
// and decides when a human should be told about it.
//
// The design follows three findings that are easy to get wrong:
//
//   - Hysteresis is asymmetric. Entering a bad state is cheap (two consecutive
//     failures, the "two evaluation cycles" rule); leaving it is expensive (two
//     consecutive successes *and* a dwell time). Alertmanager's keep_firing_for
//     and Nagios' 20-in/5-out flap thresholds both spend their hysteresis on the
//     exit, and a symmetric rule produces down/up/down/up notification storms.
//   - The unit of a health notification is the target, never the event. One dead
//     webhook and fifty events used to mean fifty identical notices.
//   - The circuit breaker and the periodic probe are one mechanism, not two. A
//     probe in the Open state *is* the half-open trial. Two state machines
//     disagree, and finding out why costs a weekend.
package health

import "time"

// State is the health of one target. It deliberately does not include an
// "disabled" value: whether a target is turned on is admin state, held on the
// target itself, and folding the two together makes "was it healthy when it was
// switched off" unanswerable and every dashboard total a lie.
type State string

const (
	// StateUnknown means never measured, or not measurable. A generic webhook
	// has no safe read-only probe, so it stays here until a real delivery says
	// otherwise. Displaying this honestly is the point: a green dot that cannot
	// be justified is worse than a grey one.
	StateUnknown State = "unknown"

	// StateOK means the last observation succeeded.
	StateOK State = "ok"

	// StatePending means failing, but not yet past the threshold. Prometheus'
	// term for the same idea. It exists so the state machine can count without
	// alerting, and the dashboard may collapse it into ok.
	StatePending State = "pending"

	// StateDown means failing past the threshold. This is the only state that
	// raises an alert.
	StateDown State = "down"

	// StateRecovering means the target is answering again but the outage is not
	// closed yet, because the recovery still owes consecutive successes, dwell
	// time, or both.
	//
	// It exists because the alternative is a red "Down for 7m" dot on a target
	// the operator has just watched answer, next to that answer quoted verbatim.
	// The asymmetric hysteresis is right and stays; reporting the wait as an
	// outage is what was wrong. A dot an operator can disprove by hand is a dot
	// they learn to ignore, and then the red one means nothing when it matters.
	//
	// It is deliberately not StateOK: the outage bookkeeping is still open and
	// the recovery notice has not been sent. It is deliverable, because the
	// target demonstrably works.
	StateRecovering State = "recovering"

	// StateFlapping means the target has changed state too often to trust.
	// Per-transition notices are suppressed while it holds, and one "unstable"
	// notice is sent instead — Nagios' design, and the only thing that actually
	// stops an oscillation storm.
	StateFlapping State = "flapping"

	// StateStale means the last observation is older than twice its interval, so
	// what we hold is no longer evidence of anything.
	StateStale State = "stale"

	// StateProbeError means the probe itself could not run — our bug, not the
	// target's. Grafana keeps this separate from "no data" for the same reason,
	// and collapsing it into StateDown blames the wrong party.
	StateProbeError State = "probe_error"
)

// Deliverable reports whether the router should attempt a send. This is the
// circuit breaker: Down is an open circuit. Everything else attempts, because a
// send is also the most authoritative health observation available and refusing
// to try is how a target stays down forever.
func (s State) Deliverable() bool { return s != StateDown }

// Kind classifies why an observation failed, which decides whether probing
// continues. It is not cosmetic: probing a confirmed-dead credential is
// actively harmful. Discord blocks an IP for about an hour after ~10k
// 401/403/429 responses in 10 minutes, and repeated SMTP auth failures feed
// fail2ban, Google's 454 throttle and Microsoft smart lockout — a monitor can
// lock the very account it is monitoring.
type Kind string

const (
	// KindNone accompanies a success.
	KindNone Kind = ""

	// KindDead is a credential the far side has rejected outright: a deleted
	// Discord webhook (404 10015), a hard Matrix logout, SMTP 535. Retrying on a
	// schedule will never help, so probing suspends after this is confirmed.
	KindDead Kind = "dead"

	// KindRefreshable is a credential that can be renewed rather than replaced —
	// a Matrix 401 carrying soft_logout: true, whose session, device ID and
	// encryption keys all stay valid. Treating this as KindDead throws away
	// working state.
	KindRefreshable Kind = "refreshable"

	// KindDenied is a valid credential without permission for the destination: a
	// Matrix bot joined to a room whose events_default was raised, an ntfy token
	// without write access. The fix is on the far side, not in our config.
	KindDenied Kind = "denied"

	// KindUnreachable is a network, DNS, TLS or 5xx failure. The target's own
	// state is unknown; keep the last one rather than blaming it.
	KindUnreachable Kind = "unreachable"

	// KindRateLimited means we are being throttled, including SMTP's 454
	// temporary auth failure. Never a reason to mark a target bad, always a
	// reason to back off — and the far side's Retry-After is binding.
	KindRateLimited Kind = "rate_limited"

	// KindUnsupported means this target type has no safe read-only probe.
	KindUnsupported Kind = "unsupported"
)

// Terminal reports whether continuing to probe can only do harm.
func (k Kind) Terminal() bool { return k == KindDead }

// Source records which tier produced an observation, because they are not
// equally trustworthy and the dashboard should say which one it is showing.
type Source string

const (
	// SourceDelivery is a real delivery outcome: free, and the most
	// authoritative signal there is. A target with recent real traffic needs no
	// probe at all, which is what keeps monitoring from costing more than the
	// work it monitors.
	SourceDelivery Source = "delivery"

	// SourceProbe is an active read-only check.
	SourceProbe Source = "probe"

	// SourceSynthetic is a real message sent to a destination the operator
	// nominated. Opt-in only, and never silently pushed into a live channel.
	SourceSynthetic Source = "synthetic"
)

// Observation is one measurement of a target, from any source.
type Observation struct {
	OK     bool
	Kind   Kind
	Source Source

	// Detail is the far side's own words, kept verbatim. It is what an operator
	// pastes into a search or sends to their provider, so it is never rewritten.
	Detail string

	// RetryAfter is the far side's binding request to wait, when it sent one.
	RetryAfter time.Duration
}

// FlapWindow is how many recent outcomes feed flap detection. Nagios uses 21
// slots over 20 possible transitions; matching it keeps the thresholds below
// meaningful.
const FlapWindow = 20

// Record is the persisted health of one target. Everything the policy needs to
// decide, and nothing it does not — it is written on every observation, so it
// stays narrow on purpose.
type Record struct {
	TargetID string
	State    State

	ConsecutiveFailures  int
	ConsecutiveSuccesses int

	LastObservedAt *time.Time
	LastSuccessAt  *time.Time
	LastFailureAt  *time.Time
	LastSource     Source
	LastKind       Kind
	LastDetail     string

	// DownSince is when this outage began, so an alert can state a duration and
	// a recovery notice can say how long it lasted.
	DownSince *time.Time

	// LastAlertAt and AlertCount drive the repeat interval. AlertCount also
	// distinguishes a first notice from a reminder in the message text.
	LastAlertAt *time.Time
	AlertCount  int

	// Transitions is the newest-first ring of outcomes feeding flap detection.
	Transitions []bool

	// ProbeSuspendedUntil stops the prober from hammering a confirmed-dead
	// credential or an endpoint that asked us to wait. Zero means not suspended.
	ProbeSuspendedUntil *time.Time

	// ProbeBackoffLevel is the exponent for the suspension above.
	ProbeBackoffLevel int

	UpdatedAt time.Time
}

// Stale reports whether the last observation is too old to be evidence.
func (r Record) Stale(interval time.Duration, now time.Time) bool {
	if r.LastObservedAt == nil {
		return false
	}
	return now.Sub(*r.LastObservedAt) > 2*interval
}

// ProbeSuspended reports whether probing is currently held off.
func (r Record) ProbeSuspended(now time.Time) bool {
	return r.ProbeSuspendedUntil != nil && now.Before(*r.ProbeSuspendedUntil)
}

// Downtime is how long this target has been failing, or zero if it is not.
func (r Record) Downtime(now time.Time) time.Duration {
	if r.DownSince == nil {
		return 0
	}
	return now.Sub(*r.DownSince)
}
