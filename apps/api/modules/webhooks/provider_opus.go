package webhooks

import (
	"fmt"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
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

// opusRender is the shared prefix of every Opus line: who did it, which task,
// and in which project. Each event below then adds only its own clause.
type opusRender struct {
	actor   string
	ref     string
	title   string
	project string
	data    opusData
}

func newOpusRender(payload opusPayload) opusRender {
	render := opusRender{actor: "someone", title: "untitled"}
	if payload.Actor != nil {
		render.actor = firstNonEmpty(payload.Actor.Name, render.actor)
	}
	render.actor = escape(render.actor)

	if task := payload.Task; task != nil {
		render.title = firstNonEmpty(task.Title, render.title)
		if task.Number != nil {
			render.ref = fmt.Sprintf("<code>#%d</code> ", *task.Number)
		}
	}
	render.title = escape(render.title)

	if payload.Project != nil && payload.Project.Name != "" {
		render.project = " in <b>" + escape(payload.Project.Name) + "</b>"
	}
	if payload.Data != nil {
		render.data = *payload.Data
	}
	return render
}

// change renders the "from x to y" clause only when both ends are known, so a
// partial payload degrades to the plain line instead of "from  to doing".
func (r opusRender) change(from, to string) string {
	if from == "" || to == "" {
		return ""
	}
	return fmt.Sprintf(" from <code>%s</code> to <code>%s</code>", escape(from), escape(to))
}

func (r opusRender) renamed() string {
	if r.data.OldTitle != "" && r.data.NewTitle != "" {
		return fmt.Sprintf("✏️ <b>[Opus]</b> %s renamed task %s<b>%s</b> → <b>%s</b>%s",
			r.actor, r.ref, escape(r.data.OldTitle), escape(r.data.NewTitle), r.project)
	}
	return fmt.Sprintf("✏️ <b>[Opus]</b> %s renamed task %s<b>%s</b>%s", r.actor, r.ref, r.title, r.project)
}

func (r opusRender) assigned() string {
	detail := ""
	if r.data.NewAssignee != "" {
		detail = " to <b>" + escape(r.data.NewAssignee) + "</b>"
	}
	return fmt.Sprintf("👤 <b>[Opus]</b> %s assigned %s<b>%s</b>%s%s", r.actor, r.ref, r.title, detail, r.project)
}

func (r opusRender) unassigned() string {
	detail := ""
	if r.data.OldAssignee != "" {
		detail = " from <b>" + escape(r.data.OldAssignee) + "</b>"
	}
	return fmt.Sprintf("👤 <b>[Opus]</b> %s unassigned %s<b>%s</b>%s%s", r.actor, r.ref, r.title, detail, r.project)
}

func opusMessage(payload opusPayload) string {
	r := newOpusRender(payload)

	switch payload.Event {
	case "task.created":
		return fmt.Sprintf("📋 <b>[Opus]</b> %s created task %s<b>%s</b>%s", r.actor, r.ref, r.title, r.project)
	case "task.status_changed":
		return fmt.Sprintf("🔄 <b>[Opus]</b> %s changed status of %s<b>%s</b>%s%s",
			r.actor, r.ref, r.title, r.change(r.data.OldStatus, r.data.NewStatus), r.project)
	case "task.priority_changed":
		return fmt.Sprintf("🏷️ <b>[Opus]</b> %s changed priority of %s<b>%s</b>%s%s",
			r.actor, r.ref, r.title, r.change(r.data.OldPriority, r.data.NewPriority), r.project)
	case "task.title_changed":
		return r.renamed()
	case "task.assignee_changed":
		return r.assigned()
	case "task.unassigned":
		return r.unassigned()
	case "task.description_changed":
		return fmt.Sprintf("📝 <b>[Opus]</b> %s updated description of %s<b>%s</b>%s", r.actor, r.ref, r.title, r.project)
	case "task.comment_created":
		return fmt.Sprintf("💬 <b>[Opus]</b> %s commented on %s<b>%s</b>%s", r.actor, r.ref, r.title, r.project)
	}

	return fmt.Sprintf("📋 <b>[Opus]</b> %s triggered <code>%s</code> on %s<b>%s</b>%s",
		r.actor, escape(payload.Event), r.ref, r.title, r.project)
}
