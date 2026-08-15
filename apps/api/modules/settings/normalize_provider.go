package settings

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalizeProviders(document map[string]json.RawMessage, fallback AppSettings) ([]Provider, int, error) {
	items, port, err := applyProvidersBlock(document["providers"], fallback)
	if err != nil {
		return nil, 0, err
	}
	if port < 1 || port > 65535 {
		return nil, 0, fmt.Errorf("provider port must be a valid TCP port")
	}
	if err := checkWebhookPaths(items); err != nil {
		return nil, 0, err
	}
	return items, port, nil
}

// applyProvidersBlock reads the incoming providers block over the current one.
// An absent block, an absent port and an absent item list each keep what is
// already stored, which is what lets the dashboard PUT a partial document.
func applyProvidersBlock(raw json.RawMessage, fallback AppSettings) ([]Provider, int, error) {
	port := fallback.Providers.Port
	items := fallback.Providers.Items
	if len(raw) == 0 {
		return items, port, nil
	}

	var incoming struct {
		Port  *json.RawMessage   `json:"port"`
		Items *[]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return nil, 0, fmt.Errorf("invalid providers block: %w", err)
	}
	if incoming.Port != nil {
		port = parseInt(*incoming.Port, port)
	}
	if incoming.Items == nil {
		return items, port, nil
	}

	existing := map[string]Provider{}
	for _, provider := range fallback.Providers.Items {
		existing[provider.ID] = provider
	}
	items = make([]Provider, 0, len(*incoming.Items))
	for _, rawProvider := range *incoming.Items {
		provider, err := normalizeProvider(rawProvider, existing)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, provider)
	}
	return items, port, nil
}

// checkWebhookPaths rejects two enabled webhook providers sharing one path.
// The intake dispatches on the path alone, so a collision would silently hand
// one provider's deliveries to the other's secret and formatter.
func checkWebhookPaths(items []Provider) error {
	seen := map[string]bool{}
	for _, provider := range items {
		if !provider.Type.IsWebhook() || !provider.Enabled {
			continue
		}
		if seen[provider.Path] {
			return describeCollision(provider.Path)
		}
		seen[provider.Path] = true
	}
	return nil
}

func normalizeProvider(raw json.RawMessage, existing map[string]Provider) (Provider, error) {
	var incoming map[string]json.RawMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return Provider{}, fmt.Errorf("invalid provider: %w", err)
	}

	seed := seedProvider(incoming, existing)
	if name := stringField(incoming, "name", ""); name != "" {
		seed.Name = name
	}
	applyBoolField(incoming, "enabled", &seed.Enabled)
	if _, ok := incoming["tags"]; ok {
		seed.Tags = stringList(incoming["tags"])
	}

	if err := applyProviderType(incoming, &seed); err != nil {
		return Provider{}, err
	}

	if seed.Tags == nil {
		seed.Tags = []string{}
	}
	return seed, nil
}

// seedProvider picks the provider the incoming fields are layered onto: the
// stored one when the id is known and the type still matches, otherwise a fresh
// default. Changing a provider's type therefore resets its type-specific fields
// rather than carrying a website's interval into an IMAP inbox.
func seedProvider(incoming map[string]json.RawMessage, existing map[string]Provider) Provider {
	providerType := ProviderType(stringField(incoming, "type", ""))
	id := stringField(incoming, "id", "")

	seed, hasSeed := existing[id]
	if !hasSeed || (providerType != "" && seed.Type != providerType) {
		if !providerType.IsValid() {
			providerType = ProviderWebsite
		}
		seed = DefaultProvider(providerType)
	}
	if id != "" {
		seed.ID = id
	}
	return seed
}

func applyProviderType(incoming map[string]json.RawMessage, seed *Provider) error {
	switch seed.Type {
	case ProviderWebsite:
		return applyWebsiteFields(incoming, seed)
	case ProviderRSS:
		applyRSSFields(incoming, seed)
		return nil
	case ProviderIMAP:
		applyIMAPFields(incoming, seed)
		return nil
	default:
		return applyWebhookFields(incoming, seed)
	}
}

func applyWebsiteFields(incoming map[string]json.RawMessage, seed *Provider) error {
	if url := stringField(incoming, "url", ""); url != "" {
		seed.URL = url
	}
	if seed.URL == "" {
		return fmt.Errorf("website providers must define a URL")
	}
	seed.Interval = positiveIntField(incoming, "interval", seed.Interval)
	return nil
}

func applyRSSFields(incoming map[string]json.RawMessage, seed *Provider) {
	if feedURL := stringField(incoming, "feedUrl", ""); feedURL != "" {
		seed.FeedURL = feedURL
	}
	seed.PollInterval = positiveIntField(incoming, "pollInterval", seed.PollInterval)
	applyBoolField(incoming, "includeContent", &seed.IncludeContent)
}

func applyIMAPFields(incoming map[string]json.RawMessage, seed *Provider) {
	if host := stringField(incoming, "host", ""); host != "" {
		seed.Host = host
	}
	seed.Port = positiveIntField(incoming, "port", seed.Port)
	applyBoolField(incoming, "secure", &seed.Secure)
	if username := stringField(incoming, "username", ""); username != "" {
		seed.Username = username
	}
	applyOptionalSecret(incoming, "password", &seed.Password)
	seed.PollInterval = positiveIntField(incoming, "pollInterval", seed.PollInterval)
	applyBoolField(incoming, "includeEmailBody", &seed.IncludeEmailBody)
}

func applyWebhookFields(incoming map[string]json.RawMessage, seed *Provider) error {
	if path := stringField(incoming, "path", ""); path != "" {
		seed.Path = path
	}
	if !strings.HasPrefix(seed.Path, "/") {
		return fmt.Errorf("webhook path for provider `%s` must start with `/`", seed.Name)
	}
	if header := stringField(incoming, "secretHeader", ""); header != "" {
		seed.SecretHeader = &header
	}
	applyOptionalSecret(incoming, "secretValue", &seed.SecretValue)
	return nil
}
