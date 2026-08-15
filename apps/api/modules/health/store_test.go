package health

import (
	"testing"
	"time"
)

func TestStoreRoundTripsEveryField(t *testing.T) {
	records, ctx := store(t)
	id := target(t, "round-trip")

	want := Record{
		TargetID:             id,
		State:                StateDown,
		ConsecutiveFailures:  4,
		ConsecutiveSuccesses: 1,
		LastSeen: LastSeen{
			LastObservedAt: moment(0),
			LastSuccessAt:  moment(-time.Hour),
			LastFailureAt:  moment(-time.Minute),
			LastSource:     SourceProbe,
			LastKind:       KindDead,
			LastDetail:     "404 Unknown Webhook (code 10015)",
		},
		DownSince:           moment(-2 * time.Hour),
		LastAlertAt:         moment(-90 * time.Minute),
		AlertCount:          3,
		Transitions:         []bool{false, false, true, false, true},
		ProbeSuspendedUntil: moment(time.Hour),
		ProbeBackoffLevel:   2,
		UpdatedAt:           *moment(0),
	}
	if err := records.Upsert(ctx, want); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := records.Load(ctx, id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertSameRecord(t, want, got)
}

func TestUpsertOverwritesInPlace(t *testing.T) {
	records, ctx := store(t)
	id := target(t, "overwrite")

	if err := records.Upsert(ctx, Record{TargetID: id, State: StatePending, ConsecutiveFailures: 1}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	if err := records.Upsert(ctx, Record{TargetID: id, State: StateDown, ConsecutiveFailures: 2, DownSince: moment(0)}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, err := records.Load(ctx, id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.State != StateDown || got.ConsecutiveFailures != 2 || got.DownSince == nil {
		t.Fatalf("the second write did not win: %+v", got)
	}

	var rows int64
	if err := testDB.Table("delivery_target_health").Where("target_id = ?", id).Count(&rows).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("rows = %d, want 1", rows)
	}
}

func TestLoadingAnUnobservedTargetIsUnknownNotAnError(t *testing.T) {
	records, ctx := store(t)

	got, err := records.Load(ctx, "never-observed")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.State != StateUnknown || got.TargetID != "never-observed" {
		t.Fatalf("got %+v, want an unknown record for the requested target", got)
	}
}

func TestLoadAllAndDelete(t *testing.T) {
	records, ctx := store(t)
	first := target(t, "all-a")
	second := target(t, "all-b")

	for _, id := range []string{first, second} {
		if err := records.Upsert(ctx, Record{TargetID: id, State: StateOK}); err != nil {
			t.Fatalf("Upsert %s: %v", id, err)
		}
	}

	all, err := records.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if !contains(all, first) || !contains(all, second) {
		t.Fatalf("LoadAll missed a record: %+v", all)
	}

	if err := records.Delete(ctx, first); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	all, err = records.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll after Delete: %v", err)
	}
	if contains(all, first) {
		t.Fatalf("the deleted record survived: %+v", all)
	}
	if !contains(all, second) {
		t.Fatalf("Delete took the wrong row: %+v", all)
	}
}

// TestDeletingATargetCascades pins the load-bearing half of the foreign key.
// settings.Store removes targets with DELETE ... WHERE id NOT IN (...) and knows
// nothing about health, so without the cascade every removed target leaves a row
// the dashboard keeps reporting on.
func TestDeletingATargetCascades(t *testing.T) {
	records, ctx := store(t)
	id := target(t, "cascade")

	if err := records.Upsert(ctx, Record{TargetID: id, State: StateDown}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := testDB.Exec("DELETE FROM delivery_targets WHERE id NOT IN (?)", "something-else").Error; err != nil {
		t.Fatalf("delete the target: %v", err)
	}

	var rows int64
	if err := testDB.Table("delivery_target_health").Where("target_id = ?", id).Count(&rows).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 0 {
		t.Fatalf("health rows left behind = %d, want 0", rows)
	}
}

func TestUpsertRejectsHealthForATargetThatDoesNotExist(t *testing.T) {
	records, ctx := store(t)

	if err := records.Upsert(ctx, Record{TargetID: "ghost", State: StateDown}); err == nil {
		t.Fatal("health was written for a target that does not exist")
	}
}

func contains(records []Record, targetID string) bool {
	for _, record := range records {
		if record.TargetID == targetID {
			return true
		}
	}
	return false
}

func assertSameRecord(t *testing.T, want, got Record) {
	t.Helper()

	if got.State != want.State ||
		got.ConsecutiveFailures != want.ConsecutiveFailures ||
		got.ConsecutiveSuccesses != want.ConsecutiveSuccesses ||
		got.LastSource != want.LastSource ||
		got.LastKind != want.LastKind ||
		got.LastDetail != want.LastDetail ||
		got.AlertCount != want.AlertCount ||
		got.ProbeBackoffLevel != want.ProbeBackoffLevel {
		t.Errorf("scalars differ:\n got %+v\nwant %+v", got, want)
	}

	times := map[string][2]*time.Time{
		"LastObservedAt":      {want.LastObservedAt, got.LastObservedAt},
		"LastSuccessAt":       {want.LastSuccessAt, got.LastSuccessAt},
		"LastFailureAt":       {want.LastFailureAt, got.LastFailureAt},
		"DownSince":           {want.DownSince, got.DownSince},
		"LastAlertAt":         {want.LastAlertAt, got.LastAlertAt},
		"ProbeSuspendedUntil": {want.ProbeSuspendedUntil, got.ProbeSuspendedUntil},
	}
	for field, pair := range times {
		expected, actual := pair[0], pair[1]
		if expected == nil || actual == nil {
			if expected != actual {
				t.Errorf("%s: got %v, want %v", field, actual, expected)
			}
			continue
		}
		if !expected.Equal(*actual) {
			t.Errorf("%s: got %v, want %v", field, actual, expected)
		}
	}
	if !want.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}

	if len(got.Transitions) != len(want.Transitions) {
		t.Fatalf("Transitions: got %v, want %v", got.Transitions, want.Transitions)
	}
	for index := range want.Transitions {
		if got.Transitions[index] != want.Transitions[index] {
			t.Fatalf("Transitions: got %v, want %v", got.Transitions, want.Transitions)
		}
	}
}
