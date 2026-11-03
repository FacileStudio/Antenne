// Package events is Antenne's activity log: every alert Antenne handled, what it was
// delivered to, and whether that worked.
package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
)

// Retention is how many entries the log keeps. It is a count rather than an age
// because the dashboard's value is "the recent past", and a quiet week should
// not empty it.
const Retention = 5000

type row struct {
	ID           string    `gorm:"primaryKey"`
	OccurredAt   time.Time `gorm:"column:occurred_at"`
	Source       string
	ProviderID   *string `gorm:"column:provider_id"`
	ProviderName string  `gorm:"column:provider_name"`
	ProviderType string  `gorm:"column:provider_type"`
	Event        string
	Message      *string
	Envelope     []byte `gorm:"type:jsonb"`
	Deliveries   []byte `gorm:"type:jsonb"`
	SearchText   string `gorm:"column:search_text"`
	ActorAvatar  string `gorm:"column:actor_avatar_url;->"`
}

func (row) TableName() string { return "events" }

// Entry is one logged event, in the shape the dashboard reads.
type Entry struct {
	ID           string             `json:"id"`
	Timestamp    string             `json:"timestamp"`
	Source       string             `json:"source"`
	ProviderID   *string            `json:"providerId"`
	ProviderName string             `json:"providerName"`
	ProviderType string             `json:"providerType"`
	Event        string             `json:"event"`
	Message      *string            `json:"message"`
	Envelope     *envelope.Envelope `json:"envelope,omitempty"`
	Deliveries   []delivery.Attempt `json:"deliveries"`
	// ActorAvatarURL is the picture of the user of this instance whose address
	// is the envelope actor's, and empty for anyone else.
	ActorAvatarURL string `json:"actorAvatarUrl,omitempty"`
}

// Stats summarize a slice of the log.
type Stats struct {
	TotalEvents          int     `json:"totalEvents"`
	LastEventAt          *string `json:"lastEventAt"`
	DeliverySuccessCount int     `json:"deliverySuccessCount"`
	DeliveryErrorCount   int     `json:"deliveryErrorCount"`
}

// New is what a caller supplies to record an event.
type New struct {
	Source       string
	ProviderID   *string
	ProviderName string
	ProviderType string
	Event        string
	Message      string
	Envelope     *envelope.Envelope
}

// Query filters the log.
type Query struct {
	ProviderID string
	TargetID   string
	Source     string
	Search     string
	Limit      int
	Offset     int
}

// Service owns the log and the live subscriber set.
type Service struct {
	db  *gorm.DB
	log *slog.Logger

	mu          sync.RWMutex
	subscribers map[chan Entry]struct{}
	sequence    uint64
}

// NewService builds the service.
func NewService(db *gorm.DB, log *slog.Logger) *Service {
	return &Service{db: db, log: log, subscribers: map[chan Entry]struct{}{}}
}

// newEntry is the shape a caller and every live subscriber see. It is separate
// from Push so the wire contract can be asserted without a database, which is
// the only way the deliveries invariant below is ever going to be checked.
func newEntry(input New, now time.Time, sequence uint64) Entry {
	entry := Entry{
		// The id has to sort with the timestamp and stay unique under a burst
		// from several providers in the same millisecond, hence the counter.
		ID:           fmt.Sprintf("%d-%06d", now.UnixMilli(), sequence),
		Timestamp:    now.Format(time.RFC3339Nano),
		Source:       input.Source,
		ProviderID:   input.ProviderID,
		ProviderName: input.ProviderName,
		ProviderType: input.ProviderType,
		Event:        input.Event,
		// An empty slice and not the nil zero value: deliveries carries no
		// omitempty, so a nil here reaches the SSE stream as "deliveries": null
		// and the dashboard reads .some() and .length off it. Nothing on the
		// read path can catch that — decode() never returns nil — so the page
		// loads fine and breaks on the first live event.
		Deliveries: []delivery.Attempt{},
	}
	if input.Message != "" {
		message := input.Message
		entry.Message = &message
	}
	entry.Envelope = input.Envelope
	return entry
}

