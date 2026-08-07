package settings

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Normalize turns whatever the dashboard PUT into a settings document Antenne can
// run, falling back to the previous value field by field.
//
// It is deliberately forgiving in the same places the TypeScript service was:
// an absent field keeps its old value, a blank string clears an optional one,
// and a field belonging to another provider type is ignored rather than
// rejected. What it does reject is a document that cannot run — a
// non-positive timeout, a webhook path that is not rooted, two enabled webhook
// providers on one path.
func Normalize(raw json.RawMessage, fallback AppSettings) (AppSettings, error) {
	var document map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &document); err != nil {
			return AppSettings{}, fmt.Errorf("invalid settings document: %w", err)
		}
	}

	next := AppSettings{}

	timeout := intField(document, "timeout", fallback.Timeout)
	if timeout < 1 {
		return AppSettings{}, fmt.Errorf("timeout must be a positive number")
	}
	next.Timeout = timeout

	system := fallback.System
	if rawSystem, ok := document["system"]; ok {
		var incoming struct {
			Events map[string]*bool `json:"events"`
		}
		if err := json.Unmarshal(rawSystem, &incoming); err == nil {
			events := &system.Events
			applyBool(incoming.Events, "startup", &events.Startup)
			applyBool(incoming.Events, "providerEnabled", &events.ProviderEnabled)
			applyBool(incoming.Events, "providerDisabled", &events.ProviderDisabled)
			applyBool(incoming.Events, "providerAdded", &events.ProviderAdded)
			applyBool(incoming.Events, "providerRemoved", &events.ProviderRemoved)
			applyBool(incoming.Events, "deliveryFailure", &events.DeliveryFailure)
		}
	}
	next.System = system

	providers, port, err := normalizeProviders(document, fallback)
	if err != nil {
		return AppSettings{}, err
	}
	next.Providers = ProvidersConfig{Port: port, Items: providers}

	targets, err := normalizeTargets(document, fallback)
	if err != nil {
		return AppSettings{}, err
	}
	next.Delivery = DeliveryConfig{Items: targets}

	next.Pool = normalizePool(document, fallback.Pool)

	return next, nil
}

func normalizeProviders(document map[string]json.RawMessage, fallback AppSettings) ([]Provider, int, error) {
	port := fallback.Providers.Port
	items := fallback.Providers.Items

	if rawProviders, ok := document["providers"]; ok {
		var incoming struct {
			Port  *json.RawMessage   `json:"port"`
			Items *[]json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(rawProviders, &incoming); err != nil {
			return nil, 0, fmt.Errorf("invalid providers block: %w", err)
		}
		if incoming.Port != nil {
			port = parseInt(*incoming.Port, port)
		}
		if incoming.Items != nil {
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
		}
	}

	if port < 1 || port > 65535 {
		return nil, 0, fmt.Errorf("provider port must be a valid TCP port")
	}

	seen := map[string]bool{}
	for _, provider := range items {
		if !provider.Type.IsWebhook() || !provider.Enabled {
			continue
		}
		if seen[provider.Path] {
			return nil, 0, describeCollision(provider.Path)
		}
		seen[provider.Path] = true
	}

	return items, port, nil
}

func normalizeProvider(raw json.RawMessage, existing map[string]Provider) (Provider, error) {
	var incoming map[string]json.RawMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return Provider{}, fmt.Errorf("invalid provider: %w", err)
	}

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

	if name := stringField(incoming, "name", ""); name != "" {
		seed.Name = name
	}
	applyBoolField(incoming, "enabled", &seed.Enabled)
	if _, ok := incoming["tags"]; ok {
		seed.Tags = stringList(incoming["tags"])
	}

	switch seed.Type {
	case ProviderWebsite:
		if url := stringField(incoming, "url", ""); url != "" {
			seed.URL = url
		}
		if seed.URL == "" {
			return Provider{}, fmt.Errorf("website providers must define a URL")
		}
		seed.Interval = positiveIntField(incoming, "interval", seed.Interval)
	case ProviderRSS:
		if feedURL := stringField(incoming, "feedUrl", ""); feedURL != "" {
			seed.FeedURL = feedURL
		}
		seed.PollInterval = positiveIntField(incoming, "pollInterval", seed.PollInterval)
		applyBoolField(incoming, "includeContent", &seed.IncludeContent)
	case ProviderIMAP:
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
	default:
		if path := stringField(incoming, "path", ""); path != "" {
			seed.Path = path
		}
		if !strings.HasPrefix(seed.Path, "/") {
			return Provider{}, fmt.Errorf("webhook path for provider `%s` must start with `/`", seed.Name)
		}
		if header := stringField(incoming, "secretHeader", ""); header != "" {
			seed.SecretHeader = &header
		}
		applyOptionalSecret(incoming, "secretValue", &seed.SecretValue)
	}

	if seed.Tags == nil {
		seed.Tags = []string{}
	}
	return seed, nil
}

