package spaces

import (
	"context"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/porte/directory"
)

func count(t *testing.T, d *Deletions, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := d.orm.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// TestErasePersonKeepsOthersAndIsIdempotent proves a deleted subject loses the
// account, the porte rows and the memberships, that another person is
// untouched, and that a second call changes nothing.
func TestErasePersonKeepsOthersAndIsIdempotent(t *testing.T) {
	db := pgtest.New(t)
	d := NewDeletions(db, directory.Client{}, func(context.Context, string, string) error { return nil }, nil)
	ctx := context.Background()

	for _, person := range []struct{ id, email, subject string }{
		{"1", "gone@example.com", "sub-gone"},
		{"2", "kept@example.com", "sub-kept"},
	} {
		db.Exec(`INSERT INTO users (id, email) VALUES (?, ?)`, person.id, person.email)
		db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, 'iss', ?)`, person.id, person.subject)
		db.Exec(`INSERT INTO porte_sessions (token_hash, user_id) VALUES (?, ?)`, "h"+person.id, person.id)
		db.Exec(`INSERT INTO spaces (id, name) VALUES ('s' || ?, 'S') ON CONFLICT DO NOTHING`, person.id)
		db.Exec(`INSERT INTO space_members (space_id, user_id) VALUES ('s' || ?, ?)`, person.id, person.email)
	}
	db.Exec(`INSERT INTO events (id, source) VALUES ('e1', 'test')`)

	for range 2 {
		if err := d.ErasePerson(ctx, "iss", "sub-gone"); err != nil {
			t.Fatal(err)
		}
	}

	if n := count(t, d, `SELECT count(*) FROM users WHERE email = 'gone@example.com'`); n != 0 {
		t.Errorf("deleted account still present")
	}
	if n := count(t, d, `SELECT count(*) FROM porte_identities WHERE subject = 'sub-gone'`); n != 0 {
		t.Errorf("deleted identity still present")
	}
	if n := count(t, d, `SELECT count(*) FROM porte_sessions WHERE user_id = 1`); n != 0 {
		t.Errorf("deleted sessions still present")
	}
	if n := count(t, d, `SELECT count(*) FROM space_members WHERE user_id = 'gone@example.com'`); n != 0 {
		t.Errorf("deleted memberships still present")
	}
	if n := count(t, d, `SELECT count(*) FROM users u JOIN porte_identities i ON i.user_id = u.id
		JOIN porte_sessions s ON s.user_id = u.id JOIN space_members m ON m.user_id = u.email
		WHERE u.email = 'kept@example.com'`); n != 1 {
		t.Errorf("the other person lost data")
	}
	if n := count(t, d, `SELECT count(*) FROM events`); n != 1 {
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
