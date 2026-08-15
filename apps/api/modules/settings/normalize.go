package settings

import (
	"encoding/json"
	"fmt"
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

	next.Bus = normalizeBus(document, fallback.Bus)

	return next, nil
}

func normalizeBus(document map[string]json.RawMessage, fallback *BusSettings) *BusSettings {
	pool := &BusSettings{}
	if fallback != nil {
		copied := *fallback
		pool = &copied
	}

	rawPool, ok := document["bus"]
	if !ok {
		rawPool, ok = document["pool"]
	}
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
