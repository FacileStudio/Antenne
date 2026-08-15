package health

import (
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type shouldProbeCase struct {
	name     string
	record   Record
	interval time.Duration
	want     bool
}

// shouldProbeCases is the prober's schedule table: when a target is due, and
// every reason it is not.
func shouldProbeCases(now time.Time, targetID string, matrix time.Duration) []shouldProbeCase {
	return append(dueCases(now, targetID, matrix), probeSuspensionCases(now, targetID, matrix)...)
}

// dueCases cover the plain schedule: never observed, too recent, or stale
// enough that the interval has elapsed.
func dueCases(now time.Time, targetID string, matrix time.Duration) []shouldProbeCase {
	return []shouldProbeCase{
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
			record:   Record{TargetID: targetID, LastSeen: LastSeen{LastObservedAt: observedAt(now, -time.Minute)}},
			interval: matrix,
			want:     false,
		},
		{
			name:     "a stale observation is due",
			record:   Record{TargetID: targetID, LastSeen: LastSeen{LastObservedAt: observedAt(now, -2*matrix)}},
			interval: matrix,
			want:     true,
		},
	}
}

func probeSuspensionCases(now time.Time, targetID string, matrix time.Duration) []shouldProbeCase {
	return []shouldProbeCase{
		{
			name: "a suspended target is left alone",
			record: Record{
				TargetID: targetID,
				LastSeen: LastSeen{
					LastObservedAt: observedAt(now, -2*matrix),
				},
				ProbeSuspendedUntil: observedAt(now, time.Hour),
			},
			interval: matrix,
			want:     false,
		},
		{
			name: "an expired suspension no longer holds",
			record: Record{
				TargetID: targetID,
				LastSeen: LastSeen{
					LastObservedAt: observedAt(now, -2*matrix),
				},
				ProbeSuspendedUntil: observedAt(now, -time.Minute),
			},
			interval: matrix,
			want:     true,
		},
		{
			name: "a confirmed-dead credential is never probed again",
			record: Record{
				TargetID: targetID,
				LastSeen: LastSeen{
					LastObservedAt: observedAt(now, -24*time.Hour),
					LastKind:       KindDead,
				},
			},
			interval: matrix,
			want:     false,
		},
		{
			name: "a rate-limited target is still probed once the wait is over",
			record: Record{
				TargetID: targetID,
				LastSeen: LastSeen{
					LastObservedAt: observedAt(now, -2*matrix),
					LastKind:       KindRateLimited,
				},
			},
			interval: matrix,
			want:     true,
		},
	}
}
