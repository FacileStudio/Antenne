package settings

import "encoding/json"

// ProviderType names a source Antenne watches or receives from.
type ProviderType string

// Provider types. The first thirteen are webhook receivers; the last three are
// pollers Antenne drives itself.
const (
	ProviderFilet      ProviderType = "filet"
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
//
// It is a function rather than a slice so that a caller who reorders or trims
// the result cannot change what every other caller sees.
func WebhookProviderTypes() []ProviderType {
	return []ProviderType{
		ProviderUptimeKuma, ProviderDocuSeal, ProviderDokploy, ProviderGit,
		ProviderVikunja, ProviderArdoise, ProviderSablier, ProviderPerception,
		ProviderOpus, ProviderVision, ProviderPlume, ProviderNuage, ProviderFilet,
	}
}

// ProviderTypes are every provider type, webhook receivers first. The order is
// the dashboard's picker order, so appending here appends there.
func ProviderTypes() []ProviderType {
	return append(WebhookProviderTypes(), ProviderIMAP, ProviderWebsite, ProviderRSS)
}

// IsWebhook reports whether this provider type receives webhooks.
func (t ProviderType) IsWebhook() bool {
	for _, candidate := range WebhookProviderTypes() {
		if candidate == t {
			return true
		}
	}
	return false
}

// IsValid reports whether the type is one Antenne knows.
func (t ProviderType) IsValid() bool {
	for _, candidate := range ProviderTypes() {
		if candidate == t {
			return true
		}
	}
	return false
}

// WebsiteSettings configures an HTTP uptime check.
//
// This and its siblings are embedded anonymously and carry no json tag, so
// encoding/json flattens them: the wire shape is the same flat object the
// dashboard has always sent, and the grouping exists only in Go.
type WebsiteSettings struct {
	URL      string `json:"url,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

// RSSSettings configures a feed poller.
type RSSSettings struct {
	FeedURL        string `json:"feedUrl,omitempty"`
	IncludeContent bool   `json:"includeContent,omitempty"`
}

// IMAPSettings configures a mailbox poller.
type IMAPSettings struct {
	Host             string  `json:"host,omitempty"`
	Port             int     `json:"port,omitempty"`
	Secure           bool    `json:"secure,omitempty"`
	Username         string  `json:"username,omitempty"`
	Password         *string `json:"password,omitempty"`
	IncludeEmailBody bool    `json:"includeEmailBody,omitempty"`
}

// WebhookSettings configures an inbound receiver: where it listens and how a
// caller proves it is entitled to post there.
type WebhookSettings struct {
	Path         string  `json:"path,omitempty"`
	SecretHeader *string `json:"secretHeader,omitempty"`
	SecretValue  *string `json:"secretValue,omitempty"`
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

	// PollInterval is shared by the IMAP and RSS pollers, so it sits above both.
	PollInterval int `json:"pollInterval,omitempty"`

	WebsiteSettings
	RSSSettings
	IMAPSettings
	WebhookSettings
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
	case ProviderFilet:
		return "filet"
	case ProviderUptimeKuma:
		return "Uptime Kuma"
	}
	return string(providerType)
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
	case ProviderFilet:
		return "x-filet-signature-256"
	case ProviderDokploy:
		return "x-dokploy-secret"
	}
	return ""
}
