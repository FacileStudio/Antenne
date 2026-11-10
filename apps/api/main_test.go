package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	troncenv "github.com/FacileStudio/tronc/env"
)

// TestEveryRouteIsDocumented fails when a route is added without an entry in a
// module's Documentation. The reference at /docs is the contract other suite
// apps integrate against, and a silently undocumented route is how it goes
// stale.
//
// Antenne's own collaborators are nil: registering their routes only takes
// method values off them, so nothing is dereferenced and no database is needed.
// porte's kits are real, because their Mount does real work — but they are
// built over a nil *sql.DB, which is untouched until a request arrives.
func TestEveryRouteIsDocumented(t *testing.T) {
	cfg := env.Config{Core: troncenv.Core{Port: 9090}, ClientDir: t.TempDir(), AdminPassword: "documented"}
	log := slog.New(slog.DiscardHandler)
	stores := portepg.New(nil)

	sessions, err := session.New(cfg.Porte, session.Deps{Sessions: stores.Sessions(), Logger: log})
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	kit, err := oidc.New(context.Background(), cfg.Porte, oidc.Deps{Sessions: sessions, Logger: log})
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

	router, err := server.BuildRouter(cfg, log, nil, nil, authService, sessions, kit, passwords, nil, nil)
	if err != nil {
		t.Fatalf("router: %v", err)
	}

	// /dashboard is a legacy redirect kept for links made before the client moved
	// to the root; it is not part of the API surface.
	if missing := apiref.Undocumented(router, server.RefConfig(), "/dashboard", "/version"); len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}

func TestRegistryIsComplete(t *testing.T) {
	if issues := apiref.Incomplete(
		server.RefConfig(),
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
