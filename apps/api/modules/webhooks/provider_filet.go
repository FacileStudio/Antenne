package webhooks

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/antenne-client/go/catalog"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// filetMaxListed caps how many findings a message names. A failing run has
// hundreds and nobody reads past the first screen; the rest are one click away
// behind the run URL.
const filetMaxListed = 5

// filetRun is where and by whom the check ran. It is embedded rather than
// nested so the wire format stays flat, which is what the action sends.
type filetRun struct {
	Repository    string `json:"repository"`
	Branch        string `json:"branch"`
	DefaultBranch string `json:"defaultBranch"`
	Commit        string `json:"commit"`
	Actor         string `json:"actor"`
	Status        string `json:"status"`
	RunURL        string `json:"runUrl"`
}

type filetCounts struct {
	Error int `json:"error"`
	Warn  int `json:"warn"`
	Info  int `json:"info"`
}

type filetFinding struct {
	Rule     string `json:"rule"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type filetTests struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type filetPayload struct {
	filetRun
	Grade    string         `json:"grade"`
	Files    int            `json:"files"`
	Lines    int            `json:"lines"`
	Counts   filetCounts    `json:"counts"`
	Findings []filetFinding `json:"findings"`
	Tests    *filetTests    `json:"tests"`
}

// filet receives a CI run from the filet GitHub action.
//
// Only a failure on the repository's default branch is worth waking anyone for.
// A green run has nothing to say, and a failure on a pull request branch is
// already an annotation on the author's own diff — so both return an empty
// MessageHTML. Alerting on every run is how a channel gets muted, and a muted
// channel does not alert on the one run that mattered either.
//
// Be precise about what an empty MessageHTML costs: the intake answers 200 and
// writes an application log line, then returns *before* Dispatch. So a silent
// run never reaches the event log and is not replayable — only delivered events
// are. Anything that wants a record of green runs has to read Journal, not the
// events API.
var filet = Definition{
	Type:         settings.ProviderFilet,
	Label:        "filet",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload filetPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		repository := firstNonEmpty(payload.Repository, "unknown")
		failed := payload.failed()

		message := ""
		if failed && payload.onDefaultBranch() {
			message = filetMessage(payload, repository)
		}

		return Result{
			Description: payload.description(repository),
			MessageHTML: message,
			EventType:   payload.eventType(),
			EventTitle:  repository,
			EventURL:    payload.RunURL,
			Actor:       &envelope.Actor{Name: str(payload.Actor)},
			Subject: &envelope.Subject{
				Type: envelope.SubjectRepository,
				ID:   str(repository),
				Name: str(repository),
				URL:  str(payload.RunURL),
			},
			Normalized: payload.normalized(repository, failed),
		}, nil
	},
}

func (p filetPayload) failed() bool {
	if p.Status != "" {
		return !strings.EqualFold(p.Status, "success")
	}
	return p.Counts.Error > 0
}

// onDefaultBranch reports whether this run is on the branch the repository
// actually ships from. An unset defaultBranch means the sender could not tell
// us, and a run that cannot be placed is treated as the branch that matters —
// a false alert is recoverable, a silently dropped one is not.
func (p filetPayload) onDefaultBranch() bool {
	if p.Branch == "" || p.DefaultBranch == "" {
		return true
	}
	return p.Branch == p.DefaultBranch
}

func (p filetPayload) eventType() string {
	if p.failed() {
		return string(catalog.FiletCheckFailed)
	}
	return string(catalog.FiletCheckPassed)
}

func (p filetPayload) description(repository string) string {
	state := "passed"
	if p.failed() {
		state = "failed"
	}
	return fmt.Sprintf("filet %s on %s", state, repository)
}

func (p filetPayload) normalized(repository string, failed bool) map[string]any {
	normalized := map[string]any{
		"repository":    repository,
		"branch":        p.Branch,
		"defaultBranch": p.DefaultBranch,
		"commit":        p.Commit,
		"actor":         p.Actor,
		"status":        p.statusName(failed),
		"grade":         p.Grade,
		"files":         p.Files,
		"lines":         p.Lines,
		"errors":        p.Counts.Error,
		"warnings":      p.Counts.Warn,
		"info":          p.Counts.Info,
		"rules":         p.ruleIDs(),
		"url":           p.RunURL,
	}
	if p.Tests != nil {
		normalized["testsPassed"] = p.Tests.Passed
		normalized["testsFailed"] = p.Tests.Failed
	}
	return normalized
}

func (p filetPayload) statusName(failed bool) string {
	if failed {
		return "failed"
	}
	return "passed"
}

// ruleIDs lists the distinct rules the run tripped, so a downstream consumer
// can diff two runs without walking every finding.
func (p filetPayload) ruleIDs() []string {
	seen := map[string]bool{}
	rules := make([]string, 0, len(p.Findings))
	for _, finding := range p.Findings {
		if finding.Rule == "" || seen[finding.Rule] {
			continue
		}
		seen[finding.Rule] = true
		rules = append(rules, finding.Rule)
	}
	return rules
}

func filetMessage(p filetPayload, repository string) string {
	lines := []string{filetHeadline(p, repository), filetTally(p)}
	lines = append(lines, filetFindingLines(p)...)
	if p.RunURL != "" {
		lines = append(lines, fmt.Sprintf(`<a href="%s">Run</a>`, escape(p.RunURL)))
	}
	return join(lines)
}

func filetHeadline(p filetPayload, repository string) string {
	where := escape(firstNonEmpty(p.Branch, "unknown"))
	if p.Commit != "" {
		where += " <code>" + escape(shortSHA(p.Commit)) + "</code>"
	}
	return fmt.Sprintf("<b>[filet]</b> %s failed on %s", escape(repository), where)
}

func filetTally(p filetPayload) string {
	tally := fmt.Sprintf("%d errors, %d warnings", p.Counts.Error, p.Counts.Warn)
	if p.Grade != "" {
		tally = "grade " + escape(p.Grade) + " — " + tally
	}
	if p.Tests != nil && p.Tests.Failed > 0 {
		tally += fmt.Sprintf(", %d failing tests", p.Tests.Failed)
	}
	return tally
}

func filetFindingLines(p filetPayload) []string {
	shown := p.Findings
	if len(shown) > filetMaxListed {
		shown = shown[:filetMaxListed]
	}

	lines := make([]string, 0, len(shown)+1)
	for _, finding := range shown {
		lines = append(lines, fmt.Sprintf("  <code>%s:%d</code> %s <i>[%s]</i>",
			escape(finding.File), finding.Line, escape(finding.Message), escape(finding.Rule)))
	}
	if remaining := p.total() - len(shown); remaining > 0 {
		lines = append(lines, fmt.Sprintf("  ... and %d more", remaining))
	}
	return lines
}

// total is how many findings the run produced, which is not how many arrived:
// the sender truncates the list so a bad run is not a megabyte of JSON. Counting
// the remainder from the array would under-report a 500-finding run as 20, so it
// comes from the severity tally, which is always complete.
func (p filetPayload) total() int {
	if tallied := p.Counts.Error + p.Counts.Warn + p.Counts.Info; tallied > 0 {
		return tallied
	}
	return len(p.Findings)
}
