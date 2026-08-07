// Package envelope builds the structured event Antenne signs and ships to HTTP and
// Perception delivery targets.
//
// This is Antenne's own schema, not the suite `enveloppe` contract. In particular
// Actor.Email is whatever a third-party payload happened to carry — it is never
// a Facile SSO identity, and nothing downstream should read it as `actor_email`.
package envelope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Source names where an event entered Antenne.
type Source string

// Event sources.
const (
	SourceWebhook Source = "webhook"
	SourceWebsite Source = "website"
	SourceIMAP    Source = "imap"
	SourceRSS     Source = "rss"
	SourceTest    Source = "test"
	SourceSystem  Source = "system"
)

// SubjectType names what an event is about.
type SubjectType string

// Subject types.
const (
	SubjectTask       SubjectType = "task"
	SubjectContract   SubjectType = "contract"
	SubjectClient     SubjectType = "client"
	SubjectDeployment SubjectType = "deployment"
	SubjectRepository SubjectType = "repository"
	SubjectEmail      SubjectType = "email"
	SubjectService    SubjectType = "service"
	SubjectSchedule   SubjectType = "schedule"
	SubjectSystem     SubjectType = "system"
	SubjectUnknown    SubjectType = "unknown"
)

// Actor is who caused the event, when the source says.
type Actor struct {
	ID    *string `json:"id"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// Subject is what the event is about.
type Subject struct {
	Type SubjectType `json:"type"`
	ID   *string     `json:"id"`
	Name *string     `json:"name"`
	URL  *string     `json:"url"`
}

// Provider identifies the Antenne provider the event came through.
type Provider struct {
	ID   *string  `json:"id"`
	Type string   `json:"type"`
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// Event is the human-facing description of what happened.
type Event struct {
	Type        string  `json:"type"`
	Title       *string `json:"title"`
	MessageHTML *string `json:"messageHtml"`
	MessageText *string `json:"messageText"`
	URL         *string `json:"url"`
}

// Payload carries the normalized fields and, when allowed, the raw source body.
type Payload struct {
	Normalized map[string]any `json:"normalized"`
	Raw        any            `json:"raw"`
}

// Privacy flags what the payload may be used for downstream.
type Privacy struct {
	ContainsPII       bool `json:"containsPii"`
	RawPayloadAllowed bool `json:"rawPayloadAllowed"`
}

// Envelope is a Antenne event, schema version 1.
type Envelope struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	OccurredAt    string   `json:"occurredAt"`
	EmittedAt     string   `json:"emittedAt"`
	Source        Source   `json:"source"`
	Provider      Provider `json:"provider"`
	Event         Event    `json:"event"`
	Actor         *Actor   `json:"actor"`
	Subject       Subject  `json:"subject"`
	Payload       Payload  `json:"payload"`
	Privacy       Privacy  `json:"privacy"`
}

// Input is what a caller supplies to Build. Every optional field falls back to
// something sensible, so a webhook provider only fills in what it knows.
type Input struct {
	ID                string
	Source            Source
	ProviderID        *string
	ProviderType      string
	ProviderName      string
	ProviderTags      []string
	EventType         string
	EventTitle        string
	MessageHTML       string
	MessageText       string
	EventURL          string
	Actor             *Actor
	Subject           *Subject
	NormalizedPayload map[string]any
	RawPayload        any
	OccurredAt        string
	EmittedAt         string
	ContainsPII       bool
	RawPayloadAllowed bool
}

// Build assembles an envelope, deriving the id, the plain-text message and the
// timestamps that were not supplied.
func Build(input Input) Envelope {
	emittedAt := input.EmittedAt
	if emittedAt == "" {
		emittedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	occurredAt := input.OccurredAt
	if occurredAt == "" {
		occurredAt = emittedAt
	}

	messageText := input.MessageText
	if messageText == "" && input.MessageHTML != "" {
		messageText = HTMLToPlainText(input.MessageHTML)
	}

	normalized := input.NormalizedPayload
	if normalized == nil {
		normalized = map[string]any{}
	}

	title := input.EventTitle
	if title == "" {
		title = input.EventType
	}

	subject := Subject{Type: SubjectUnknown}
	if input.Subject != nil {
		subject = *input.Subject
		if subject.Type == "" {
			subject.Type = SubjectUnknown
		}
		subject.ID = compact(subject.ID)
		subject.Name = compact(subject.Name)
		subject.URL = compact(subject.URL)
	}

	eventURL := nilIfEmpty(input.EventURL)
	if eventURL == nil {
		eventURL = subject.URL
	}

	var raw any
	if input.RawPayloadAllowed {
		raw = input.RawPayload
	}

	tags := input.ProviderTags
	if tags == nil {
		tags = []string{}
	}

	built := Envelope{
		SchemaVersion: 1,
		OccurredAt:    occurredAt,
		EmittedAt:     emittedAt,
		Source:        input.Source,
		Provider: Provider{
			ID:   input.ProviderID,
			Type: input.ProviderType,
			Name: input.ProviderName,
			Tags: tags,
		},
		Event: Event{
			Type:        input.EventType,
			Title:       nilIfEmpty(title),
			MessageHTML: nilIfEmpty(input.MessageHTML),
			MessageText: nilIfEmpty(messageText),
			URL:         eventURL,
		},
		Actor:   normalizeActor(input.Actor),
		Subject: subject,
		Payload: Payload{Normalized: normalized, Raw: raw},
		Privacy: Privacy{ContainsPII: input.ContainsPII, RawPayloadAllowed: input.RawPayloadAllowed},
	}

	built.ID = input.ID
	if built.ID == "" {
		built.ID = deriveID(input, occurredAt)
	}

	return built
}

// RedactRaw returns a copy with the raw payload stripped, for a target that is
// not allowed to see it.
func RedactRaw(source Envelope) Envelope {
	redacted := source
	redacted.Payload.Raw = nil
	redacted.Privacy.RawPayloadAllowed = false
	return redacted
}

// deriveID hashes the identifying fields so the same source event replayed
// twice carries the same id, which is what makes a downstream consumer able to
// deduplicate.
func deriveID(input Input, occurredAt string) string {
	hash := sha256.New()
	if input.ProviderID != nil {
		hash.Write([]byte(*input.ProviderID))
	}
	hash.Write([]byte{0})
	hash.Write([]byte(input.ProviderType))
	hash.Write([]byte{0})
	hash.Write([]byte(input.EventType))
	hash.Write([]byte{0})
	hash.Write([]byte(occurredAt))
	hash.Write([]byte{0})

	switch {
	case len(input.NormalizedPayload) > 0:
		hash.Write([]byte(stableJSON(input.NormalizedPayload)))
	case input.RawPayload != nil:
		hash.Write([]byte(stableJSON(input.RawPayload)))
	case input.MessageText != "":
		hash.Write([]byte(stableJSON(input.MessageText)))
	default:
		hash.Write([]byte(stableJSON(input.MessageHTML)))
	}

	return "antenne_" + hex.EncodeToString(hash.Sum(nil))[:32]
}

// stableJSON serializes with object keys sorted, so an id derived from a map
// does not change between two runs over the same data.
func stableJSON(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			encodedKey, _ := json.Marshal(key)
			parts = append(parts, fmt.Sprintf("%s:%s", encodedKey, stableJSON(typed[key])))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, stableJSON(item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}

	// Anything else goes through encoding/json, which is already stable for
	// scalars; a struct is re-read as a map first so its keys sort too.
	encoded, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	var reparsed any
	if err := json.Unmarshal(encoded, &reparsed); err != nil {
		return string(encoded)
	}
	if _, isScalar := reparsed.(map[string]any); !isScalar {
		if _, isList := reparsed.([]any); !isList {
			return string(encoded)
		}
	}
	return stableJSON(reparsed)
}

func normalizeActor(actor *Actor) *Actor {
	if actor == nil {
		return nil
	}
	normalized := Actor{
		ID:    compact(actor.ID),
		Name:  compact(actor.Name),
		Email: compact(actor.Email),
	}
	if normalized.ID == nil && normalized.Name == nil && normalized.Email == nil {
		return nil
	}
	return &normalized
}

func compact(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func nilIfEmpty(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// Str is a convenience for the pointer fields, which are pointers because the
// wire format distinguishes "absent" from "empty".
func Str(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
