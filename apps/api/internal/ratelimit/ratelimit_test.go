package ratelimit

import (
	"testing"
	"time"
)

// TestFailuresBlockAndSuccessResets pins the login throttle's rule: the fifth
// failure within the window blocks the key, another key keeps its budget, and
// a success clears the count.
func TestFailuresBlockAndSuccessResets(t *testing.T) {
	limiter := New(5, 10*time.Minute, 10*time.Minute)

	for range 4 {
		limiter.Fail("1.2.3.4")
	}
	if limiter.Blocked("1.2.3.4") {
		t.Fatal("four failures blocked the key")
	}
	limiter.Reset("1.2.3.4")
	for range 4 {
		limiter.Fail("1.2.3.4")
	}
	if limiter.Blocked("1.2.3.4") {
		t.Fatal("a reset did not clear the earlier failures")
	}

	limiter.Fail("1.2.3.4")
	if !limiter.Blocked("1.2.3.4") {
		t.Fatal("the fifth failure did not block the key")
	}
	if limiter.Blocked("5.6.7.8") {
		t.Fatal("a blocked key leaked into another caller's budget")
	}
}
