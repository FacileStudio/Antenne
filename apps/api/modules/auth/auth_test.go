package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
)

// TestABareUsernameBecomesAnAddress covers the one translation porte forced.
// Antenne is configured with a login label and porte/local resolves an account
// by address, so the two have to agree on the derivation — the seed writes the
// identity under this address and the login form looks it up under it.
func TestABareUsernameBecomesAnAddress(t *testing.T) {
	for _, testCase := range []struct {
		username string
		want     string
	}{
		{"admin", "admin@antenne.local"},
		{"Admin", "admin@antenne.local"},
		{"ops@facile.studio", "ops@facile.studio"},
		{"  ops@facile.studio  ", "ops@facile.studio"},
	} {
		if got := bootstrapEmail(testCase.username); got != testCase.want {
			t.Errorf("bootstrapEmail(%q) = %q, want %q", testCase.username, got, testCase.want)
		}
	}
}

// TestTheLoginThrottleCountsRefusalsAndForgivesSuccess pins the behaviour that
// moved when porte took the login handler.
//
// The throttle reads the outcome off the status code now, because the handler
// belongs to the kit. A test matters here precisely because that indirection is
// easy to get backwards: a bucket that counted successes would lock out the
// only account after five ordinary logins.
func TestTheLoginThrottleCountsRefusalsAndForgivesSuccess(t *testing.T) {
	var status int
	throttled := ThrottleLogins()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))

	attempt := func() int {
		request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		request.Header.Set("X-Forwarded-For", "203.0.113.7")
		recorder := httptest.NewRecorder()
		throttled.ServeHTTP(recorder, request)
		return recorder.Code
	}

	status = http.StatusUnauthorized
	for i := range 5 {
		if got := attempt(); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d, want 401", i+1, got)
		}
	}
	if got := attempt(); got != http.StatusTooManyRequests {
		t.Fatalf("the sixth refusal got %d, want 429", got)
	}

	status = http.StatusOK
	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	request.Header.Set("X-Forwarded-For", "198.51.100.4")
	recorder := httptest.NewRecorder()
	throttled.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("a different address got %d, want 200 — the bucket is not per-address", recorder.Code)
	}
}

// TestAnAnonymousContextIsNotAuthenticated guards the zero value, which is what
// a handler sees when the middleware chain is misassembled. It must never read
// as a caller.
func TestAnAnonymousContextIsNotAuthenticated(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	if identity := authcontext.From(request); identity.IsAuthenticated() || identity.IsAdmin() {
		t.Fatalf("the zero identity reads as authenticated=%v admin=%v", identity.IsAuthenticated(), identity.IsAdmin())
	}
}

func TestHandleSessionResolvesIdentity(t *testing.T) {
	service := &Service{cfg: Config{AdminUsername: "admin", AdminPassword: "secret"}}

	for _, tc := range []struct {
		name         string
		identity     authcontext.Identity
		wantUsername string
		wantEmail    string
		wantName     string
		wantRole     string
	}{
		{
			name: "bootstrap admin",
			identity: authcontext.Identity{
				UserID: 1,
				Email:  "admin@antenne.local",
				Role:   authcontext.RoleAdmin,
			},
			wantUsername: "admin",
			wantEmail:    "admin@antenne.local",
			wantRole:     "ADMIN",
		},
		{
			name: "oidc user with name",
			identity: authcontext.Identity{
				UserID: 2,
				Email:  "saravenpi@facile.studio",
				Name:   "saravenpi",
				Role:   authcontext.RoleAdmin,
			},
			wantUsername: "saravenpi",
			wantEmail:    "saravenpi@facile.studio",
			wantName:     "saravenpi",
			wantRole:     "ADMIN",
		},
		{
			name: "oidc user without name falls back to local email part",
			identity: authcontext.Identity{
				UserID: 3,
				Email:  "jane.doe@facile.studio",
				Role:   authcontext.RoleAdmin,
			},
			wantUsername: "jane.doe",
			wantEmail:    "jane.doe@facile.studio",
			wantRole:     "ADMIN",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
			req = req.WithContext(authcontext.With(req.Context(), tc.identity))
			rec := httptest.NewRecorder()

			service.handleSession(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("got status %d, want 200", rec.Code)
			}

			var resp SessionResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if !resp.Authenticated {
				t.Errorf("got authenticated = false, want true")
			}
			if resp.Username != tc.wantUsername {
				t.Errorf("got username %q, want %q", resp.Username, tc.wantUsername)
			}
			if resp.Email != tc.wantEmail {
				t.Errorf("got email %q, want %q", resp.Email, tc.wantEmail)
			}
			if resp.Name != tc.wantName {
				t.Errorf("got name %q, want %q", resp.Name, tc.wantName)
			}
			if resp.Role != tc.wantRole {
				t.Errorf("got role %q, want %q", resp.Role, tc.wantRole)
			}
		})
	}
}
