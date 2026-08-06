package webhooks

import (
	"fmt"
	"strconv"
	"time"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type sablierPayload struct {
	Event string `json:"event"`
	Data  *struct {
		ID          int    `json:"id"`
		ProjectID   int    `json:"project_id"`
		ProjectName string `json:"project_name"`
		TaskID      int    `json:"task_id"`
		TaskName    string `json:"task_name"`
		UserID      int    `json:"user_id"`
		UserEmail   string `json:"user_email"`
		Description string `json:"description"`
		StartedAt   string `json:"started_at"`
		StoppedAt   string `json:"stopped_at"`
	} `json:"data"`
}

var sablier = Definition{
	Type:  settings.ProviderSablier,
	Label: "Sablier",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload sablierPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		if payload.Event == "" || payload.Data == nil {
			return Result{Description: firstNonEmpty(payload.Event, "unknown event")}, nil
		}
		data := payload.Data

		status := ""
		completedDate := ""
		switch payload.Event {
		case "timer_started":
			status = "started"
		case "timer_stopped":
			status = "stopped"
			completedDate = data.StoppedAt
		}

		result := Result{
			Description: payload.Event,
			MessageHTML: sablierMessage(payload),
			EventType:   payload.Event,
			EventTitle:  firstNonEmpty(data.TaskName, data.ProjectName, payload.Event, "Sablier event"),
			OccurredAt:  firstNonEmpty(data.StoppedAt, data.StartedAt),
			Actor: &envelope.Actor{
				ID:    str(strconv.Itoa(data.UserID)),
				Email: str(data.UserEmail),
			},
			Subject: &envelope.Subject{
				Type: envelope.SubjectSchedule,
				ID:   str(strconv.Itoa(data.ID)),
				Name: str(firstNonEmpty(data.TaskName, data.ProjectName, data.Description)),
			},
			Normalized: map[string]any{
				"event":         payload.Event,
				"timeEntryId":   data.ID,
				"projectId":     data.ProjectID,
				"projectName":   data.ProjectName,
				"taskId":        data.TaskID,
				"taskName":      data.TaskName,
				"startedAt":     data.StartedAt,
				"stoppedAt":     data.StoppedAt,
				"status":        status,
				"completedDate": completedDate,
			},
			ContainsPII: true,
		}

		return result, nil
	},
}

func sablierMessage(payload sablierPayload) string {
	data := payload.Data

	user, project, task, description := "", "", "", ""
	if data.UserEmail != "" {
		user = " <code>" + escape(data.UserEmail) + "</code>"
	}
	if data.ProjectName != "" {
		project = " on <i>" + escape(data.ProjectName) + "</i>"
	}
	if data.TaskName != "" {
		task = " / <i>" + escape(data.TaskName) + "</i>"
	}
	if data.Description != "" {
		description = ": " + escape(data.Description)
	}

	switch payload.Event {
	case "timer_started":
		return fmt.Sprintf("<b>[Sablier]</b> Timer started%s%s%s%s", user, project, task, description)
	case "timer_stopped":
		return fmt.Sprintf("<b>[Sablier]</b> Timer stopped%s%s%s%s%s",
			user, project, task, description, sablierDuration(data.StartedAt, data.StoppedAt))
	}

	return fmt.Sprintf("<b>[Sablier]</b> <code>%s</code>", escape(payload.Event))
}

func sablierDuration(startedAt, stoppedAt string) string {
	if startedAt == "" || stoppedAt == "" {
		return ""
	}
	start, err := parseTime(startedAt)
	if err != nil {
		return ""
	}
	stop, err := parseTime(stoppedAt)
	if err != nil {
		return ""
	}

	span := stop.Sub(start)
	if span < 0 {
		return ""
	}

	hours := int(span / time.Hour)
	minutes := int(span/time.Minute) % 60
	switch {
	case hours > 0 && minutes > 0:
		return fmt.Sprintf(" (%dh %dm)", hours, minutes)
	case hours > 0:
		return fmt.Sprintf(" (%dh)", hours)
	}
	return fmt.Sprintf(" (%dm)", minutes)
}
