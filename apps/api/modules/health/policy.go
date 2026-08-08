package health

import "time"

// probeBackoffBase is the first suspension applied to a confirmed-dead
// credential, doubling per level up to probeBackoffCap. An hour is the floor
// that matters: Discord blocks an IP for about an hour after roughly 10k
// 401/403/429 responses in ten minutes, and a monitor that keeps probing a
// deleted webhook is the thing that earns that block.
const (
	probeBackoffBase = time.Hour
	probeBackoffCap  = 24 * time.Hour
)

// Evaluate folds one observation into a target's health and says what, if
// anything, should be announced. It is pure: no clock, no database, no network,
// no logger, no globals. Every input is an argument and the only output is the
// returned Decision, which is what makes the hysteresis and the repeat cadence
// testable with a table instead of a running instance.
//
// Two things it deliberately does not do. It does not jitter the repeat
// interval — the caller does, so that the same inputs always produce the same
// decision here. And it does not consider staleness: Record.Stale is a read-side
// judgement about evidence we already hold, and folding it in would mean
// inventing an observation nobody made.
func Evaluate(current Record, obs Observation, cfg Thresholds, now time.Time) Decision {
	// The entry threshold is guarded because a zero value would send a target
	// down on its first blip. The exit threshold is used exactly as given: the
	// contract says it must be >= FailuresToDown, and quietly raising it here
	// would hide a misconfiguration and disarm the test that pins the
	// asymmetry.
	if cfg.FailuresToDown < 1 {
		cfg.FailuresToDown = 1
	}

	at := now
	next := current
	next.Transitions = append([]bool(nil), current.Transitions...)
	next.LastObservedAt = &at
	next.LastSource = obs.Source
	next.LastKind = obs.Kind
	next.LastDetail = obs.Detail
	next.UpdatedAt = at

	// The far side's Retry-After is binding regardless of what else this
	// observation means.
	if obs.RetryAfter > 0 {
		suspendProbe(&next, at.Add(obs.RetryAfter))
	}

	// Three kinds say nothing about the target's health, so none of them touch
	// the counters, the transition ring or the state machine. They only ever
	// accompany a failure, which is why they are matched on the kind alone.
	switch obs.Kind {
	case KindUnsupported:
		// No safe read-only probe exists for this target type. Grey is the
		// honest colour; a green dot that cannot be justified is worse.
		next.State = StateUnknown
		return Decision{Next: next}
	case KindRateLimited:
		// Being throttled is a fact about us, not about the target. Moving it
		// toward StateDown here would open the circuit breaker on our own
		// traffic and then blame the destination for it.
		return Decision{Next: next}
	case KindUnreachable:
		// DNS, TLS, a dead network or a 5xx: the target's own state is unknown,
		// so keep the last one rather than blaming it. Grafana's keep-last-state
		// for the same reason.
		next.LastFailureAt = &at
		return Decision{Next: next}
	}

	if obs.OK {
		next.ConsecutiveSuccesses = current.ConsecutiveSuccesses + 1
		next.ConsecutiveFailures = 0
		next.LastSuccessAt = &at
		// A success proves the credential is alive, so any suspension earned by
		// a previous confirmed-dead verdict is void and the backoff restarts
		// from zero.
		next.ProbeSuspendedUntil = nil
		next.ProbeBackoffLevel = 0
		if obs.RetryAfter > 0 {
			suspendProbe(&next, at.Add(obs.RetryAfter))
		}
	} else {
		next.ConsecutiveFailures = current.ConsecutiveFailures + 1
		next.ConsecutiveSuccesses = 0
		next.LastFailureAt = &at
		if obs.Kind.Terminal() {
			suspendProbe(&next, at.Add(probeBackoff(current.ProbeBackoffLevel)))
			next.ProbeBackoffLevel = current.ProbeBackoffLevel + 1
		}
	}
	next.Transitions = pushTransition(next.Transitions, obs.OK)

	percent, measurable := flapPercent(next.Transitions)

	if current.State == StateFlapping {
		if !measurable || percent >= cfg.FlapLeave {
			next.State = StateFlapping
			return Decision{Next: next}
		}
		return resolve(current, next, obs, cfg, at, true)
	}

	if measurable && percent > cfg.FlapEnter {
		// One notice in place of the per-transition storm it replaces. It fires
		// once because it fires on the transition into StateFlapping, and
		// leaving that state is deliberately hard.
		next.State = StateFlapping
		next.LastAlertAt = &at
		next.AlertCount = current.AlertCount + 1
		return Decision{Next: next, Alert: AlertUnstable}
	}

	return resolve(current, next, obs, cfg, at, false)
}

