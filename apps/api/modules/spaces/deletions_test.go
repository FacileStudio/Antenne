package spaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	portepg "github.com/FacileStudio/porte/pg"
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

// accountStore is Antenne's account store over the test schema, built as the
// app builds it, with eraseData for the rows the erased person owns.
func accountStore(t *testing.T, sqlDB *sql.DB, eraseData func(context.Context, *sql.Tx, int64) error) *kit.Store {
	t.Helper()
	store, err := auth.Accounts(context.Background(), sqlDB, eraseData)
	if err != nil {
		t.Fatalf("accounts: %v", err)
	}
	return store
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
	accounts := accountStore(t, sqlDB, EraseMemberships)
	d := NewDeletions(db, directory.Client{}, Erasers{Account: accounts.EraseAccount, Identity: func(context.Context, string, string) error { return nil }}, nil)
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

// TestErasureDeletesTheBusEventsNamingThePerson erases one person with an account
// and one Registre subject Antenne never saw sign in, and proves that every bus event
// naming either is gone, matched case-blind at any depth, while an event naming an
// address that merely contains the erased one stays.
func TestErasureDeletesTheBusEventsNamingThePerson(t *testing.T) {
	db := pgtest.New(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	eraseData := func(ctx context.Context, tx *sql.Tx, userID int64) error {
		if err := EraseMemberships(ctx, tx, userID); err != nil {
			return err
		}
		return ErasePoolEvents(ctx, tx, userID)
	}
	accounts := accountStore(t, sqlDB, eraseData)
	d := NewDeletions(db, directory.Client{}, Erasers{Account: accounts.EraseAccount, Identity: func(context.Context, string, string) error { return nil }}, nil)
	seedPerson(t, db, 1, "gone@example.com")
	seedPerson(t, db, 2, "kept@example.com")
	payloads := map[string]string{
		"by-email":     `{"data":{"owner_email":"GONE@example.com","name":"p"}}`,
		"by-subject":   `{"actor":{"sub":"sub-gone@example.com"}}`,
		"by-ghost":     `{"items":[{"owner":"ghost-subject"}]}`,
		"kept":         `{"owner_email":"kept@example.com"}`,
		"lookalike":    `{"owner_email":"jimgone@example.com"}`,
		"in-free-text": `{"note":"see gone@example.com later"}`,
	}
	offset := 0
	for id, payload := range payloads {
		offset++
		if err := db.Exec(`INSERT INTO pool_events (id, channel, offset_num, payload, sender) VALUES (?, 'project.created', ?, ?::jsonb, 'Opus')`, id, offset, payload).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, subject := range []string{"sub-gone@example.com", "ghost-subject"} {
		if err := d.receiver.OnPersonDeleted(context.Background(), "iss", subject); err != nil {
			t.Fatal(err)
		}
	}

	var left []string
	if err := db.Raw(`SELECT id FROM pool_events ORDER BY id`).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(left, ","); got != "in-free-text,kept,lookalike" {
		t.Fatalf("pool_events left = %s, want in-free-text,kept,lookalike", got)
	}
}

// TestARecreatedAccountForAnErasedSubjectIsErasedAgain erases a person through
// Poll, re-creates their account as a sign-in racing the erasure would, and
// proves the next poll of the same receiver erases it again, that the guard
// porte/oidc consults answers true for that subject only, and that a stale
// directory snapshot cannot link the erased subject to an account again.
func TestARecreatedAccountForAnErasedSubjectIsErasedAgain(t *testing.T) {
	db := pgtest.New(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	registre := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": []directory.Deleted{{Subject: "sub-gone", DeletedAt: time.Now().Add(-time.Minute)}}})
	}))
	t.Cleanup(registre.Close)
	client := directory.Client{Issuer: registre.URL, ClientID: "antenne", ClientSecret: "s"}
	accounts := accountStore(t, sqlDB, EraseMemberships)
	identities := portepg.New(sqlDB)
	d := NewDeletions(db, client, Erasers{Account: accounts.EraseAccount, Identity: identities.EraseIdentity, Holds: identities.HoldsIdentity}, nil)
	seed := func(id int64, email, subject string) {
		t.Helper()
		if err := db.Exec(`INSERT INTO users (id, email) VALUES (?, ?)`, id, email).Error; err != nil {
			t.Fatal(err)
		}
		if subject != "" {
			if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, id, registre.URL, subject).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	seed(1, "gone@example.com", "sub-gone")
	seed(2, "kept@example.com", "sub-kept")
	ctx := context.Background()

	if err := d.receiver.Poll(ctx); err != nil {
		t.Fatal(err)
	}
	if erased, err := d.Erased(ctx, registre.URL, "sub-gone"); err != nil || !erased {
		t.Fatalf("Erased(sub-gone) = %v, %v; want true: sign-in and provisioning would not be refused", erased, err)
	}
	if erased, err := d.Erased(ctx, registre.URL, "sub-kept"); err != nil || erased {
		t.Fatalf("Erased(sub-kept) = %v, %v; want false", erased, err)
	}

	seed(3, "gone@example.com", "sub-gone")
	if err := d.receiver.Poll(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT count(*) FROM users WHERE email = 'gone@example.com'`); n != 0 {
		t.Fatalf("the re-created account survived the next poll: %d rows", n)
	}
	if n := count(t, db, `SELECT count(*) FROM users WHERE email = 'kept@example.com'`); n != 1 {
		t.Fatalf("another person's account was touched: %d rows", n)
	}

	seed(4, "gone@example.com", "")
	service := NewService(db, client, nil)
	service.AttachDeletions(d)
	snapshot := directory.Snapshot{Version: 1, Spaces: []directory.Space{{ID: "r-1", Name: "Team", Members: []directory.Member{{Subject: "sub-gone", Email: "gone@example.com", Role: "member"}}}}}
	if err := service.Apply(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT count(*) FROM porte_identities WHERE subject = 'sub-gone'`); n != 0 {
		t.Fatalf("the import linked an erased subject again: %d identities", n)
	}
	if n := count(t, db, `SELECT count(*) FROM space_members WHERE lower(user_id) = 'gone@example.com'`); n != 0 {
		t.Fatalf("the import kept an erased person as a member: %d rows", n)
	}
}
