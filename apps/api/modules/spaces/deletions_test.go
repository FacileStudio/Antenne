package spaces

import (
	"context"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"gorm.io/gorm"
)

func count(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// userTables is every table that references a person, with the query that
// counts the rows naming the account of the given address.
var userTables = map[string]string{
	"users":             `SELECT count(*) FROM users WHERE email = @email`,
	"porte_identities":  `SELECT count(*) FROM porte_identities i JOIN users u ON u.id = i.user_id WHERE u.email = @email OR i.subject = 'sub-' || @email`,
	"porte_sessions":    `SELECT count(*) FROM porte_sessions WHERE token_hash = 'h-' || @email`,
	"porte_login_codes": `SELECT count(*) FROM porte_login_codes WHERE code_hash = 'c-' || @email`,
	"sessions":          `SELECT count(*) FROM sessions WHERE token_hash = 'l-' || @email`,
	"space_members":     `SELECT count(*) FROM space_members WHERE lower(user_id) = @email`,
}

// seedPerson gives one person a row in every table of userTables.
func seedPerson(t *testing.T, db *gorm.DB, id int64, email string) {
	t.Helper()
	statements := []string{
		`INSERT INTO users (id, email) VALUES (@id, @email)`,
		`INSERT INTO porte_identities (user_id, provider, subject) VALUES (@id, 'iss', 'sub-' || @email)`,
		`INSERT INTO porte_sessions (token_hash, user_id) VALUES ('h-' || @email, @id)`,
		`INSERT INTO porte_login_codes (code_hash, user_id, expires_at) VALUES ('c-' || @email, @id, now())`,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('l-' || @email, @id, now())`,
		`INSERT INTO spaces (id, name) VALUES (@space, 'S')`,
		`INSERT INTO space_members (space_id, user_id) VALUES (@space, upper(@email))`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement, map[string]any{"id": id, "email": email, "space": "s-" + email}).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

// TestErasureLeavesNoRowNamingThePerson erases a person through the callback
// the directory receiver calls, porte/kit's EraseAccount with
// EraseMemberships, and proves that no table still names them, that another
// person keeps every row, that the instance's event log is untouched, and that
// a replayed notice changes nothing.
func TestErasureLeavesNoRowNamingThePerson(t *testing.T) {
	db := pgtest.New(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	accounts := &kit.Store{DB: sqlDB, Columns: kit.Columns{Picture: "picture_url"}, EraseData: EraseMemberships}
	d := NewDeletions(db, directory.Client{}, accounts.EraseAccount, func(context.Context, string, string) error { return nil }, nil)
	seedPerson(t, db, 1, "gone@example.com")
	seedPerson(t, db, 2, "kept@example.com")
	db.Exec(`INSERT INTO events (id, source) VALUES ('e1', 'test')`)

	for range 2 {
		if err := d.receiver.OnPersonDeleted(context.Background(), "iss", "sub-gone@example.com"); err != nil {
			t.Fatal(err)
		}
	}

	for table, query := range userTables {
		if n := count(t, db, query, map[string]any{"email": "gone@example.com"}); n != 0 {
			t.Errorf("%s still holds %d rows naming the erased person", table, n)
		}
		if n := count(t, db, query, map[string]any{"email": "kept@example.com"}); n == 0 {
			t.Errorf("%s lost the other person's row", table)
		}
	}
	if n := count(t, db, `SELECT count(*) FROM events`); n != 1 {
		t.Errorf("the instance's event log was touched")
	}
}

// TestDeletionStateRoundTrips checks the persisted StateStore.
func TestDeletionStateRoundTrips(t *testing.T) {
	state := deletionState{orm: pgtest.New(t)}
	ctx := context.Background()
	if got, err := state.Load(ctx); err != nil || got != nil {
		t.Fatalf("empty load = %q, %v", got, err)
	}
	for _, value := range []string{"one", "two"} {
		if err := state.Save(ctx, []byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := state.Load(ctx); string(got) != "two" {
		t.Fatalf("load = %q, want two", got)
	}
}
