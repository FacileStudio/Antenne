package server

import (
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func TestBuildHealthPageJoinsRecordsAndKeepsUnknownHonest(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	brokeAt := now.Add(-90 * time.Minute)

	targets := []settings.DeliveryTarget{
		{ID: "a", Name: "Ops room", Type: settings.TargetMatrix, Enabled: true},
		{ID: "b", Name: "Generic hook", Type: settings.TargetWebhook, Enabled: true},
	}
	records := []health.Record{{
		TargetID:  "a",
		State:     health.StateDown,
		DownSince: &brokeAt,
		LastSeen: health.LastSeen{
			LastSource: health.SourceProbe,
			LastKind:   health.KindDead,
			LastDetail: "M_UNKNOWN_TOKEN: Invalid access token passed.",
		},
	}}

	page := buildHealthPage(targets, records, now)

	if len(page.Targets) != 2 {
		t.Fatalf("got %d rows, want one per target", len(page.Targets))
	}

	down := page.Targets[0]
	if down.State != "down" || down.Deliverable {
		t.Errorf("a down target reported state %q deliverable %v", down.State, down.Deliverable)
	}
	if down.DowntimeSeconds != 5400 {
		t.Errorf("downtime = %ds, want 5400", down.DowntimeSeconds)
	}
	if down.Detail != "M_UNKNOWN_TOKEN: Invalid access token passed." {
		t.Errorf("the far side's words were rewritten: %q", down.Detail)
	}
	if down.Source != "probe" {
		t.Errorf("source = %q, want probe", down.Source)
	}

	// A target with no record has never been measured. It must read as unknown
	// and it must still be deliverable — refusing to send to a target nobody has
	// measured is how one stays unmeasured forever.
	never := page.Targets[1]
	if never.State != "unknown" {
		t.Errorf("an unmeasured target reported state %q, want unknown", never.State)
	}
	if !never.Deliverable {
		t.Error("an unmeasured target was reported undeliverable")
	}
	if never.DowntimeSeconds != 0 {
		t.Errorf("an unmeasured target reported %ds of downtime", never.DowntimeSeconds)
	}

	// One reachable-enough target is all the banner needs.
	if !page.AnyDeliverable {
		t.Error("AnyDeliverable is false while a target can still receive")
	}
}

func TestBuildHealthPageIgnoresDisabledTargetsForDeliverability(t *testing.T) {
	now := time.Now()
	targets := []settings.DeliveryTarget{
		{ID: "a", Name: "Ops room", Type: settings.TargetMatrix, Enabled: true},
		{ID: "b", Name: "Spare", Type: settings.TargetNtfy, Enabled: false},
	}
	records := []health.Record{{TargetID: "a", State: health.StateDown}}

	// "b" is healthy by omission but switched off, so nothing can be delivered.
	// Counting it would hide a total outage behind a target that is not running.
	if buildHealthPage(targets, records, now).AnyDeliverable {
		t.Error("a disabled target counted towards deliverability")
	}
}
