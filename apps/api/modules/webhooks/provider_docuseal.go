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

// docuSealLines are the three optional lines every submission event shares.
type docuSealLines struct {
	openLink   string
	submission string
	signer     string
}

func docuSealCommonLines(submitter, submissionID, submissionURL string) docuSealLines {
	lines := docuSealLines{}
	if submissionURL != "" {
		lines.openLink = fmt.Sprintf(`<a href="%s">Open submission</a>`, escape(submissionURL))
	}
	if submissionID != "" {
		lines.submission = "Submission: <code>" + escape(submissionID) + "</code>"
	}
	if submitter != "" {
		lines.signer = "Signer: <b>" + escape(submitter) + "</b>"
	}
	return lines
}

func docuSealMessage(payload docuSealPayload, templateName, submitter, submissionID, submissionURL string) string {
	lines := docuSealCommonLines(submitter, submissionID, submissionURL)

	switch {
	case payload.EventType == "form.completed":
		return docuSealCompleted(payload, templateName, lines)
	case payload.EventType == "form.declined":
		return docuSealDeclined(payload, templateName, lines)
	case payload.EventType == "form.started", payload.EventType == "form.viewed":
		return docuSealOpened(payload, templateName, lines)
	case strings.HasPrefix(payload.EventType, "template."):
		return docuSealTemplate(payload, templateName)
	case strings.HasPrefix(payload.EventType, "submission."):
		return join([]string{
			fmt.Sprintf("📨 <b>[DocuSeal] Submission %s</b>", escape(strings.TrimPrefix(payload.EventType, "submission."))),
			"Template: <b>" + escape(templateName) + "</b>",
			lines.submission,
		})
	}

	return "ℹ️ <b>[DocuSeal]</b> " + escape(payload.EventType)
}

func docuSealCompleted(payload docuSealPayload, templateName string, lines docuSealLines) string {
	externalLine := ""
	if payload.Data.ExternalID != "" {
		externalLine = "External ID: <code>" + escape(payload.Data.ExternalID) + "</code>"
	}
	return join([]string{
		fmt.Sprintf("<b>[DocuSeal] %s</b> completed", escape(templateName)),
		lines.signer, lines.submission, externalLine, lines.openLink,
		docuSealDocuments(payload.Data.Documents),
	})
}

func docuSealDeclined(payload docuSealPayload, templateName string, lines docuSealLines) string {
	reasonLine := ""
	if payload.Data.DeclineReason != "" {
		reasonLine = "Reason: <i>" + escape(payload.Data.DeclineReason) + "</i>"
	}
	return join([]string{
		fmt.Sprintf("<b>[DocuSeal] %s</b> declined", escape(templateName)),
		lines.signer, reasonLine, lines.submission, lines.openLink,
	})
}

func docuSealOpened(payload docuSealPayload, templateName string, lines docuSealLines) string {
	verb := "viewed"
	if payload.EventType == "form.started" {
		verb = "started"
	}
	return join([]string{
		fmt.Sprintf("<b>[DocuSeal] %s</b> %s", escape(templateName), verb),
		lines.signer, lines.submission, lines.openLink,
	})
}

func docuSealTemplate(payload docuSealPayload, templateName string) string {
	slugLine := ""
	if payload.Data.Slug != "" {
		slugLine = "Slug: <code>" + escape(payload.Data.Slug) + "</code>"
	}
	return join([]string{
		fmt.Sprintf("🧾 <b>[DocuSeal] Template %s</b>", escape(strings.TrimPrefix(payload.EventType, "template."))),
		"Name: <b>" + escape(templateName) + "</b>",
		"Template ID: <code>" + escape(strconv.Itoa(payload.Data.ID)) + "</code>",
		slugLine,
	})
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
