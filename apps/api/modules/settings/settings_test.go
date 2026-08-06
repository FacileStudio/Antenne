package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func baseline() AppSettings {
	secret := "s3cr3t"
	current := Defaults(9090, 10)
	current.Providers.Items = []Provider{
		{
			ID: "p1", Type: ProviderDokploy, Name: "Dokploy", Enabled: true,
			Tags: []string{"infra"}, Path: "/webhook/dokploy", SecretValue: &secret,
		},
	}
	current.Delivery.Items = []DeliveryTarget{
		{
			ID: "t1", Type: TargetMatrix, Name: "Matrix", Enabled: true,
			ProviderIDs: []string{"p1"}, ProviderTags: []string{},
			Homeserver: "https://matrix.org", AccessToken: "syt_live", RoomIDs: []string{"!room:matrix.org"},
		},
	}
	return current
}

// A dashboard round-trip must not erase a secret it was never shown. This is
// the whole point of the redaction marker, and the failure mode is silent.
func TestRedactedSecretsSurviveARoundTrip(t *testing.T) {
	current := baseline()

	redacted := Redact(current)
	if got := *redacted.Providers.Items[0].SecretValue; got != Redacted {
		t.Fatalf("provider secret was not redacted: %q", got)
	}
	if redacted.Delivery.Items[0].AccessToken != Redacted {
		t.Fatalf("matrix token was not redacted: %q", redacted.Delivery.Items[0].AccessToken)
	}

	sentBack, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	merged := MergeSecrets(sentBack, current)
	next, err := Normalize(merged, current)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if got := *next.Providers.Items[0].SecretValue; got != "s3cr3t" {
		t.Errorf("provider secret after round-trip = %q, want the stored value", got)
	}
	if got := next.Delivery.Items[0].AccessToken; got != "syt_live" {
		t.Errorf("matrix token after round-trip = %q, want the stored value", got)
	}
}

// A real edit still has to win over the stored value, or a secret could never
// be changed from the dashboard.
func TestANewSecretReplacesTheStoredOne(t *testing.T) {
	current := baseline()

	edited := []byte(`{"providers":{"port":9090,"items":[
		{"id":"p1","type":"dokploy","name":"Dokploy","enabled":true,"tags":["infra"],
		 "path":"/webhook/dokploy","secretValue":"rotated"}]},
		"delivery":{"items":[]},"timeout":10}`)

	next, err := Normalize(MergeSecrets(edited, current), current)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got := *next.Providers.Items[0].SecretValue; got != "rotated" {
		t.Errorf("provider secret = %q, want the edited value", got)
	}
}

func TestNormalizeRejectsTwoEnabledProvidersOnOnePath(t *testing.T) {
	current := Defaults(9090, 10)

	document := []byte(`{"providers":{"port":9090,"items":[
		{"id":"a","type":"dokploy","name":"A","enabled":true,"path":"/webhook/dokploy"},
		{"id":"b","type":"dokploy","name":"B","enabled":true,"path":"/webhook/dokploy"}]}}`)

	_, err := Normalize(document, current)
	if err == nil {
		t.Fatal("a path collision was accepted")
	}
	if !strings.Contains(err.Error(), "/webhook/dokploy") {
		t.Errorf("the error does not name the path: %v", err)
	}
}

func TestNormalizeRejectsAWebhookPathThatIsNotRooted(t *testing.T) {
	document := []byte(`{"providers":{"port":9090,"items":[
		{"id":"a","type":"dokploy","name":"A","enabled":true,"path":"webhook/dokploy"}]}}`)

	if _, err := Normalize(document, Defaults(9090, 10)); err == nil {
		t.Fatal("an unrooted webhook path was accepted")
	}
}

// Only the fields belonging to a type may reach the wire; the dashboard branches
// on their presence.
func TestMarshalEmitsOnlyTheFieldsOfTheType(t *testing.T) {
	encoded, err := json.Marshal(Provider{ID: "p", Type: ProviderWebsite, Name: "Site", URL: "https://x", Interval: 60})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, leaked := decoded["path"]; leaked {
		t.Error("a website provider carried the webhook `path` field")
	}
	if _, missing := decoded["url"]; !missing {
		t.Error("a website provider is missing `url`")
	}
}

// A client whose JSON encoder escapes non-ASCII sends the marker as the ASCII
// escape sequence for U+2022, eight times. Decoded it is the same string, so the
// merge has to find it — a byte-level scan of the request body does not, and
// skipping the merge writes the marker itself over every secret in the document.
func TestRedactedSecretsSurviveAnEscapingEncoder(t *testing.T) {
	current := baseline()

	escaped := []byte(`{"timeout":10,
	  "providers":{"port":9090,"items":[
	    {"id":"p1","type":"dokploy","name":"Dokploy","enabled":true,"tags":["infra"],
	     "path":"/webhook/dokploy","secretHeader":"x-dokploy-secret",
	     "secretValue":"\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"}]},
	  "delivery":{"items":[
	    {"id":"t1","type":"matrix","name":"Matrix","enabled":true,
	     "providerIds":["p1"],"providerTags":[],"homeserver":"https://matrix.org",
	     "accessToken":"\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022",
	     "roomIds":["!room:matrix.org"]}]}}`)

	// The fixture must carry the escape sequence, not the character: a literal
	// marker here would pass against the very bug this pins.
	if strings.Contains(string(escaped), Redacted) {
		t.Fatal("the fixture holds the literal marker, so it does not exercise an escaping encoder")
	}
	if !strings.Contains(string(escaped), `\u2022`) {
		t.Fatal("the fixture is not escaped at all")
	}

	next, err := Normalize(MergeSecrets(escaped, current), current)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if got := *next.Providers.Items[0].SecretValue; got != "s3cr3t" {
		t.Errorf("provider secret = %q, want the stored value", got)
	}
	if got := next.Delivery.Items[0].AccessToken; got != "syt_live" {
		t.Errorf("matrix token = %q, want the stored value", got)
	}
}

// Second line of defence: even with MergeSecrets bypassed entirely, the marker
// must never reach storage as a credential. It is not a value anyone means.
func TestNormalizeNeverStoresTheMarkerAsASecret(t *testing.T) {
	current := baseline()

	document := []byte(`{"timeout":10,
	  "providers":{"port":9090,"items":[
	    {"id":"p1","type":"dokploy","name":"Dokploy","enabled":true,"tags":["infra"],
	     "path":"/webhook/dokploy","secretValue":"` + Redacted + `"}]},
	  "delivery":{"items":[
	    {"id":"t1","type":"matrix","name":"Matrix","enabled":true,
	     "providerIds":["p1"],"providerTags":[],"homeserver":"https://matrix.org",
	     "accessToken":"` + Redacted + `","roomIds":["!room:matrix.org"]}]}}`)

	// Deliberately no MergeSecrets call.
	next, err := Normalize(document, current)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if got := *next.Providers.Items[0].SecretValue; got == Redacted {
		t.Error("the redaction marker was stored as the provider secret")
	} else if got != "s3cr3t" {
		t.Errorf("provider secret = %q, want the stored value kept", got)
	}
	if got := next.Delivery.Items[0].AccessToken; got == Redacted {
		t.Error("the redaction marker was stored as the matrix token")
	} else if got != "syt_live" {
		t.Errorf("matrix token = %q, want the stored value kept", got)
	}
}
