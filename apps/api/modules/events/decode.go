package events

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
)

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
	}
	entry.Deliveries = decodeDeliveries(stored.Deliveries)
	if len(stored.Envelope) > 0 {
		decoded := envelope.Envelope{}
		if json.Unmarshal(stored.Envelope, &decoded) == nil {
			entry.Envelope = &decoded
		}
	}
	return entry
}

// decodeDeliveries reads the stored attempt list, keeping whatever parsed and
// never handing back nil.
//
// A blob that no longer decodes costs the event its delivery record and nothing
// else. The attempts are a log of what already happened rather than state
// anything acts on, so refusing the row would hide the event itself from the
// activity log that exists to explain it.
func decodeDeliveries(blob []byte) []delivery.Attempt {
	attempts := []delivery.Attempt{}
	if len(blob) > 0 {
		json.Unmarshal(blob, &attempts)
	}
	if attempts == nil {
		return []delivery.Attempt{}
	}
	return attempts
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
