package webhooks

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type dokployPayload struct {
	Title           string `json:"title"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"`
	ProjectName     string `json:"projectName"`
	ApplicationName string `json:"applicationName"`
	ApplicationType string `json:"applicationType"`
	BuildLink       string `json:"buildLink"`
	Status          string `json:"status"`
	Type            string `json:"type"`
}

var dokploy = Definition{
	Type:  settings.ProviderDokploy,
	Label: "Dokploy",
	Handle: func(body []byte, ctx Context) (Result, error) {
		var payload dokployPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		title := strings.TrimSpace(payload.Title)
		message := strings.TrimSpace(payload.Message)
		timestamp := strings.TrimSpace(payload.Timestamp)
		project := strings.TrimSpace(payload.ProjectName)
		application := strings.TrimSpace(payload.ApplicationName)

		description := firstNonEmpty(payload.Title, payload.Message, "unknown")
		if label := firstNonEmpty(application, project); label != "" {
			description = fmt.Sprintf("%s (%s)", description, label)
		}

		name := strings.Join(filterEmpty([]string{project, application}), " / ")
		if name == "" {
			name = firstNonEmpty(title, "Dokploy")
		}

		result := Result{
			Description: description,
			EventType:   firstNonEmpty(payload.Type, payload.Status, title, "dokploy.event"),
			EventTitle:  firstNonEmpty(title, name),
			EventURL:    payload.BuildLink,
			OccurredAt:  timestamp,
			Subject: &envelope.Subject{
				Type: envelope.SubjectDeployment,
				ID:   str(firstNonEmpty(application, project)),
				Name: str(name),
				URL:  str(payload.BuildLink),
			},
			Normalized: map[string]any{
				"title":           payload.Title,
				"message":         payload.Message,
				"projectName":     payload.ProjectName,
				"applicationName": payload.ApplicationName,
				"applicationType": payload.ApplicationType,
				"status":          payload.Status,
				"type":            payload.Type,
				"buildLink":       payload.BuildLink,
			},
		}

		// A payload with none of the three human fields carries nothing an
		// operator could act on, so it is accepted and not delivered.
		if title == "" && message == "" && timestamp == "" {
			return result, nil
		}

		heading := firstNonEmpty(title, ctx.ProviderName, "Dokploy")
		lines := []string{fmt.Sprintf("<b>[Dokploy] %s</b>", escape(heading))}

		if project != "" || application != "" {
			segments := make([]string, 0, 2)
			if project != "" {
				segments = append(segments, escape(project))
			}
			if application != "" {
				segments = append(segments, escape(application))
			}
			lines = append(lines, "Service: <b>"+strings.Join(segments, " / ")+"</b>")
		}
		if message != "" {
			lines = append(lines, strings.ReplaceAll(escape(message), "\n", "<br/>"))
		}
		if timestamp != "" {
			lines = append(lines, "<code>"+escape(formatTimestamp(timestamp))+"</code>")
		}

		result.MessageHTML = join(lines)
		return result, nil
	},
}

func filterEmpty(values []string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			kept = append(kept, value)
		}
	}
	return kept
}
