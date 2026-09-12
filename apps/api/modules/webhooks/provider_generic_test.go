package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type genericCase struct {
	name       string
	body       string
	eventType  string
	eventTitle string
	contains   []string
}

func assertGenericCase(t *testing.T, tc genericCase) {
	t.Helper()
	result, err := generic.Handle([]byte(tc.body), Context{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if result.EventType != tc.eventType {
		t.Errorf("eventType = %q, want %q", result.EventType, tc.eventType)
	}
	if result.EventTitle != tc.eventTitle {
		t.Errorf("eventTitle = %q, want %q", result.EventTitle, tc.eventTitle)
	}
	if result.Description == "" {
		t.Error("a generic event still has to be described in the log")
	}
	if result.MessageHTML == "" {
		t.Fatal("expected a delivered message")
	}
	for _, want := range tc.contains {
		if !strings.Contains(result.MessageHTML, want) {
			t.Errorf("message is missing %q:\n%s", want, result.MessageHTML)
		}
	}
}

func TestGenericHandle(t *testing.T) {
	for _, tc := range []genericCase{
		{
			name: "full payload",
			body: `{"title":"Deploy failed","message":"api returned 502","event":"deploy.failed",` +
				`"url":"https://example.com/deploys/7","source":"dokploy","data":{"app":"api","attempt":3}}`,
			eventType:  "deploy.failed",
			eventTitle: "Deploy failed",
			contains:   []string{"<b>[dokploy] Deploy failed</b>", "api returned 502", "app: api", "attempt: 3"},
		},
		{
			name:      "data only is still an alert",
			body:      `{"data":{"deploy":"staging"}}`,
			eventType: "generic.event",
			contains:  []string{"deploy: staging"},
		},
		{
			name:       "everything is escaped",
			body:       `{"source":"<cron>","title":"<script>alert(1)</script>","message":"<img src=x>","data":{"q":"<a href>"}}`,
			eventType:  "generic.event",
			eventTitle: "<script>alert(1)</script>",
			contains:   []string{"&lt;script&gt;alert(1)&lt;/script&gt;", "&lt;img src=x&gt;", "q: &lt;a href&gt;"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertGenericCase(t, tc)
		})
	}
}

func TestGenericEmptyObjectIsSilent(t *testing.T) {
	result, err := generic.Handle([]byte(`{}`), Context{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result.MessageHTML != "" {
		t.Errorf("must be accepted silently, got:\n%s", result.MessageHTML)
	}
	if result.Description == "" {
		t.Error("a silent event still has to be described in the log")
	}
	if result.EventType != "generic.event" {
		t.Errorf("eventType = %q, want generic.event", result.EventType)
	}
}

func TestGenericPlainBodies(t *testing.T) {
	for _, tc := range []genericCase{
		{
			name:      "plain array body",
			body:      `["deploy","staging"]`,
			eventType: "generic.event",
			contains:  []string{`["deploy","staging"]`},
		},
		{
			name:      "plain string body",
			body:      `"webhook ping"`,
			eventType: "generic.event",
			contains:  []string{"webhook ping"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertGenericCase(t, tc)
		})
	}
}

func TestGenericFullPayloadCarriesTheKnownFields(t *testing.T) {
	result, err := generic.Handle([]byte(`{"title":"Deploy failed","event":"deploy.failed","url":"https://example.com/d","source":"dokploy","data":{"app":"api"}}`), Context{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if result.EventURL != "https://example.com/d" {
		t.Errorf("eventURL = %q, want https://example.com/d", result.EventURL)
	}
	if result.Description != "dokploy event" {
		t.Errorf("description = %q, want %q", result.Description, "dokploy event")
	}
	if result.Normalized["app"] != "api" {
		t.Errorf("normalized[app] = %v, want api", result.Normalized["app"])
	}
	if result.Normalized["event"] != "deploy.failed" || result.Normalized["source"] != "dokploy" {
		t.Errorf("normalized = %v, want the known fields alongside data", result.Normalized)
	}
}

func signGeneric(key string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func genericAuth(t *testing.T, secret, signature string, body []byte) bool {
	t.Helper()
	header := settings.WebhookDefaultSecretHeader(settings.ProviderGeneric)
	value := secret
	provider := settings.Provider{
		WebhookSettings: settings.WebhookSettings{SecretHeader: &header, SecretValue: &value},
	}
	request := &http.Request{Header: http.Header{}}
	request.Header.Set(header, signature)
	return generic.Authenticate(request, body, provider)
}

func TestGenericSignatureVerification(t *testing.T) {
	const secret = "webhook-secret"
	body := []byte(`{"message":"signed","data":{"n":1}}`)

	if !genericAuth(t, secret, signGeneric(secret, body), body) {
		t.Error("a correct sha256= signature must authenticate")
	}
	if genericAuth(t, secret, signGeneric("other-secret", body), body) {
		t.Error("a signature under another secret must be rejected")
	}
	if genericAuth(t, secret, signGeneric(secret, []byte(`{"message":"tampered"}`)), body) {
		t.Error("a signature over a different body must be rejected")
	}
	if genericAuth(t, secret, "", body) {
		t.Error("a missing signature must be rejected")
	}
	if !genericAuth(t, "", signGeneric("unused", body), body) {
		t.Error("an unconfigured secret accepts the call")
	}
}
