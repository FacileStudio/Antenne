// Package settings owns Antenne's configuration: the provider list, the delivery
// target list, the system event switches and the pool settings.
//
// The JSON these types marshal to is the dashboard's contract and is unchanged
// from the pre-Go service, so a config exported from the old dashboard imports
// here untouched.
package settings

import (
	"encoding/json"
	"fmt"
)

// SystemProviderID is the reserved provider id Antenne's own events carry. A
// delivery target must list it explicitly to receive them; there is no
// catch-all.
const SystemProviderID = "__antenne_system__"

// Redacted is what a secret reads as over the wire. A PUT echoing it back means
// "keep the stored value".
const Redacted = "••••••••"

// ProviderType names a source Antenne watches or receives from.
type ProviderType string

// Provider types. The first twelve are webhook receivers; the last three are
// pollers Antenne drives itself.
const (
	ProviderUptimeKuma ProviderType = "uptime-kuma"
	ProviderDocuSeal   ProviderType = "docuseal"
	ProviderDokploy    ProviderType = "dokploy"
	ProviderGit        ProviderType = "git"
	ProviderVikunja    ProviderType = "vikunja"
	ProviderArdoise    ProviderType = "ardoise"
	ProviderSablier    ProviderType = "sablier"
	ProviderPerception ProviderType = "perception"
	ProviderOpus       ProviderType = "opus"
	ProviderVision     ProviderType = "vision"
	ProviderPlume      ProviderType = "plume"
	ProviderNuage      ProviderType = "nuage"
	ProviderIMAP       ProviderType = "imap"
	ProviderWebsite    ProviderType = "website"
	ProviderRSS        ProviderType = "rss"
)

// WebhookProviderTypes are the provider types that receive HTTP webhooks.
var WebhookProviderTypes = []ProviderType{
	ProviderUptimeKuma, ProviderDocuSeal, ProviderDokploy, ProviderGit,
	ProviderVikunja, ProviderArdoise, ProviderSablier, ProviderPerception,
	ProviderOpus, ProviderVision, ProviderPlume, ProviderNuage,
}

// ProviderTypes are every provider type, webhook receivers first.
var ProviderTypes = append(append([]ProviderType{}, WebhookProviderTypes...),
	ProviderIMAP, ProviderWebsite, ProviderRSS)

// IsWebhook reports whether this provider type receives webhooks.
func (t ProviderType) IsWebhook() bool {
	for _, candidate := range WebhookProviderTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// IsValid reports whether the type is one Antenne knows.
func (t ProviderType) IsValid() bool {
	for _, candidate := range ProviderTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// Provider is one configured source.
//
// It is a single flat struct rather than an interface because the settings
// document is edited as a whole by the dashboard, and MarshalJSON emits only
// the fields that belong to the type — so the wire shape stays exactly as
// narrow as the TypeScript union it replaces.
type Provider struct {
	ID      string       `json:"id"`
	Type    ProviderType `json:"type"`
	Name    string       `json:"name"`
	Enabled bool         `json:"enabled"`
	Tags    []string     `json:"tags"`

	// website
	URL      string `json:"url,omitempty"`
	Interval int    `json:"interval,omitempty"`

	// rss
	FeedURL        string `json:"feedUrl,omitempty"`
	IncludeContent bool   `json:"includeContent,omitempty"`

	// imap
	Host             string  `json:"host,omitempty"`
	Port             int     `json:"port,omitempty"`
	Secure           bool    `json:"secure,omitempty"`
	Username         string  `json:"username,omitempty"`
	Password         *string `json:"password,omitempty"`
	IncludeEmailBody bool    `json:"includeEmailBody,omitempty"`

	// imap + rss
	PollInterval int `json:"pollInterval,omitempty"`

	// webhook
	Path         string  `json:"path,omitempty"`
	SecretHeader *string `json:"secretHeader,omitempty"`
	SecretValue  *string `json:"secretValue,omitempty"`
}

// MarshalJSON writes only the fields that belong to the provider's type.
func (p Provider) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"id":      p.ID,
		"type":    p.Type,
		"name":    p.Name,
		"enabled": p.Enabled,
		"tags":    emptySlice(p.Tags),
	}

	switch {
	case p.Type == ProviderWebsite:
		out["url"] = p.URL
		out["interval"] = p.Interval
	case p.Type == ProviderRSS:
		out["feedUrl"] = p.FeedURL
		out["pollInterval"] = p.PollInterval
		out["includeContent"] = p.IncludeContent
	case p.Type == ProviderIMAP:
		out["host"] = p.Host
		out["port"] = p.Port
		out["secure"] = p.Secure
		out["username"] = p.Username
		out["password"] = p.Password
		out["pollInterval"] = p.PollInterval
		out["includeEmailBody"] = p.IncludeEmailBody
	default:
		out["path"] = p.Path
		out["secretHeader"] = p.SecretHeader
		out["secretValue"] = p.SecretValue
	}

	return json.Marshal(out)
}

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
var DeliveryTargetTypes = []DeliveryTargetType{
	TargetMatrix, TargetDiscord, TargetWebhook, TargetSMTP, TargetNtfy, TargetPerception,
}

