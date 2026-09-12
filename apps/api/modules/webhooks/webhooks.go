// Package webhooks receives third-party events and turns them into Antenne alerts.
//
// One file per provider under providers.go's registry; each provider owns its
// payload shape, its message rendering and its envelope mapping, and nothing
// else in Antenne knows what a DocuSeal submission or a Vikunja task looks like.
package webhooks

import (
	"crypto/hmac"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Context tells a provider which of its configured instances is handling the
// request. The same provider type can be configured several times on different
// paths with different secrets.
type Context struct {
	ProviderID   string
	ProviderName string
	ProviderType settings.ProviderType
	Path         string
}

// Result is what a provider made of a payload.
//
// An empty MessageHTML means the event is understood but not worth an alert;
// the intake still answers 200 and logs it, which is what keeps a chatty source
// from turning every heartbeat into a Matrix message.
type Result struct {
	Description       string
	MessageHTML       string
	EventType         string
	EventTitle        string
	EventURL          string
	OccurredAt        string
	Actor             *envelope.Actor
	Subject           *envelope.Subject
	Normalized        map[string]any
	ContainsPII       bool
	RawPayloadAllowed bool
}

// Definition is one provider type's behaviour.
type Definition struct {
	Type  settings.ProviderType
	Label string

	// Authenticate is optional. Without it, the intake compares the configured
	// secret against the configured header in constant time. Providers that
	// sign the body implement it instead.
	Authenticate func(r *http.Request, body []byte, provider settings.Provider) bool

	// Handle parses the raw body. Returning an error rejects the request as
	// malformed; returning a Result with no MessageHTML accepts it silently.
	Handle func(body []byte, ctx Context) (Result, error)
}

// Lookup resolves the definition a provider type is handled by, and reports
// whether Antenne knows that type at all. Callers must treat the second result
// as authoritative: a provider row can name a type this build does not carry.
func Lookup(providerType settings.ProviderType) (Definition, bool) {
	definition, known := registry[providerType]
	return definition, known
}

// registry is built once rather than per call: Lookup sits on the webhook
// intake path, which is the one path here a third party can drive.
var registry = map[settings.ProviderType]Definition{
	settings.ProviderSonde:      sonde,
	settings.ProviderUptimeKuma: uptimeKuma,
	settings.ProviderDocuSeal:   docuSeal,
	settings.ProviderDokploy:    dokploy,
	settings.ProviderGit:        git,
	settings.ProviderVikunja:    vikunja,
	settings.ProviderArdoise:    ardoise,
	settings.ProviderSablier:    sablier,
	settings.ProviderPerception: perception,
	settings.ProviderOpus:       opus,
	settings.ProviderVision:     vision,
	settings.ProviderPlume:      plume,
	settings.ProviderNuage:      nuage,
	settings.ProviderFilet:      filet,
	settings.ProviderJournal:    journal,
	settings.ProviderGeneric:    generic,
}

// Authenticate applies the provider's own check, or the default header
// comparison when it has none.
func Authenticate(definition Definition, provider settings.Provider, r *http.Request, body []byte) bool {
	if definition.Authenticate != nil {
		return definition.Authenticate(r, body, provider)
	}

	if provider.SecretValue == nil || *provider.SecretValue == "" {
		return true
	}
	if provider.SecretHeader == nil || *provider.SecretHeader == "" {
		return false
	}

	presented := r.Header.Get(*provider.SecretHeader)
	return hmac.Equal([]byte(presented), []byte(*provider.SecretValue))
}

// ReadBody reads the request body under a cap, so an oversized POST cannot be
// used to exhaust memory before authentication has even run.
func ReadBody(r *http.Request, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, limit))
}

// signedWith verifies an HMAC-SHA256 body signature under the provider's
// secret, with the given prefix. An unconfigured secret accepts everything,
// which is what makes a provider usable before its secret is exchanged.
func signedWith(prefix string) func(*http.Request, []byte, settings.Provider) bool {
	return func(r *http.Request, body []byte, provider settings.Provider) bool {
		if provider.SecretValue == nil || *provider.SecretValue == "" {
			return true
		}
		header := ""
		if provider.SecretHeader != nil {
			header = *provider.SecretHeader
		}
		if header == "" {
			return false
		}
		return envelope.VerifyHexSignature(*provider.SecretValue, prefix, r.Header.Get(header), body)
	}
}

func decode(body []byte, into any) error {
	return json.Unmarshal(body, into)
}

func escape(value string) string { return envelope.EscapeHTML(value) }

func join(parts []string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "<br/>")
}

func str(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
