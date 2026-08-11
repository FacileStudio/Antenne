package webhooks

import (
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
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