// IsValid reports whether the type is one Antenne can deliver to.
func (t DeliveryTargetType) IsValid() bool {
	for _, candidate := range DeliveryTargetTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// DeliveryTarget is one configured delivery channel.
type DeliveryTarget struct {
	ID           string             `json:"id"`
	Type         DeliveryTargetType `json:"type"`
	Name         string             `json:"name"`
	Enabled      bool               `json:"enabled"`
	ProviderIDs  []string           `json:"providerIds"`
	ProviderTags []string           `json:"providerTags"`

	// matrix
	Homeserver   string   `json:"homeserver,omitempty"`
	AccessToken  string   `json:"accessToken,omitempty"`
	RefreshToken *string  `json:"refreshToken,omitempty"`
	UserID       string   `json:"userId,omitempty"`
	DeviceID     *string  `json:"deviceId,omitempty"`
	RoomIDs      []string `json:"roomIds,omitempty"`

	// discord
	WebhookURL string `json:"webhookUrl,omitempty"`

	// webhook + perception
	URL               string  `json:"url,omitempty"`
	SecretValue       *string `json:"secretValue,omitempty"`
	IncludeRawPayload bool    `json:"includeRawPayload,omitempty"`
	Workspace         string  `json:"workspace,omitempty"`

	// smtp
	Host          string   `json:"host,omitempty"`
	Port          int      `json:"port,omitempty"`
	Secure        bool     `json:"secure,omitempty"`
	Username      string   `json:"username,omitempty"`
	FromAddress   string   `json:"fromAddress,omitempty"`
	ToAddresses   []string `json:"toAddresses,omitempty"`
	SubjectPrefix string   `json:"subjectPrefix,omitempty"`

	// ntfy
	ServerURL string   `json:"serverUrl,omitempty"`
	Topic     string   `json:"topic,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	ClickURL  *string  `json:"clickUrl,omitempty"`

	// matrix + smtp
	Password *string `json:"password,omitempty"`
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

// SystemEvents are the switches for Antenne's own events.
type SystemEvents struct {
	Startup          bool `json:"startup"`
	ProviderEnabled  bool `json:"providerEnabled"`
	ProviderDisabled bool `json:"providerDisabled"`
	ProviderAdded    bool `json:"providerAdded"`
	ProviderRemoved  bool `json:"providerRemoved"`
	DeliveryFailure  bool `json:"deliveryFailure"`
}

// SystemSettings wraps the system event switches.
type SystemSettings struct {
	Events SystemEvents `json:"events"`
}

// BusSettings configures the Antenne Pool.
type BusSettings struct {
	Secret      string `json:"secret,omitempty"`
	LogActivity *bool  `json:"logActivity,omitempty"`
}

// LogActivityEnabled reports whether pool traffic reaches the activity feed.
func (p *BusSettings) LogActivityEnabled() bool {
	return p == nil || p.LogActivity == nil || *p.LogActivity
}

// DeliveryConfig is the ordered delivery target list.
type DeliveryConfig struct {
	Items []DeliveryTarget `json:"items"`
}

// ProvidersConfig is the ordered provider list plus the listen port.
type ProvidersConfig struct {
	Port  int        `json:"port"`
	Items []Provider `json:"items"`
}

// AppSettings is the whole configuration document.
type AppSettings struct {
	Delivery  DeliveryConfig  `json:"delivery"`
	System    SystemSettings  `json:"system"`
	Timeout   int             `json:"timeout"`
	Providers ProvidersConfig `json:"providers"`
	Bus       *BusSettings    `json:"bus,omitempty"`
}

// FindProvider returns the provider with this id, if any.
func (s AppSettings) FindProvider(id string) (Provider, bool) {
	for _, provider := range s.Providers.Items {
		if provider.ID == id {
			return provider, true
		}
	}
	return Provider{}, false
}

// FindTarget returns the delivery target with this id, if any.
func (s AppSettings) FindTarget(id string) (DeliveryTarget, bool) {
	for _, target := range s.Delivery.Items {
		if target.ID == id {
			return target, true
		}
	}
	return DeliveryTarget{}, false
}

// ProviderTags returns the tags of the named provider, or nothing for the
// system pseudo-provider.
func (s AppSettings) ProviderTags(providerID string) []string {
	if providerID == SystemProviderID {
		return nil
	}
	if provider, ok := s.FindProvider(providerID); ok {
		return provider.Tags
	}
	return nil
}

// RuntimeStatus is what the dashboard shows about the running process.
type RuntimeStatus struct {
	ServerPort             int    `json:"serverPort"`
	SettingsPath           string `json:"settingsPath"`
	UsingPersistedSettings bool   `json:"usingPersistedSettings"`
	DashboardBuilt         bool   `json:"dashboardBuilt"`
	LastReloadedAt         string `json:"lastReloadedAt"`
}

// ProviderLabel is the human name for a provider type.
func ProviderLabel(providerType ProviderType) string {
	switch providerType {
	case ProviderWebsite:
		return "Website"
	case ProviderIMAP:
		return "IMAP"
	case ProviderRSS:
		return "RSS"
	case ProviderDocuSeal:
		return "DocuSeal"
	case ProviderDokploy:
		return "Dokploy"
	case ProviderGit:
		return "Git"
	case ProviderVikunja:
		return "Vikunja"
	case ProviderArdoise:
		return "Ardoise"
	case ProviderSablier:
		return "Sablier"
	case ProviderPerception:
		return "Perception"
	case ProviderOpus:
		return "Opus"
	case ProviderVision:
		return "Vision"
	case ProviderPlume:
		return "Plume"
	case ProviderNuage:
		return "Nuage"
	case ProviderUptimeKuma:
		return "Uptime Kuma"
	}
	return string(providerType)
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

// WebhookDefaultPath is the path a new webhook provider of this type listens on.
func WebhookDefaultPath(providerType ProviderType) string {
	if providerType.IsWebhook() {
		return "/webhook/" + string(providerType)
	}
	return ""
}

// WebhookDefaultSecretHeader is the header a new webhook provider reads its
// secret or signature from.
func WebhookDefaultSecretHeader(providerType ProviderType) string {
	switch providerType {
	case ProviderUptimeKuma:
		return "x-webhook-secret"
	case ProviderDocuSeal:
		return "x-docuseal-secret"
	case ProviderGit:
		return "x-hub-signature-256"
	case ProviderVikunja:
		return "x-vikunja-signature"
	case ProviderArdoise:
		return "x-ardoise-signature-256"
	case ProviderSablier:
		return "x-sablier-signature"
	case ProviderPerception:
		return "x-perception-secret"
	case ProviderOpus:
		return "x-opus-signature"
	case ProviderVision:
		return "x-vision-signature-256"
	case ProviderPlume:
		return "x-plume-signature-256"
	case ProviderNuage:
		return "x-nuage-signature-256"
	case ProviderDokploy:
		return "x-dokploy-secret"
	}
	return ""
}

// emptySlice keeps a nil slice from marshalling as null. The dashboard iterates
// these without a null guard, exactly as it did against the old service.
func emptySlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func describeCollision(path string) error {
	return fmt.Errorf("provider path collision detected for `%s`", path)
}
