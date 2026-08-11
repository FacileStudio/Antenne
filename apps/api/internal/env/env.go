// Package env loads the Antenne API configuration from the process environment
// once, at startup, so a misconfigured service refuses to boot instead of
// running half-configured.
package env

import (
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/porte"
	troncenv "github.com/FacileStudio/tronc/env"
)

// DefaultPort is the port Antenne binds when PORT is unset. It is the port the
// published webhook URLs and the pool clients already point at, so it does not
// follow troncenv.Core's 8080 default.
const DefaultPort = 9090

// AuthPrefix is where porte's routes are mounted. They sit under /api with the
// rest of the dashboard API rather than at the root: unlike the webhook intake
// and the pool endpoints, nothing third-party has them registered, and the
// /api subtree is the one that answers an unknown path with JSON.
const AuthPrefix = "/api"

// Config is the Antenne server configuration.
//
// It embeds troncenv.Core so the shared fields carry the same names as in every
// other Facile API. Core is filled field by field rather than through
// troncenv.LoadCore only because Antenne keeps its own PORT default; DATABASE_URL
// is still required, and Required is called for it explicitly.
type Config struct {
	troncenv.Core

	AdminUsername string
	AdminPassword string
	PoolSecret    string
	ClientDir     string
	Timeout       int

	// PublicURL is the origin Antenne is reached on. It exists because porte
	// derives the Secure attribute and the __Host- cookie prefix from its own
	// configuration rather than from a proxy header, and it reads that from
	// the OIDC redirect and success URLs alone. An instance running on the
	// password login only would therefore have no https anywhere in its
	// configuration and would issue an unprefixed, non-Secure session cookie
	// over TLS. This is the value that keeps that from depending on whether
	// SSO happens to be switched on.
	PublicURL string

	// Porte is the suite's auth contract: OIDC_ISSUER enables single sign-on,
	// SSO_ONLY makes it the only way in. Both may be unset, in which case the
	// bootstrap password below is the only credential.
	Porte porte.Config

	// HeartbeatURL is an external dead man's switch — healthchecks.io, Dead Man's
	// Snitch, Grafana OnCall. Empty turns the feature off: it is the only failure
	// report that survives this process dying, but it needs a third party, so it
	// cannot be a default.
	HeartbeatURL      string
	HeartbeatInterval time.Duration
}

// Load reads and validates the configuration. Every error it returns is a
// reason not to start.
func Load() (Config, error) {
	port, err := troncenv.Int("PORT", DefaultPort)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("env: PORT must be a valid TCP port, got %d", port)
	}

	databaseURL, err := troncenv.Required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	timeout, err := troncenv.Int("TIMEOUT", 10)
	if err != nil {
		return Config{}, err
	}
	if timeout < 1 {
		return Config{}, fmt.Errorf("env: TIMEOUT must be a positive number of seconds, got %d", timeout)
	}

	// The receiver's own resolution bounds anything useful here: Grafana OnCall
	// polls every 13 minutes and wants 15 or more, healthchecks.io has no floor.
	// Five minutes suits the latter; an OnCall user raises it.
	heartbeatInterval, err := troncenv.Duration("HEARTBEAT_INTERVAL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	if heartbeatInterval < time.Minute {
		return Config{}, fmt.Errorf("env: HEARTBEAT_INTERVAL must be at least 1m, got %s", heartbeatInterval)
	}

	// Every field troncenv.Core grows has to be repeated here, because this
	// literal exists to skip troncenv.LoadCore. A field left out keeps its
	// zero value, which for TrustedProxies means TRUSTED_PROXIES is ignored
	// while the deployment panel shows it set.
	trustedProxies, err := troncenv.TrustedProxies()
	if err != nil {
		return Config{}, err
	}
	cdnProxies, cdnHeader := troncenv.CDN()

	cfg := Config{
		Core: troncenv.Core{
			TrustedProxies:     trustedProxies,
			CDNProxies:         cdnProxies,
			CDNHeader:          cdnHeader,
			AppEnv:             troncenv.ParseEnvironment(troncenv.String("APP_ENV", string(troncenv.Development))),
			Port:               port,
			LogLevel:           troncenv.String("LOG_LEVEL", "info"),
			DatabaseURL:        databaseURL,
			CORSAllowedOrigins: corsOrigins(),
			JournalURL:         troncenv.String("JOURNAL_URL", ""),
			JournalToken:       troncenv.String("JOURNAL_TOKEN", ""),
		},
		AdminUsername: troncenv.String("ANTENNE_ADMIN_USERNAME", "admin"),
		AdminPassword: troncenv.String("ANTENNE_ADMIN_PASSWORD", ""),
		PoolSecret:    troncenv.String("ANTENNE_SECRET", ""),
		ClientDir:     troncenv.String("CLIENT_DIR", "./client"),
		Timeout:       timeout,
		PublicURL:     strings.TrimSuffix(troncenv.String("PUBLIC_URL", ""), "/"),

		HeartbeatURL:      troncenv.String("HEARTBEAT_URL", ""),
		HeartbeatInterval: heartbeatInterval,
	}

	ssoOnly, err := troncenv.Bool("SSO_ONLY", false)
	if err != nil {
		return Config{}, err
	}

	cfg.Porte = porte.Config{
		Issuer:       troncenv.String("OIDC_ISSUER", ""),
		ClientID:     troncenv.String("OIDC_CLIENT_ID", ""),
		ClientSecret: troncenv.String("OIDC_CLIENT_SECRET", ""),
		RedirectURL:  troncenv.String("OIDC_REDIRECT_URL", cfg.defaultRedirectURL()),
		SuccessURL:   troncenv.String("OIDC_SUCCESS_URL", cfg.PublicURL+"/"),
		FailureURL:   troncenv.String("OIDC_FAILURE_URL", ""),
		ClaimsScope:  troncenv.String("OIDC_CLAIMS_SCOPE", ""),
		SSOOnly:      ssoOnly,
	}
	if err := cfg.Porte.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, cfg.validate()
}

