package auth

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/porte/kit"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
)

// Accounts is the federated half of an account: porte/kit's store over
// Antenne's users table. It resolves a sign-in, backfills pictures and erases
// a person Registre erased. The kit has no admin flag for this table, because
// Antenne's is a role, so OnCreate writes the one role Antenne has.
func Accounts(db *sql.DB) *kit.Store {
	return &kit.Store{
		DB:       db,
		Columns:  kit.Columns{Picture: "picture_url"},
		OnCreate: grantAdmin,
	}
}

// grantAdmin makes a new account an administrator. Antenne has one tier: the
// dashboard holds the provider credentials, the delivery secrets and the pool
// secret, so who may sign in at all is the issuer's decision, not this one.
func grantAdmin(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET role = $1 WHERE id = $2`, string(authcontext.RoleAdmin), userID)
	return err
}
