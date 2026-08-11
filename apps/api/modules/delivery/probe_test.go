package delivery

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func probeSender() *Sender {
	return NewSender(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func matrixTarget(homeserver string, rooms ...string) settings.DeliveryTarget {
	return settings.DeliveryTarget{
		ID: "m", Type: settings.TargetMatrix, Name: "matrix", Enabled: true,
		AccessToken: "token",
		MatrixSettings: settings.MatrixSettings{
			Homeserver: homeserver, UserID: "@antenne:example.org", RoomIDs: rooms,
		},
	}
}

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

// Discord bans an IP for about an hour after ~10k invalid requests in 10
// minutes, so a deleted webhook has to be terminal, not retried on a schedule.
func TestDiscordUnknownWebhookIsTerminallyDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("the Discord probe used %s; it must be read-only", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"message":"Unknown Webhook","code":10015}`)
	}))
	defer server.Close()

	target := settings.DeliveryTarget{
		ID: "d", Type: settings.TargetDiscord, Name: "discord", Enabled: true,
		DiscordSettings: settings.DiscordSettings{WebhookURL: server.URL + "/api/webhooks/123/abc?wait=true"},
	}

	observation := probeSender().Probe(context.Background(), target)
	if observation.Kind != health.KindDead {
		t.Errorf("Discord 10015 classified as %q, want %q", observation.Kind, health.KindDead)
	}
	if !observation.Kind.Terminal() {
		t.Error("a deleted Discord webhook stayed probeable, which is how an IP gets banned")
	}
}

func TestDiscordRateLimitCarriesRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"message":"You are being rate limited.","retry_after":5.0}`)
	}))
	defer server.Close()

	target := settings.DeliveryTarget{
		ID: "d", Type: settings.TargetDiscord, Name: "discord", Enabled: true,
		DiscordSettings: settings.DiscordSettings{WebhookURL: server.URL + "/api/webhooks/123/abc"},
	}

	observation := probeSender().Probe(context.Background(), target)
	if observation.Kind != health.KindRateLimited {
		t.Errorf("429 classified as %q, want %q", observation.Kind, health.KindRateLimited)
	}
	if observation.RetryAfter != 5*time.Second {
		t.Errorf("RetryAfter %v, want 5s: the far side's wait is binding", observation.RetryAfter)
	}
	if observation.Kind.Terminal() {
		t.Error("being throttled suspended probing permanently")
	}
}

// /v1/account is allowed anonymously, so a token that never reached the server
// comes back 200 as the anonymous user. Trusting the status code here is a false
// green, which is the worst answer a health check can give.
func TestNtfyAnonymousAnswerIsNotHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			io.WriteString(w, `{"healthy":true}`)
		case "/v1/account":
			io.WriteString(w, `{"username":"*","role":"anonymous"}`)
		default:
			t.Errorf("unexpected probe request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	target := settings.DeliveryTarget{
		ID: "n", Type: settings.TargetNtfy, Name: "ntfy", Enabled: true,
		AccessToken:  "tk_secret",
		NtfySettings: settings.NtfySettings{ServerURL: server.URL, Topic: "alerts"},
	}

	observation := probeSender().Probe(context.Background(), target)
	if observation.OK {
		t.Fatal("ntfy answering as the anonymous user was reported healthy")
	}
	if observation.Kind != health.KindDenied {
		t.Errorf("anonymous answer classified as %q, want %q", observation.Kind, health.KindDenied)
	}
}

func TestNtfyAuthenticatedAnswerIsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			io.WriteString(w, `{"healthy":true}`)
		case "/v1/account":
			if r.Header.Get("Authorization") != "Bearer tk_secret" {
				t.Error("the ntfy probe did not send the access token")
			}
			io.WriteString(w, `{"username":"antenne","role":"user"}`)
		}
	}))
	defer server.Close()

	target := settings.DeliveryTarget{
		ID: "n", Type: settings.TargetNtfy, Name: "ntfy", Enabled: true,
		AccessToken:  "tk_secret",
		NtfySettings: settings.NtfySettings{ServerURL: server.URL, Topic: "alerts"},
	}

	if observation := probeSender().Probe(context.Background(), target); !observation.OK {
		t.Errorf("a working ntfy token was reported unhealthy: %q", observation.Detail)
	}
}

