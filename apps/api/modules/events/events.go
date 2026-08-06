// Package events is Nook's activity log: every alert Nook handled, what it was
// delivered to, and whether that worked.
package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/delivery"
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

// Push records an event and fans it out to every live subscriber.
func (s *Service) Push(input New) Entry {
	now := time.Now().UTC()

	s.mu.Lock()
	s.sequence++
	sequence := s.sequence
	s.mu.Unlock()

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
		Deliveries:   []delivery.Attempt{},
	}
	if input.Message != "" {
		message := input.Message
		entry.Message = &message
	}
	entry.Envelope = input.Envelope

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

// List returns a filtered page of the log, newest first, with the total that
// matched the filter.
func (s *Service) List(query Query) ([]Entry, int64, error) {
	base := s.db.Model(&row{})

	if query.ProviderID != "" {
		base = base.Where("provider_id = ?", query.ProviderID)
	}
	if query.Source != "" {
		base = base.Where("source = ?", query.Source)
	}
	if query.TargetID != "" {
		// The attempts live in a jsonb array; @> asks Postgres whether one of
		// them names this target, which the GIN-less index still answers fast
		// at this table size.
		base = base.Where("deliveries @> ?", fmt.Sprintf(`[{"targetId":%q}]`, query.TargetID))
	}
	for _, term := range strings.Fields(strings.ToLower(query.Search)) {
		base = base.Where("search_text LIKE ?", "%"+term+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	var stored []row
	if err := base.Order("occurred_at desc, id desc").
		Limit(limit).Offset(query.Offset).Find(&stored).Error; err != nil {
		return nil, 0, err
	}

	entries := make([]Entry, 0, len(stored))
	for _, item := range stored {
		entries = append(entries, decode(item))
	}
	return entries, total, nil
}

// StatsFor summarizes the log under the same filter List takes.
func (s *Service) StatsFor(query Query) (Stats, error) {
	base := s.db.Model(&row{})
	if query.ProviderID != "" {
		base = base.Where("provider_id = ?", query.ProviderID)
	}
	if query.TargetID != "" {
		base = base.Where("deliveries @> ?", fmt.Sprintf(`[{"targetId":%q}]`, query.TargetID))
	}

	var aggregate struct {
		Total       int64
		LastEventAt *time.Time
		Successes   int64
		Errors      int64
	}

	err := base.Select(`
		count(*) AS total,
		max(occurred_at) AS last_event_at,
		coalesce(sum((SELECT count(*) FROM jsonb_array_elements(deliveries) AS d WHERE d->>'status' = 'success')), 0) AS successes,
		coalesce(sum((SELECT count(*) FROM jsonb_array_elements(deliveries) AS d WHERE d->>'status' = 'error')), 0) AS errors
	`).Scan(&aggregate).Error
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		TotalEvents:          int(aggregate.Total),
		DeliverySuccessCount: int(aggregate.Successes),
		DeliveryErrorCount:   int(aggregate.Errors),
	}
	if aggregate.LastEventAt != nil {
		formatted := aggregate.LastEventAt.UTC().Format(time.RFC3339Nano)
		stats.LastEventAt = &formatted
	}
	return stats, nil
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

func decode(stored row) Entry {
	entry := Entry{
		ID:           stored.ID,
		Timestamp:    stored.OccurredAt.UTC().Format(time.RFC3339Nano),
		Source:       stored.Source,
		ProviderID:   stored.ProviderID,
		ProviderName: stored.ProviderName,
		ProviderType: stored.ProviderType,
		Event:        stored.Event,
		Message:      stored.Message,
		Deliveries:   []delivery.Attempt{},
	}
	if len(stored.Deliveries) > 0 {
		_ = json.Unmarshal(stored.Deliveries, &entry.Deliveries)
		if entry.Deliveries == nil {
			entry.Deliveries = []delivery.Attempt{}
		}
	}
	if len(stored.Envelope) > 0 {
		decoded := envelope.Envelope{}
		if json.Unmarshal(stored.Envelope, &decoded) == nil {
			entry.Envelope = &decoded
		}
	}
	return entry
}

// buildSearchText flattens everything a human might search for into one
// lowercase column, so the activity search is a single LIKE instead of a
// jsonb traversal per term.
func buildSearchText(entry Entry) string {
	parts := []string{entry.Event, entry.ProviderName, entry.ProviderType, entry.Source}

	if entry.Envelope != nil {
		parts = append(parts, entry.Envelope.Event.Type)
		parts = appendPointer(parts, entry.Envelope.Event.Title)
		parts = appendPointer(parts, entry.Envelope.Event.MessageText)
		parts = appendPointer(parts, entry.Envelope.Event.URL)
		if entry.Envelope.Actor != nil {
			parts = appendPointer(parts, entry.Envelope.Actor.Name)
			parts = appendPointer(parts, entry.Envelope.Actor.Email)
		}
		parts = appendPointer(parts, entry.Envelope.Subject.Name)
		parts = appendPointer(parts, entry.Envelope.Subject.URL)
		for _, value := range entry.Envelope.Payload.Normalized {
			if text, ok := value.(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
	} else if entry.Message != nil {
		parts = append(parts, envelope.HTMLToPlainText(*entry.Message))
	}

	for _, attempt := range entry.Deliveries {
		parts = append(parts, attempt.TargetName, attempt.TargetType, attempt.Status)
		if attempt.Error != "" {
			parts = append(parts, attempt.Error)
		}
	}

	return strings.ToLower(strings.Join(parts, " "))
}

func appendPointer(parts []string, value *string) []string {
	if value != nil && *value != "" {
		return append(parts, *value)
	}
	return parts
}
