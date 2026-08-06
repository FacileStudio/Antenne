package settings

import (
	"encoding/json"
	"strings"
)

// Redact replaces every stored secret with the redaction marker. The dashboard
// only ever holds redacted settings, so a screenshot, a browser cache or an
// exported config never carries a live credential.
func Redact(source AppSettings) AppSettings {
	redacted := cloneSettings(source)

	for index := range redacted.Providers.Items {
		provider := &redacted.Providers.Items[index]
		switch provider.Type {
		case ProviderIMAP:
			provider.Password = redactPointer(provider.Password)
		case ProviderWebsite, ProviderRSS:
		default:
			provider.SecretValue = redactPointer(provider.SecretValue)
		}
	}

	for index := range redacted.Delivery.Items {
		target := &redacted.Delivery.Items[index]
		switch target.Type {
		case TargetMatrix:
			target.AccessToken = redactValue(target.AccessToken)
			target.RefreshToken = redactPointer(target.RefreshToken)
			target.Password = redactPointer(target.Password)
		case TargetDiscord:
			target.WebhookURL = redactValue(target.WebhookURL)
		case TargetSMTP:
			target.Password = redactPointer(target.Password)
		case TargetNtfy:
			target.AccessToken = redactValue(target.AccessToken)
		case TargetWebhook, TargetPerception:
			target.SecretValue = redactPointer(target.SecretValue)
		}
	}

	if redacted.Pool != nil && redacted.Pool.Secret != "" {
		redacted.Pool.Secret = Redacted
	}

	return redacted
}

// MergeSecrets rewrites every redaction marker in an incoming document back to
// the stored value, so a dashboard save that never saw the real secret cannot
// erase it.
//
// It works on the raw JSON rather than on a parsed document because
// normalization happens afterwards: a marker has to become the real secret
// before any "blank clears the field" rule sees it.
func MergeSecrets(raw json.RawMessage, current AppSettings) json.RawMessage {
	if !strings.Contains(string(raw), Redacted) {
		return raw
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		return raw
	}

	if merged, ok := mergeProviderSecrets(document["providers"], current); ok {
		document["providers"] = merged
	}
	if merged, ok := mergeTargetSecrets(document["delivery"], current); ok {
		document["delivery"] = merged
	}
	if merged, ok := mergePoolSecret(document["pool"], current); ok {
		document["pool"] = merged
	}

	rewritten, err := json.Marshal(document)
	if err != nil {
		return raw
	}
	return rewritten
}

func mergeProviderSecrets(raw json.RawMessage, current AppSettings) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}

	var block map[string]json.RawMessage
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, false
	}

	items, ok := decodeItems(block["items"])
	if !ok {
		return nil, false
	}

	for index, item := range items {
		id := itemID(item)
		stored, found := current.FindProvider(id)
		if !found {
			continue
		}
		switch stored.Type {
		case ProviderIMAP:
			restorePointer(item, "password", stored.Password)
		case ProviderWebsite, ProviderRSS:
		default:
			restorePointer(item, "secretValue", stored.SecretValue)
		}
		items[index] = item
	}

	return reencodeItems(block, items)
}

func mergeTargetSecrets(raw json.RawMessage, current AppSettings) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}

	var block map[string]json.RawMessage
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, false
	}

	items, ok := decodeItems(block["items"])
	if !ok {
		return nil, false
	}

	for index, item := range items {
		stored, found := current.FindTarget(itemID(item))
		if !found {
			continue
		}
		switch stored.Type {
		case TargetMatrix:
			restoreValue(item, "accessToken", stored.AccessToken)
			restorePointer(item, "refreshToken", stored.RefreshToken)
			restorePointer(item, "password", stored.Password)
		case TargetDiscord:
			restoreValue(item, "webhookUrl", stored.WebhookURL)
		case TargetSMTP:
			restorePointer(item, "password", stored.Password)
		case TargetNtfy:
			restoreValue(item, "accessToken", stored.AccessToken)
		case TargetWebhook, TargetPerception:
			restorePointer(item, "secretValue", stored.SecretValue)
		}
		items[index] = item
	}

	return reencodeItems(block, items)
}

func mergePoolSecret(raw json.RawMessage, current AppSettings) (json.RawMessage, bool) {
	if len(raw) == 0 || current.Pool == nil || current.Pool.Secret == "" {
		return nil, false
	}

	var block map[string]json.RawMessage
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, false
	}
	restoreValue(block, "secret", current.Pool.Secret)

	rewritten, err := json.Marshal(block)
	if err != nil {
		return nil, false
	}
	return rewritten, true
}

func decodeItems(raw json.RawMessage) ([]map[string]json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func reencodeItems(block map[string]json.RawMessage, items []map[string]json.RawMessage) (json.RawMessage, bool) {
	encodedItems, err := json.Marshal(items)
	if err != nil {
		return nil, false
	}
	block["items"] = encodedItems

	encoded, err := json.Marshal(block)
	if err != nil {
		return nil, false
	}
	return encoded, true
}

func itemID(item map[string]json.RawMessage) string {
	var id string
	if err := json.Unmarshal(item["id"], &id); err != nil {
		return ""
	}
	return id
}

func restoreValue(item map[string]json.RawMessage, key, stored string) {
	if !isRedacted(item[key]) {
		return
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return
	}
	item[key] = encoded
}

func restorePointer(item map[string]json.RawMessage, key string, stored *string) {
	if !isRedacted(item[key]) {
		return
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return
	}
	item[key] = encoded
}

func isRedacted(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value == Redacted
}

func redactValue(value string) string {
	if value == "" {
		return ""
	}
	return Redacted
}

func redactPointer(value *string) *string {
	if value == nil || *value == "" {
		return value
	}
	marker := Redacted
	return &marker
}
