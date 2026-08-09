package delivery

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// TestMatrixLoginAlwaysNamesTheSameDevice is the one that matters for the
// account rather than for the message.
//
// A Matrix login that names no device_id makes the homeserver mint a new
// device, and the session cache lives in the Sender, so it dies with the
// process. That is one new device per restart: the account's device list fills
// with identical entries until it reaches its device limit, and then the login
// that would fix it is the login that fails.
//
// The two Senders here are two boots of the same container.
func TestMatrixLoginAlwaysNamesTheSameDevice(t *testing.T) {
	target := settings.DeliveryTarget{
		ID:         "9f1c0c1e-2f2a-4f6b-9b7e-0a1d2c3b4a59",
		Type:       settings.TargetMatrix,
		Homeserver: "https://matrix.org",
		UserID:     "@nook-facile:matrix.org",
	}

	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	first := NewSender(discard).matrixSessionFor(target).deviceID
	second := NewSender(discard).matrixSessionFor(target).deviceID

	if first == "" {
		t.Fatal("a target with no configured deviceId logs in anonymously, minting a device per restart")
	}
	if first != second {
		t.Errorf("the device id did not survive a restart: %q then %q", first, second)
	}
	if first != settings.MatrixDeviceID(target.ID) {
		t.Errorf("device id = %q, want the value derived from the target id", first)
	}
}

// TestAConfiguredDeviceIDWins keeps the derivation from overriding an operator
// who pointed a target at a device that already exists.
func TestAConfiguredDeviceIDWins(t *testing.T) {
	configured := "ANTENNEDEADBEEF01"
	target := settings.DeliveryTarget{
		ID:       "9f1c0c1e-2f2a-4f6b-9b7e-0a1d2c3b4a59",
		Type:     settings.TargetMatrix,
		DeviceID: &configured,
	}

	session := NewSender(slog.New(slog.NewTextHandler(io.Discard, nil))).matrixSessionFor(target)
	if session.deviceID != configured {
		t.Errorf("device id = %q, want the configured %q", session.deviceID, configured)
	}
}

// TestDerivedDeviceIDsDoNotCollide keeps two targets on one account from
// sharing a device, which would have them silently revoking each other's token
// on every login.
func TestDerivedDeviceIDsDoNotCollide(t *testing.T) {
	one := settings.MatrixDeviceID("9f1c0c1e-2f2a-4f6b-9b7e-0a1d2c3b4a59")
	two := settings.MatrixDeviceID("0b2d1d2f-3a3b-4a7c-8c8f-1b2e3d4c5b6a")
	if one == two {
		t.Fatalf("two targets derived the same device id: %q", one)
	}
}

// TestProbeRelogsInWhenTheTokenWasRevoked is the aftermath of removing a device
// in Element: that revokes the device's access token, so the token Antenne
// still holds answers M_UNKNOWN_TOKEN.
//
// The credential is fine — the password is right there, and the send path
// already re-logs in on this errcode. A probe that instead reported KindDead
// suspended itself for an hour and raised an alert telling the operator to
// rotate a credential that still worked.
func TestProbeRelogsInWhenTheTokenWasRevoked(t *testing.T) {
	var logins int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_matrix/client/v3/login":
			logins++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh","user_id":"@antenne:example.org","device_id":"ANTENNE01"}`))
		case r.Header.Get("Authorization") != "Bearer fresh":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errcode":"M_UNKNOWN_TOKEN","error":"Invalid access token"}`))
		case r.URL.Path == "/_matrix/client/v3/account/whoami":
			_, _ = w.Write([]byte(`{"user_id":"@antenne:example.org","device_id":"ANTENNE01"}`))
		default:
			_, _ = w.Write([]byte(`{"joined_rooms":["!room:example.org"]}`))
		}
	}))
	defer server.Close()

	password := "hunter2"
	target := settings.DeliveryTarget{
		ID:         "9f1c0c1e-2f2a-4f6b-9b7e-0a1d2c3b4a59",
		Type:       settings.TargetMatrix,
		Enabled:    true,
		Homeserver: server.URL,
		UserID:     "@antenne:example.org",
		Password:   &password,
		RoomIDs:    []string{"!room:example.org"},
	}

	sender := NewSender(slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Seed the cache with the token the deleted device was using.
	sender.matrixSessionFor(target).accessToken = "revoked"

	observation := sender.Probe(context.Background(), target)
	if !observation.OK {
		t.Fatalf("a revoked token was not recovered by a re-login: kind=%q detail=%q",
			observation.Kind, observation.Detail)
	}
	if observation.Kind == health.KindDead {
		t.Error("a revoked token was reported as a dead credential, suspending probing for an hour")
	}
	if logins != 1 {
		t.Errorf("logins = %d, want exactly 1: the probe must not log in on a loop", logins)
	}
}
