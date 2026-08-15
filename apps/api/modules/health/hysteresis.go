package health

import "time"

// evaluation is one pass of the state machine: the record as it stands, the
// record being built from it, and the inputs both are judged against.
type evaluation struct {
	current      Record
	next         Record
	obs          Observation
	cfg          Thresholds
	at           time.Time
	leftFlapping bool
}

// afterFlap applies flap detection, which outranks the down/up hysteresis
// because an unstable target is not usefully described as either.
func (e evaluation) afterFlap() Decision {
	percent, measurable := flapPercent(e.next.Transitions)

	if e.current.State == StateFlapping {
		if !measurable || percent >= e.cfg.FlapLeave {
			e.next.State = StateFlapping
			return Decision{Next: e.next}
		}
		e.leftFlapping = true
		return e.resolve()
	}

	if measurable && percent > e.cfg.FlapEnter {
		// One notice in place of the per-transition storm it replaces. It fires
		// once because it fires on the transition into StateFlapping, and
		// leaving that state is deliberately hard.
		e.next.State = StateFlapping
		e.next.LastAlertAt = &e.at
		e.next.AlertCount = e.current.AlertCount + 1
		return Decision{Next: e.next, Alert: AlertUnstable}
	}

	return e.resolve()
}

// resolve applies the down/up hysteresis to a record whose counters are already
// updated.
func (e evaluation) resolve() Decision {
	if e.obs.OK {
		return e.resolveRecovery()
	}
	return e.resolveFailure()
}

// resolveRecovery closes an outage, or declines to.
//
// leftFlapping is the one case where a recovery has to be announced without a
// recorded outage: the unstable notice was sent, so the loop still has to be
// closed.
func (e evaluation) resolveRecovery() Decision {
	next := e.next
	if e.current.DownSince == nil {
		next.State = StateOK
		if e.leftFlapping && e.current.AlertCount > 0 {
			clearOutage(&next)
			return Decision{Next: next, Alert: AlertRecovered}
		}
		return Decision{Next: next}
	}

	// The asymmetry: consecutive successes AND a minimum dwell. Either one
	// alone produces down/up/down/up and a notification for each.
	recovered := next.ConsecutiveSuccesses >= e.cfg.SuccessesToRecover &&
		e.at.Sub(*e.current.DownSince) >= e.cfg.MinDownDwell
	if !recovered {
		// Answering, but the outage is not closed yet. Saying "down" here is
		// the one thing an operator can disprove by hand.
		next.State = StateRecovering
		return Decision{Next: next}
	}

	downtime := e.at.Sub(*e.current.DownSince)
	announced := e.current.AlertCount > 0
	next.State = StateOK
	clearOutage(&next)
	if announced {
		return Decision{Next: next, Alert: AlertRecovered, Downtime: downtime}
	}
	// Broke and recovered inside the threshold: nobody was told, so nobody is
	// told it is over. A recovery notice for an outage that was never announced
	// is pure noise.
	return Decision{Next: next}
}

/*
resolveFailure opens an outage, promotes a pending one, or repeats a reminder.

DownSince, not the state, is what says an outage is open — and it is the only
thing that survives a target answering once in the middle of one.

Reading the state instead had two costs. A failure arriving after a success
inside an open outage restarted the failure count, so the target fell back to
StatePending: an amber "not confirmed yet" dot on an outage already announced to
everyone. And the next failure then found a state that was not StateDown, took
the first-notice path, and sent a second "broke" alert for an outage that never
ended.
*/
func (e evaluation) resolveFailure() Decision {
	next := e.next
	if e.current.DownSince == nil {
		if !e.confirmedDown() {
			next.State = StatePending
			return Decision{Next: next}
		}
		next.DownSince = &e.at
		next.AlertCount = 0
	}
	next.State = StateDown

	// StateRecovering is the same outage still open, so it owes no first notice.
	if e.current.State != StateDown && e.current.State != StateRecovering {
		next.LastAlertAt = &e.at
		next.AlertCount++
		return Decision{Next: next, Alert: AlertBroke, Downtime: e.at.Sub(*next.DownSince)}
	}

	if !repeatDue(e.current, e.cfg, e.at) {
		return Decision{Next: next}
	}
	next.LastAlertAt = &e.at
	next.AlertCount = e.current.AlertCount + 1
	return Decision{
		Next:     next,
		Alert:    AlertStillBroken,
		Downtime: e.at.Sub(*next.DownSince),
		Reminder: next.AlertCount - 1,
	}
}

/*
confirmedDown reports whether this failure is enough to open an outage.

A terminal kind skips the entry threshold, because it is certainty rather than a
sample. The threshold exists to filter blips; `M_FORBIDDEN` on a login, a
deleted Discord webhook and SMTP 535 are not blips, and waiting for a second
opinion on them buys nothing.

It is also load-bearing rather than tidy. A terminal kind suspends probing for an
hour and then longer, which is what stops us hammering a far side that has
already said no — so a target parked in StatePending would never receive the
second failure that promotes it, and would sit broken and silent indefinitely.
Production found this within a minute of the first deploy: a Matrix account over
its device limit, correctly suspended, correctly diagnosed, and reported to
nobody.
*/
func (e evaluation) confirmedDown() bool {
	return e.next.ConsecutiveFailures >= e.cfg.FailuresToDown || e.obs.Kind.Terminal()
}
