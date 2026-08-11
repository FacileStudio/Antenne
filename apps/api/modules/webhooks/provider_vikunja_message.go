package webhooks

import (
	"fmt"
	"strings"
)

// vikunjaRender holds the pieces every Vikunja line shares, so each event
// family below reads as one format call instead of re-deriving the actor and
// the task reference.
type vikunjaRender struct {
	payload   vikunjaPayload
	actor     string
	taskRef   string
	taskTitle string
}

func newVikunjaRender(payload vikunjaPayload) vikunjaRender {
	render := vikunjaRender{
		payload:   payload,
		actor:     escape(payload.Data.Doer.display()),
		taskTitle: "untitled",
	}

	task := payload.Data.Task
	if task == nil {
		return render
	}
	if task.Identifier != "" {
		render.taskRef = "<code>" + escape(task.Identifier) + "</code>"
	} else if task.ID != 0 {
		render.taskRef = fmt.Sprintf("<code>#%d</code>", task.ID)
	}
	render.taskTitle = firstNonEmpty(task.Title, render.taskTitle)
	return render
}

func (r vikunjaRender) taskLine(verb string) string {
	return fmt.Sprintf("📋 <b>[Vikunja]</b> %s %s task %s <b>%s</b>", r.actor, verb, r.taskRef, escape(r.taskTitle))
}

func (r vikunjaRender) onTask(icon, verb string) string {
	spacer := ""
	if r.taskRef != "" {
		spacer = " " + r.taskRef
	}
	return fmt.Sprintf("%s <b>[Vikunja]</b> %s %s%s <b>%s</b>", icon, r.actor, verb, spacer, escape(r.taskTitle))
}

func (r vikunjaRender) taskUpdated() string {
	task := r.payload.Data.Task
	if task == nil || task.Done == nil {
		return r.taskLine("updated")
	}
	if *task.Done {
		return r.taskLine("completed")
	}
	return r.taskLine("reopened")
}

func (r vikunjaRender) taskMessage(event string) string {
	switch event {
	case "task.created":
		return r.taskLine("created")
	case "task.updated":
		return r.taskUpdated()
	case "task.deleted":
		return r.taskLine("deleted")
	case "task.reminder.fired":
		return r.taskLine("reminder fired for")
	case "task.overdue", "tasks.overdue":
		return r.taskLine("is overdue:")
	case "task.assignee.created":
		return r.onTask("👤", "assigned "+escape(r.payload.Data.Assignee.display())+" to")
	case "task.assignee.deleted":
		return r.onTask("👤", "unassigned "+escape(r.payload.Data.Assignee.display())+" from")
	case "task.comment.created":
		return r.onTask("💬", "commented on")
	case "task.comment.edited":
		return r.onTask("💬", "edited a comment on")
	case "task.comment.deleted":
		return r.onTask("💬", "deleted a comment on")
	case "task.relation.created":
		return r.onTask("🔗", "added a relation to")
	case "task.relation.deleted":
		return r.onTask("🔗", "removed a relation from")
	case "task.attachment.created":
		return r.onTask("📎", "attached a file to")
	case "task.attachment.deleted":
		return r.onTask("📎", "removed an attachment from")
	}
	return ""
}

func (r vikunjaRender) projectMessage(event string) string {
	title := "untitled"
	if r.payload.Data.Project != nil {
		title = firstNonEmpty(r.payload.Data.Project.Title, title)
	}
	title = escape(title)

	switch event {
	case "project.created", "project.updated", "project.deleted":
		return fmt.Sprintf("📁 <b>[Vikunja]</b> %s %s project <b>%s</b>",
			r.actor, strings.TrimPrefix(event, "project."), title)
	case "project.shared.user":
		return fmt.Sprintf("🔗 <b>[Vikunja]</b> %s shared project <b>%s</b> with a user", r.actor, title)
	case "project.shared.team":
		return fmt.Sprintf("🔗 <b>[Vikunja]</b> %s shared project <b>%s</b> with a team", r.actor, title)
	}
	return ""
}

func (r vikunjaRender) teamMessage(event string) string {
	name := "unnamed"
	if r.payload.Data.Team != nil {
		name = firstNonEmpty(r.payload.Data.Team.Name, name)
	}
	name = escape(name)

	switch event {
	case "team.created", "team.deleted":
		return fmt.Sprintf("👥 <b>[Vikunja]</b> %s %s team <b>%s</b>",
			r.actor, strings.TrimPrefix(event, "team."), name)
	case "team.member.added":
		return r.teamMemberLine("added", name)
	case "team.member.removed":
		return r.teamMemberLine("removed", name)
	}
	return ""
}

func (r vikunjaRender) teamMemberLine(verb, name string) string {
	return fmt.Sprintf("👥 <b>[Vikunja]</b> %s %s %s in team <b>%s</b>",
		r.actor, verb, escape(r.payload.Data.Member.display()), name)
}

func vikunjaMessage(payload vikunjaPayload) string {
	render := newVikunjaRender(payload)
	event := payload.EventName

	var message string
	switch {
	case strings.HasPrefix(event, "task.") || event == "tasks.overdue":
		message = render.taskMessage(event)
	case strings.HasPrefix(event, "project."):
		message = render.projectMessage(event)
	case strings.HasPrefix(event, "team."):
		message = render.teamMessage(event)
	}
	if message != "" {
		return message
	}

	return fmt.Sprintf("📋 <b>[Vikunja]</b> %s triggered <code>%s</code>", render.actor, escape(event))
}
