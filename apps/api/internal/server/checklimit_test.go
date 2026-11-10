package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTheCheckLimitIsTenPerMinute replays a held-down button: five checks
// pass at once, then one every twelve seconds, and the sixty seconds from the
// first press admit no more than ten.
func TestTheCheckLimitIsTenPerMinute(t *testing.T) {
	limiter := newCheckLimiter()
	start := time.Unix(0, 0)
	admitted := 0
	for second := range 61 {
		for range 3 {
			if ok, _ := limiter.AllowAt("192.0.2.1", start.Add(time.Duration(second)*time.Second)); ok {
				admitted++
			}
		}
	}
	if admitted != 10 {
		t.Fatalf("a minute of presses admitted %d checks, want 10", admitted)
	}
}

// TestARefusedCheckSaysHowLongToWait exhausts one address's budget and proves
// the refusal is the 429 envelope with a Retry-After in seconds, keyed on
// RemoteAddr: a forged X-Forwarded-For does not buy a fresh budget.
func TestARefusedCheckSaysHowLongToWait(t *testing.T) {
	api := &API{checks: newCheckLimiter()}
	for range 5 {
		api.checks.AllowAt("192.0.2.1", time.Now())
	}

	request := httptest.NewRequest(http.MethodPost, "/api/delivery-health/t1/check", nil)
	request.RemoteAddr = "192.0.2.1:4242"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")
	recorder := httptest.NewRecorder()
	api.handleCheckTargetHealth(recorder, request)

	want := `{"error":{"code":"rate_limited","message":"too many checks from this client; wait before checking again"}}` + "\n"
	if recorder.Code != http.StatusTooManyRequests || recorder.Body.String() != want {
		t.Fatalf("refusal = %d %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Retry-After"); got != "12" {
		t.Fatalf("Retry-After = %q, want 12", got)
	}
}
