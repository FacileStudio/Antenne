package webhooks

import (
	"fmt"
	"strconv"
	"strings"
)

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
