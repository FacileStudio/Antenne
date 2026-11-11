package spaces

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/porte/directory"
	"gorm.io/gorm"
)

// deletionStateName is the one row the receiver's state lives in.
const deletionStateName = "registre"

// Deletions erases the people Registre erased. It polls Registre's deletion
// list with the same client credentials as the snapshot sync and accepts
// Registre's signed push on directory.DeletionsPath.
type Deletions struct {
	orm      *gorm.DB
	receiver *directory.Deletions
}

// NewDeletions builds the receiver over Antenne's tables. eraseAccount deletes
// the account behind a Registre identity, porte/kit's Store.EraseAccount with
// EraseMemberships and ErasePoolEvents in its EraseData. The bus events naming
// the subject go next, for a person who never had an Antenne account. erase
// then removes porte's own rows for the identity, pg.Store.EraseIdentity.
func NewDeletions(orm *gorm.DB, client directory.Client, eraseAccount, erase Eraser, logger *slog.Logger) *Deletions {
	d := &Deletions{orm: orm}
	d.receiver = &directory.Deletions{
		Client: client,
		OnPersonDeleted: func(ctx context.Context, provider, subject string) error {
			if err := eraseAccount(ctx, provider, subject); err != nil {
				return err
			}
			return orm.WithContext(ctx).Exec(poolEventsNaming+`(SELECT lower(?::text)))`, subject).Error
		},
		Erase:  erase,
		State:  deletionState{orm: orm},
		Logger: logger,
	}
	return d
}

// Run polls until ctx ends. Start it in exactly one replica.
func (d *Deletions) Run(ctx context.Context) {
	d.receiver.Run(ctx)
}

// Handler verifies and applies Registre's signed user.deleted push.
func (d *Deletions) Handler() http.Handler {
	return d.receiver.Handler()
}

// Eraser removes what one (provider, subject) identity owns.
type Eraser func(ctx context.Context, provider, subject string) error

// EraseMemberships deletes the erased person's membership of every space. It
// is the EraseData of the account store, so it runs in the transaction that
// deletes the users row, which takes the porte identity, sessions, API tokens
// and login codes with it through their foreign keys. The event log, the
// providers and the delivery targets belong to the instance and stay.
func EraseMemberships(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM space_members WHERE lower(user_id) = (SELECT lower(email) FROM users WHERE id = $1)`, userID)
	return err
}

// poolEventsNaming deletes every bus event whose payload holds, as any string value
// at any depth, one of the lowered names the appended subquery returns. An exact
// value match, so erasing bob@example.com leaves jimbob@example.com's events.
const poolEventsNaming = `DELETE FROM pool_events e WHERE EXISTS (
	SELECT 1 FROM jsonb_path_query(e.payload, 'strict $.**') AS v
	WHERE jsonb_typeof(v) = 'string' AND lower(v #>> '{}') IN `

// ErasePoolEvents deletes the bus events that name the erased person by email or by
// any of their identity subjects, in the transaction that deletes the account. Apps
// publish payloads such as owner_email and the Registre subject on the bus, and
// retention alone keeps them for up to 30 days.
func ErasePoolEvents(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, poolEventsNaming+`(
		SELECT lower(email) FROM users WHERE id = $1 AND email <> ''
		UNION SELECT lower(subject) FROM porte_identities WHERE user_id = $1 AND subject <> ''))`, userID)
	return err
}

// deletionState keeps the receiver's cursor and handled subjects in one row.
type deletionState struct {
	orm *gorm.DB
}

// Load returns the saved state, or nil before the first save.
func (s deletionState) Load(ctx context.Context) ([]byte, error) {
	var rows []struct{ State []byte }
	err := s.orm.WithContext(ctx).Raw(`SELECT state FROM deletion_state WHERE name = ?`, deletionStateName).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].State, nil
}

// Save replaces the saved state.
func (s deletionState) Save(ctx context.Context, state []byte) error {
	return s.orm.WithContext(ctx).Exec(
		`INSERT INTO deletion_state (name, state, updated_at) VALUES (?, ?, now())
		 ON CONFLICT (name) DO UPDATE SET state = EXCLUDED.state, updated_at = now()`,
		deletionStateName, state).Error
}
