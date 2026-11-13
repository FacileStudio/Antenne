package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/Antenne/apps/api/modules/mcp"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	troncenv "github.com/FacileStudio/tronc/env"
	"github.com/go-chi/chi/v5"
)

// TestEveryRouteIsDocumented fails when a route is added without an entry in a
// module's Documentation, with passwords and with single sign-on. The
// reference at /docs is the contract other suite apps integrate against, and a
// silently undocumented route is how it goes stale.
func TestEveryRouteIsDocumented(t *testing.T) {
	for name, cfg := range map[string]env.Config{"passwords": {}, "sso": ssoConfig(t)} {
		if missing := apiref.Undocumented(documentedRouter(t, cfg), server.RefConfig(mcp.Documentation()), undocumented...); len(missing) > 0 {
			t.Errorf("%s: routes missing from the API registry: %v", name, missing)
		}
	}
}

// ssoConfig points porte at a stub issuer and sets a CLI audience, which is
// what it takes to see the OIDC routes at all: porte registers none of them
// without OIDC_ISSUER, and the device exchange only with OIDC_CLI_AUDIENCE.
func ssoConfig(t *testing.T) env.Config {
	t.Helper()
	var issuer string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}))
	t.Cleanup(provider.Close)
	issuer = provider.URL
	return env.Config{Porte: porte.Config{
		Issuer:       issuer,
		ClientID:     "antenne",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:9090/api/auth/oidc/callback",
		SuccessURL:   "http://localhost:9090/",
		CLIAudience:  "facile-cli",
	}}
}

// documentedRouter builds the real router for cfg. Antenne's own
// collaborators are nil: registering their routes only takes method values
// off them, so nothing is dereferenced and no database is needed. porte's kits
// are real, because their Mount does real work, but they are built over a nil
// *sql.DB, which is untouched until a request arrives.
func documentedRouter(t *testing.T, cfg env.Config) chi.Router {
	t.Helper()
	cfg.Core = troncenv.Core{Port: 9090}
	cfg.ClientDir = t.TempDir()
	cfg.AdminPassword = "documented"
	log := slog.New(slog.DiscardHandler)
	stores := portepg.New(nil)

	sessions, err := session.New(cfg.Porte, session.Deps{Sessions: stores.Sessions(), Logger: log})
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	oidcKit, err := oidc.New(context.Background(), cfg.Porte, oidc.Deps{Users: &kit.Store{}, Identities: stores.Identities(), Codes: stores.LoginCodes(), Sessions: sessions, Logger: log})
	if err != nil {
		t.Fatalf("oidc: %v", err)
	}
	authService := auth.NewService(nil, log, auth.Config{AdminUsername: "admin", AdminPassword: "documented"}, nil, stores.Identities(), sessions)
	passwords, err := local.New(local.Config{}, local.Deps{
		Users:      auth.NewStore(nil),
		Identities: stores.Identities(),
		Sessions:   sessions,
		Logger:     log,
		Count:      func(context.Context) (int64, error) { return 1, nil },
	})
	if err != nil {
		t.Fatalf("local: %v", err)
	}

	router, err := server.BuildRouter(cfg, log, nil, nil, server.NewAPI(nil), authService, sessions, oidcKit, passwords, nil, nil, nil, (*mcp.Service)(nil))
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	return router
}

func TestRegistryIsComplete(t *testing.T) {
	if issues := apiref.Incomplete(
		server.RefConfig(mcp.Documentation()),
		"/api/auth/logout",
		"/api/auth/oidc",
		"/api/auth/oidc/callback",
		"/api/auth/backchannel-logout",
		"/api/auth/sync-profile",
		"/api/registre/sync",
		"/api/registre/deletions",
		"/api/test-alert",
		"/api/delivery-health/{targetId}/check",
		"/api/events/stream",
		"/antenne/ws",
		"/api/antenne/ws",
	); len(issues) > 0 {
		t.Errorf("incomplete documentation routes:\n  %s", strings.Join(issues, "\n  "))
	}
}

// TestTheReferenceRendersEveryModule catches an empty or malformed registry,
// which would otherwise serve a valid-looking but useless reference page.
func TestTheReferenceRendersEveryModule(t *testing.T) {
	document := apiref.OpenAPI(server.RefConfig())

	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("the OpenAPI document has no paths object")
	}

	for _, expected := range []string{
		"/api/settings",
		"/api/events",
		"/api/spaces/{id}/members",
		"/webhook/{provider}",
		"/antenne/ws",
	} {
		if _, found := paths[expected]; !found {
			t.Errorf("the reference is missing %s", expected)
		}
	}
}

// TestMaintenanceRunsEveryChoreAtBoot pins the retention fix: with a ticker far longer
// than the test, every chore still runs once, because a process redeployed several
// times a day never reaches the first tick.
func TestMaintenanceRunsEveryChoreAtBoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ran := make(chan int, 2)
	go maintenance(ctx, 24*time.Hour, func() { ran <- 1 }, func() { ran <- 2 })
	for _, want := range []int{1, 2} {
		select {
		case got := <-ran:
			if got != want {
				t.Fatalf("chore %d ran, want %d", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("chore %d did not run at boot", want)
		}
	}
}
