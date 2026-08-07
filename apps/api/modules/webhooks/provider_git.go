package webhooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// gitPayload is the union of the GitHub, Gitea and GitLab push/issue/PR bodies.
// The three forges disagree on nearly every field name, so every accessor below
// falls through the alternatives rather than branching on a forge flag no
// payload actually carries.
type gitPayload struct {
	Ref     string `json:"ref"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Compare string `json:"compare"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		URL     string `json:"url"`
		Author  struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"commits"`
	HeadCommit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Sender struct {
		Login    string `json:"login"`
		Username string `json:"username"`
	} `json:"sender"`
	Repository struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	UserName string `json:"user_name"`
	User     struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	} `json:"project"`
	TotalCommitsCount *int   `json:"total_commits_count"`
	Action            string `json:"action"`
	Issue             *struct {
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	} `json:"issue"`
	PullRequest *struct {
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
		Merged  *bool  `json:"merged"`
	} `json:"pull_request"`
	ObjectAttributes *struct {
		Title  string `json:"title"`
		URL    string `json:"url"`
		Action string `json:"action"`
		State  string `json:"state"`
		IID    int    `json:"iid"`
	} `json:"object_attributes"`
}

var git = Definition{
	Type:  settings.ProviderGit,
	Label: "Git",
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload gitPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		repository := payload.repositoryName()
		actor := payload.actorName()
		primaryURL := payload.primaryURL()

		var eventType, description, message string
		switch {
		case payload.isPush():
			eventType, description = "git.push", "push to "+repository
			message = payload.pushMessage(repository, actor)
		case payload.isPullRequest():
			eventType, description = "git.pull_request", "PR on "+repository
			message = payload.pullRequestMessage(repository, actor)
		case payload.isIssue():
			eventType, description = "git.issue", "issue on "+repository
			message = payload.issueMessage(repository, actor)
		default:
			eventType, description = "git.event", "event on "+repository
			message = fmt.Sprintf("📡 <b>[%s]</b> %s triggered a webhook event", escape(repository), escape(actor))
		}

		commitSHA := payload.After
		commitMessage := ""
		if payload.HeadCommit != nil {
			commitSHA = firstNonEmpty(payload.HeadCommit.ID, commitSHA)
			commitMessage = firstLine(payload.HeadCommit.Message)
		} else if len(payload.Commits) > 0 {
			commitSHA = firstNonEmpty(payload.Commits[0].ID, commitSHA)
			commitMessage = firstLine(payload.Commits[0].Message)
		}

		return Result{
			Description: description,
			MessageHTML: message,
			EventType:   eventType,
			EventTitle:  repository,
			EventURL:    primaryURL,
			Actor:       &envelope.Actor{Name: str(actor)},
			Subject: &envelope.Subject{
				Type: envelope.SubjectRepository,
				ID:   str(repository),
				Name: str(repository),
				URL:  str(firstNonEmpty(payload.Repository.HTMLURL, payload.Project.WebURL)),
			},
			Normalized: map[string]any{
				"repository":    repository,
				"ref":           shortRef(payload.Ref),
				"before":        payload.Before,
				"after":         payload.After,
				"commitSha":     commitSHA,
				"commitMessage": commitMessage,
				"commitCount":   payload.commitCount(),
				"action":        payload.actionName(),
				"url":           primaryURL,
				"status":        payload.status(eventType),
			},
		}, nil
	},
}

func (p gitPayload) repositoryName() string {
	return firstNonEmpty(p.Repository.FullName, p.Project.PathWithNamespace, p.Repository.Name, "unknown")
}

func (p gitPayload) actorName() string {
	return firstNonEmpty(p.Pusher.Name, p.Sender.Login, p.Sender.Username, p.UserName, p.User.Name, p.User.Username, "someone")
}

func (p gitPayload) isPush() bool {
	return p.Ref != "" && (len(p.Commits) > 0 || p.TotalCommitsCount != nil)
}

func (p gitPayload) isPullRequest() bool {
	return p.PullRequest != nil || (p.ObjectAttributes != nil && p.Issue == nil)
}

func (p gitPayload) isIssue() bool { return p.Issue != nil }

func (p gitPayload) commitCount() int {
	if len(p.Commits) > 0 {
		return len(p.Commits)
	}
	if p.TotalCommitsCount != nil {
		return *p.TotalCommitsCount
	}
	return 0
}

func (p gitPayload) actionName() string {
	if p.Action != "" {
		return p.Action
	}
	if p.ObjectAttributes != nil {
		return p.ObjectAttributes.Action
	}
	return ""
}

func (p gitPayload) primaryURL() string {
	url := ""
	if p.PullRequest != nil {
		url = p.PullRequest.HTMLURL
	}
	if url == "" && p.Issue != nil {
		url = p.Issue.HTMLURL
	}
	if url == "" && p.ObjectAttributes != nil {
		url = p.ObjectAttributes.URL
	}
	return firstNonEmpty(url, p.Compare, p.Repository.HTMLURL, p.Project.WebURL)
}

func (p gitPayload) status(eventType string) string {
	switch eventType {
	case "git.push":
		return "pushed"
	case "git.pull_request":
		if p.PullRequest != nil && p.PullRequest.Merged != nil && *p.PullRequest.Merged {
			return "merged"
		}
		return p.actionName()
	case "git.issue":
		return p.actionName()
	}
	return ""
}

func (p gitPayload) pushMessage(repository, actor string) string {
	branch := "unknown"
	if p.Ref != "" {
		branch = shortRef(p.Ref)
	}
	count := p.commitCount()
	plural := "s"
	if count == 1 {
		plural = ""
	}

	lines := []string{fmt.Sprintf("<b>[%s]</b> %s pushed %d commit%s to <code>%s</code>",
		escape(repository), escape(actor), count, plural, escape(branch))}

	shown := p.Commits
	if len(shown) > 3 {
		shown = shown[:3]
	}
	for _, commit := range shown {
		sha := ""
		if commit.ID != "" {
			sha = "<code>" + escape(shortSHA(commit.ID)) + "</code>"
		}
		summary := firstLine(commit.Message)
		if summary == "" {
			summary = "no message"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", sha, escape(summary)))
	}
	if len(p.Commits) > 3 {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(p.Commits)-3))
	}
	if p.Compare != "" {
		lines = append(lines, fmt.Sprintf(`<a href="%s">Compare</a>`, escape(p.Compare)))
	}

	return join(lines)
}

func (p gitPayload) pullRequestMessage(repository, actor string) string {
	title, number, url := "untitled", "", ""
	if p.PullRequest != nil {
		title = firstNonEmpty(p.PullRequest.Title, title)
		number = strconv.Itoa(p.PullRequest.Number)
		url = p.PullRequest.HTMLURL
	} else if p.ObjectAttributes != nil {
		title = firstNonEmpty(p.ObjectAttributes.Title, title)
		number = strconv.Itoa(p.ObjectAttributes.IID)
		url = p.ObjectAttributes.URL
	} else {
		return ""
	}

	action := firstNonEmpty(p.actionName(), "updated")
	label := "#" + number
	if url != "" {
		label = fmt.Sprintf(`<a href="%s">#%s</a>`, escape(url), number)
	}

	return fmt.Sprintf("<b>[%s]</b> %s %s PR %s: %s",
		escape(repository), escape(actor), escape(action), label, escape(title))
}

func (p gitPayload) issueMessage(repository, actor string) string {
	if p.Issue == nil {
		return ""
	}

	title := firstNonEmpty(p.Issue.Title, "untitled")
	action := firstNonEmpty(p.actionName(), "updated")
	label := "#" + strconv.Itoa(p.Issue.Number)
	if p.Issue.HTMLURL != "" {
		label = fmt.Sprintf(`<a href="%s">#%d</a>`, escape(p.Issue.HTMLURL), p.Issue.Number)
	}

	return fmt.Sprintf("<b>[%s]</b> %s %s issue %s: %s",
		escape(repository), escape(actor), escape(action), label, escape(title))
}

func shortRef(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	return strings.TrimPrefix(ref, "refs/tags/")
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}
