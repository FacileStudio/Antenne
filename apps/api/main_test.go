package main

import (
	"log/slog"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/tronc/apiref"
	troncenv "github.com/FacileStudio/tronc/env"
)

// TestEveryRouteIsDocumented fails when a route is added without an entry in a
// module's Documentation. The reference at /docs is the contract other suite
// apps integrate against, and a silently undocumented route is how it goes
// stale.
//
// buildRouter's collaborators are nil: registering routes only takes method
// values off them, so nothing is dereferenced and no database is needed.
func TestEveryRouteIsDocumented(t *testing.T) {
	cfg := env.Config{Core: troncenv.Core{Port: 9090}, ClientDir: t.TempDir()}
	router := buildRouter(cfg, slog.New(slog.DiscardHandler), nil, nil, nil, nil, nil)

	// /dashboard is a legacy redirect kept for links made before the client moved
	// to the root; it is not part of the API surface.
	if missing := apiref.Undocumented(router, refConfig(), "/dashboard"); len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}

// TestTheReferenceRendersEveryModule catches an empty or malformed registry,
// which would otherwise serve a valid-looking but useless reference page.
func TestTheReferenceRendersEveryModule(t *testing.T) {
	document := apiref.OpenAPI(refConfig())

	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("the OpenAPI document has no paths object")
	}

	for _, expected := range []string{
		"/api/settings",
		"/api/events",
		"/api/spaces/{id}/members/{userId}",
		"/webhook/{provider}",
		"/antenne/ws",
	} {
		if _, found := paths[expected]; !found {
			t.Errorf("the reference is missing %s", expected)
		}
	}
}