// resolve applies the down/up hysteresis to a record whose counters are already
// updated. leftFlapping says the target has just settled out of StateFlapping,
// which is the one case where a recovery has to be announced without a recorded
// outage: the unstable notice was sent, so the loop still has to be closed.
func resolve(current, next Record, obs Observation, cfg Thresholds, at time.Time, leftFlapping bool) Decision {
	if obs.OK {
		if current.DownSince != nil {
			// The asymmetry: consecutive successes AND a minimum dwell. Either
			// one alone produces down/up/down/up and a notification for each.
			recovered := next.ConsecutiveSuccesses >= cfg.SuccessesToRecover &&
				at.Sub(*current.DownSince) >= cfg.MinDownDwell
			if !recovered {
				next.State = StateDown
				return Decision{Next: next}
			}
			downtime := at.Sub(*current.DownSince)
			announced := current.AlertCount > 0
			next.State = StateOK
			clearOutage(&next)
			if announced {
				return Decision{Next: next, Alert: AlertRecovered, Downtime: downtime}
			}
			// Broke and recovered inside the threshold: nobody was told, so
			// nobody is told it is over. A recovery notice for an outage that
			// was never announced is pure noise.
			return Decision{Next: next}
		}
		next.State = StateOK
		if leftFlapping && current.AlertCount > 0 {
			clearOutage(&next)
			return Decision{Next: next, Alert: AlertRecovered}
		}
		return Decision{Next: next}
	}

	if next.ConsecutiveFailures < cfg.FailuresToDown {
		next.State = StatePending
		return Decision{Next: next}
	}

	if current.State != StateDown {
		if next.DownSince == nil {
			next.DownSince = &at
			next.AlertCount = 0
		}
		next.State = StateDown
		next.LastAlertAt = &at
		next.AlertCount++
		return Decision{Next: next, Alert: AlertBroke, Downtime: at.Sub(*next.DownSince)}
	}

	next.State = StateDown
	if !repeatDue(current, cfg, at) {
		return Decision{Next: next}
	}
	next.LastAlertAt = &at
	next.AlertCount = current.AlertCount + 1
	return Decision{
		Next:     next,
		Alert:    AlertStillBroken,
		Downtime: at.Sub(*next.DownSince),
		Reminder: next.AlertCount - 1,
	}
}

// repeatDue reports whether a still-broken reminder is owed. Alertmanager's rule
// applies: a repeat is skipped when the state changed since the last send.
// Structurally, any re-entry into StateDown goes through the AlertBroke path and
// re-arms LastAlertAt, and a recovery clears it — so the residual case to guard
// is a send that predates the current outage entirely.
func repeatDue(current Record, cfg Thresholds, at time.Time) bool {
	if cfg.RepeatInterval <= 0 || current.LastAlertAt == nil || current.AlertCount == 0 {
		return false
	}
	if current.DownSince != nil && current.LastAlertAt.Before(*current.DownSince) {
		return false
	}
	return at.Sub(*current.LastAlertAt) >= cfg.RepeatInterval
}

func clearOutage(record *Record) {
	record.DownSince = nil
	record.LastAlertAt = nil
	record.AlertCount = 0
}

// suspendProbe holds probing off until at least the given instant. It never
// shortens an existing suspension: whichever of the two reasons to wait is
// longer wins.
func suspendProbe(record *Record, until time.Time) {
	if record.ProbeSuspendedUntil != nil && record.ProbeSuspendedUntil.After(until) {
		return
	}
	record.ProbeSuspendedUntil = &until
}

func probeBackoff(level int) time.Duration {
	backoff := probeBackoffBase
	for i := 0; i < level && backoff < probeBackoffCap; i++ {
		backoff *= 2
	}
	if backoff > probeBackoffCap {
		return probeBackoffCap
	}
	return backoff
}

// pushTransition keeps the ring newest-first and capped at FlapWindow.
func pushTransition(ring []bool, outcome bool) []bool {
	ring = append([]bool{outcome}, ring...)
	if len(ring) > FlapWindow {
		ring = ring[:FlapWindow]
	}
	return ring
}

// flapPercent is the recency-weighted percentage of state changes across the
// ring, and reports whether it is worth believing.
//
// Nagios weights the newest change about 50% more than the oldest; this is the
// linear reading of that: weights fall evenly from 1.5 on the newest change slot
// to 1.0 on the oldest, and the result is the weighted share of slots where the
// outcome flipped.
//
// It refuses to answer until the ring is full. With a partial ring a single
// break-and-recover is a large fraction of very few slots, so a target's first
// outage would be called unstable — which is precisely the false verdict flap
// detection exists to avoid.
func flapPercent(ring []bool) (float64, bool) {
	if len(ring) < FlapWindow {
		return 0, false
	}
	slots := len(ring) - 1
	var changed, total float64
	for index := 0; index < slots; index++ {
		weight := 1.5
		if slots > 1 {
			weight = 1.5 - 0.5*float64(index)/float64(slots-1)
		}
		total += weight
		if ring[index] != ring[index+1] {
			changed += weight
		}
	}
	if total == 0 {
		return 0, false
	}
	return 100 * changed / total, true
}
