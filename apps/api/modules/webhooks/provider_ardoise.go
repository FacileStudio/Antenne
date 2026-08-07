package webhooks

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type ardoisePayload struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	CreatedAt string `json:"createdAt"`
	Source    string `json:"source"`
	Actor     *struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"actor"`
	Data *struct {
		Entity        string   `json:"entity"`
		EntityID      string   `json:"entityId"`
		ChangedFields []string `json:"changedFields"`
		Record        *struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"record"`
	} `json:"data"`
}

var ardoise = Definition{
	Type:         settings.ProviderArdoise,
	Label:        "Ardoise",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload ardoisePayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		if payload.Event == "" {
			return Result{
				Description: "ardoise event",
				MessageHTML: "📡 <b>[Ardoise]</b> received an event",
				EventType:   "ardoise.event",
				EventTitle:  "Ardoise event",
				ContainsPII: true,
			}, nil
		}

		entity := "client"
		entityID := ""
		var changedFields []string
		recordEmail := ""
		recordName := ""

		if payload.Data != nil {
			entity = firstNonEmpty(payload.Data.Entity, eventSegment(payload.Event, 1), entity)
			entityID = payload.Data.EntityID
			changedFields = payload.Data.ChangedFields
			if record := payload.Data.Record; record != nil {
				recordEmail = record.Email
				recordName = strings.TrimSpace(strings.Join(filterEmpty([]string{record.FirstName, record.LastName}), " "))
				if entityID == "" {
					entityID = record.ID
				}
			}
		}
		if entityID == "" {
			entityID = payload.ID
		}
		if changedFields == nil {
			changedFields = []string{}
		}

		result := Result{
			Description: payload.Event,
			MessageHTML: ardoiseMessage(payload, recordName),
			EventType:   payload.Event,
			EventTitle:  titleCase(payload.Event),
			OccurredAt:  payload.CreatedAt,
			Subject: &envelope.Subject{
				Type: envelope.SubjectClient,
				ID:   str(entityID),
				Name: str(firstNonEmpty(recordName, recordEmail, entity)),
			},
			Normalized: map[string]any{
				"event":         payload.Event,
				"source":        payload.Source,
				"entity":        entity,
				"entityId":      entityID,
				"changedFields": changedFields,
				"recordEmail":   recordEmail,
				"status":        ardoiseStatus(payload.Event),
			},
			ContainsPII: true,
		}

		if payload.Actor != nil {
			result.Actor = &envelope.Actor{
				ID:    str(payload.Actor.ID),
				Email: str(payload.Actor.Email),
				Name:  str(payload.Actor.Role),
			}
		}

		return result, nil
	},
}

func ardoiseMessage(payload ardoisePayload, recordName string) string {
	actor := "someone"
	if payload.Actor != nil {
		actor = firstNonEmpty(payload.Actor.Email, payload.Actor.ID, actor)
	}

	switch {
	case strings.HasPrefix(payload.Event, "admin."):
		entity := firstNonEmpty(eventSegment(payload.Event, 1), "record")
		action := firstNonEmpty(eventSegment(payload.Event, 2), "updated")

		entityID, email := "", ""
		if payload.Data != nil {
			entityID = payload.Data.EntityID
			if record := payload.Data.Record; record != nil {
				email = record.Email
				if entityID == "" {
					entityID = record.ID
				}
			}
		}

		lines := []string{fmt.Sprintf("<b>[Ardoise]</b> %s %s %s", escape(actor), escape(action), escape(entity))}
		if entityID != "" {
			lines = append(lines, "ID: <code>"+escape(entityID)+"</code>")
		}
		if recordName != "" {
			lines = append(lines, "Name: <b>"+escape(recordName)+"</b>")
		}
		if email != "" {
			lines = append(lines, "Email: <code>"+escape(email)+"</code>")
		}
		return join(lines)

	case payload.Event == "settings.logo.updated":
		return "🖼️ <b>[Ardoise]</b> logo updated"

	case strings.HasPrefix(payload.Event, "settings."):
		lines := []string{"⚙️ <b>[Ardoise]</b> settings updated"}
		if payload.Data != nil && len(payload.Data.ChangedFields) > 0 {
			fields := make([]string, 0, len(payload.Data.ChangedFields))
			for _, field := range payload.Data.ChangedFields {
				fields = append(fields, "<code>"+escape(field)+"</code>")
			}
			lines = append(lines, "Fields: "+strings.Join(fields, ", "))
		}
		return join(lines)
	}

	return "📡 <b>[Ardoise]</b> " + escape(titleCase(payload.Event))
}

func ardoiseStatus(event string) string {
	parts := strings.Split(event, ".")
	switch parts[len(parts)-1] {
	case "create", "created":
		return "created"
	case "update", "updated":
		return "updated"
	case "delete", "deleted":
		return "deleted"
	}
	return ""
}

func eventSegment(event string, index int) string {
	parts := strings.Split(event, ".")
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func titleCase(event string) string {
	parts := strings.Split(event, ".")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
