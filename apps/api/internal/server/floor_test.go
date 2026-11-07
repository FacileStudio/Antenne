package server_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	troncenv "github.com/FacileStudio/tronc/env"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
)

const poolSecret = "pool-secret"

func floorRouter(t *testing.T, db *gorm.DB) (http.Handler, *session.Manager) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	cfg := env.Config{
		Core:          troncenv.Core{Port: 9090},
		ClientDir:     t.TempDir(),
		AdminUsername: "admin",
		AdminPassword: "bootstrap-password",
		Version:       "v-test",
		Porte:         porte.Config{Issuer: "https://sso.example", ClientID: "antenne"},
	}
	stores := portepg.New(sqlDB)
	sessions, err := session.New(cfg.Porte, session.Deps{Sessions: stores.Sessions(), Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	userStore := auth.NewStore(db)
	service := auth.NewService(db, log, auth.Config{
		AdminUsername: "admin", AdminPassword: "bootstrap-password", Issuer: cfg.Porte.Issuer,
	}, userStore, stores.Identities(), sessions)
	passwords, err := local.New(local.Config{AllowRegistration: true}, local.Deps{
		Users: userStore, Identities: stores.Identities(), Sessions: sessions, Logger: log, Count: userStore.CountUsers,
	})
	if err != nil {
		t.Fatal(err)
	}
	kit, err := oidc.New(context.Background(), porte.Config{}, oidc.Deps{Sessions: sessions, Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	router := server.BuildRouter(cfg, log, sqlDB, nil, service, sessions, kit, passwords, nil, bus.NewServer(db, log, poolSecret))
	return router, sessions
}

func send(router http.Handler, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range header {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// TestSessionlessCallersStillWork covers what the Registre-only sign-in must
// not break: a CLI or another app holding a bearer token reaches the API with
// no cookie, a pool client registers with the shared secret and no session,
// the dashboard API refuses an anonymous caller, and no password or
// registration route answers once an issuer is configured.
func TestSessionlessCallersStillWork(t *testing.T) {
	db := pgtest.New(t)
	router, sessions := floorRouter(t, db)

	user := auth.User{Email: "cli@example.com", Role: "ADMIN"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	token, _, err := sessions.Issue(context.Background(), user.ID, "cli")
	if err != nil {
		t.Fatal(err)
	}

	bearer := send(router, "GET", "/api/session", "", map[string]string{"Authorization": "Bearer " + token})
	if bearer.Code != http.StatusOK || !strings.Contains(bearer.Body.String(), "cli@example.com") {
		t.Errorf("bearer token: %d %s", bearer.Code, bearer.Body.String())
	}
	if anonymous := send(router, "GET", "/api/session", "", nil); anonymous.Code != http.StatusUnauthorized {
		t.Errorf("anonymous dashboard call: %d, want 401", anonymous.Code)
	}

	register := `{"app":"opus","instance_id":"i1","secret":"` + poolSecret + `"}`
	for _, path := range []string{"/antenne/register", "/api/antenne/register"} {
		if got := send(router, "POST", path, register, nil); got.Code != http.StatusOK {
			t.Errorf("pool register %s: %d %s", path, got.Code, got.Body.String())
		}
	}

	for _, path := range []string{"/api/auth/login", "/api/auth/register"} {
		body := `{"email":"admin@antenne.local","password":"bootstrap-password","name":"x"}`
		if got := send(router, "POST", path, body, nil); got.Code != http.StatusNotFound {
			t.Errorf("%s answered %d, want 404 with an issuer configured", path, got.Code)
		}
	}

	if got := send(router, "GET", "/version", "", nil); !strings.Contains(got.Body.String(), "v-test") {
		t.Errorf("/version: %s", got.Body.String())
	}
}