// Push records an event and fans it out to every live subscriber.
func (s *Service) Push(input New) Entry {
	now := time.Now().UTC()

	s.mu.Lock()
	s.sequence++
	sequence := s.sequence
	s.mu.Unlock()

	entry := newEntry(input, now, sequence)

	stored := row{
		ID:           entry.ID,
		OccurredAt:   now,
		Source:       entry.Source,
		ProviderID:   entry.ProviderID,
		ProviderName: entry.ProviderName,
		ProviderType: entry.ProviderType,
		Event:        entry.Event,
		Message:      entry.Message,
		Deliveries:   []byte("[]"),
		SearchText:   buildSearchText(entry),
	}
	if entry.Envelope != nil {
		if encoded, err := json.Marshal(entry.Envelope); err == nil {
			stored.Envelope = encoded
		}
	}

	if err := s.db.Create(&stored).Error; err != nil {
		s.log.Error("failed to record event", slog.Any("error", err))
	}

	s.broadcast(entry)
	return entry
}

// SetDeliveries replaces the delivery attempts recorded for an event.
func (s *Service) SetDeliveries(eventID string, attempts []delivery.Attempt) {
	if attempts == nil {
		attempts = []delivery.Attempt{}
	}
	encoded, err := json.Marshal(attempts)
	if err != nil {
		return
	}

	updates := map[string]any{"deliveries": encoded}
	if entry, ok := s.byID(eventID); ok {
		entry.Deliveries = attempts
		updates["search_text"] = buildSearchText(entry)
	}

	if err := s.db.Model(&row{}).Where("id = ?", eventID).Updates(updates).Error; err != nil {
		s.log.Error("failed to record deliveries", slog.Any("error", err))
	}
}

// MarkRetry rewrites one delivery attempt after a retry settled.
func (s *Service) MarkRetry(eventID, targetID, status string) {
	entry, ok := s.byID(eventID)
	if !ok {
		return
	}
	for index := range entry.Deliveries {
		if entry.Deliveries[index].TargetID != targetID {
			continue
		}
		entry.Deliveries[index].Status = status
		if status == delivery.StatusSuccess {
			entry.Deliveries[index].Error = ""
		}
	}
	s.SetDeliveries(eventID, entry.Deliveries)
}

// ByID returns one entry.
func (s *Service) ByID(eventID string) (Entry, bool) { return s.byID(eventID) }

func (s *Service) byID(eventID string) (Entry, bool) {
	var stored row
	if err := s.db.Where("id = ?", eventID).First(&stored).Error; err != nil {
		return Entry{}, false
	}
	return decode(stored), true
}

// Subscribe returns a channel of new entries and the function that unsubscribes
// it. The channel is buffered and dropped-on-full, so a stalled SSE client
// slows nothing down.
func (s *Service) Subscribe() (<-chan Entry, func()) {
	stream := make(chan Entry, 32)

	s.mu.Lock()
	s.subscribers[stream] = struct{}{}
	s.mu.Unlock()

	return stream, func() {
		s.mu.Lock()
		if _, ok := s.subscribers[stream]; ok {
			delete(s.subscribers, stream)
			close(stream)
		}
		s.mu.Unlock()
	}
}

func (s *Service) broadcast(entry Entry) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for stream := range s.subscribers {
		select {
		case stream <- entry:
		default:
		}
	}
}

// Prune drops everything past the retention count.
func (s *Service) Prune() {
	err := s.db.Exec(`
		DELETE FROM events WHERE id IN (
			SELECT id FROM events ORDER BY occurred_at DESC, id DESC OFFSET ?
		)`, Retention).Error
	if err != nil {
		s.log.Error("failed to prune the event log", slog.Any("error", err))
	}
}
