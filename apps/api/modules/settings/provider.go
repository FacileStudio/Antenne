package settings

import "encoding/json"

type ProviderType string

const (
	ProviderSonde      ProviderType = "sonde"
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
	ProviderJournal    ProviderType = "journal"
	ProviderGeneric    ProviderType = "generic"
)

func WebhookProviderTypes() []ProviderType {
	return []ProviderType{
		ProviderSonde, ProviderUptimeKuma, ProviderDocuSeal, ProviderDokploy, ProviderGit,
		ProviderVikunja, ProviderArdoise, ProviderSablier, ProviderPerception,
		ProviderOpus, ProviderVision, ProviderPlume, ProviderNuage, ProviderFilet,
		ProviderJournal, ProviderGeneric,
	}
}

func ProviderTypes() []ProviderType {
	return append(WebhookProviderTypes(), ProviderIMAP, ProviderWebsite, ProviderRSS)
}

func (t ProviderType) IsWebhook() bool {
	for _, candidate := range WebhookProviderTypes() {
		if candidate == t {
			return true
		}
	}
	return false
}

func (t ProviderType) IsValid() bool {
	for _, candidate := range ProviderTypes() {
		if candidate == t {
			return true
		}
	}
	return false
}

type WebsiteSettings struct {
	URL      string `json:"url,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

type RSSSettings struct {
	FeedURL        string `json:"feedUrl,omitempty"`
	IncludeContent bool   `json:"includeContent,omitempty"`
}

type IMAPSettings struct {
	Host             string  `json:"host,omitempty"`
	Port             int     `json:"port,omitempty"`
	Secure           bool    `json:"secure,omitempty"`
	Username         string  `json:"username,omitempty"`
	Password         *string `json:"password,omitempty"`
	IncludeEmailBody bool    `json:"includeEmailBody,omitempty"`
}

type WebhookSettings struct {
	Path         string  `json:"path,omitempty"`
	SecretHeader *string `json:"secretHeader,omitempty"`
	SecretValue  *string `json:"secretValue,omitempty"`
}

type Provider struct {
	ID           string       `json:"id"`
	Type         ProviderType `json:"type"`
	Name         string       `json:"name"`
	Enabled      bool         `json:"enabled"`
	Tags         []string     `json:"tags"`
	PollInterval int          `json:"pollInterval,omitempty"`
	WebsiteSettings
	RSSSettings
	IMAPSettings
	WebhookSettings
}

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
	case ProviderJournal:
		return "Journal"
	case ProviderGeneric:
		return "Generic"
	case ProviderSonde:
		return "Sonde"
	case ProviderUptimeKuma:
		return "Uptime Kuma"
	}
	return string(providerType)
}

func WebhookDefaultPath(providerType ProviderType) string {
	if providerType.IsWebhook() {
		return "/webhook/" + string(providerType)
	}
	return ""
}

func WebhookDefaultSecretHeader(providerType ProviderType) string {
	switch providerType {
	case ProviderSonde:
		return "x-sonde-signature-256"
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
	case ProviderJournal:
		return "x-journal-signature-256"
	case ProviderGeneric:
		return "x-generic-signature-256"
	case ProviderDokploy:
		return "x-dokploy-secret"
	}
	return ""
}
