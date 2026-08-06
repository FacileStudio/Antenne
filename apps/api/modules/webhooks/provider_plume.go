package webhooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type plumePayload struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	Owner      *struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"owner"`
	Document *struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		FileName   string `json:"file_name"`
		URL        string `json:"url"`
		Sequential bool   `json:"sequential"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	} `json:"document"`
	Signer *struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Role       string `json:"role"`
		Status     string `json:"status"`
		OrderNum   int    `json:"order_num"`
		SigningURL string `json:"signing_url"`
		SignedAt   string `json:"signed_at"`
		ViewedAt   string `json:"viewed_at"`
	} `json:"signer"`
}

var plume = Definition{
	Type:         settings.ProviderPlume,
	Label:        "Plume",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload plumePayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		if payload.EventType == "" || payload.Document == nil {
			return Result{Description: "unknown plume event"}, nil
		}

		document := payload.Document
		documentName := firstNonEmpty(document.Name, document.FileName, "document")

		occurredAt := payload.OccurredAt
		if payload.Signer != nil {
			occurredAt = firstNonEmpty(occurredAt, payload.Signer.SignedAt, payload.Signer.ViewedAt)
		}
		occurredAt = firstNonEmpty(occurredAt, document.UpdatedAt)

		subjectLabel := documentName
		if payload.Signer != nil && payload.Signer.Name != "" {
			subjectLabel = payload.Signer.Name
		}

		normalized := map[string]any{
			"eventId":           payload.EventID,
			"eventType":         payload.EventType,
			"documentId":        document.ID,
			"documentName":      documentName,
			"documentStatus":    document.Status,
			"documentFileName":  document.FileName,
			"documentCreatedAt": document.CreatedAt,
			"documentUpdatedAt": document.UpdatedAt,
			"sequential":        document.Sequential,
			"status":            plumeStatus(payload.EventType),
		}

		var actor *envelope.Actor
		if signer := payload.Signer; signer != nil {
			normalized["signerId"] = signer.ID
			normalized["signerName"] = signer.Name
			normalized["signerEmail"] = signer.Email
			normalized["signerRole"] = signer.Role
			normalized["signerStatus"] = signer.Status
			normalized["signerOrderNum"] = signer.OrderNum
			normalized["signedAt"] = signer.SignedAt
			normalized["viewedAt"] = signer.ViewedAt
			actor = &envelope.Actor{
				ID:    str(strconv.Itoa(signer.ID)),
				Name:  str(signer.Name),
				Email: str(signer.Email),
			}
		}
		if owner := payload.Owner; owner != nil {
			normalized["ownerId"] = owner.ID
			normalized["ownerName"] = owner.Name
			normalized["ownerEmail"] = owner.Email
			if actor == nil {
				actor = &envelope.Actor{
					ID:    str(strconv.Itoa(owner.ID)),
					Name:  str(owner.Name),
					Email: str(owner.Email),
				}
			}
		}

		eventURL := document.URL
		if eventURL == "" && payload.Signer != nil {
			eventURL = payload.Signer.SigningURL
		}

		return Result{
			Description: fmt.Sprintf("%s: %s", payload.EventType, subjectLabel),
			MessageHTML: plumeMessage(payload, documentName),
			EventType:   "plume." + payload.EventType,
			EventTitle:  documentName,
			EventURL:    eventURL,
			OccurredAt:  occurredAt,
			Actor:       actor,
			Subject: &envelope.Subject{
				Type: envelope.SubjectContract,
				ID:   str(strconv.Itoa(document.ID)),
				Name: str(documentName),
				URL:  str(document.URL),
			},
			Normalized:  normalized,
			ContainsPII: true,
		}, nil
	},
}

func plumeMessage(payload plumePayload, documentName string) string {
	icon := plumeEmoji(payload.EventType)
	verb := plumeVerb(payload.EventType)

	lines := make([]string, 0, 3)
	if payload.Signer != nil && strings.HasPrefix(payload.EventType, "signer.") {
		lines = append(lines, fmt.Sprintf("%s <b>%s</b> %s <b>%s</b>",
			icon, escape(payload.Signer.Name), escape(verb), escape(documentName)))
	} else {
		lines = append(lines, fmt.Sprintf("%s <b>%s</b> %s", icon, escape(documentName), escape(verb)))
	}

	if payload.Signer != nil && payload.Signer.Email != "" {
		lines = append(lines, "Signer: "+escape(payload.Signer.Email))
	}
	if payload.Document.URL != "" {
		lines = append(lines, fmt.Sprintf(`<a href="%s">Open document</a>`, escape(payload.Document.URL)))
	}

	return join(lines)
}

func plumeVerb(eventType string) string {
	switch eventType {
	case "document.created":
		return "created"
	case "document.sent":
		return "sent for signing"
	case "document.completed":
		return "completed"
	case "document.declined", "signer.declined":
		return "declined"
	case "document.deleted":
		return "deleted"
	case "signer.added":
		return "added a signer to"
	case "signer.viewed":
		return "opened"
	case "signer.signed":
		return "signed"
	case "signer.reminded":
		return "was reminded about"
	case "test.ping":
		return "test event from"
	}
	return eventType
}

func plumeEmoji(eventType string) string {
	switch eventType {
	case "document.completed", "signer.signed":
		return "✅"
	case "document.declined", "signer.declined":
		return "❌"
	case "document.sent":
		return "📨"
	case "signer.viewed":
		return "👀"
	case "signer.reminded":
		return "🔔"
	case "document.deleted":
		return "🗑️"
	case "document.created", "signer.added":
		return "📝"
	case "test.ping":
		return "🛎️"
	}
	return "ℹ️"
}

func plumeStatus(eventType string) string {
	switch eventType {
	case "document.completed", "signer.signed":
		return "completed"
	case "document.declined", "signer.declined":
		return "declined"
	case "document.sent", "signer.added", "signer.viewed", "signer.reminded":
		return "pending"
	case "document.created":
		return "draft"
	case "document.deleted":
		return "deleted"
	}
	return ""
}
