// Package ratelimit is a fixed-window in-process counter, used for login
// attempts. It is per-instance, which is enough for a single-container service.
package ratelimit

import (
	"sync"
	"time"
)

type attempt struct {
	count        int
	firstFailure time.Time
	blockedUntil time.Time
}

// Limiter blocks a key for Block once it records Max failures within Window.
type Limiter struct {
	Max    int
	Window time.Duration
	Block  time.Duration

	mu       sync.Mutex
	attempts map[string]*attempt
}

// New returns a limiter allowing max failures per window, blocking for block.
func New(max int, window, block time.Duration) *Limiter {
	return &Limiter{Max: max, Window: window, Block: block, attempts: map[string]*attempt{}}
}

// Blocked reports whether key is currently blocked, expiring stale windows.
func (l *Limiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, ok := l.attempts[key]
	if !ok {
		return false
	}

	now := time.Now()
	if current.blockedUntil.After(now) {
		return true
	}
	if current.firstFailure.Add(l.Window).Before(now) {
		delete(l.attempts, key)
	}
	return false
}

// Fail records one failure for key.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	current, ok := l.attempts[key]
	if !ok || current.firstFailure.Add(l.Window).Before(now) {
		l.attempts[key] = &attempt{count: 1, firstFailure: now}
		return
	}

	current.count++
	if current.count >= l.Max {
		current.blockedUntil = now.Add(l.Block)
	}
}

// Reset clears every failure recorded for key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
