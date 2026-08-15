package health

import (
	"hash/fnv"
	"time"
)

// ShouldProbe answers the scheduler's only question. It is pure so the answer
// can be pinned in a table, because every clause here is a way to do harm:
//
//   - A zero interval means the type has no safe read-only probe.
//   - A suspended target was told to wait, or has a credential the far side has
//     already rejected outright.
//   - A terminal kind is a confirmed-dead credential. Probing it on a schedule
//     can only get the instance blocked; it takes a config change to fix, and a
//     config change re-applies the runtime.
//   - A target with a fresh observation needs nothing. A real delivery outcome is
//     free and more authoritative than any probe, and monitoring that costs more
//     than the work it monitors gets switched off by the first operator who
//     notices.
//
// startedAt is when the scheduler came up, and is used only for a target that has
// never been observed: without it every such target would be probed on the very
// first sweep, which is exactly the restart spike the phase exists to prevent.
func ShouldProbe(record Record, targetID string, interval time.Duration, startedAt, now time.Time) bool {
	if interval <= 0 {
		return false
	}
	if record.ProbeSuspended(now) {
		return false
	}
	if record.LastKind.Terminal() {
		return false
	}
	if record.LastObservedAt == nil {
		return !startedAt.Add(probePhase(targetID, interval)).After(now)
	}
	return !NextProbeAt(targetID, interval, *record.LastObservedAt).After(now)
}

// NextProbeAt is when this target may next be probed, given its last
// observation from any source.
//
// Two properties, and both matter:
//
//   - The gap is never shorter than the interval. Aligning to a shared grid
//     instead would let a delivery that landed one second before a slot be
//     followed immediately by a probe, which is the opposite of passive-first.
//   - The phase is derived from the target id, not from process start, so a
//     restart does not re-align every target onto the same tick. It is capped at
//     half the interval to keep the effective gap inside the researched band —
//     30 minutes of SMTP becomes 30 to 45, not 30 to 60.
func NextProbeAt(targetID string, interval time.Duration, since time.Time) time.Time {
	if interval <= 0 {
		return since
	}
	return since.Add(interval).Add(probePhase(targetID, interval/2))
}

// probePhase spreads targets across the interval so a restart does not line
// every probe up on the same second.
//
// There is no error path because hash.Hash.Write is documented never to return
// one; fnv has nothing to fail at.
func probePhase(targetID string, span time.Duration) time.Duration {
	if span <= 0 {
		return 0
	}
	sum := fnv.New64a()
	sum.Write([]byte(targetID))
	return time.Duration(sum.Sum64() % uint64(span))
}
