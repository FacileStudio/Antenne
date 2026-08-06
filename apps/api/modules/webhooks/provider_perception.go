package webhooks

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type perceptionAlert struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	EntityName string `json:"entity_name"`
}

type perceptionPayload struct {
	Source      string            `json:"source"`
	Event       string            `json:"event"`
	Count       int               `json:"count"`
	Alerts      []perceptionAlert `json:"alerts"`
	GeneratedAt string            `json:"generated_at"`
}

var perceptionSeverityEmoji = map[string]string{
	"critical": "🔴",
	"warning":  "🟠",
	"info":     "🔵",
}

var perceptionTypeLabels = map[string]string{
	"entity_velocity_spike":      "Velocity Spike",
	"service_health_degradation": "Health Degradation",
	"client_silence":             "Gone Quiet",
	"actor_absence":              "Actor Absent",
}

var perception = Definition{
	Type:  settings.ProviderPerception,
	Label: "Perception",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload perceptionPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		// Perception posts one digest shape; anything else on this path is a
		// misrouted body and is accepted without an alert.
		if payload.Source != "perception" || payload.Event != "digest" || len(payload.Alerts) == 0 {
			return Result{Description: "unknown perception event"}, nil
		}

		noun := "anomalies"
		if payload.Count == 1 {
			noun = "anomaly"
		}
		summary := fmt.Sprintf("%d %s detected", payload.Count, noun)

		rendered := make([]string, 0, len(payload.Alerts))
		types := make([]string, 0, len(payload.Alerts))
		entities := make([]string, 0, len(payload.Alerts))
		seenTypes := map[string]bool{}
		topSeverity := "info"

		for _, alert := range payload.Alerts {
			emoji := perceptionSeverityEmoji[alert.Severity]
			if emoji == "" {
				emoji = perceptionSeverityEmoji["info"]
			}
			label := perceptionTypeLabels[alert.Type]
			if label == "" {
				label = alert.Type
			}
			entity := ""
			if alert.EntityName != "" {
				entity = " <code>" + escape(alert.EntityName) + "</code>"
				entities = append(entities, alert.EntityName)
			}
			rendered = append(rendered, fmt.Sprintf("%s <b>%s</b>%s<br/><i>%s</i> <sub>[%s]</sub>",
				emoji, escape(alert.Title), entity, escape(alert.Message), escape(label)))

			if !seenTypes[alert.Type] {
				seenTypes[alert.Type] = true
				types = append(types, alert.Type)
			}
			if alert.Severity == "critical" {
				topSeverity = "critical"
			} else if alert.Severity == "warning" && topSeverity != "critical" {
				topSeverity = "warning"
			}
		}

		header := fmt.Sprintf("<b>[Perception]</b> %s", summary)

		return Result{
			Description: summary,
			MessageHTML: header + "<br/><br/>" + strings.Join(rendered, "<br/><br/>"),
			EventType:   "perception.digest",
			EventTitle:  fmt.Sprintf("%d %s", payload.Count, noun),
			OccurredAt:  payload.GeneratedAt,
			Normalized: map[string]any{
				"count":        payload.Count,
				"top_severity": topSeverity,
				"types":        types,
				"entities":     entities,
			},
		}, nil
	},
}
