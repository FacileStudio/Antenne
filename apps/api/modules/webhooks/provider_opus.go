package webhooks

import (
	"fmt"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type opusPayload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Project   *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
	Task *struct {
		ID       string `json:"id"`
		Number   *int   `json:"number"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
		URL      string `json:"url"`
	} `json:"task"`
	Actor *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"actor"`
	Data *opusData `json:"data"`
}

type opusData struct {
	OldStatus   string `json:"oldStatus"`
	NewStatus   string `json:"newStatus"`
	OldPriority string `json:"oldPriority"`
	NewPriority string `json:"newPriority"`
	OldTitle    string `json:"oldTitle"`
	NewTitle    string `json:"newTitle"`
	OldAssignee string `json:"oldAssignee"`
	NewAssignee string `json:"newAssignee"`
}

// Opus signs the bare hex digest, without the `sha256=` prefix the other suite
// apps use.
var opus = Definition{
	Type:         settings.ProviderOpus,
	Label:        "Opus",
	Authenticate: signedWith(""),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload opusPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		event := payload.Event
		if event == "" {
			return Result{
				Description: "opus event",
				MessageHTML: "📋 <b>[Opus]</b> received an event",
				EventType:   "opus.event",
				EventTitle:  "opus.event",
			}, nil
		}

		taskID, taskTitle, taskURL, taskPriority := "", event, "", ""
		var taskNumber *int
		if task := payload.Task; task != nil {
			taskID = task.ID
			taskTitle = firstNonEmpty(task.Title, event)
			taskURL = task.URL
			taskPriority = task.Priority
			taskNumber = task.Number
		}

		projectID, projectName := "", ""
		if payload.Project != nil {
			projectID = payload.Project.ID
			projectName = payload.Project.Name
		}

		status := ""
		switch {
		case event == "task.created":
			status = "open"
		case event == "task.status_changed" && payload.Data != nil:
			status = payload.Data.NewStatus
		}

		priority := taskPriority
		if event == "task.priority_changed" && payload.Data != nil {
			priority = payload.Data.NewPriority
		}

		result := Result{
			Description: event,
			MessageHTML: opusMessage(payload),
			EventType:   event,
			EventTitle:  taskTitle,
			EventURL:    taskURL,
			OccurredAt:  payload.Timestamp,
			Subject: &envelope.Subject{
				Type: envelope.SubjectTask,
				ID:   str(taskID),
				Name: str(taskTitle),
				URL:  str(taskURL),
			},
			Normalized: map[string]any{
				"event":       event,
				"taskId":      taskID,
				"taskNumber":  taskNumber,
				"taskTitle":   taskTitle,
				"projectId":   projectID,
				"projectName": projectName,
				"status":      status,
				"priority":    priority,
			},
		}

		if payload.Actor != nil {
			result.Actor = &envelope.Actor{ID: str(payload.Actor.ID), Name: str(payload.Actor.Name)}
		}

		return result, nil
	},
}

func opusMessage(payload opusPayload) string {
	actor := "someone"
	if payload.Actor != nil {
		actor = firstNonEmpty(payload.Actor.Name, actor)
	}
	actor = escape(actor)

	title := "untitled"
	ref := ""
	if task := payload.Task; task != nil {
		title = firstNonEmpty(task.Title, title)
		if task.Number != nil {
			ref = fmt.Sprintf("<code>#%d</code> ", *task.Number)
		}
	}
	title = escape(title)

	project := ""
	if payload.Project != nil && payload.Project.Name != "" {
		project = " in <b>" + escape(payload.Project.Name) + "</b>"
	}

	change := func(from, to string) string {
		if from == "" || to == "" {
			return ""
		}
		return fmt.Sprintf(" from <code>%s</code> to <code>%s</code>", escape(from), escape(to))
	}

	data := payload.Data
	if data == nil {
		data = &opusData{}
	}

	switch payload.Event {
	case "task.created":
		return fmt.Sprintf("📋 <b>[Opus]</b> %s created task %s<b>%s</b>%s", actor, ref, title, project)
	case "task.status_changed":
		return fmt.Sprintf("🔄 <b>[Opus]</b> %s changed status of %s<b>%s</b>%s%s",
			actor, ref, title, change(data.OldStatus, data.NewStatus), project)
	case "task.priority_changed":
		return fmt.Sprintf("🏷️ <b>[Opus]</b> %s changed priority of %s<b>%s</b>%s%s",
			actor, ref, title, change(data.OldPriority, data.NewPriority), project)
	case "task.title_changed":
		if data.OldTitle != "" && data.NewTitle != "" {
			return fmt.Sprintf("✏️ <b>[Opus]</b> %s renamed task %s<b>%s</b> → <b>%s</b>%s",
				actor, ref, escape(data.OldTitle), escape(data.NewTitle), project)
		}
		return fmt.Sprintf("✏️ <b>[Opus]</b> %s renamed task %s<b>%s</b>%s", actor, ref, title, project)
	case "task.assignee_changed":
		detail := ""
		if data.NewAssignee != "" {
			detail = " to <b>" + escape(data.NewAssignee) + "</b>"
		}
		return fmt.Sprintf("👤 <b>[Opus]</b> %s assigned %s<b>%s</b>%s%s", actor, ref, title, detail, project)
	case "task.unassigned":
		detail := ""
		if data.OldAssignee != "" {
			detail = " from <b>" + escape(data.OldAssignee) + "</b>"
		}
		return fmt.Sprintf("👤 <b>[Opus]</b> %s unassigned %s<b>%s</b>%s%s", actor, ref, title, detail, project)
	case "task.description_changed":
		return fmt.Sprintf("📝 <b>[Opus]</b> %s updated description of %s<b>%s</b>%s", actor, ref, title, project)
	case "task.comment_created":
		return fmt.Sprintf("💬 <b>[Opus]</b> %s commented on %s<b>%s</b>%s", actor, ref, title, project)
	}

	return fmt.Sprintf("📋 <b>[Opus]</b> %s triggered <code>%s</code> on %s<b>%s</b>%s",
		actor, escape(payload.Event), ref, title, project)
}