// defaultRedirectURL is the callback Authentik is configured with. Deriving it
// from PublicURL keeps the two from drifting: the value registered on the
// provider is the one path porte cannot discover for itself, and an app that
// spells it a second time in the deployment panel is an app where changing the
// domain breaks login on the next restart rather than at the next login.
func (c Config) defaultRedirectURL() string {
	if c.PublicURL == "" {
		return ""
	}
	return c.PublicURL + AuthPrefix + porte.RouteCallback
}

// corsOrigins drops schemeless entries. troncenv.CORSOrigins also reads DOMAIN,
// which deployments set to a bare hostname for Traefik; a schemeless value never
// matches an Origin header and would only confuse anything that reads the list
// back as a URL.
func corsOrigins() []string {
	origins := troncenv.CORSOrigins()
	kept := make([]string, 0, len(origins))
	for _, origin := range origins {
		if len(origin) > 7 && (origin[:7] == "http://" || origin[:8] == "https://") {
			kept = append(kept, origin)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// validate rejects a configuration that leaves no way in, or a way in that
// cannot be secured.
//
// There is no open mode any more. Antenne used to serve a synthetic admin to
// every caller when no password was set, which made a local run's dashboard and
// its whole API reachable by anyone who could open the port — and made any auth
// assertion written against such an instance worthless, because it passed for
// reasons that had nothing to do with the credential. A missing credential is a
// refusal to boot in every environment now, not a warning in production.
func (c Config) validate() error {
	if c.AdminPassword == "" && !c.Porte.Enabled() {
		return fmt.Errorf("env: set ANTENNE_ADMIN_PASSWORD or OIDC_ISSUER — with neither there is no way to sign in, and Antenne no longer serves an open dashboard")
	}
	if c.Porte.SSOOnly && !c.Porte.Enabled() {
		return fmt.Errorf("env: SSO_ONLY=true with no OIDC_ISSUER leaves no way to sign in")
	}
	if c.IsProduction() && !strings.HasPrefix(strings.ToLower(c.PublicURL), "https://") {
		// porte reads the Secure attribute and the __Host- cookie prefix off
		// this value, not off X-Forwarded-Proto, so that a proxy which stops
		// sending the header downgrades nothing. Left unset in production the
		// session cookie ships without either.
		return fmt.Errorf("env: APP_ENV=production requires PUBLIC_URL to be the https origin Antenne is reached on, otherwise the session cookie is issued without Secure")
	}
	return nil
}
