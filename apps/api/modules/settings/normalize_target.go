package settings

import (
	"encoding/json"
	"fmt"
)

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

	seed, err := seedTarget(incoming, existing)
	if err != nil {
		return DeliveryTarget{}, err
	}

	if name := stringField(incoming, "name", ""); name != "" {
		seed.Name = name
	}
	applyBoolField(incoming, "enabled", &seed.Enabled)
	seed.ProviderIDs = stringList(incoming["providerIds"])
	seed.ProviderTags = stringList(incoming["providerTags"])

	applyTargetType(incoming, &seed)
	return seed, nil
}

// seedTarget picks the target the incoming fields are layered onto: the stored
// one when the id is known and the type still matches, otherwise a fresh
// default. Switching a target's type therefore drops the credentials of the old
// one instead of leaving a Matrix access token on an SMTP target.
func seedTarget(incoming map[string]json.RawMessage, existing map[string]DeliveryTarget) (DeliveryTarget, error) {
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
	return seed, nil
}

// applyTargetType layers the fields only this target type owns. The default
// arm covers the plain HTTP webhook, so an unrecognised type still normalizes
// into something the delivery loop can refuse cleanly rather than a zero value.
func applyTargetType(incoming map[string]json.RawMessage, seed *DeliveryTarget) {
	switch seed.Type {
	case TargetMatrix:
		applyMatrixFields(incoming, seed)
	case TargetDiscord:
		applySecret(incoming, "webhookUrl", &seed.WebhookURL)
	case TargetSMTP:
		applySMTPFields(incoming, seed)
	case TargetNtfy:
		applyNtfyFields(incoming, seed)
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
}

func applyMatrixFields(incoming map[string]json.RawMessage, seed *DeliveryTarget) {
	applyString(incoming, "homeserver", &seed.Homeserver)
	applySecret(incoming, "accessToken", &seed.AccessToken)
	applyOptionalSecret(incoming, "refreshToken", &seed.RefreshToken)
	applyString(incoming, "userId", &seed.UserID)
	applyOptionalSecret(incoming, "password", &seed.Password)
	applyOptionalString(incoming, "deviceId", &seed.DeviceID)
	if _, ok := incoming["roomIds"]; ok {
		seed.RoomIDs = stringList(incoming["roomIds"])
	}
}

func applySMTPFields(incoming map[string]json.RawMessage, seed *DeliveryTarget) {
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
}

func applyNtfyFields(incoming map[string]json.RawMessage, seed *DeliveryTarget) {
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
}
