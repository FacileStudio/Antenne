package webhooks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type sondeEnvelope struct {
	Version        int             `json:"version"`
	App            string          `json:"app"`
	Object         string          `json:"object"`
	Action         string          `json:"action"`
	Timestamp      string          `json:"timestamp"`
	IdempotencyKey string          `json:"idempotency_key"`
	Payload        json.RawMessage `json:"payload"`
}

type sondePayload struct {
	sondeEnvelope

	Event      string `json:"event"`
	EventType  string `json:"event_type"`
	MonitorID  any    `json:"monitor_id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	LatencyMS  int64  `json:"latency_ms"`
	URL        string `json:"url"`
	Error      string `json:"error"`
	OccurredAt string `json:"occurred_at"`
	ActorEmail string `json:"actor_email"`

	Monitor *struct {
		ID        any     `json:"id"`
		MonitorID any     `json:"monitor_id"`
		Name      string  `json:"name"`
		Slug      string  `json:"slug"`
		Status    string  `json:"status"`
		URL       string  `json:"url"`
		Target    string  `json:"target"`
		LatencyMS int64   `json:"latency_ms"`
		Error     *string `json:"error"`
	} `json:"monitor"`

	Incident *struct {
		ID         any     `json:"id"`
		StartedAt  string  `json:"started_at"`
		ResolvedAt *string `json:"resolved_at"`
		Cause      string  `json:"cause"`
	} `json:"incident"`
}

type sondeInnerPayload struct {
	MonitorID  any     `json:"monitor_id"`
	FacileID   string  `json:"facile_id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	LatencyMS  int64   `json:"latency_ms"`
	Error      *string `json:"error"`
	ActorEmail string  `json:"actor_email"`
}

var sonde = Definition{
	Type:         settings.ProviderSonde,
	Label:        "Sonde",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload sondePayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		name := payload.Name
		slug := payload.Slug
		status := strings.ToLower(payload.Status)
		urlStr := payload.URL
		errStr := payload.Error
		latencyMS := payload.LatencyMS
		actorEmail := payload.ActorEmail
		occurredAt := firstNonEmpty(payload.OccurredAt, payload.Timestamp)
		monitorID := stringify(payload.MonitorID)

		if len(payload.Payload) > 0 {
			var inner sondeInnerPayload
			if err := json.Unmarshal(payload.Payload, &inner); err == nil {
				if inner.Name != "" {
					name = inner.Name
				}
				if inner.Slug != "" {
					slug = inner.Slug
				}
				if inner.Status != "" {
					status = strings.ToLower(inner.Status)
				}
				if inner.LatencyMS > 0 {
					latencyMS = inner.LatencyMS
				}
				if inner.Error != nil && *inner.Error != "" {
					errStr = *inner.Error
				}
				if inner.ActorEmail != "" {
					actorEmail = inner.ActorEmail
				}
				if inner.MonitorID != nil {
					monitorID = stringify(inner.MonitorID)
				}
			}
		}

		if payload.Monitor != nil {
			m := payload.Monitor
			if m.Name != "" {
				name = m.Name
			}
			if m.Slug != "" {
				slug = m.Slug
			}
			if m.Status != "" {
				status = strings.ToLower(m.Status)
			}
			if m.URL != "" {
				urlStr = m.URL
			} else if m.Target != "" {
				urlStr = m.Target
			}
			if m.LatencyMS > 0 {
				latencyMS = m.LatencyMS
			}
			if m.Error != nil && *m.Error != "" {
				errStr = *m.Error
			}
			if monitorID == "" {
				if m.ID != nil {
					monitorID = stringify(m.ID)
				} else if m.MonitorID != nil {
					monitorID = stringify(m.MonitorID)
				}
			}
		}

		if payload.Incident != nil {
			inc := payload.Incident
			if inc.Cause != "" && errStr == "" {
				errStr = inc.Cause
			}
			if inc.ResolvedAt != nil && *inc.ResolvedAt != "" {
				status = "up"
				if occurredAt == "" {
					occurredAt = *inc.ResolvedAt
				}
			} else if inc.StartedAt != "" {
				status = "down"
				if occurredAt == "" {
					occurredAt = inc.StartedAt
				}
			}
		}

		eventKey := firstNonEmpty(payload.Event, payload.EventType)
		if strings.HasSuffix(eventKey, ".down") {
			status = "down"
		} else if strings.HasSuffix(eventKey, ".up") {
			status = "up"
		}

		if name == "" {
			name = firstNonEmpty(slug, monitorID, "unknown")
		}

		eventType := "sonde.monitor." + firstNonEmpty(status, "event")
		eventTitle := fmt.Sprintf("%s %s", name, firstNonEmpty(status, "event"))

		var messageHTML string
		switch status {
		case "down":
			messageHTML = fmt.Sprintf("<b>[Sonde] %s</b> is DOWN", escape(name))
			if errStr != "" {
				messageHTML += " — " + escape(errStr)
			}
			if urlStr != "" {
				messageHTML += "<br/><code>" + escape(urlStr) + "</code>"
			}
		case "up":
			messageHTML = fmt.Sprintf("<b>[Sonde] %s</b> is back UP", escape(name))
			if latencyMS > 0 {
				messageHTML += fmt.Sprintf(" (latency: %dms)", latencyMS)
			}
		default:
			if eventKey != "" {
				messageHTML = fmt.Sprintf("<b>[Sonde] %s</b> — %s", escape(name), escape(eventKey))
			} else {
				messageHTML = fmt.Sprintf("<b>[Sonde] %s</b> status: %s", escape(name), escape(status))
			}
		}

		result := Result{
			Description: fmt.Sprintf("%s %s", name, firstNonEmpty(status, eventKey, "event")),
			MessageHTML: messageHTML,
			EventType:   eventType,
			EventTitle:  eventTitle,
			EventURL:    urlStr,
			OccurredAt:  occurredAt,
			Actor: &envelope.Actor{
				Email: str(actorEmail),
			},
			Subject: &envelope.Subject{
				Type: envelope.SubjectService,
				ID:   str(monitorID),
				Name: str(name),
				URL:  str(urlStr),
			},
			Normalized: map[string]any{
				"monitorId":   monitorID,
				"monitorName": name,
				"slug":        slug,
				"status":      status,
				"latencyMs":   latencyMS,
				"url":         urlStr,
				"error":       errStr,
			},
			ContainsPII: actorEmail != "",
		}

		return result, nil
	},
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
