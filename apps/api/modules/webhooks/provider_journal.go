package webhooks

import (
	"fmt"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type journalPayload struct {
	Alert struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		SavedQueryID  int64  `json:"saved_query_id"`
		Provider      string `json:"provider"`
		Threshold     int    `json:"threshold"`
		WindowMinutes int    `json:"window_minutes"`
		WebhookURL    string `json:"webhook_url"`
		Enabled       bool   `json:"enabled"`
	} `json:"alert"`
	Count   int64  `json:"count"`
	Since   string `json:"since"`
	Until   string `json:"until"`
	Entries []struct {
		ID        int64  `json:"id"`
		App       string `json:"app"`
		Level     string `json:"level"`
		Message   string `json:"message"`
		Meta      any    `json:"meta,omitempty"`
		CreatedAt string `json:"created_at"`
	} `json:"entries"`
}

var journal = Definition{
	Type:  settings.ProviderJournal,
	Label: "Journal",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload journalPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		alert := payload.Alert
		if alert.Name == "" {
			return Result{Description: "unknown journal alert"}, nil
		}

		result := Result{
			Description: alert.Name,
			MessageHTML: journalMessage(payload),
			EventType:   "alert.fired",
			EventTitle:  alert.Name,
			OccurredAt:  payload.Until,
			Actor: &envelope.Actor{
				Name: str("Journal Alert"),
			},
			Subject: &envelope.Subject{
				Type: envelope.SubjectUnknown,
				ID:   str(fmt.Sprintf("alert_%d", alert.ID)),
				Name: str(alert.Name),
			},
			Normalized: map[string]any{
				"alert_id":       alert.ID,
				"alert_name":     alert.Name,
				"alert_provider": alert.Provider,
				"threshold":      alert.Threshold,
				"window_minutes": alert.WindowMinutes,
				"count":          payload.Count,
				"since":          payload.Since,
				"until":          payload.Until,
				"entry_count":    len(payload.Entries),
			},
			ContainsPII: false,
		}

		return result, nil
	},
}

func journalMessage(payload journalPayload) string {
	alert := payload.Alert
	entries := payload.Entries

	message := fmt.Sprintf("<b>[Journal]</b> Alert fired: <i>%s</i>", escape(alert.Name))
	message += fmt.Sprintf("<br/>Threshold: ≥ %d in %dm", alert.Threshold, alert.WindowMinutes)
	message += fmt.Sprintf("<br/>Count: %d", payload.Count)

	if len(entries) > 0 {
		message += "<br/><br/><b>Recent entries:</b>"
		for i, entry := range entries {
			if i >= 5 {
				break
			}
			levelClass := ""
			switch entry.Level {
			case "error":
				levelClass = "tone-danger"
			case "warn":
				levelClass = "tone-warning"
			case "info":
				levelClass = "tone-info"
			case "debug":
				levelClass = "tone-muted"
			}
			message += fmt.Sprintf("<br/><span class=\"badge %s\">%s</span> %s: %s",
				levelClass, escape(entry.Level), escape(entry.App), escape(entry.Message))
		}
	}

	return message
}
