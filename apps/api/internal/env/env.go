// Package env loads the Antenne API configuration from the process environment
// once, at startup, so a misconfigured service refuses to boot instead of
// running half-configured.
package env

import (
	"fmt"
	"time"

	troncenv "github.com/FacileStudio/tronc/env"
)

// DefaultPort is the port Antenne binds when PORT is unset. It is the port the
// published webhook URLs and the pool clients already point at, so it does not
// follow troncenv.Core's 8080 default.
const DefaultPort = 9090

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
	SessionSecret string
	PoolSecret    string
	ClientDir     string
	Timeout       int

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
		SessionSecret: troncenv.String("ANTENNE_SESSION_SECRET", ""),
		PoolSecret:    troncenv.String("ANTENNE_SECRET", ""),
		ClientDir:     troncenv.String("CLIENT_DIR", "./client"),
		Timeout:       timeout,

		HeartbeatURL:      troncenv.String("HEARTBEAT_URL", ""),
		HeartbeatInterval: heartbeatInterval,
	}

	return cfg, cfg.validate()
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

func (c Config) validate() error {
	if c.IsProduction() && c.AdminPassword == "" {
		return fmt.Errorf("env: APP_ENV=production requires ANTENNE_ADMIN_PASSWORD, otherwise the dashboard and its API are open to anyone who can reach this server")
	}
	if c.IsProduction() && c.SessionSecret == "" {
		return fmt.Errorf("env: APP_ENV=production requires ANTENNE_SESSION_SECRET, otherwise every session dies on restart")
	}
	return nil
}
