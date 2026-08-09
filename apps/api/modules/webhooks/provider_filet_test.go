package webhooks

import (
	"encoding/json"
	"strings"
	"testing"
)

func filetBody(t *testing.T, overrides map[string]any) []byte {
	t.Helper()
	payload := map[string]any{
		"repository":    "FacileStudio/Sablier",
		"branch":        "main",
		"defaultBranch": "main",
		"commit":        "abc1234def",
		"actor":         "saravenpi",
		"status":        "failure",
		"runUrl":        "https://github.com/FacileStudio/Sablier/actions/runs/1",
		"grade":         "C",
		"files":         12,
		"lines":         3000,
		"counts":        map[string]int{"error": 2, "warn": 5, "info": 1},
		"findings": []map[string]any{
			{"rule": "go.err.wrap", "file": "apps/api/main.go", "line": 12,
				"severity": "error", "message": "%v loses the error chain"},
		},
	}
	for key, value := range overrides {
		payload[key] = value
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return body
}

func handleFilet(t *testing.T, body []byte) Result {
	t.Helper()
	result, err := filet.Handle(body, Context{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	return result
}

func TestFiletFailureOnDefaultBranchAlerts(t *testing.T) {
	result := handleFilet(t, filetBody(t, nil))

	if result.MessageHTML == "" {
		t.Fatal("a failure on the default branch must be delivered")
	}
	if result.EventType != "filet.check.failed" {
		t.Errorf("eventType = %q, want filet.check.failed", result.EventType)
	}
	for _, want := range []string{"FacileStudio/Sablier", "grade C", "go.err.wrap", "abc1234"} {
		if !strings.Contains(result.MessageHTML, want) {
			t.Errorf("message is missing %q:\n%s", want, result.MessageHTML)
		}
	}
}

func TestFiletCountsTheRemainderFromTheTallyNotTheTruncatedList(t *testing.T) {
	findings := make([]map[string]any, 0, 20)
	for i := range 20 {
		findings = append(findings, map[string]any{
			"rule": "go.doc.missing", "file": "a.go", "line": i + 1,
			"severity": "warn", "message": "no doc comment",
		})
	}
	result := handleFilet(t, filetBody(t, map[string]any{
		"counts":   map[string]int{"error": 1, "warn": 138, "info": 362},
		"findings": findings,
	}))

	if !strings.Contains(result.MessageHTML, "and 496 more") {
		t.Errorf("want the remainder of all 501 findings, not of the 20 that arrived:\n%s",
			result.MessageHTML)
	}
}

func TestFiletStaysSilentWhenNobodyNeedsWaking(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]any
	}{
		{"green run", map[string]any{"status": "success"}},
		{"failure on a pull request branch", map[string]any{"branch": "feature/x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := handleFilet(t, filetBody(t, tc.overrides))
			if result.MessageHTML != "" {
				t.Errorf("must be logged and not delivered, got:\n%s", result.MessageHTML)
			}
			if result.Description == "" {
				t.Error("a silent event still has to be described in the log")
			}
		})
	}
}

func TestFiletAlertsWhenTheBranchCannotBePlaced(t *testing.T) {
	result := handleFilet(t, filetBody(t, map[string]any{"defaultBranch": ""}))

	if result.MessageHTML == "" {
		t.Fatal("a run that cannot be placed on a branch must alert, not vanish")
	}
}

func TestFiletFallsBackToTheErrorCountWithoutAStatus(t *testing.T) {
	result := handleFilet(t, filetBody(t, map[string]any{
		"status": "",
		"counts": map[string]int{"error": 0, "warn": 3, "info": 0},
	}))

	if result.EventType != "filet.check.passed" {
		t.Errorf("eventType = %q, want filet.check.passed", result.EventType)
	}
	if result.MessageHTML != "" {
		t.Error("warnings alone are not worth an alert")
	}
}

func TestFiletNormalizedListsDistinctRules(t *testing.T) {
	result := handleFilet(t, filetBody(t, map[string]any{
		"findings": []map[string]any{
			{"rule": "go.err.wrap", "file": "a.go", "line": 1, "severity": "error", "message": "x"},
			{"rule": "go.err.wrap", "file": "b.go", "line": 2, "severity": "error", "message": "y"},
			{"rule": "gen.todo", "file": "c.go", "line": 3, "severity": "info", "message": "z"},
		},
	}))

	rules, ok := result.Normalized["rules"].([]string)
	if !ok {
		t.Fatalf("rules = %T, want []string", result.Normalized["rules"])
	}
	if len(rules) != 2 || rules[0] != "go.err.wrap" || rules[1] != "gen.todo" {
		t.Errorf("rules = %v, want the two distinct ids in first-seen order", rules)
	}
}
