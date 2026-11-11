package spaces

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/porte/directory"
)

const testIssuer = "https://sso.example.test"

// seedUser inserts a user, with a porte identity under testIssuer when subject
// is set, and removes both after the test.
func seedUser(t *testing.T, email, subject string) {
	t.Helper()
	var id int64
	if err := testDB.Raw(`INSERT INTO users (email) VALUES (?) RETURNING id`, email).Scan(&id).Error; err != nil {
		t.Fatalf("seeding %s: %v", email, err)
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM users WHERE email = ?", email) })
	if subject == "" {
		return
	}
	if err := testDB.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, id, testIssuer, subject).Error; err != nil {
		t.Fatalf("seeding the identity of %s: %v", email, err)
	}
}

// state is everything Apply may touch, read in a stable order.
func state(t *testing.T) map[string]any {
	t.Helper()
	read := func(query string) []map[string]any {
		rows := []map[string]any{}
		if err := testDB.Raw(query).Scan(&rows).Error; err != nil {
			t.Fatalf("reading state: %v", err)
		}
		return rows
	}
	return map[string]any{
		"spaces":     read(`SELECT id, name, registre_space_id, updated_at FROM spaces ORDER BY id`),
		"members":    read(`SELECT space_id, user_id, role, joined_at FROM space_members ORDER BY space_id, user_id`),
		"users":      read(`SELECT id, email, name, picture_url, updated_at FROM users ORDER BY id`),
		"identities": read(`SELECT user_id, provider, subject FROM porte_identities ORDER BY provider, subject`),
	}
}

func roles(t *testing.T, spaceID string) map[string]Role {
	t.Helper()
	members := []Member{}
	if err := testDB.Where("space_id = ?", spaceID).Find(&members).Error; err != nil {
		t.Fatalf("reading members: %v", err)
	}
	got := map[string]Role{}
	for _, member := range members {
		got[member.UserID] = member.Role
	}
	return got
}

// TestApplyMirrorsTheSnapshot covers the whole contract of Apply: link by name,
// create, hold a member who never signed in by address with no account, link an
// existing user by address, role mapping, the removal that spares local-only
// members, and a second apply changing nothing.
func TestApplyMirrorsTheSnapshot(t *testing.T) {
	studio := newSpace(t, user("local@antenne.local"), "Studio")
	seedUser(t, "gone@example.test", "sub-gone")
	seedUser(t, "kept@example.test", "sub-kept")
	seedUser(t, "match@example.test", "")
	t.Cleanup(func() { testDB.Exec("DELETE FROM users WHERE email = 'new@example.test'") })
	addMember(t, studio, "gone@example.test", RoleMember)
	addMember(t, studio, "kept@example.test", RoleAdmin)

	snapshot := directory.Snapshot{Version: 7, Spaces: []directory.Space{
		{ID: "reg-studio", Name: "studio", Members: []directory.Member{
			{Subject: "sub-kept", Email: "kept@example.test", Role: directory.RoleMember},
			{Subject: "sub-new", Email: "New@Example.test", Name: "New Mate", Picture: testIssuer + "/avatars/sub-new", Role: directory.RoleOwner},
			{Subject: "sub-match", Email: "match@example.test", Role: directory.RoleAdmin},
			{Subject: "sub-odd", Email: "", Role: "viewer"},
		}},
		{ID: "reg-fresh", Name: "Fresh", Members: []directory.Member{
			{Subject: "sub-kept", Email: "kept@example.test", Role: "viewer"},
		}},
	}}
	service := NewService(testDB, directory.Client{Issuer: testIssuer}, slog.New(slog.DiscardHandler))
	if err := service.Apply(context.Background(), snapshot); err != nil {
		t.Fatalf("apply: %v", err)
	}

	linked, err := service.Get(studio)
	if err != nil || linked.RegistreSpaceID == nil || *linked.RegistreSpaceID != "reg-studio" || linked.Name != "studio" {
		t.Fatalf("Studio was not linked by name and renamed: %+v %v", linked, err)
	}
	want := map[string]Role{
		"local@antenne.local": RoleOwner,
		"kept@example.test":   RoleMember,
		"new@example.test":    RoleOwner,
		"match@example.test":  RoleAdmin,
	}
	if got := roles(t, studio); !reflect.DeepEqual(got, want) {
		t.Errorf("studio members: got %v, want %v", got, want)
	}

	var fresh Space
	if err := testDB.Where("registre_space_id = ?", "reg-fresh").Take(&fresh).Error; err != nil {
		t.Fatalf("Fresh was not created: %v", err)
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM spaces WHERE id = ?", fresh.ID) })
	if got := roles(t, fresh.ID); !reflect.DeepEqual(got, map[string]Role{"kept@example.test": RoleMember}) {
		t.Errorf("an unknown role did not map to member: %v", got)
	}

	var accounts int64
	testDB.Raw(`SELECT count(*) FROM users WHERE lower(email) = 'new@example.test'`).Scan(&accounts)
	if accounts != 0 {
		t.Errorf("the import created an account for a member who never signed in")
	}
	var matched int64
	testDB.Raw(`SELECT count(*) FROM porte_identities JOIN users ON users.id = porte_identities.user_id
	             WHERE users.email = 'match@example.test' AND subject = 'sub-match'`).Scan(&matched)
	if matched != 1 {
		t.Errorf("the existing user was not given the registre identity")
	}

	before := state(t)
	if err := service.Apply(context.Background(), snapshot); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if after := state(t); !reflect.DeepEqual(before, after) {
		t.Errorf("a second apply changed state:\nbefore %v\nafter  %v", before, after)
	}

	snapshot.Spaces[0].Members = snapshot.Spaces[0].Members[:1]
	if err := service.Apply(context.Background(), snapshot); err != nil {
		t.Fatalf("third apply: %v", err)
	}
	want = map[string]Role{"local@antenne.local": RoleOwner, "kept@example.test": RoleMember}
	if got := roles(t, studio); !reflect.DeepEqual(got, want) {
		t.Errorf("a member without an account was not removed with the others: got %v, want %v", got, want)
	}
}

// TestImportRunsBeforeTheFirstSync pins the order that keeps a sync from
// removing members Registre has not heard of: while the import fails, here
// with the 404 an undeployed Registre answers, no snapshot is pulled.
func TestImportRunsBeforeTheFirstSync(t *testing.T) {
	if testDB == nil {
		t.Skip(skipReason)
	}
	var (
		mu    sync.Mutex
		calls []string
	)
	pulled := make(chan struct{}, 1)
	registre := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/api/directory/import" && len(calls) <= 2 {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(directory.Snapshot{Version: 1})
		if r.URL.Path == "/api/directory/snapshot" {
			select {
			case pulled <- struct{}{}:
			default:
			}
		}
	}))
	defer registre.Close()

	service := NewService(testDB, directory.Client{Issuer: registre.URL, ClientID: "antenne", ClientSecret: "s"}, slog.New(slog.DiscardHandler))
	service.retry = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { service.Run(ctx); close(done) }()

	select {
	case <-pulled:
	case <-time.After(5 * time.Second):
		t.Fatal("no snapshot was pulled after the import succeeded")
	}
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	want := []string{"POST /api/directory/import", "POST /api/directory/import", "POST /api/directory/import", "GET /api/directory/snapshot"}
	if !reflect.DeepEqual(calls[:4], want) {
		t.Errorf("calls: got %v, want %v first", calls, want)
	}
}
