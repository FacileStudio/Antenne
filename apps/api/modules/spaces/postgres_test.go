package spaces

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/migrate"
	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
)

// testDB is the scoped handle every test in this package shares, and skipReason
// is why there is none. The guard under test is a membership lookup against the
// Postgres DDL in apps/api/migrations, so no other engine would prove anything
// about it.
var (
	testDB     *gorm.DB
	skipReason string
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// Loud on stderr as well as skipped: `go test` without -v shows a skip
		// nowhere, and an authorization test that quietly does not run is worse
		// than one that fails.
		skipReason = "TEST_DATABASE_URL is not set, so the spaces guard was NOT exercised against Postgres"
		fmt.Fprintf(os.Stderr, "spaces: %s\n", skipReason)
		os.Exit(m.Run())
	}

	schema := fmt.Sprintf("spaces_test_%d", time.Now().UnixNano())
	admin, err := open(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spaces: TEST_DATABASE_URL is set but unusable: %v\n", err)
		os.Exit(1)
	}
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		fmt.Fprintf(os.Stderr, "spaces: could not create the scratch schema: %v\n", err)
		os.Exit(1)
	}

	scoped, err := open(withSearchPath(dsn, schema))
	if err != nil {
		fmt.Fprintf(os.Stderr, "spaces: could not open the scratch schema: %v\n", err)
		os.Exit(1)
	}
	sqlDB, err := scoped.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "spaces: no database handle: %v\n", err)
		os.Exit(1)
	}
	if err := migrate.Run(context.Background(), migrate.Config{
		DB:     sqlDB,
		FS:     os.DirFS("../../migrations"),
		Logger: slog.New(slog.DiscardHandler),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "spaces: migrations failed: %v\n", err)
		os.Exit(1)
	}

	testDB = scoped
	code := m.Run()

	if err := sqlDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "spaces: closing the test connection failed: %v\n", err)
	}
	if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
		fmt.Fprintf(os.Stderr, "spaces: dropping schema %s failed, it is now orphaned: %v\n", schema, err)
	}
	os.Exit(code)
}

func open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  gormlogger.Discard,
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
}

func withSearchPath(dsn, schema string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn + " search_path=" + schema
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// user is a signed-in caller. Antenne's own admin role is deliberately not a
// space role, so admin(...) exists only to prove it buys nothing here.
func user(email string) authcontext.Identity {
	return authcontext.Identity{UserID: 1, Email: email, Role: authcontext.RoleUser}
}

func admin(email string) authcontext.Identity {
	return authcontext.Identity{UserID: 2, Email: email, Role: authcontext.RoleAdmin}
}

// call runs one request through the real router with identity already on the
// context, which is what auth.RequireAuth does in production.
func call(t *testing.T, identity authcontext.Identity, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	if testDB == nil {
		t.Skip(skipReason)
	}

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authcontext.With(r.Context(), identity)))
		})
	})
	NewService(testDB).RegisterRoutes(router)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// newSpace creates a space through the API, so the owner is seeded the way a
// real caller would seed it.
func newSpace(t *testing.T, owner authcontext.Identity, name string) string {
	t.Helper()
	return newSpaceWithBody(t, owner, `{"name":"`+name+`"}`)
}

func newSpaceWithBody(t *testing.T, owner authcontext.Identity, body string) string {
	t.Helper()
	recorder := call(t, owner, http.MethodPost, "/spaces", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("creating a space: got %d, body %s", recorder.Code, recorder.Body)
	}

	var payload struct {
		Space Space `json:"space"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding the created space: %v", err)
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM spaces WHERE id = ?", payload.Space.ID) })
	return payload.Space.ID
}

func addMember(t *testing.T, actor authcontext.Identity, spaceID, email string, role Role) {
	t.Helper()
	body := `{"userId":"` + email + `","role":"` + string(role) + `"}`
	recorder := call(t, actor, http.MethodPost, "/spaces/"+spaceID+"/members", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("adding %s as %s: got %d, body %s", email, role, recorder.Code, recorder.Body)
	}
}

// probe is one request the guard must refuse.
type probe struct {
	method string
	path   string
	body   string
}

// refusedProbes is every route that takes a space id, so a new route added
// without a guard shows up here as a missing line rather than as a silent hole.
func refusedProbes(id string) []probe {
	return []probe{
		{http.MethodGet, "/spaces/" + id, ""},
		{http.MethodPut, "/spaces/" + id, `{"name":"Renamed"}`},
		{http.MethodDelete, "/spaces/" + id, ""},
		{http.MethodGet, "/spaces/" + id + "/members", ""},
		{http.MethodPost, "/spaces/" + id + "/members", `{"userId":"intruder@example.test","role":"owner"}`},
		{http.MethodPut, "/spaces/" + id + "/members/owner@example.test", `{"role":"member"}`},
		{http.MethodDelete, "/spaces/" + id + "/members/owner@example.test", ""},
	}
}
