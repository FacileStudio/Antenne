package delivery

import (
	"io"
	"log/slog"
	"testing"

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
