package settings

import "testing"

var headerExceptions = map[ProviderType]string{
	ProviderUptimeKuma: "x-webhook-secret",
	ProviderDocuSeal:   "x-docuseal-secret",
	ProviderDokploy:    "x-dokploy-secret",
	ProviderGit:        "x-hub-signature-256",
	ProviderVikunja:    "x-vikunja-signature",
	ProviderSablier:    "x-sablier-signature",
	ProviderPerception: "x-perception-secret",
	ProviderOpus:       "x-opus-signature",
	ProviderJournal:    "x-journal-signature-256",
}

func TestTheActionCanDeriveEveryWebhookHeader(t *testing.T) {
	for _, providerType := range WebhookProviderTypes() {
		derived := "x-" + string(providerType) + "-signature-256"
		actual := WebhookDefaultSecretHeader(providerType)

		if expected, ok := headerExceptions[providerType]; ok {
			if actual != expected {
				t.Errorf("%s: header is %q, the exception list says %q", providerType, actual, expected)
			}
			continue
		}
		if actual != derived {
			t.Errorf("%s: header is %q, but the action derives %q", providerType, actual, derived)
		}
	}
}

func TestEveryWebhookProviderHasADefaultHeader(t *testing.T) {
	for _, providerType := range WebhookProviderTypes() {
		if WebhookDefaultSecretHeader(providerType) == "" {
			t.Errorf("%s receives webhooks but has no default secret header", providerType)
		}
	}
}
