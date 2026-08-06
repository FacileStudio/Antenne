package webhooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

type vikunjaDoer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (d *vikunjaDoer) display() string {
	if d == nil {
		return "someone"
	}
	return firstNonEmpty(d.Name, d.Username, d.Email, "someone")
}

type vikunjaTask struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Done       *bool  `json:"done"`
	Priority   *int   `json:"priority"`
	DueDate    string `json:"due_date"`
	ProjectID  int    `json:"project_id"`
	Identifier string `json:"identifier"`
}

type vikunjaPayload struct {
	EventName string `json:"event_name"`
	Time      string `json:"time"`
	Data      struct {
		Task    *vikunjaTask `json:"task"`
		Doer    *vikunjaDoer `json:"doer"`
		Project *struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"project"`
		Team *struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		Assignee *vikunjaDoer `json:"assignee"`
		Member   *vikunjaDoer `json:"member"`
	} `json:"data"`
}

var vikunja = Definition{
	Type:  settings.ProviderVikunja,
	Label: "Vikunja",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload vikunjaPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		event := payload.EventName
		if event == "" {
			return Result{Description: "unknown event"}, nil
		}

		task := payload.Data.Task
		project := payload.Data.Project
		doer := payload.Data.Doer

		subjectName := event
		subjectID := ""
		switch {
		case task != nil && task.Title != "":
			subjectName = task.Title
			subjectID = firstNonEmpty(task.Identifier, strconv.Itoa(task.ID))
		case project != nil && project.Title != "":
			subjectName = project.Title
			subjectID = strconv.Itoa(project.ID)
		case payload.Data.Team != nil && payload.Data.Team.Name != "":
			subjectName = payload.Data.Team.Name
		}

		subjectType := envelope.SubjectUnknown
		if strings.HasPrefix(event, "task.") || event == "tasks.overdue" {
			subjectType = envelope.SubjectTask
		}

		normalized := map[string]any{
			"event":  event,
			"status": vikunjaStatus(event, task),
		}
		if task != nil {
			normalized["taskId"] = task.ID
			normalized["taskIdentifier"] = task.Identifier
			normalized["taskTitle"] = task.Title
			normalized["taskDone"] = task.Done
			normalized["priority"] = task.Priority
			normalized["dueDate"] = task.DueDate
			normalized["projectId"] = task.ProjectID
			if task.Done != nil && *task.Done {
				normalized["completedDate"] = payload.Time
			}
		}
		if project != nil {
			normalized["projectTitle"] = project.Title
			if normalized["projectId"] == nil {
				normalized["projectId"] = project.ID
			}
		}
		if payload.Data.Assignee != nil {
			normalized["assignee"] = payload.Data.Assignee.display()
		}

		containsPII := (doer != nil && doer.Email != "") ||
			(payload.Data.Assignee != nil && payload.Data.Assignee.Email != "")

		result := Result{
			Description: event,
			MessageHTML: vikunjaMessage(payload),
			EventType:   event,
			EventTitle:  subjectName,
			OccurredAt:  payload.Time,
			Subject: &envelope.Subject{
				Type: subjectType,
				ID:   str(subjectID),
				Name: str(subjectName),
			},
			Normalized:  normalized,
			ContainsPII: containsPII,
		}
		if doer != nil {
			result.Actor = &envelope.Actor{
				ID:    str(strconv.Itoa(doer.ID)),
				Name:  str(firstNonEmpty(doer.Name, doer.Username)),
				Email: str(doer.Email),
			}
		}

		return result, nil
	},
}

func vikunjaStatus(event string, task *vikunjaTask) string {
	switch event {
	case "task.created":
		return "open"
	case "task.updated":
		if task != nil && task.Done != nil {
			if *task.Done {
				return "completed"
			}
			return "open"
		}
	case "task.deleted":
		return "deleted"
	case "task.overdue", "tasks.overdue":
		return "overdue"
	}
	return ""
}

func vikunjaMessage(payload vikunjaPayload) string {
	event := payload.EventName
	actor := escape(payload.Data.Doer.display())

	taskRef := ""
	taskTitle := "untitled"
	if task := payload.Data.Task; task != nil {
		if task.Identifier != "" {
			taskRef = "<code>" + escape(task.Identifier) + "</code>"
		} else if task.ID != 0 {
			taskRef = fmt.Sprintf("<code>#%d</code>", task.ID)
		}
		taskTitle = firstNonEmpty(task.Title, taskTitle)
	}
	taskLine := func(verb string) string {
		return fmt.Sprintf("📋 <b>[Vikunja]</b> %s %s task %s <b>%s</b>", actor, verb, taskRef, escape(taskTitle))
	}
	onTask := func(icon, verb string) string {
		spacer := ""
		if taskRef != "" {
			spacer = " " + taskRef
		}
		return fmt.Sprintf("%s <b>[Vikunja]</b> %s %s%s <b>%s</b>", icon, actor, verb, spacer, escape(taskTitle))
	}

	switch event {
	case "task.created":
		return taskLine("created")
	case "task.updated":
		if task := payload.Data.Task; task != nil && task.Done != nil {
			if *task.Done {
				return taskLine("completed")
			}
			return taskLine("reopened")
		}
		return taskLine("updated")
	case "task.deleted":
		return taskLine("deleted")
	case "task.reminder.fired":
		return taskLine("reminder fired for")
	case "task.overdue", "tasks.overdue":
		return taskLine("is overdue:")

	case "task.assignee.created":
		return onTask("👤", "assigned "+escape(payload.Data.Assignee.display())+" to")
	case "task.assignee.deleted":
		return onTask("👤", "unassigned "+escape(payload.Data.Assignee.display())+" from")
	case "task.comment.created":
		return onTask("💬", "commented on")
	case "task.comment.edited":
		return onTask("💬", "edited a comment on")
	case "task.comment.deleted":
		return onTask("💬", "deleted a comment on")
	case "task.relation.created":
		return onTask("🔗", "added a relation to")
	case "task.relation.deleted":
		return onTask("🔗", "removed a relation from")
	case "task.attachment.created":
		return onTask("📎", "attached a file to")
	case "task.attachment.deleted":
		return onTask("📎", "removed an attachment from")

	case "project.created", "project.updated", "project.deleted":
		title := "untitled"
		if payload.Data.Project != nil {
			title = firstNonEmpty(payload.Data.Project.Title, title)
		}
		verb := strings.TrimPrefix(event, "project.")
		return fmt.Sprintf("📁 <b>[Vikunja]</b> %s %s project <b>%s</b>", actor, verb, escape(title))

	case "project.shared.user", "project.shared.team":
		title := "untitled"
		if payload.Data.Project != nil {
			title = firstNonEmpty(payload.Data.Project.Title, title)
		}
		target := "a user"
		if event == "project.shared.team" {
			target = "a team"
		}
		return fmt.Sprintf("🔗 <b>[Vikunja]</b> %s shared project <b>%s</b> with %s", actor, escape(title), target)

	case "team.created", "team.deleted":
		name := "unnamed"
		if payload.Data.Team != nil {
			name = firstNonEmpty(payload.Data.Team.Name, name)
		}
		verb := strings.TrimPrefix(event, "team.")
		return fmt.Sprintf("👥 <b>[Vikunja]</b> %s %s team <b>%s</b>", actor, verb, escape(name))

	case "team.member.added", "team.member.removed":
		name := "unnamed"
		if payload.Data.Team != nil {
			name = firstNonEmpty(payload.Data.Team.Name, name)
		}
		verb := "added"
		if event == "team.member.removed" {
			verb = "removed"
		}
		return fmt.Sprintf("👥 <b>[Vikunja]</b> %s %s %s in team <b>%s</b>",
			actor, verb, escape(payload.Data.Member.display()), escape(name))
	}

	return fmt.Sprintf("📋 <b>[Vikunja]</b> %s triggered <code>%s</code>", actor, escape(event))
}
