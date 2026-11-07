package spaces

import (
	"context"
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

// NewDeletions builds the receiver over Antenne's tables. erase removes porte's
// own rows for the identity, pg.Store.EraseIdentity.
func NewDeletions(
	orm *gorm.DB, client directory.Client, erase func(context.Context, string, string) error, logger *slog.Logger,
) *Deletions {
	d := &Deletions{orm: orm}
	d.receiver = &directory.Deletions{
		Client:          client,
		OnPersonDeleted: d.ErasePerson,
		Erase:           erase,
		State:           deletionState{orm: orm},
		Logger:          logger,
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

// ErasePerson deletes the account behind a Registre identity: the user row,
// which takes the porte identity, sessions, API tokens and login codes with it
// through their foreign keys, and the person's membership of every space. The
// event log, the providers and the delivery targets belong to the instance and
// stay. A subject that is already gone is a no-op, so a replayed notice
// changes nothing.
func (d *Deletions) ErasePerson(ctx context.Context, provider, subject string) error {
	var account struct {
		ID    int64
		Email string
	}
	err := d.orm.WithContext(ctx).Raw(
		`SELECT u.id, u.email FROM users u
		   JOIN porte_identities i ON i.user_id = u.id
		  WHERE i.provider = ? AND i.subject = ?`,
		provider, subject).Scan(&account).Error
	if err != nil || account.ID == 0 {
		return err
	}
	return d.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM space_members WHERE lower(user_id) = lower(?)`, account.Email).Error; err != nil {
			return err
		}
		return tx.Exec(`DELETE FROM users WHERE id = ?`, account.ID).Error
	})
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
