package settings

import (
	"encoding/json"
	"strconv"
	"strings"
)

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
