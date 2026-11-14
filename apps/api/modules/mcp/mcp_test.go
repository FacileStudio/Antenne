package mcp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestConformance runs tronc's MCP conformance kit, the replay confirmation included.
func TestConformance(t *testing.T) {
	f := newFixture(t)
	mcptest.Run(t, mcptest.Config{
		URL:   f.resource,
		Token: f.token,
		Confirm: []mcptest.ConfirmCase{{
			Tool: "antenne_replay_event",
			Setup: func(testing.TB) (map[string]any, func() int) {
				id := f.events.add("replay")
				return map[string]any{"event_id": id, "target_id": "matrix"}, func() int { return f.pipeline.runs(id, "matrix") }
			},
		}},
	})
}

// TestNoResultCarriesASecret calls every read tool and the switch over a configuration
// holding a credential everywhere one can sit, and checks none comes back.
func TestNoResultCarriesASecret(t *testing.T) {
	f := newFixture(t)
	id := f.events.add("down")
	session := mcptest.Connect(t, f.resource, f.token(t), mcptest.ClientOptions{})
	calls := map[string]map[string]any{
		"antenne_list_events":    {"response_format": "detailed"},
		"antenne_get_event":      {"event_id": id},
		"antenne_list_providers": {},
		"antenne_list_targets":   {},
		"antenne_pool_status":    {},
		"antenne_check_target":   {"target_id": "discord"},
		"antenne_set_enabled":    {"kind": "target", "id": "discord", "enabled": false},
	}
	for name, args := range calls {
		result := mcptest.Call(t, session, name, args)
		raw, err := json.Marshal(result)
		must(t, err)
		if result.IsError {
			t.Errorf("%s failed: %s", name, mcptest.Text(result))
		}
		for _, secret := range []string{secretPool, secretHook, secretToken, secretIMAP} {
			if strings.Contains(string(raw), secret) {
				t.Errorf("%s returned the credential %s", name, secret)
			}
		}
	}
}

// TestSwitchKeepsEveryCredential turns a target off and checks the save kept the
// credentials the dashboard never sees, and touched nothing else.
func TestSwitchKeepsEveryCredential(t *testing.T) {
	f := newFixture(t)
	session := mcptest.Connect(t, f.resource, f.token(t), mcptest.ClientOptions{})
	var out struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	mcptest.Decode(t, mcptest.Call(t, session, "antenne_set_enabled", map[string]any{"kind": "target", "id": "matrix", "enabled": false}), &out)
	current := f.pipeline.Settings()
	matrix, _ := current.FindTarget("matrix")
	discord, _ := current.FindTarget("discord")
	inbox, _ := current.FindProvider("inbox")
	switch {
	case out.Name != "Matrix" || out.Enabled || matrix.Enabled:
		t.Errorf("want Matrix off, got %+v and enabled %v", out, matrix.Enabled)
	case matrix.AccessToken != secretToken || !strings.HasSuffix(discord.WebhookURL, secretHook) || !discord.Enabled:
		t.Errorf("the save lost a target credential or switched the wrong target: %+v %+v", matrix, discord)
	case inbox.Password == nil || *inbox.Password != secretIMAP || current.Bus == nil || current.Bus.Secret != secretPool:
		t.Errorf("the save lost the IMAP password or the pool secret")
	}
}

