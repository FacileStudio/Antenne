package settings

import "encoding/json"

// DeliveryTargetType names a delivery channel.
type DeliveryTargetType string

// Delivery target types.
const (
	TargetMatrix     DeliveryTargetType = "matrix"
	TargetDiscord    DeliveryTargetType = "discord"
	TargetWebhook    DeliveryTargetType = "webhook"
	TargetSMTP       DeliveryTargetType = "smtp"
	TargetNtfy       DeliveryTargetType = "ntfy"
	TargetPerception DeliveryTargetType = "perception"
)

// DeliveryTargetTypes are every delivery target type.
//
// It is a function rather than a slice so that a caller who reorders or trims
// the result cannot change what every other caller sees.
func DeliveryTargetTypes() []DeliveryTargetType {
	return []DeliveryTargetType{
		TargetMatrix, TargetDiscord, TargetWebhook, TargetSMTP, TargetNtfy, TargetPerception,
	}
}

// IsValid reports whether the type is one Antenne can deliver to.
func (t DeliveryTargetType) IsValid() bool {
	for _, candidate := range DeliveryTargetTypes() {
		if candidate == t {
			return true
		}
	}
	return false
}

// MatrixSettings addresses a homeserver and the rooms an alert lands in.
//
// This and its siblings are embedded anonymously and carry no json tag, so
// encoding/json flattens them: the wire shape is the same flat object the
// dashboard has always sent, and the grouping exists only in Go.
type MatrixSettings struct {
	Homeserver   string   `json:"homeserver,omitempty"`
	RefreshToken *string  `json:"refreshToken,omitempty"`
	UserID       string   `json:"userId,omitempty"`
	DeviceID     *string  `json:"deviceId,omitempty"`
	RoomIDs      []string `json:"roomIds,omitempty"`
}

// DiscordSettings holds the incoming-webhook URL Discord issues per channel.
type DiscordSettings struct {
	WebhookURL string `json:"webhookUrl,omitempty"`
}

// HTTPSettings configures the two targets Antenne POSTs JSON to: a generic
// webhook and Perception, which only adds a workspace.
type HTTPSettings struct {
	URL               string  `json:"url,omitempty"`
	SecretValue       *string `json:"secretValue,omitempty"`
	IncludeRawPayload bool    `json:"includeRawPayload,omitempty"`
	Workspace         string  `json:"workspace,omitempty"`
}

// SMTPSettings configures the mail relay and the envelope Antenne sends with.
type SMTPSettings struct {
	Host          string   `json:"host,omitempty"`
	Port          int      `json:"port,omitempty"`
	Secure        bool     `json:"secure,omitempty"`
	Username      string   `json:"username,omitempty"`
	FromAddress   string   `json:"fromAddress,omitempty"`
	ToAddresses   []string `json:"toAddresses,omitempty"`
	SubjectPrefix string   `json:"subjectPrefix,omitempty"`
}

// NtfySettings configures the topic an alert is published to.
type NtfySettings struct {
	ServerURL string   `json:"serverUrl,omitempty"`
	Topic     string   `json:"topic,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	ClickURL  *string  `json:"clickUrl,omitempty"`
}

// DeliveryTarget is one configured delivery channel.
type DeliveryTarget struct {
	ID           string             `json:"id"`
	Type         DeliveryTargetType `json:"type"`
	Name         string             `json:"name"`
	Enabled      bool               `json:"enabled"`
	ProviderIDs  []string           `json:"providerIds"`
	ProviderTags []string           `json:"providerTags"`

	// AccessToken is Matrix's and ntfy's; Password is Matrix's and SMTP's. Both
	// stay above the per-channel groups because one field backs two channels and
	// duplicating either would give the same wire key two homes.
	AccessToken string  `json:"accessToken,omitempty"`
	Password    *string `json:"password,omitempty"`

	MatrixSettings
	DiscordSettings
	HTTPSettings
	SMTPSettings
	NtfySettings
}

// MarshalJSON writes only the fields that belong to the target's type.
func (t DeliveryTarget) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"id":           t.ID,
		"type":         t.Type,
		"name":         t.Name,
		"enabled":      t.Enabled,
		"providerIds":  emptySlice(t.ProviderIDs),
		"providerTags": emptySlice(t.ProviderTags),
	}

	switch t.Type {
	case TargetMatrix:
		out["homeserver"] = t.Homeserver
		out["accessToken"] = t.AccessToken
		out["refreshToken"] = t.RefreshToken
		out["userId"] = t.UserID
		out["password"] = t.Password
		out["deviceId"] = t.DeviceID
		out["roomIds"] = emptySlice(t.RoomIDs)
	case TargetDiscord:
		out["webhookUrl"] = t.WebhookURL
	case TargetSMTP:
		out["host"] = t.Host
		out["port"] = t.Port
		out["secure"] = t.Secure
		out["username"] = t.Username
		out["password"] = t.Password
		out["fromAddress"] = t.FromAddress
		out["toAddresses"] = emptySlice(t.ToAddresses)
		out["subjectPrefix"] = t.SubjectPrefix
	case TargetNtfy:
		out["serverUrl"] = t.ServerURL
		out["topic"] = t.Topic
		// ntfy's token is nullable on the wire where Matrix's is a plain
		// string; an empty value has to read back as null, not "".
		out["accessToken"] = nilIfEmpty(t.AccessToken)
		out["priority"] = t.Priority
		out["tags"] = emptySlice(t.Tags)
		out["clickUrl"] = t.ClickURL
	case TargetPerception:
		out["url"] = t.URL
		out["workspace"] = t.Workspace
		out["secretValue"] = t.SecretValue
		out["includeRawPayload"] = t.IncludeRawPayload
	default:
		out["url"] = t.URL
		out["secretValue"] = t.SecretValue
		out["includeRawPayload"] = t.IncludeRawPayload
	}

	return json.Marshal(out)
}

// TargetLabel is the human name for a delivery target type.
func TargetLabel(targetType DeliveryTargetType) string {
	switch targetType {
	case TargetMatrix:
		return "Matrix"
	case TargetDiscord:
		return "Discord webhook"
	case TargetSMTP:
		return "SMTP"
	case TargetNtfy:
		return "ntfy"
	case TargetPerception:
		return "Perception"
	}
	return "HTTP webhook"
}
