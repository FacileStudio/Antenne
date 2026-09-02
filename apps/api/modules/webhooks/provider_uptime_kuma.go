package webhooks

import (
	"fmt"
	"strconv"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type uptimeKumaPayload struct {
	Heartbeat *struct {
		MonitorID int      `json:"monitorID"`
		Status    *int     `json:"status"`
		Time      string   `json:"time"`
		Msg       string   `json:"msg"`
		Ping      *float64 `json:"ping"`
		Duration  *float64 `json:"duration"`
	} `json:"heartbeat"`
	Monitor *struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		URL      string `json:"url"`
		Type     string `json:"type"`
		Hostname string `json:"hostname"`
	} `json:"monitor"`
	Msg string `json:"msg"`
}

var uptimeKuma = Definition{
	Type:  settings.ProviderUptimeKuma,
	Label: "Uptime Kuma",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload uptimeKumaPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		name := "unknown"
		monitorURL := ""
		monitorID := ""
		if payload.Monitor != nil {
			name = firstNonEmpty(payload.Monitor.Name, payload.Monitor.URL, "unknown")
			monitorURL = payload.Monitor.URL
			if payload.Monitor.ID != 0 {
				monitorID = strconv.Itoa(payload.Monitor.ID)
			}
		}

		status := "unknown"
		reason := payload.Msg
		occurredAt := ""
		if payload.Heartbeat != nil {
			if payload.Heartbeat.Msg != "" {
				reason = payload.Heartbeat.Msg
			}
			occurredAt = payload.Heartbeat.Time
			if monitorID == "" && payload.Heartbeat.MonitorID != 0 {
				monitorID = strconv.Itoa(payload.Heartbeat.MonitorID)
			}
			if payload.Heartbeat.Status != nil {
				switch *payload.Heartbeat.Status {
				case 1:
					status = "up"
				case 0:
					status = "down"
				}
			}
		}

		result := Result{
			Description: firstNonEmpty(payload.Msg, name),
			EventType:   "uptime-kuma." + status,
			EventTitle:  fmt.Sprintf("%s %s", name, status),
			EventURL:    monitorURL,
			OccurredAt:  occurredAt,
			Subject: &envelope.Subject{
				Type: envelope.SubjectService,
				ID:   str(monitorID),
				Name: str(name),
				URL:  str(monitorURL),
			},
			Normalized: map[string]any{
				"status":      status,
				"monitorId":   monitorID,
				"monitorName": name,
				"url":         monitorURL,
				"message":     reason,
			},
		}

		if payload.Heartbeat != nil {
			result.Normalized["ping"] = payload.Heartbeat.Ping
			result.Normalized["duration"] = payload.Heartbeat.Duration
		}

		switch status {
		case "up":
			result.MessageHTML = fmt.Sprintf("🟢 <b>[Uptime Kuma] %s</b> is back UP", escape(name))
			if reason != "" {
				result.MessageHTML += "<br/><i>" + escape(reason) + "</i>"
			}
		case "down":
			result.MessageHTML = fmt.Sprintf("🚨 <b>[Uptime Kuma] %s</b> is DOWN", escape(name))
			if reason != "" {
				result.MessageHTML += " — " + escape(reason)
			}
			if monitorURL != "" {
				result.MessageHTML += "<br/><code>" + escape(monitorURL) + "</code>"
			}
		default:
			// No heartbeat status at all is a bare notification; only a message
			// makes it worth forwarding.
			if payload.Msg != "" {
				result.MessageHTML = fmt.Sprintf("<b>[Uptime Kuma] %s</b> — %s", escape(name), escape(payload.Msg))
			}
		}

		return result, nil
	},
}
