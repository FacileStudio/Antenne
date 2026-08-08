package ratelimit

import (
	"testing"
	"time"
)

func TestRetryAfterReportsTheRemainingBlock(t *testing.T) {
	limiter := New(2, time.Minute, 30*time.Second)

	if wait := limiter.RetryAfter("1.2.3.4"); wait != 0 {
		t.Fatalf("an untouched key reported a wait of %s", wait)
	}

	limiter.Fail("1.2.3.4")
	if wait := limiter.RetryAfter("1.2.3.4"); wait != 0 {
		t.Fatalf("a key still inside its budget reported a wait of %s", wait)
	}

	limiter.Fail("1.2.3.4")
	wait := limiter.RetryAfter("1.2.3.4")
	if wait <= 0 || wait > 30*time.Second {
		t.Fatalf("a blocked key reported a wait of %s, want (0s, 30s]", wait)
	}

	// A blocked key must not leak into another caller's budget: the header the
	// handler sends is per-client, and a shared one would tell an innocent
	// browser to back off.
	if wait := limiter.RetryAfter("5.6.7.8"); wait != 0 {
		t.Fatalf("an unrelated key reported a wait of %s", wait)
	}
}