func normalizeTargets(document map[string]json.RawMessage, fallback AppSettings) ([]DeliveryTarget, error) {
	rawDelivery, ok := document["delivery"]
	if !ok {
		return fallback.Delivery.Items, nil
	}

	var incoming struct {
		Items *[]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rawDelivery, &incoming); err != nil {
		return nil, fmt.Errorf("invalid delivery block: %w", err)
	}
	if incoming.Items == nil {
		return fallback.Delivery.Items, nil
	}

	existing := map[string]DeliveryTarget{}
	for _, target := range fallback.Delivery.Items {
		existing[target.ID] = target
	}

	targets := make([]DeliveryTarget, 0, len(*incoming.Items))
	for _, rawTarget := range *incoming.Items {
		target, err := normalizeTarget(rawTarget, existing)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func normalizeTarget(raw json.RawMessage, existing map[string]DeliveryTarget) (DeliveryTarget, error) {
	var incoming map[string]json.RawMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return DeliveryTarget{}, fmt.Errorf("invalid delivery target: %w", err)
	}

	targetType := DeliveryTargetType(stringField(incoming, "type", ""))
	if targetType != "" && !targetType.IsValid() {
		return DeliveryTarget{}, fmt.Errorf("unknown delivery target type `%s`", targetType)
	}

	id := stringField(incoming, "id", "")
	seed, hasSeed := existing[id]
	if !hasSeed || (targetType != "" && seed.Type != targetType) {
		if targetType == "" {
			targetType = TargetWebhook
		}
		seed = DefaultTarget(targetType)
	}
	if id != "" {
		seed.ID = id
	}

	if name := stringField(incoming, "name", ""); name != "" {
		seed.Name = name
	}
	applyBoolField(incoming, "enabled", &seed.Enabled)
	seed.ProviderIDs = stringList(incoming["providerIds"])
	seed.ProviderTags = stringList(incoming["providerTags"])

	switch seed.Type {
	case TargetMatrix:
		applyString(incoming, "homeserver", &seed.Homeserver)
		applySecret(incoming, "accessToken", &seed.AccessToken)
		applyOptionalSecret(incoming, "refreshToken", &seed.RefreshToken)
		applyString(incoming, "userId", &seed.UserID)
		applyOptionalSecret(incoming, "password", &seed.Password)
		applyOptionalString(incoming, "deviceId", &seed.DeviceID)
		if _, ok := incoming["roomIds"]; ok {
			seed.RoomIDs = stringList(incoming["roomIds"])
		}
	case TargetDiscord:
		applySecret(incoming, "webhookUrl", &seed.WebhookURL)
	case TargetSMTP:
		applyString(incoming, "host", &seed.Host)
		seed.Port = positiveIntField(incoming, "port", seed.Port)
		applyBoolField(incoming, "secure", &seed.Secure)
		applyString(incoming, "username", &seed.Username)
		applyOptionalSecret(incoming, "password", &seed.Password)
		applyString(incoming, "fromAddress", &seed.FromAddress)
		if _, ok := incoming["toAddresses"]; ok {
			seed.ToAddresses = stringList(incoming["toAddresses"])
		}
		applyString(incoming, "subjectPrefix", &seed.SubjectPrefix)
	case TargetNtfy:
		applyString(incoming, "serverUrl", &seed.ServerURL)
		applyString(incoming, "topic", &seed.Topic)
		applySecret(incoming, "accessToken", &seed.AccessToken)
		if rawPriority, ok := incoming["priority"]; ok {
			seed.Priority = optionalPositiveInt(rawPriority, seed.Priority)
		}
		if _, ok := incoming["tags"]; ok {
			seed.Tags = stringList(incoming["tags"])
		}
		applyOptionalString(incoming, "clickUrl", &seed.ClickURL)
	case TargetPerception:
		applyString(incoming, "url", &seed.URL)
		applyString(incoming, "workspace", &seed.Workspace)
		applyOptionalSecret(incoming, "secretValue", &seed.SecretValue)
		applyBoolField(incoming, "includeRawPayload", &seed.IncludeRawPayload)
	default:
		applyString(incoming, "url", &seed.URL)
		applyOptionalSecret(incoming, "secretValue", &seed.SecretValue)
		applyBoolField(incoming, "includeRawPayload", &seed.IncludeRawPayload)
	}

	return seed, nil
}

func normalizePool(document map[string]json.RawMessage, fallback *PoolSettings) *PoolSettings {
	pool := &PoolSettings{}
	if fallback != nil {
		copied := *fallback
		pool = &copied
	}

	rawPool, ok := document["pool"]
	if !ok {
		return pool
	}

	var incoming struct {
		Secret      *string `json:"secret"`
		LogActivity *bool   `json:"logActivity"`
	}
	if err := json.Unmarshal(rawPool, &incoming); err != nil {
		return pool
	}
	if incoming.Secret != nil && strings.TrimSpace(*incoming.Secret) != Redacted {
		pool.Secret = strings.TrimSpace(*incoming.Secret)
	}
	if incoming.LogActivity != nil {
		pool.LogActivity = incoming.LogActivity
	}
	return pool
}

func stringField(document map[string]json.RawMessage, key, fallback string) string {
	raw, ok := document[key]
	if !ok {
		return fallback
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return fallback
	}
	return strings.TrimSpace(value)
}

// applySecret is applyString for a field that Redact rewrites. A value equal to
// the marker means the caller is echoing back what it was shown, so the stored
// value is kept — a second line of defence behind MergeSecrets, and the reason a
// client whose encoder mangles the marker can no longer erase a credential.
func applySecret(document map[string]json.RawMessage, key string, target *string) {
	if isRedactedField(document, key) {
		return
	}
	applyString(document, key, target)
}

// applyOptionalSecret is applyOptionalString with the same guard.
func applyOptionalSecret(document map[string]json.RawMessage, key string, target **string) {
	if isRedactedField(document, key) {
		return
	}
	applyOptionalString(document, key, target)
}

func isRedactedField(document map[string]json.RawMessage, key string) bool {
	raw, ok := document[key]
	if !ok {
		return false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value == Redacted
}

func applyString(document map[string]json.RawMessage, key string, target *string) {
	if _, ok := document[key]; !ok {
		return
	}
	*target = stringField(document, key, *target)
}

func applyOptionalString(document map[string]json.RawMessage, key string, target **string) {
	if _, ok := document[key]; !ok {
		return
	}
	*target = optionalString(document[key])
}

func optionalString(raw json.RawMessage) *string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func applyBool(values map[string]*bool, key string, target *bool) {
	if value, ok := values[key]; ok && value != nil {
		*target = *value
	}
}

func applyBoolField(document map[string]json.RawMessage, key string, target *bool) {
	raw, ok := document[key]
	if !ok {
		return
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		*target = value
	}
}

func intField(document map[string]json.RawMessage, key string, fallback int) int {
	raw, ok := document[key]
	if !ok {
		return fallback
	}
	return parseInt(raw, fallback)
}

func positiveIntField(document map[string]json.RawMessage, key string, fallback int) int {
	raw, ok := document[key]
	if !ok {
		return fallback
	}
	value := parseInt(raw, fallback)
	if value < 1 {
		return fallback
	}
	return value
}

func optionalPositiveInt(raw json.RawMessage, fallback *int) *int {
	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		value := int(asNumber)
		if value < 1 {
			return fallback
		}
		return &value
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if asString == "" {
			return fallback
		}
		if value, err := strconv.Atoi(strings.TrimSpace(asString)); err == nil && value >= 1 {
			return &value
		}
	}
	return fallback
}

// parseInt accepts both a JSON number and a numeric string, because the
// dashboard's number inputs hand back strings when the field is edited by hand.
func parseInt(raw json.RawMessage, fallback int) int {
	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return int(asNumber)
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if value, err := strconv.Atoi(strings.TrimSpace(asString)); err == nil {
			return value
		}
	}
	return fallback
}

func stringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		cleaned := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		return cleaned
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		cleaned := []string{}
		for _, value := range strings.Split(single, ",") {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		return cleaned
	}

	return []string{}
}
