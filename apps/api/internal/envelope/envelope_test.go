package envelope

import (
	"net/http"
	"strings"
	"testing"
)

// The id is what lets a downstream consumer deduplicate a replayed event, so
// the same input has to hash the same however the payload map was ordered.
func TestTheIDIsStableAcrossMapOrdering(t *testing.T) {
	input := func() Input {
		return Input{
			Source:       SourceWebhook,
			ProviderID:   Str("p1"),
			ProviderType: "dokploy",
			ProviderName: "Dokploy",
			EventType:    "deploy.succeeded",
			OccurredAt:   "2026-08-06T10:00:00Z",
			NormalizedPayload: map[string]any{
				"zeta":  1,
				"alpha": "two",
				"mid":   []any{"a", "b"},
			},
		}
	}

	first := Build(input())
	second := Build(input())

	if first.ID != second.ID {
		t.Errorf("the same event hashed to %s and %s", first.ID, second.ID)
	}
	if !strings.HasPrefix(first.ID, "antenne_") {
		t.Errorf("id %q is missing the antenne_ prefix", first.ID)
	}
}

func TestADifferentPayloadHashesDifferently(t *testing.T) {
	base := Input{
		Source: SourceWebhook, ProviderType: "git", EventType: "git.push",
		OccurredAt:        "2026-08-06T10:00:00Z",
		NormalizedPayload: map[string]any{"commitSha": "aaa"},
	}
	other := base
	other.NormalizedPayload = map[string]any{"commitSha": "bbb"}

	if Build(base).ID == Build(other).ID {
		t.Error("two different payloads produced one id")
	}
}

// A target that is not allowed the raw payload must never receive it, and the
// flag has to travel with the redaction so the receiver can tell.
func TestRawPayloadIsWithheldUnlessAllowed(t *testing.T) {
	built := Build(Input{
		Source:       SourceWebhook,
		ProviderType: "docuseal",
		EventType:    "form.completed",
		RawPayload:   map[string]any{"email": "someone@example.com"},
	})

	if built.Payload.Raw != nil {
		t.Error("the raw payload was carried without rawPayloadAllowed")
	}

	allowed := Build(Input{
		Source: SourceWebhook, ProviderType: "vision", EventType: "vision.weekly",
		RawPayload: map[string]any{"pageviews": 10}, RawPayloadAllowed: true,
	})
	if allowed.Payload.Raw == nil {
		t.Fatal("an allowed raw payload was dropped")
	}

	redacted := RedactRaw(allowed)
	if redacted.Payload.Raw != nil {
		t.Error("RedactRaw left the payload in place")
	}
	if redacted.Privacy.RawPayloadAllowed {
		t.Error("RedactRaw left rawPayloadAllowed true, which misleads the receiver")
	}
}

func TestHTMLToPlainTextRendersTheMessageSubset(t *testing.T) {
	got := HTMLToPlainText("<b>Site</b> is DOWN<br/><code>https://x</code><p>tail</p>")
	want := "Site is DOWN\nhttps://xtail"
	if got != want {
		t.Errorf("HTMLToPlainText = %q, want %q", got, want)
	}
}

// The signature binds the timestamp and the event id, not just the body, so a
// receiver can reject a replay. Verify the header actually covers all three.
func TestSignedHeadersCoverTimestampEventAndBody(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	headers := SignedHeaders("secret", "antenne_abc", body)

	signature := headers.Get("x-antenne-signature-256")
	if signature == "" {
		t.Fatal("no signature was written")
	}

	timestamp := headers.Get("x-antenne-timestamp")
	signed := []byte(timestamp + ".antenne_abc." + string(body))
	if !VerifyHexSignature("secret", "sha256=", signature, signed) {
		t.Error("the signature does not verify over timestamp.eventId.body")
	}
	if VerifyHexSignature("secret", "sha256=", signature, body) {
		t.Error("the signature verified over the body alone, so a replay would pass")
	}
}

func TestNoSecretMeansNoSignature(t *testing.T) {
	headers := SignedHeaders("", "antenne_abc", []byte("{}"))
	if headers.Get("x-antenne-signature-256") != "" {
		t.Error("an unconfigured secret still produced a signature header")
	}
	if headers.Get("x-antenne-schema-version") != "1" {
		t.Error("the schema version header is missing")
	}
	var _ http.Header = headers
}
