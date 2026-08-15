package health

import (
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func TestShouldProbe(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	started := now.Add(-time.Hour)
	const targetID = "target-1"

	matrix := ProbeInterval(settings.TargetMatrix)

	cases := shouldProbeCases(now, targetID, matrix)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ShouldProbe(testCase.record, targetID, testCase.interval, started, now)
			if got != testCase.want {
				t.Fatalf("ShouldProbe = %v, want %v", got, testCase.want)
			}
		})
	}
}
