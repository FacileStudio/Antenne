package delivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
)

// A 401 M_UNKNOWN_TOKEN carrying soft_logout keeps the device id and the
// encryption keys valid, and a refresh token can revive it. Calling it dead
// throws that away *and* suspends probing, so this one is pinned.
func TestMatrixSoftLogoutIsRefreshable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"errcode":"M_UNKNOWN_TOKEN","error":"Soft logged out","soft_logout":true}`)
	}))
	defer server.Close()

	observation := probeSender().Probe(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if observation.OK {
		t.Fatal("a soft-logged-out token was reported healthy")
	}
	if observation.Kind != health.KindRefreshable {
		t.Errorf("soft logout classified as %q, want %q", observation.Kind, health.KindRefreshable)
	}
	if observation.Kind.Terminal() {
		t.Error("soft logout suspended probing; the session is still recoverable")
	}
}

func TestMatrixHardLogoutIsDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"errcode":"M_UNKNOWN_TOKEN","error":"Invalid access token"}`)
	}))
	defer server.Close()

	observation := probeSender().Probe(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if observation.Kind != health.KindDead || !observation.Kind.Terminal() {
		t.Errorf("a revoked token classified as %q, want a terminal %q", observation.Kind, health.KindDead)
	}
}

// A working token with a missing room is a permission problem on the far side,
// not a credential problem — and treating it as dead stops the probing that
// would notice the bot being invited back.
func TestMatrixValidTokenWithMissingRoomIsDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_matrix/client/v3/account/whoami":
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@antenne:example.org", "device_id": "DEV"})
		case "/_matrix/client/v3/joined_rooms":
			json.NewEncoder(w).Encode(map[string][]string{"joined_rooms": {"!other:example.org"}})
		default:
			t.Errorf("unexpected probe request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	observation := probeSender().Probe(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if observation.OK {
		t.Fatal("a target the bot is not joined to was reported healthy")
	}
	if observation.Kind != health.KindDenied {
		t.Errorf("missing room classified as %q, want %q", observation.Kind, health.KindDenied)
	}
	if observation.Kind.Terminal() {
		t.Error("a missing room suspended probing")
	}
}

func TestMatrixTokenForAnotherAccountIsDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"user_id": "@someone:example.org", "device_id": "DEV"})
	}))
	defer server.Close()

	observation := probeSender().Probe(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if observation.OK || observation.Kind != health.KindDenied {
		t.Errorf("a token belonging to another account gave OK=%v kind=%q", observation.OK, observation.Kind)
	}
}

func TestMatrixHealthyTargetIsOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_matrix/client/v3/account/whoami":
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@antenne:example.org", "device_id": "DEV"})
		default:
			json.NewEncoder(w).Encode(map[string][]string{"joined_rooms": {"!room:example.org"}})
		}
	}))
	defer server.Close()

	observation := probeSender().Probe(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if !observation.OK || observation.Kind != health.KindNone {
		t.Errorf("a healthy Matrix target gave OK=%v kind=%q detail=%q", observation.OK, observation.Kind, observation.Detail)
	}
	if observation.Source != health.SourceProbe {
		t.Errorf("source %q, want %q", observation.Source, health.SourceProbe)
	}
}

// A missing m.room.power_levels event means the spec default applies, which
// anyone in the room clears. A 404 here is healthy.
func TestMatrixPowerLevelsMissingEventIsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"errcode":"M_NOT_FOUND"}`)
	}))
	defer server.Close()

	observation := probeSender().ProbeMatrixPowerLevels(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if !observation.OK {
		t.Errorf("a room with no power-levels event was reported unhealthy: %q", observation.Detail)
	}
}

func TestMatrixPowerLevelsBelowThresholdIsDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"events_default": 50,
			"users":          map[string]int{"@antenne:example.org": 0},
		})
	}))
	defer server.Close()

	observation := probeSender().ProbeMatrixPowerLevels(context.Background(), matrixTarget(server.URL, "!room:example.org"))
	if observation.OK || observation.Kind != health.KindDenied {
		t.Errorf("a muted bot gave OK=%v kind=%q", observation.OK, observation.Kind)
	}
}
