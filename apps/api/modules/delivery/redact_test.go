package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// The token is in the *path* of a Discord webhook URL, which is what makes this
// worse than a query-string secret: nothing about the shape of the string warns
// you, and every http.Client error quotes the whole thing.
const secretToken = "aVerySecretWebhookToken0123456789"

func TestSafeErrorTextDropsTheURL(t *testing.T) {
	endpoint := "https://discord.com/api/webhooks/123456789/" + secretToken + "?wait=true"

	cases := map[string]error{
		"transport failure": &url.Error{
			Op:  "Get",
			URL: endpoint,
			Err: errors.New("dial tcp 1.2.3.4:443: i/o timeout"),
		},
		"wrapped twice": fmt.Errorf("probe: %w", &url.Error{
			Op:  "Post",
			URL: endpoint,
			Err: errors.New("connection reset by peer"),
		}),
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			text := SafeErrorText(err)

			if strings.Contains(text, secretToken) {
				t.Fatalf("the webhook token reached the error text: %q", text)
			}
			if strings.Contains(text, "123456789") {
				t.Errorf("the webhook id reached the error text: %q", text)
			}
			// Still diagnostic: an operator must be able to tell a timeout
			// reaching Discord from a rejected credential.
			if !strings.Contains(text, "discord.com") {
				t.Errorf("the host was dropped too, leaving nothing useful: %q", text)
			}
			if !strings.Contains(text, "timeout") && !strings.Contains(text, "reset") {
				t.Errorf("the cause was dropped: %q", text)
			}
		})
	}
}

// Belt and braces: drive the real probe against a URL that cannot be reached, so
// the assertion covers whatever the http client actually produces rather than an
// error I hand-built to match my own fix.
func TestProbeDiscordNeverLeaksTheTokenOnAnUnreachableHost(t *testing.T) {
	sender := NewSender(slog.New(slog.DiscardHandler))

	target := settings.DeliveryTarget{
		ID:      "discord-1",
		Type:    settings.TargetDiscord,
		Name:    "Alerts",
		Enabled: true,
		// Reserved by RFC 6761 and guaranteed not to resolve.
		DiscordSettings: settings.DiscordSettings{
			WebhookURL: "https://nonexistent.invalid/api/webhooks/999/" + secretToken,
		},
	}

	observation := sender.Probe(context.Background(), target)

	if observation.OK {
		t.Fatalf("an unreachable host probed OK: %+v", observation)
	}
	if strings.Contains(observation.Detail, secretToken) {
		t.Fatalf("the webhook token reached Observation.Detail: %q", observation.Detail)
	}
	if observation.Detail == "" {
		t.Error("Detail was emptied entirely; a redacted message must still say something")
	}
}
