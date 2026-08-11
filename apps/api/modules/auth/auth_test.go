package auth

import (
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
