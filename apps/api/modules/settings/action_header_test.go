package settings

import "testing"

// The GitHub action at the repository root derives its signature header from
// the webhook URL as `x-<provider>-signature-256`, because that is what this
// package hands a newly created provider. A provider whose default departs from
// that pattern is unreachable by the action unless the caller overrides
// `header:`, and nothing else in the tree would say so — hence this list.
//
// Adding a provider here is the decision: either name it as an exception, or
// give it the pattern header and let the action find it on its own.
var headerExceptions = map[ProviderType]string{
	ProviderUptimeKuma: "x-webhook-secret",
	ProviderDocuSeal:   "x-docuseal-secret",
	ProviderDokploy:    "x-dokploy-secret",
	ProviderGit:        "x-hub-signature-256",
	ProviderVikunja:    "x-vikunja-signature",
	ProviderSablier:    "x-sablier-signature",
	ProviderPerception: "x-perception-secret",
	ProviderOpus:       "x-opus-signature",
}

func TestTheActionCanDeriveEveryWebhookHeader(t *testing.T) {
	for _, providerType := range WebhookProviderTypes {
		derived := "x-" + string(providerType) + "-signature-256"
		actual := WebhookDefaultSecretHeader(providerType)

		if expected, ok := headerExceptions[providerType]; ok {
			if actual != expected {
				t.Errorf("%s: header is %q, the exception list says %q", providerType, actual, expected)
			}
			continue
		}
		if actual != derived {
			t.Errorf("%s: header is %q, but the action derives %q — give it the pattern header "+
				"or add it to headerExceptions", providerType, actual, derived)
		}
	}
}

func TestEveryWebhookProviderHasADefaultHeader(t *testing.T) {
	for _, providerType := range WebhookProviderTypes {
		if WebhookDefaultSecretHeader(providerType) == "" {
			t.Errorf("%s receives webhooks but has no default secret header, so a new one "+
				"is created unauthenticated", providerType)
		}
	}
}
