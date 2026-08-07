package delivery

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

// retryDelays are the waits before attempts 1, 2 and 3. The spread is
// deliberate: a target that failed on a blip recovers in seconds, one that is
// down for a deploy gets five minutes before the last try.
var retryDelays = []time.Duration{5 * time.Second, 30 * time.Second, 5 * time.Minute}

// MaxRetryAttempts is how many times a failed delivery is re-sent before the
// queue gives up. It is exported so an operator-facing message can say it.
const MaxRetryAttempts = 3

type retryEntry struct {
	eventID string
	target  settings.DeliveryTarget
	message Message
	attempt int
	dueAt   time.Time
}

// RetryOutcome reports a retry's final result to whoever owns the event log.
type RetryOutcome func(eventID, targetID, status string)

// Exhausted reports a delivery that has run out of attempts.
//
// It is separate from RetryOutcome because the two have different audiences: the
// event log wants every outcome, success included, while an operator only wants
// the one that is still broken after the queue has done all it can.
type Exhausted func(target settings.DeliveryTarget, message Message)

// RetryQueue re-sends failed deliveries in the background.
//
// It is in-memory on purpose: a retry is only useful within minutes of the
// original attempt, and the event log already records the permanent failure for
// anything that outlives the process.
type RetryQueue struct {
	sender  *Sender
	log     *slog.Logger
	outcome RetryOutcome

	mu        sync.Mutex
	entries   []retryEntry
	exhausted Exhausted
}

// NewRetryQueue builds the queue. outcome may be nil.
func NewRetryQueue(sender *Sender, log *slog.Logger, outcome RetryOutcome) *RetryQueue {
	return &RetryQueue{sender: sender, log: log, outcome: outcome}
}

// OnExhausted installs the handler for a delivery that has run out of attempts.
func (q *RetryQueue) OnExhausted(handler Exhausted) {
	q.mu.Lock()
	q.exhausted = handler
	q.mu.Unlock()
}

// Enqueue schedules the first retry for a failed delivery.
func (q *RetryQueue) Enqueue(eventID string, target settings.DeliveryTarget, message Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, retryEntry{
		eventID: eventID,
		target:  target,
		message: message,
		attempt: 1,
		dueAt:   time.Now().Add(retryDelays[0]),
	})
}

// Size reports how many retries are pending.
func (q *RetryQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.entries)
}

// Start runs the queue until ctx is cancelled.
func (q *RetryQueue) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				q.process(ctx)
			}
		}
	}()
}

func (q *RetryQueue) process(ctx context.Context) {
	q.mu.Lock()
	now := time.Now()
	ready := make([]retryEntry, 0)
	pending := q.entries[:0]
	for _, entry := range q.entries {
		if entry.dueAt.After(now) {
			pending = append(pending, entry)
			continue
		}
		ready = append(ready, entry)
	}
	q.entries = append([]retryEntry{}, pending...)
	q.mu.Unlock()

	for _, entry := range ready {
		err := q.sender.SendTo(ctx, entry.target, entry.message)
		if err == nil {
			q.log.Info("retry succeeded",
				slog.String("target", entry.target.Name),
				slog.Int("attempt", entry.attempt))
			q.report(entry.eventID, entry.target.ID, StatusSuccess)
			continue
		}

		if entry.attempt >= MaxRetryAttempts {
			q.log.Error("retry exhausted",
				slog.String("target", entry.target.Name),
				slog.Int("attempts", MaxRetryAttempts),
				slog.Any("error", err))
			q.report(entry.eventID, entry.target.ID, StatusError)
			q.reportExhausted(entry.target, entry.message)
			continue
		}

		delay := retryDelays[min(entry.attempt, len(retryDelays)-1)]
		q.log.Warn("retry failed, scheduling another",
			slog.String("target", entry.target.Name),
			slog.Int("attempt", entry.attempt),
			slog.Duration("next_in", delay))

		next := entry
		next.attempt++
		next.dueAt = time.Now().Add(delay)

		q.mu.Lock()
		q.entries = append(q.entries, next)
		q.mu.Unlock()
	}
}

func (q *RetryQueue) report(eventID, targetID, status string) {
	if q.outcome != nil {
		q.outcome(eventID, targetID, status)
	}
}

func (q *RetryQueue) reportExhausted(target settings.DeliveryTarget, message Message) {
	q.mu.Lock()
	handler := q.exhausted
	q.mu.Unlock()
	if handler != nil {
		handler(target, message)
	}
}
