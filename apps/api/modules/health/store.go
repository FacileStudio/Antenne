package health

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// healthRow is the delivery_target_health table. It is a table of its own rather
// than columns on delivery_targets because settings.Store rewrites
// delivery_targets.config from its in-memory copy on every dashboard save, which
// would clobber anything health wrote there.
type healthRow struct {
	TargetID string `gorm:"column:target_id;primaryKey"`
	State    string

	ConsecutiveFailures  int
	ConsecutiveSuccesses int

	LastObservedAt *time.Time
	LastSuccessAt  *time.Time
	LastFailureAt  *time.Time
	LastSource     string
	LastKind       string
	LastDetail     string

	DownSince   *time.Time
	LastAlertAt *time.Time
	AlertCount  int

	Transitions []byte `gorm:"type:jsonb"`

	ProbeSuspendedUntil *time.Time
	ProbeBackoffLevel   int

	UpdatedAt time.Time
}

func (healthRow) TableName() string { return "delivery_target_health" }

// mutableColumns is every column an upsert may overwrite. target_id is the
// conflict key and so is absent.
var mutableColumns = []string{
	"state",
	"consecutive_failures",
	"consecutive_successes",
	"last_observed_at",
	"last_success_at",
	"last_failure_at",
	"last_source",
	"last_kind",
	"last_detail",
	"down_since",
	"last_alert_at",
	"alert_count",
	"transitions",
	"probe_suspended_until",
	"probe_backoff_level",
	"updated_at",
}

// Store persists one health record per delivery target.
//
// Unlike settings.Store there is no in-memory copy: health is written on every
// observation from both the delivery path and the prober, and a cache in front
// of a row that two goroutines update is a lost-update bug waiting for a busy
// afternoon. The table holds one row per target, so every read here is a page or
// two.
type Store struct {
	db *gorm.DB
}

// NewStore builds a health store over an existing connection.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// Load returns the health of one target. A target that has never been observed
// has no row, and gets StateUnknown rather than an error: "never measured" is a
// legitimate health, and making the caller distinguish it from a database
// failure only invites it to treat both as green.
func (s *Store) Load(ctx context.Context, targetID string) (Record, error) {
	var row healthRow
	err := s.db.WithContext(ctx).First(&row, "target_id = ?", targetID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Record{TargetID: targetID, State: StateUnknown}, nil
	}
	if err != nil {
		return Record{}, err
	}
	return row.record(), nil
}

// LoadAll returns every stored record in a stable order, so a dashboard and a
// sweep never disagree about which row they are looking at.
func (s *Store) LoadAll(ctx context.Context) ([]Record, error) {
	var rows []healthRow
	if err := s.db.WithContext(ctx).Order("target_id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, row.record())
	}
	return records, nil
}

// Upsert writes one record in a single statement. One statement matters: this is
// called on every delivery outcome and every probe, and a read-then-write pair
// here would be both twice the round trips and a race.
//
// A record for a target that no longer exists fails on the foreign key, which is
// the honest outcome: health for a deleted target is not data worth keeping.
func (s *Store) Upsert(ctx context.Context, record Record) error {
	transitions, err := json.Marshal(emptyRing(record.Transitions))
	if err != nil {
		return err
	}

	row := healthRow{
		TargetID:             record.TargetID,
		State:                string(record.State),
		ConsecutiveFailures:  record.ConsecutiveFailures,
		ConsecutiveSuccesses: record.ConsecutiveSuccesses,
		LastObservedAt:       record.LastObservedAt,
		LastSuccessAt:        record.LastSuccessAt,
		LastFailureAt:        record.LastFailureAt,
		LastSource:           string(record.LastSource),
		LastKind:             string(record.LastKind),
		LastDetail:           record.LastDetail,
		DownSince:            record.DownSince,
		LastAlertAt:          record.LastAlertAt,
		AlertCount:           record.AlertCount,
		Transitions:          transitions,
		ProbeSuspendedUntil:  record.ProbeSuspendedUntil,
		ProbeBackoffLevel:    record.ProbeBackoffLevel,
		UpdatedAt:            record.UpdatedAt,
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	}
	if row.State == "" {
		row.State = string(StateUnknown)
	}

	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns(mutableColumns),
	}).Create(&row).Error
}

// Delete drops the health of one target. Deleting a target through the settings
// store already cascades; this is for the sweep that reconciles rows left by a
// target removed while the process was down.
func (s *Store) Delete(ctx context.Context, targetID string) error {
	return s.db.WithContext(ctx).Where("target_id = ?", targetID).Delete(&healthRow{}).Error
}

func (r healthRow) record() Record {
	record := Record{
		TargetID:             r.TargetID,
		State:                State(r.State),
		ConsecutiveFailures:  r.ConsecutiveFailures,
		ConsecutiveSuccesses: r.ConsecutiveSuccesses,
		LastObservedAt:       r.LastObservedAt,
		LastSuccessAt:        r.LastSuccessAt,
		LastFailureAt:        r.LastFailureAt,
		LastSource:           Source(r.LastSource),
		LastKind:             Kind(r.LastKind),
		LastDetail:           r.LastDetail,
		DownSince:            r.DownSince,
		LastAlertAt:          r.LastAlertAt,
		AlertCount:           r.AlertCount,
		ProbeSuspendedUntil:  r.ProbeSuspendedUntil,
		ProbeBackoffLevel:    r.ProbeBackoffLevel,
		UpdatedAt:            r.UpdatedAt,
	}
	// A ring that will not parse is dropped rather than fatal: flap detection
	// rebuilds it from the next FlapWindow observations, and refusing to report
	// a target's state because its history is malformed is the wrong trade.
	if len(r.Transitions) > 0 {
		_ = json.Unmarshal(r.Transitions, &record.Transitions)
	}
	return record
}

func emptyRing(ring []bool) []bool {
	if ring == nil {
		return []bool{}
	}
	return ring
}