// TestTheConfirmPageSendsOnceForTheSessionOnly asks for a replay from a client that
// cannot elicit, lists it on the confirm page, refuses a delegated token, then confirms it
// from the browser session.
func TestTheConfirmPageSendsOnceForTheSessionOnly(t *testing.T) {
	f := newFixture(t)
	id, confirmURL := askReplay(t, f, "page")
	var list struct {
		Actions []struct {
			ID     string                  `json:"id"`
			State  string                  `json:"state"`
			Event  *struct{ Title string } `json:"event"`
			Target *struct{ Name string }  `json:"target"`
		} `json:"actions"`
	}
	getJSON(t, pendingBase(f), &list)
	if len(list.Actions) != 1 || confirmURL != pendingBase(f)+"/"+list.Actions[0].ID+"/confirm" || list.Actions[0].State != "ready" ||
		list.Actions[0].Event == nil || list.Actions[0].Event.Title != "page" || list.Actions[0].Target == nil || list.Actions[0].Target.Name != "Matrix" {
		t.Fatalf("want the replay with its event and target, got %+v", list)
	}

	f.session.Actor = "bulle"
	if status := post(t, confirmURL); status != http.StatusForbidden || f.pipeline.runs(id, "matrix") != 0 {
		t.Errorf("a delegated token must not confirm, got %d and %d runs", status, f.pipeline.runs(id, "matrix"))
	}
	f.session.Actor = ""
	if status := post(t, confirmURL); status != http.StatusOK || f.pipeline.runs(id, "matrix") != 1 {
		t.Errorf("the session must confirm once, got %d and %d runs", status, f.pipeline.runs(id, "matrix"))
	}
	if status := post(t, confirmURL); status == http.StatusOK || f.pipeline.runs(id, "matrix") != 1 {
		t.Errorf("a second confirm must fail, got %d and %d runs", status, f.pipeline.runs(id, "matrix"))
	}
}

// TestACLISessionCannotConfirm refuses a replay confirmed from a session no browser
// sign-in opened, and leaves the event unsent.
func TestACLISessionCannotConfirm(t *testing.T) {
	f := newFixture(t)
	id, confirmURL := askReplay(t, f, "cli")
	f.session.BrowserSession = false
	if status := post(t, confirmURL); status != http.StatusForbidden || f.pipeline.runs(id, "matrix") != 0 {
		t.Errorf("a CLI session must not confirm, got %d and %d runs", status, f.pipeline.runs(id, "matrix"))
	}
}

// askReplay asks for a replay of a new event titled title from a client that cannot
// elicit, and returns the event and the confirm endpoint of the pending action.
func askReplay(t *testing.T, f *fixture, title string) (string, string) {
	t.Helper()
	id := f.events.add(title)
	session := mcptest.Connect(t, f.resource, f.token(t), mcptest.ClientOptions{})
	var asked struct {
		ConfirmURL string `json:"confirm_url"`
	}
	mcptest.Decode(t, mcptest.Call(t, session, "antenne_replay_event", map[string]any{"event_id": id, "target_id": "matrix"}), &asked)
	action := asked.ConfirmURL[strings.LastIndex(asked.ConfirmURL, "=")+1:]
	return id, pendingBase(f) + "/" + action + "/confirm"
}

// pendingBase is the confirm page API of the fixture.
func pendingBase(f *fixture) string {
	return strings.TrimSuffix(f.resource, "/mcp") + "/api/mcp/pending"
}

// TestAFailedTestSendIsAResultWithoutTheURL checks a refused send reads as failed, with
// the webhook path cut from the error.
func TestAFailedTestSendIsAResultWithoutTheURL(t *testing.T) {
	f := newFixture(t)
	f.pipeline.sendErr = &urlError{}
	session := mcptest.Connect(t, f.resource, f.token(t), mcptest.ClientOptions{})
	var out struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	mcptest.Decode(t, mcptest.Call(t, session, "antenne_test_target", map[string]any{"target_id": "discord"}), &out)
	if out.Status != "failed" || strings.Contains(out.Error, secretHook) || !strings.Contains(out.Error, "https://discord.com") {
		t.Errorf("want failed with the host only, got %+v", out)
	}
}

// urlError is a send failure whose text carries the webhook URL.
type urlError struct{}

func (*urlError) Error() string {
	return "discord answered 404 for https://discord.com/api/webhooks/1/" + secretHook
}

// getJSON reads url into out.
func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	must(t, err)
	defer resp.Body.Close()
	must(t, json.NewDecoder(resp.Body).Decode(out))
}

// post sends an empty POST and returns its status.
func post(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	must(t, err)
	resp.Body.Close()
	return resp.StatusCode
}