// Perception's health endpoint sits beside its ingest URL, so an instance served
// under a path prefix still resolves.
func TestPerceptionProbeUsesTheSiblingHealthEndpoint(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	secret := "s3cret"
	target := settings.DeliveryTarget{
		ID: "p", Type: settings.TargetPerception, Name: "perception", Enabled: true,
		HTTPSettings: settings.HTTPSettings{
			URL: server.URL + "/api/ingest", Workspace: "studio", SecretValue: &secret,
		},
	}

	observation := probeSender().Probe(context.Background(), target)
	if !observation.OK {
		t.Errorf("a healthy Perception was reported unhealthy: %q", observation.Detail)
	}
	if seen != "/api/health" {
		t.Errorf("probed %q, want /api/health", seen)
	}
}

// There is no safe read-only probe for an arbitrary third-party URL: a GET may
// be unimplemented, may mean something else entirely, or may be answered by a
// proxy that never reaches the handler. unknown is the honest answer.
func TestGenericWebhookHasNoProbe(t *testing.T) {
	target := settings.DeliveryTarget{
		ID: "w", Type: settings.TargetWebhook, Name: "webhook", Enabled: true,
		HTTPSettings: settings.HTTPSettings{URL: "https://example.invalid/hook"},
	}

	observation := probeSender().Probe(context.Background(), target)
	if observation.OK {
		t.Fatal("a generic webhook was painted green")
	}
	if observation.Kind != health.KindUnsupported {
		t.Errorf("generic webhook classified as %q, want %q", observation.Kind, health.KindUnsupported)
	}
}

func TestSMTPReplyClassification(t *testing.T) {
	cases := []struct {
		code    int
		message string
		ok      bool
		kind    health.Kind
	}{
		{235, "2.7.0 Authentication successful", true, health.KindNone},
		{535, "5.7.8 Username and Password not accepted", false, health.KindDead},
		// The distinction that stops the monitor locking the account it monitors.
		{454, "4.7.0 Too many login attempts, please try again later", false, health.KindRateLimited},
		{432, "4.7.12 A password transition is needed", false, health.KindRefreshable},
		{421, "4.4.5 Service not available", false, health.KindUnreachable},
		{530, "5.7.0 Authentication required", false, health.KindDenied},
		{538, "5.7.11 Encryption required", false, health.KindDenied},
		{550, "5.7.1 Relay access denied", false, health.KindDenied},
	}

	for _, testCase := range cases {
		observation := classifySMTPReply(testCase.code, testCase.message)
		if observation.OK != testCase.ok || observation.Kind != testCase.kind {
			t.Errorf("SMTP %d gave OK=%v kind=%q, want OK=%v kind=%q",
				testCase.code, observation.OK, observation.Kind, testCase.ok, testCase.kind)
		}
		if observation.Detail == "" {
			t.Errorf("SMTP %d dropped the server's own words", testCase.code)
		}
	}

	if classifySMTPReply(535, "nope").Kind.Terminal() != true {
		t.Error("SMTP 535 is not terminal, so the prober will keep feeding fail2ban")
	}
	if classifySMTPReply(454, "slow down").Kind.Terminal() {
		t.Error("SMTP 454 is terminal, so a throttle permanently kills the target")
	}
}

func TestAnUnconfiguredTargetIsNotProbed(t *testing.T) {
	target := settings.DeliveryTarget{ID: "x", Type: settings.TargetDiscord, Name: "half-built", Enabled: true}

	observation := probeSender().Probe(context.Background(), target)
	if observation.OK || observation.Kind != health.KindUnsupported {
		t.Errorf("an unconfigured target gave OK=%v kind=%q", observation.OK, observation.Kind)
	}
}

func TestRetryAfterAcceptsBothForms(t *testing.T) {
	if got := retryAfterFrom(http.Header{"Retry-After": []string{"7"}}); got != 7*time.Second {
		t.Errorf("seconds form gave %v, want 7s", got)
	}
	future := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
	if got := retryAfterFrom(http.Header{"Retry-After": []string{future}}); got < 30*time.Second {
		t.Errorf("HTTP-date form gave %v, want roughly 90s", got)
	}
	if got := retryAfterFrom(http.Header{}); got != 0 {
		t.Errorf("a missing header gave %v, want 0", got)
	}
}
