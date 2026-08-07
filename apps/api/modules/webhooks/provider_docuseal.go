package webhooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type docuSealDocument struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type docuSealPayload struct {
	EventType string `json:"event_type"`
	Timestamp string `json:"timestamp"`
	Data      *struct {
		ID            int    `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Role          string `json:"role"`
		ExternalID    string `json:"external_id"`
		DeclineReason string `json:"decline_reason"`
		Status        string `json:"status"`
		OpenedAt      string `json:"opened_at"`
		CompletedAt   string `json:"completed_at"`
		DeclinedAt    string `json:"declined_at"`
		Slug          string `json:"slug"`
		Submission    *struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			URL    string `json:"url"`
		} `json:"submission"`
		Template *struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"template"`
		Documents []docuSealDocument `json:"documents"`
	} `json:"data"`
}

var docuSeal = Definition{
	Type:  settings.ProviderDocuSeal,
	Label: "DocuSeal",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload docuSealPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		description := payload.EventType
		if description == "" {
			description = "unknown"
		}

		data := payload.Data
		if payload.EventType == "" || data == nil {
			return Result{Description: description}, nil
		}

		templateName := "Untitled template"
		templateID := ""
		if data.Template != nil && data.Template.Name != "" {
			templateName = data.Template.Name
			templateID = strconv.Itoa(data.Template.ID)
		} else if data.Name != "" {
			templateName = data.Name
		}

		submitter := firstNonEmpty(data.Name, data.Email, data.Role)

		submissionID := ""
		submissionURL := ""
		submissionStatus := data.Status
		if data.Submission != nil {
			if data.Submission.ID != 0 {
				submissionID = strconv.Itoa(data.Submission.ID)
			}
			submissionURL = data.Submission.URL
			if data.Submission.Status != "" {
				submissionStatus = data.Submission.Status
			}
		}
		if submissionID == "" && data.ID != 0 {
			submissionID = strconv.Itoa(data.ID)
		}

		result := Result{
			Description: description,
			EventType:   payload.EventType,
			EventTitle:  templateName,
			EventURL:    submissionURL,
			OccurredAt:  firstNonEmpty(payload.Timestamp, data.CompletedAt, data.DeclinedAt, data.OpenedAt),
			Actor: &envelope.Actor{
				ID:    str(strconv.Itoa(data.ID)),
				Name:  str(firstNonEmpty(data.Name, data.Role)),
				Email: str(data.Email),
			},
			Subject: &envelope.Subject{
				Type: envelope.SubjectContract,
				ID:   str(submissionID),
				Name: str(templateName),
				URL:  str(submissionURL),
			},
			Normalized: map[string]any{
				"eventType":        payload.EventType,
				"submissionId":     submissionID,
				"submissionStatus": submissionStatus,
				"templateId":       templateID,
				"templateName":     templateName,
				"completedAt":      data.CompletedAt,
				"declinedAt":       data.DeclinedAt,
				"openedAt":         data.OpenedAt,
				"status":           docuSealStatus(payload.EventType, submissionStatus),
				"completedDate":    data.CompletedAt,
			},
			ContainsPII: true,
		}

		result.MessageHTML = docuSealMessage(payload, templateName, submitter, submissionID, submissionURL)
		return result, nil
	},
}

func docuSealStatus(eventType, explicit string) string {
	if explicit != "" {
		return explicit
	}
	switch eventType {
	case "form.completed", "submission.completed":
		return "completed"
	case "form.declined", "submission.declined":
		return "declined"
	case "form.started", "form.viewed":
		return "pending"
	}
	return ""
}

func docuSealMessage(payload docuSealPayload, templateName, submitter, submissionID, submissionURL string) string {
	data := payload.Data
	openLink := ""
	if submissionURL != "" {
		openLink = fmt.Sprintf(`<a href="%s">Open submission</a>`, escape(submissionURL))
	}
	submissionLine := ""
	if submissionID != "" {
		submissionLine = "Submission: <code>" + escape(submissionID) + "</code>"
	}
	signerLine := ""
	if submitter != "" {
		signerLine = "Signer: <b>" + escape(submitter) + "</b>"
	}

	switch {
	case payload.EventType == "form.completed":
		externalLine := ""
		if data.ExternalID != "" {
			externalLine = "External ID: <code>" + escape(data.ExternalID) + "</code>"
		}
		return join([]string{
			fmt.Sprintf("<b>[DocuSeal] %s</b> completed", escape(templateName)),
			signerLine, submissionLine, externalLine, openLink,
			docuSealDocuments(data.Documents),
		})

	case payload.EventType == "form.declined":
		reasonLine := ""
		if data.DeclineReason != "" {
			reasonLine = "Reason: <i>" + escape(data.DeclineReason) + "</i>"
		}
		return join([]string{
			fmt.Sprintf("<b>[DocuSeal] %s</b> declined", escape(templateName)),
			signerLine, reasonLine, submissionLine, openLink,
		})

	case payload.EventType == "form.started", payload.EventType == "form.viewed":
		verb := "viewed"
		if payload.EventType == "form.started" {
			verb = "started"
		}
		return join([]string{
			fmt.Sprintf("<b>[DocuSeal] %s</b> %s", escape(templateName), verb),
			signerLine, submissionLine, openLink,
		})

	case strings.HasPrefix(payload.EventType, "template."):
		slugLine := ""
		if data.Slug != "" {
			slugLine = "Slug: <code>" + escape(data.Slug) + "</code>"
		}
		return join([]string{
			fmt.Sprintf("🧾 <b>[DocuSeal] Template %s</b>", escape(strings.TrimPrefix(payload.EventType, "template."))),
			"Name: <b>" + escape(templateName) + "</b>",
			"Template ID: <code>" + escape(strconv.Itoa(data.ID)) + "</code>",
			slugLine,
		})

	case strings.HasPrefix(payload.EventType, "submission."):
		return join([]string{
			fmt.Sprintf("📨 <b>[DocuSeal] Submission %s</b>", escape(strings.TrimPrefix(payload.EventType, "submission."))),
			"Template: <b>" + escape(templateName) + "</b>",
			submissionLine,
		})
	}

	return "ℹ️ <b>[DocuSeal]</b> " + escape(payload.EventType)
}

func docuSealDocuments(documents []docuSealDocument) string {
	lines := make([]string, 0, len(documents))
	for _, document := range documents {
		if document.URL == "" {
			continue
		}
		label := document.Name
		if label == "" {
			label = "document"
		}
		lines = append(lines, fmt.Sprintf(`• <a href="%s">%s</a>`, escape(document.URL), escape(label)))
	}
	if len(lines) == 0 {
		return ""
	}
	return "Documents:<br/>" + strings.Join(lines, "<br/>")
}
