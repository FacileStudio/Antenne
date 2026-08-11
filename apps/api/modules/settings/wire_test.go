package settings

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// The dashboard and every exported config file speak this flat object. Grouping
// the Go fields into embedded structs must not move a single key, so the key set
// per type is pinned here rather than left to review.

func keysOf(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func fullProvider(providerType ProviderType) Provider {
	secret := "s"
	return Provider{
		ID: "p", Type: providerType, Name: "n", Enabled: true, Tags: []string{"t"},
		PollInterval:    60,
		WebsiteSettings: WebsiteSettings{URL: "https://x", Interval: 30},
		RSSSettings:     RSSSettings{FeedURL: "https://f", IncludeContent: true},
		IMAPSettings: IMAPSettings{
			Host: "h", Port: 993, Secure: true, Username: "u",
			Password: &secret, IncludeEmailBody: true,
		},
		WebhookSettings: WebhookSettings{Path: "/webhook/x", SecretHeader: &secret, SecretValue: &secret},
	}
}

func fullTarget(targetType DeliveryTargetType) DeliveryTarget {
	secret := "s"
	priority := 4
	return DeliveryTarget{
		ID: "d", Type: targetType, Name: "n", Enabled: true,
		ProviderIDs: []string{"p"}, ProviderTags: []string{"t"},
		AccessToken: "tok", Password: &secret,
		MatrixSettings: MatrixSettings{
			Homeserver: "https://m", RefreshToken: &secret, UserID: "@a:m",
			DeviceID: &secret, RoomIDs: []string{"!r:m"},
		},
		DiscordSettings: DiscordSettings{WebhookURL: "https://d"},
		HTTPSettings:    HTTPSettings{URL: "https://u", SecretValue: &secret, IncludeRawPayload: true, Workspace: "w"},
		SMTPSettings: SMTPSettings{
			Host: "h", Port: 587, Secure: true, Username: "u",
			FromAddress: "a@b", ToAddresses: []string{"c@d"}, SubjectPrefix: "[x]",
		},
		NtfySettings: NtfySettings{
			ServerURL: "https://n", Topic: "alerts", Priority: &priority,
			Tags: []string{"warning"}, ClickURL: &secret,
		},
	}
}

func TestProviderWireKeysAreUnchanged(t *testing.T) {
	cases := map[ProviderType]string{
		ProviderWebsite: "enabled,id,interval,name,tags,type,url",
		ProviderRSS:     "enabled,feedUrl,id,includeContent,name,pollInterval,tags,type",
		ProviderIMAP: "enabled,host,id,includeEmailBody,name,password,pollInterval," +
			"port,secure,tags,type,username",
		ProviderDokploy: "enabled,id,name,path,secretHeader,secretValue,tags,type",
	}

	for providerType, want := range cases {
		if got := keysOf(t, fullProvider(providerType)); got != want {
			t.Errorf("%s provider keys = %s, want %s", providerType, got, want)
		}
	}
}

func TestDeliveryTargetWireKeysAreUnchanged(t *testing.T) {
	core := "enabled,id,name,providerIds,providerTags,type,"
	cases := map[DeliveryTargetType]string{
		TargetMatrix:     core + "accessToken,deviceId,homeserver,password,refreshToken,roomIds,userId",
		TargetDiscord:    core + "webhookUrl",
		TargetSMTP:       core + "fromAddress,host,password,port,secure,subjectPrefix,toAddresses,username",
		TargetNtfy:       core + "accessToken,clickUrl,priority,serverUrl,tags,topic",
		TargetPerception: core + "includeRawPayload,secretValue,url,workspace",
		TargetWebhook:    core + "includeRawPayload,secretValue,url",
	}

	for targetType, want := range cases {
		want = sortedCSV(want)
		if got := keysOf(t, fullTarget(targetType)); got != want {
			t.Errorf("%s target keys = %s, want %s", targetType, got, want)
		}
	}
}

func sortedCSV(value string) string {
	parts := strings.Split(value, ",")
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// The embedded groups carry no json tag, so encoding/json must flatten them on
// the way in too — a tag slipping onto one would nest that group and silently
// drop every one of its values from an imported config.
func TestAFlatDocumentStillFillsTheGroupedFields(t *testing.T) {
	document := []byte(`{"id":"p","type":"imap","name":"Mailbox","enabled":true,
		"tags":["ops"],"host":"imap.example","port":993,"secure":true,
		"username":"antenne","password":"pw","pollInterval":120,
		"includeEmailBody":true,"url":"https://x","interval":30,
		"feedUrl":"https://f","includeContent":true,
		"path":"/webhook/x","secretHeader":"x-h","secretValue":"sv"}`)

	var provider Provider
	if err := json.Unmarshal(document, &provider); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if provider.Host != "imap.example" || provider.Port != 993 || !provider.Secure {
		t.Errorf("imap group not filled: %+v", provider.IMAPSettings)
	}
	if provider.Password == nil || *provider.Password != "pw" {
		t.Error("imap password not filled")
	}
	if provider.URL != "https://x" || provider.Interval != 30 {
		t.Errorf("website group not filled: %+v", provider.WebsiteSettings)
	}
	if provider.FeedURL != "https://f" || !provider.IncludeContent {
		t.Errorf("rss group not filled: %+v", provider.RSSSettings)
	}
	if provider.Path != "/webhook/x" || provider.SecretValue == nil {
		t.Errorf("webhook group not filled: %+v", provider.WebhookSettings)
	}
	if provider.PollInterval != 120 {
		t.Errorf("pollInterval = %d, want 120", provider.PollInterval)
	}
}

// Same guarantee for the delivery target, whose groups overlap on two shared
// credentials that live above them.
func TestAFlatTargetDocumentStillFillsTheGroupedFields(t *testing.T) {
	document := []byte(`{"id":"d","type":"smtp","name":"Relay","enabled":true,
		"providerIds":["p"],"providerTags":[],"accessToken":"tok","password":"pw",
		"homeserver":"https://m","userId":"@a:m","roomIds":["!r:m"],
		"webhookUrl":"https://d","url":"https://u","workspace":"w",
		"includeRawPayload":true,"host":"smtp.example","port":587,"secure":true,
		"username":"u","fromAddress":"a@b","toAddresses":["c@d"],
		"subjectPrefix":"[x]","serverUrl":"https://n","topic":"alerts",
		"priority":4,"tags":["warning"],"clickUrl":"https://c"}`)

	var target DeliveryTarget
	if err := json.Unmarshal(document, &target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if target.AccessToken != "tok" || target.Password == nil || *target.Password != "pw" {
		t.Error("the shared credentials were not filled")
	}
	if target.Homeserver != "https://m" || len(target.RoomIDs) != 1 {
		t.Errorf("matrix group not filled: %+v", target.MatrixSettings)
	}
	if target.WebhookURL != "https://d" {
		t.Errorf("discord group not filled: %+v", target.DiscordSettings)
	}
	if target.URL != "https://u" || target.Workspace != "w" || !target.IncludeRawPayload {
		t.Errorf("http group not filled: %+v", target.HTTPSettings)
	}
	if target.Host != "smtp.example" || target.Port != 587 || target.FromAddress != "a@b" {
		t.Errorf("smtp group not filled: %+v", target.SMTPSettings)
	}
	if target.ServerURL != "https://n" || target.Topic != "alerts" || target.Priority == nil {
		t.Errorf("ntfy group not filled: %+v", target.NtfySettings)
	}
}
