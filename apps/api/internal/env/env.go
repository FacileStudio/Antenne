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

	// JournalBrowserURL and JournalBrowserKey are the dashboard's half of
	// Journal, handed to the browser over GET /api/auth/config rather than baked
	// into the bundle: the client is adapter-static, so it has no environment to
	// read and rotating a baked key would be a rebuild. They are a separate pair
	// from JOURNAL_URL and JOURNAL_TOKEN because the server SDK talks to Journal
	// over the docker network, which no browser can resolve, and because the
	// browser key is public by design where the server token is not.
	JournalBrowserURL string
	JournalBrowserKey string

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
	core, err := loadCore()
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

	heartbeatInterval, err := loadHeartbeatInterval()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Core:          core,
		AdminUsername: troncenv.String("ANTENNE_ADMIN_USERNAME", "admin"),
		AdminPassword: troncenv.String("ANTENNE_ADMIN_PASSWORD", ""),
		PoolSecret:    troncenv.String("ANTENNE_SECRET", ""),
		ClientDir:     troncenv.String("CLIENT_DIR", "./client"),
		Timeout:       timeout,
		PublicURL:     strings.TrimSuffix(troncenv.String("PUBLIC_URL", ""), "/"),

		JournalBrowserURL: strings.TrimSuffix(troncenv.String("JOURNAL_BROWSER_URL", ""), "/"),
		JournalBrowserKey: troncenv.String("JOURNAL_BROWSER_KEY", ""),

		HeartbeatURL:      troncenv.String("HEARTBEAT_URL", ""),
		HeartbeatInterval: heartbeatInterval,
	}

	if cfg.Porte, err = loadPorte(cfg); err != nil {
		return Config{}, err
	}

	return cfg, cfg.validate()
}

// loadCore fills troncenv.Core field by field, which is what Antenne trades for
// keeping its own PORT default instead of calling troncenv.LoadCore. Every field
// Core grows has to be repeated here: one left out keeps its zero value, which
// for TrustedProxies means TRUSTED_PROXIES is ignored while the deployment panel
// shows it set.
func loadCore() (troncenv.Core, error) {
	port, err := troncenv.Int("PORT", DefaultPort)
	if err != nil {
		return troncenv.Core{}, err
	}
	if port < 1 || port > 65535 {
		return troncenv.Core{}, fmt.Errorf("env: PORT must be a valid TCP port, got %d", port)
	}

	databaseURL, err := troncenv.Required("DATABASE_URL")
	if err != nil {
		return troncenv.Core{}, err
	}

	trustedProxies, err := troncenv.TrustedProxies()
	if err != nil {
		return troncenv.Core{}, err
	}
	cdnProxies, cdnHeader := troncenv.CDN()

	return troncenv.Core{
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
	}, nil
}

// loadHeartbeatInterval bounds the interval below by the receiver's own
// resolution: Grafana OnCall polls every 13 minutes and wants 15 or more,
// healthchecks.io has no floor. Five minutes suits the latter; an OnCall user
// raises it.
func loadHeartbeatInterval() (time.Duration, error) {
	interval, err := troncenv.Duration("HEARTBEAT_INTERVAL", 5*time.Minute)
	if err != nil {
		return 0, err
	}
	if interval < time.Minute {
		return 0, fmt.Errorf("env: HEARTBEAT_INTERVAL must be at least 1m, got %s", interval)
	}
	return interval, nil
}

// loadPorte reads the auth half of the configuration. It takes the Config it
// belongs to because two of porte's URLs default off PublicURL, so this can only
// run once the rest of the configuration exists.
func loadPorte(c Config) (porte.Config, error) {
	ssoOnly, err := troncenv.Bool("SSO_ONLY", false)
	if err != nil {
		return porte.Config{}, err
	}

	cfg := porte.Config{
		Issuer:       troncenv.String("OIDC_ISSUER", ""),
		ClientID:     troncenv.String("OIDC_CLIENT_ID", ""),
		ClientSecret: troncenv.String("OIDC_CLIENT_SECRET", ""),
		RedirectURL:  troncenv.String("OIDC_REDIRECT_URL", c.defaultRedirectURL()),
		SuccessURL:   troncenv.String("OIDC_SUCCESS_URL", c.PublicURL+"/"),
		FailureURL:   troncenv.String("OIDC_FAILURE_URL", ""),
		ClaimsScope:  troncenv.String("OIDC_CLAIMS_SCOPE", ""),
		SSOOnly:      ssoOnly,
	}
	if err := cfg.Validate(); err != nil {
		return porte.Config{}, err
	}
	return cfg, nil
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
	if c.JournalBrowserURL != "" && !strings.HasSuffix(c.JournalBrowserURL, "/api") {
		// Journal's dashboard is the catch-all on its own domain, so a base URL
		// without the /api suffix answers every report with 200 and an HTML
		// document. The browser SDK never throws, which means the misconfiguration
		// is invisible: the dashboard reports diligently into nowhere and Journal
		// shows nothing. Refusing at boot is the only place it is loud.
		return fmt.Errorf("env: JOURNAL_BROWSER_URL must end in /api (got %q), otherwise Journal's SPA catch-all answers every report with 200 and HTML and the reports are lost in silence", c.JournalBrowserURL)
	}
	return nil
}
