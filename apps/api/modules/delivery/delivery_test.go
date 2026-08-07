package delivery

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

func target(id string, providerIDs, providerTags []string) settings.DeliveryTarget {
	return settings.DeliveryTarget{
		ID: id, Type: settings.TargetDiscord, Name: id, Enabled: true,
		ProviderIDs: providerIDs, ProviderTags: providerTags,
		WebhookURL: "https://discord.example/hook",
	}
}

// Routing is opt-in with no fallthrough. A target that names nothing receives
// nothing — this is the single most common "why is my target silent" cause, so
// it is pinned by a test rather than left to the reader.
func TestAnEmptyFilterSelectsNothing(t *testing.T) {
	if Accepts(target("t", nil, nil), "p1", []string{"infra"}) {
		t.Error("a target with no provider ids and no tags accepted an event")
	}
}

func TestAProviderIDSelects(t *testing.T) {
	if !Accepts(target("t", []string{"p1"}, nil), "p1", nil) {
		t.Error("a target naming the provider id rejected its event")
	}
	if Accepts(target("t", []string{"p2"}, nil), "p1", nil) {
		t.Error("a target naming another provider accepted the event")
	}
}

func TestASharedTagSelects(t *testing.T) {
	if !Accepts(target("t", nil, []string{"infra"}), "p1", []string{"infra", "prod"}) {
		t.Error("a shared tag did not select the target")
	}
	if Accepts(target("t", nil, []string{"infra"}), "p1", []string{"prod"}) {
		t.Error("a target matched on a tag it does not carry")
	}
}

// Nook's own events need the reserved id listed explicitly: enabling a system
// event is not enough on its own.
func TestSystemEventsNeedTheReservedProviderID(t *testing.T) {
	if Accepts(target("t", []string{"p1"}, nil), settings.SystemProviderID, nil) {
		t.Error("a target that lists only a real provider received a system event")
	}
	if !Accepts(target("t", []string{settings.SystemProviderID}, nil), settings.SystemProviderID, nil) {
		t.Error("a target that lists the system provider did not receive a system event")
	}
}

func TestSummarizeSeparatesFilteredFromUnconfigured(t *testing.T) {
	config := settings.DeliveryConfig{Items: []settings.DeliveryTarget{
		target("selected", []string{"p1"}, nil),
		target("filtered", []string{"p2"}, nil),
		{
			ID: "unconfigured", Type: settings.TargetDiscord, Name: "unconfigured",
			Enabled: true, ProviderIDs: []string{"p1"},
		},
	}}

	route := Summarize(config, "p1", nil)

	if len(route.Selected) != 2 {
		t.Errorf("selected %d targets, want 2", len(route.Selected))
	}
	if len(route.Sendable) != 1 || route.Sendable[0].ID != "selected" {
		t.Errorf("sendable = %v, want just the configured one", route.Sendable)
	}
	if route.SkippedByFilter != 1 {
		t.Errorf("skipped by filter = %d, want 1", route.SkippedByFilter)
	}
	if len(route.SkippedUnconfigured) != 1 {
		t.Errorf("skipped unconfigured = %d, want 1", len(route.SkippedUnconfigured))
	}
}

func TestCanSendChecksTheFieldsEachTypeNeeds(t *testing.T) {
	cases := []struct {
		name   string
		target settings.DeliveryTarget
		want   bool
	}{
		{"disabled", settings.DeliveryTarget{Type: settings.TargetDiscord, WebhookURL: "https://x"}, false},
		{"discord without a url", settings.DeliveryTarget{Type: settings.TargetDiscord, Enabled: true}, false},
		{"matrix without a room", settings.DeliveryTarget{
			Type: settings.TargetMatrix, Enabled: true, Homeserver: "https://m", AccessToken: "t",
		}, false},
		{"matrix ready", settings.DeliveryTarget{
			Type: settings.TargetMatrix, Enabled: true, Homeserver: "https://m",
			AccessToken: "t", RoomIDs: []string{"!r:m"},
		}, true},
		{"perception without a secret", settings.DeliveryTarget{
			Type: settings.TargetPerception, Enabled: true, URL: "https://p", Workspace: "w",
		}, false},
	}

	for _, testCase := range cases {
		if got := CanSend(testCase.target); got != testCase.want {
			t.Errorf("%s: CanSend = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

// A delivery that never recovers has to announce itself exactly once, when the
// queue gives up — not on the first attempt, where nearly every failure turns
// out to be a blip the retries fix, and not once per attempt.
func TestTheQueueAnnouncesADeliveryItGivesUpOnExactlyOnce(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	queue := NewRetryQueue(NewSender(discard), discard, nil)

	announced := 0
	queue.OnExhausted(func(settings.DeliveryTarget, Message) { announced++ })

	// An empty URL fails inside the sender, before any network call.
	queue.Enqueue("event-1", settings.DeliveryTarget{
		ID: "t1", Name: "Broken", Type: settings.TargetWebhook, Enabled: true,
	}, Message{})

	for range MaxRetryAttempts {
		queue.mu.Lock()
		for i := range queue.entries {
			queue.entries[i].dueAt = time.Now().Add(-time.Second)
		}
		queue.mu.Unlock()
		queue.process(context.Background())
	}

	if announced != 1 {
		t.Errorf("a delivery that never recovered announced itself %d times, want exactly 1", announced)
	}
	if queue.Size() != 0 {
		t.Errorf("%d entries left after the attempts ran out, want 0", queue.Size())
	}
}
